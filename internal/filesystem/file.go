package filesystem

import (
	"context"
	"fmt"
	"syscall"

	"github.com/amirdaaee/TGMon/internal/db"
	"github.com/amirdaaee/TGMon/internal/log"
	"github.com/amirdaaee/TGMon/internal/stream"
	"github.com/amirdaaee/TGMon/internal/types"
	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
	"github.com/sirupsen/logrus"
)

// MediaFile represents a single media file in the filesystem
type MediaFile struct {
	fs.Inode
	media            *types.MediaFileDoc
	dbContainer      db.IDbContainer
	streamWorkerPool stream.IWorkerPool
}

var _ fs.NodeOpener = (*MediaFile)(nil)
var _ fs.NodeGetattrer = (*MediaFile)(nil)

// Getattr returns file attributes
func (mf *MediaFile) Getattr(ctx context.Context, f fs.FileHandle, out *fuse.AttrOut) syscall.Errno {
	out.Mode = fuse.S_IFREG | 0444
	out.Size = uint64(mf.media.Meta.FileSize)
	out.Mtime = uint64(mf.media.CreatedAt.Unix())
	out.Atime = uint64(mf.media.UpdatedAt.Unix())
	out.Ctime = uint64(mf.media.CreatedAt.Unix())
	return 0
}

// Open opens the file for reading
func (mf *MediaFile) Open(ctx context.Context, flags uint32) (fs.FileHandle, uint32, syscall.Errno) {
	ll := mf.getLogger("Open")
	ll.Debugf("Opening file: %s (flags: %d)", mf.media.ID.Hex(), flags)

	// Only allow read operations
	if flags&fuse.O_ANYWRITE != 0 {
		return nil, 0, syscall.EACCES
	}

	// Create a cancelable context for this file handle
	// This context will be canceled when the file is closed
	fileCtx, cancel := context.WithCancel(ctx)

	fileHandle := &MediaFileHandle{
		media:            mf.media,
		streamWorkerPool: mf.streamWorkerPool,
		ctx:              fileCtx,
		cancel:           cancel,
	}

	return fileHandle, fuse.FOPEN_KEEP_CACHE, 0
}

func (mf *MediaFile) getLogger(fn string) *logrus.Entry {
	return log.GetLogger(log.WebModule).WithField("func", fmt.Sprintf("%T.%s", mf, fn))
}
