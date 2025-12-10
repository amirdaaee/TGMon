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

type uidMapEntryType struct {
	data  *types.FuseStateDoc
	inode uint64
	mu    sync.RWMutex
}

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
func (e *uidMapEntryType) IncrementNameSuffix() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.data.NameSuffix++
}

// ...
type uidMapType struct {
	mappedUID       map[string]*uidMapEntryType // uid -> entry
	inodeNumCounter uint64
	seenNames       map[string]bool
	dbColl          mongo.ICollection[types.FuseStateDoc]
	mu              sync.RWMutex
}

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
func (r *uidMapType) Get(uid string, src string) (*uidMapEntryType, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	key := r.getKey(uid, src)
	v, ok := r.mappedUID[key]
	return v, ok
}

func (r *uidMapType) Exists(uid string, src string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	key := r.getKey(uid, src)
	_, ok := r.mappedUID[key]
	return ok
}
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

// fetchs latest state from db. this is intended to be called at init
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
func (r *uidMapType) getFreeInodeNum() uint64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.inodeNumCounter++
	return r.inodeNumCounter
}

func (r *uidMapType) getKey(uid string, src string) string {
	return fmt.Sprintf("%s-%s", uid, src)
}
func (r *uidMapType) getLogger(at string) *logrus.Entry {
	return log.GetLogger(log.FuseModule).WithField("at", fmt.Sprintf("%T.%s", r, at))
}
