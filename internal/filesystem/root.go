package filesystem

import (
	"context"
	"fmt"
	"sort"
	"syscall"
	"time"

	"github.com/amirdaaee/TGMon/internal/db"
	"github.com/amirdaaee/TGMon/internal/filesystem/src"
	"github.com/amirdaaee/TGMon/internal/log"
	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
	"github.com/sirupsen/logrus"
)

type RootFS struct {
	fs.Inode
	uidMap uidMapType
	srcs   map[string]src.ISrc
}

var _ fs.NodeOnAdder = (*RootFS)(nil)
var _ fs.NodeReaddirer = (*RootFS)(nil)
var _ fs.NodeLookuper = (*RootFS)(nil)
var _ fs.NodeGetattrer = (*RootFS)(nil)
var _ fs.NodeOpendirer = (*RootFS)(nil)
var _ fs.NodeRenamer = (*RootFS)(nil)

// var _ fs.NodeUnlinker = (*RootFS)(nil)

// OnAdd is called when the filesystem is mounted
func (mfs *RootFS) OnAdd(ctx context.Context) {
	mfs.getLogger("OnAdd").Info("MediaFS mounted")
}

// Opendir opens a directory for reading
func (mfs *RootFS) Opendir(ctx context.Context) syscall.Errno {
	mfs.getLogger("Opendir").Debug("Opening directory")
	return 0
}

// Getattr returns directory attributes for the root directory
func (mfs *RootFS) Getattr(ctx context.Context, f fs.FileHandle, out *fuse.AttrOut) syscall.Errno {
	// Use 0755 permissions to allow directory traversal
	// When allow-other is enabled, the mount point itself will have 0777
	out.Mode = fuse.S_IFDIR | 0755 // Directory, read and execute permissions
	out.Nlink = 2                  // Standard for directories (., ..)
	out.Size = 4096                // Typical directory size
	now := uint64(time.Now().Unix())
	out.Mtime = now
	out.Atime = now
	out.Ctime = now
	return 0
}

// Readdir lists all media files in the root directory
func (mfs *RootFS) Readdir(ctx context.Context) (fs.DirStream, syscall.Errno) {
	ll := mfs.getLogger("Readdir")
	ll.Debug("Reading directory")

	// Create a context with timeout for database operations
	// Increased timeout for large directories (1000+ files)
	// This prevents the filesystem from hanging if the database is slow
	queryCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Get all media files from database
	entries, err := mfs.listFiles(queryCtx)
	if err != nil {
		// Check if error is due to context cancellation or timeout
		if ctx.Err() != nil || queryCtx.Err() != nil {
			ll.WithError(err).Debug("Context canceled or timed out during getMediaFiles")
		} else {
			ll.WithError(err).Error("Failed to get media files, returning empty directory")
		}
		// Return empty directory instead of error to prevent I/O errors
		// This is safer for container access - they can retry later
		return fs.NewListDirStream([]fuse.DirEntry{}), 0
	}

	// Sort entries by filename for deterministic ordering
	// This is important for proper directory scanning behavior
	// Using sort.Slice is efficient even for large directories (1000+ files)
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name < entries[j].Name
	})

	ll.Debugf("Returning %d entries", len(entries))
	return fs.NewListDirStream(entries), 0
}

// Lookup finds a file by name and returns a file node
func (mfs *RootFS) Lookup(ctx context.Context, name string, out *fuse.EntryOut) (*fs.Inode, syscall.Errno) {
	ll := mfs.getLogger("Lookup")
	ll.Debugf("Looking up file: %s", name)

	entry, ok := mfs.uidMap.GetByName(name)
	if !ok {
		return nil, syscall.ENOENT // TODO
	}
	// ...
	qContext, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	file, err := mfs.srcs[entry.data.SrcID].Lookup(qContext, entry.data.UID)
	if err != nil {
		if err == context.Canceled || err == context.DeadlineExceeded {
			ll.Warn("Context canceled or timed out during src Lookup")
			return nil, syscall.EINTR // TODO
		}
		ll.WithError(err).Error("failed to lookup file")
		return nil, syscall.EIO // TODO
	}

	// Set entry attributes
	out.Mode = fuse.S_IFREG | 0444
	out.Size = file.Size()
	out.Mtime = file.Mtime()
	out.Atime = file.Atime()
	out.Ctime = file.Ctime()

	stable := fs.StableAttr{
		Mode: fuse.S_IFREG,
		Ino:  entry.inode,
	}
	ll.Debugf("Found file: %s (size: %d)", name, file.Size())
	return mfs.NewInode(ctx, file, stable), 0
}

