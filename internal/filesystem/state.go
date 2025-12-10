package filesystem

import (
	"context"
	"fmt"
	"sync"

	"github.com/amirdaaee/TGMon/internal/db/mongo"
	"github.com/amirdaaee/TGMon/internal/log"
	"github.com/amirdaaee/TGMon/internal/types"
	"github.com/chenmingyong0423/go-mongox/v2/bsonx"
	"github.com/chenmingyong0423/go-mongox/v2/builder/update"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// uidMapEntryType represents a single entry in the UID map.
// It stores file metadata, inode number, and provides thread-safe access
// to the entry's name and properties.
type uidMapEntryType struct {
	data  *types.FuseStateDoc
	inode uint64
	mu    sync.RWMutex
}

// Name returns the display name for the entry.
// It returns the renamed name if set, otherwise constructs the name
// from the base name, optional suffix, and extension, sanitized for filesystem use.
func (e *uidMapEntryType) Name() string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if e.data.Rename != "" {
		return e.data.Rename
	}
	v := e.data.Name
	if e.data.NameSuffix > 0 {
		v = fmt.Sprintf("%s-%d", v, e.data.NameSuffix)
	}
	return sanitizeFilename(v) + e.data.Ext
}

// IncrementNameSuffix increments the name suffix counter.
// This is used to handle filename conflicts by appending a numeric suffix.
func (e *uidMapEntryType) IncrementNameSuffix() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.data.NameSuffix++
}

// uidMapType maintains a mapping between unique identifiers (UID + source ID)
// and filesystem entries. It handles name conflicts, inode assignment,
// and persistence to the database. All operations are thread-safe.
type uidMapType struct {
	mappedUID       map[string]*uidMapEntryType // uid -> entry
	inodeNumCounter uint64
	seenNames       map[string]bool
	dbColl          mongo.ICollection[types.FuseStateDoc]
	mu              sync.RWMutex
}

// Add adds a new entry to the UID map.
// It automatically handles name conflicts by appending a numeric suffix
// and persists the entry to the database.
func (r *uidMapType) Add(ctx context.Context, uid string, src string, name, ext string) error {
	key := r.getKey(uid, src)
	if r.Exists(uid, src) {
		return fmt.Errorf("uid %s already exists in src %s", uid, src)
	}
	inode := r.getFreeInodeNum()
	r.mu.Lock()

	data := types.FuseStateDoc{
		SrcID:      src,
		UID:        uid,
		Name:       name,
		NameSuffix: 0,
		Ext:        ext,
	}
	entry := uidMapEntryType{
		data:  &data,
		inode: inode,
	}
	r.mappedUID[key] = &entry
	for {
		mappedName := entry.Name()
		if _, ok := r.seenNames[mappedName]; !ok {
			r.seenNames[mappedName] = true
			break
		}
		entry.IncrementNameSuffix()
	}
	r.mu.Unlock()
	if _, err := r.dbColl.Creator().InsertOne(ctx, entry.data); err != nil {
		r.mu.Lock()
		delete(r.mappedUID, key)
		r.mu.Unlock()
		return fmt.Errorf("failed to insert doc into db: %w", err)
	}
	return nil
}

// Get retrieves an entry from the UID map by UID and source ID.
// Returns the entry and true if found, nil and false otherwise.
func (r *uidMapType) Get(uid string, src string) (*uidMapEntryType, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	key := r.getKey(uid, src)
	v, ok := r.mappedUID[key]
	return v, ok
}

// Exists checks if an entry exists in the UID map for the given UID and source ID.
func (r *uidMapType) Exists(uid string, src string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	key := r.getKey(uid, src)
	_, ok := r.mappedUID[key]
	return ok
}

// GetByName retrieves an entry from the UID map by its display name.
// Returns the entry and true if found, nil and false otherwise.
func (r *uidMapType) GetByName(name string) (*uidMapEntryType, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, v := range r.mappedUID {
		if v.Name() == name {
			return v, true
		}
	}
	return nil, false
}

// DeleteByName removes an entry from the UID map by its display name.
// It also removes the entry from the database and the seen names cache.
func (r *uidMapType) DeleteByName(ctx context.Context, name string) error {
	entry, ok := r.GetByName(name)
	if !ok {
		return fmt.Errorf("entry not found: %s", name)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, err := r.dbColl.Deleter().Filter(bsonx.Id(entry.data.ID)).DeleteOne(ctx); err != nil {
		return fmt.Errorf("failed to delete doc from db: %w", err)
	}
	key := r.getKey(entry.data.UID, entry.data.SrcID)
	delete(r.mappedUID, key)
	delete(r.seenNames, name)
	return nil
}

// RenameByName renames an entry in the UID map by updating its display name.
// The change is persisted to the database.
func (r *uidMapType) RenameByName(ctx context.Context, oldName string, newName string) error {
	entry, ok := r.GetByName(oldName)
	if !ok {
		return fmt.Errorf("entry not found: %s", oldName)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	entry.data.Rename = newName
	if _, err := r.dbColl.Updater().Filter(bsonx.Id(entry.data.ID)).Updates([]bson.D{update.Set(types.FuseStateDoc__RenameField, newName)}).UpdateOne(ctx); err != nil {
		return fmt.Errorf("failed to update doc in db: %w", err)
	}
	return nil
}

// SyncDB fetches the latest state from the database and rebuilds the in-memory UID map.
// This is intended to be called at initialization to restore the filesystem state.
func (r *uidMapType) SyncDB(ctx context.Context) error {
	ll := r.getLogger("SyncDB")
	allDocs, err := r.dbColl.Finder().Find(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch all docs from db: %w", err)
	}
	ll.Debugf("fetched %d docs from db", len(allDocs))
	r.mu.Lock()
	defer r.mu.Unlock()
	r.mappedUID = make(map[string]*uidMapEntryType)
	r.inodeNumCounter = uint64(len(allDocs))
	r.seenNames = make(map[string]bool)
	for c, doc := range allDocs {
		v := uidMapEntryType{
			data:  doc,
			inode: uint64(c + 1),
		}
		r.mappedUID[r.getKey(doc.UID, doc.SrcID)] = &v
		r.seenNames[v.Name()] = true
	}
	return nil
}

// getFreeInodeNum generates and returns a new inode number.
func (r *uidMapType) getFreeInodeNum() uint64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.inodeNumCounter++
	return r.inodeNumCounter
}

// getKey generates a unique key for a UID and source ID combination.
func (r *uidMapType) getKey(uid string, src string) string {
	return fmt.Sprintf("%s-%s", uid, src)
}
func (r *uidMapType) getLogger(at string) *logrus.Entry {
	return log.GetLogger(log.FuseModule).WithField("at", fmt.Sprintf("%T.%s", r, at))
}