// Lookup finds a file by name and returns a file node
func (mfs *RootFS) Rename(ctx context.Context, name string, newParent fs.InodeEmbedder, newName string, flags uint32) syscall.Errno {
	ll := mfs.getLogger("Rename")
	ll.Debugf("Renaming file: %s to %s", name, newName)

	_, ok := mfs.uidMap.GetByName(name)
	if !ok {
		return syscall.ENOENT // TODO
	}
	// don't allow renaming to a different parent
	if newParent.EmbeddedInode().Path(nil) != mfs.EmbeddedInode().Path(nil) {
		return syscall.EPERM // TODO
	}

	return 0
}

// func (mfs *RootFS) Unlink(ctx context.Context, name string) syscall.Errno {
// 	mfs.cacheMutex.Lock()
// 	defer mfs.cacheMutex.Unlock()
// 	ll := mfs.getLogger("Unlink")
// 	uname := types.RemoveExtension(name)
// 	ll.Infof("removing media file: %s", uname)
// 	filter := bsonx.NewD().Add(types.MediaFileDoc__UnameField, uname).Build()
// 	// ...
// 	if _, err := mfs.mediaFacade.DeleteOne(ctx, filter); err != nil {
// 		ll.WithError(err).Error("failed to delete media file")
// 		return syscall.EIO
// 	}
// 	ll.Debugf("deleted media file: %s", uname)
// 	// ...
// 	delete(mfs.mediaCache, name)
// 	// mfs.RmChild(name)
// 	// ...
// 	return 0
// }

// listFiles retrieves all files from all data sources
func (mfs *RootFS) listFiles(ctx context.Context) ([]fuse.DirEntry, error) {
	ll := mfs.getLogger("listFiles")
	entries := make([]fuse.DirEntry, 0)
	for _, src := range mfs.srcs {
		llSrc := ll.WithField("src", src.UID())
		srcEntries, err := src.List(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list files from src %s: %w", src.UID(), err)
		}
		for _, _f := range srcEntries {
			llEntry := llSrc.WithField("file", _f.Name()).WithField("uid", _f.UID())
			if !mfs.uidMap.Exists(_f.UID(), src.UID()) {
				if err := mfs.uidMap.Add(ctx, _f.UID(), src.UID(), _f.Name(), _f.Ext()); err != nil {
					llEntry.WithError(err).Error("failed to add entry to uid map. skipping file.")
					continue
				}
			}
			entr, _ := mfs.uidMap.Get(_f.UID(), src.UID())
			entries = append(entries, fuse.DirEntry{
				Name: entr.Name(),
				Mode: fuse.S_IFREG | 0444,
				Ino:  entr.inode,
			})
		}
	}
	return entries, nil
}

func (mfs *RootFS) getLogger(fn string) *logrus.Entry {
	return log.GetLogger(log.FuseModule).WithField("func", fmt.Sprintf("%T.%s", mfs, fn))
}

// ...

// NewMediaFS creates a new MediaFS filesystem
func NewMediaFS(srcs []src.ISrc, dbContainer db.IDbContainer) *RootFS {
	v := RootFS{
		uidMap: uidMapType{
			mappedUID: make(map[string]*uidMapEntryType),
			seenNames: make(map[string]bool),
			dbColl:    dbContainer.GetMongoContainer().GetFuseRenameCollection(),
		},
		srcs: make(map[string]src.ISrc),
	}
	for _, src := range srcs {
		v.srcs[src.UID()] = src
	}
	return &v
}
