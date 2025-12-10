package src

import (
	"context"
	"fmt"
	"io"
	"syscall"

	ftypes "github.com/amirdaaee/TGMon/internal/facade/types"
	"github.com/amirdaaee/TGMon/internal/log"
	"github.com/amirdaaee/TGMon/internal/stream"
	"github.com/amirdaaee/TGMon/internal/types"
	"github.com/chenmingyong0423/go-mongox/v2/bsonx"
	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// MediaFileSrc is a data source implementation that provides media files
// from the database. It uses a facade to access media file documents and
// a stream worker pool for reading file data.
type MediaFileSrc struct {
	facade           ftypes.IFacade[types.MediaFileDoc]
	streamWorkerPool stream.IWorkerPool
}

var _ ISrc = (*MediaFileSrc)(nil)

// List retrieves all media files from the database and returns them as IFile instances.
func (mfs *MediaFileSrc) List(ctx context.Context) ([]IFile, error) {
	docs, err := mfs.facade.GetCollection().Finder().Sort(bson.D{{Key: "_id", Value: 1}}).Find(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list media files from db: %w", err)
	}
	files := make([]IFile, 0, len(docs))
	for _, doc := range docs {
		files = append(files, &MediaFile{media: doc, streamWorkerPool: mfs.streamWorkerPool})
	}
	return files, nil
}

// Lookup finds a media file by its UID.
func (mfs *MediaFileSrc) Lookup(ctx context.Context, uid string) (IFile, error) {
	qID, err := mfs.getIdQ(uid)
	if err != nil {
		return nil, fmt.Errorf("failed to get id query: %w", err)
	}
	doc, err := mfs.facade.GetCollection().Finder().Filter(qID).FindOne(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to find media file from db: %w", err)
	}
	return &MediaFile{media: doc, streamWorkerPool: mfs.streamWorkerPool}, nil
}

// UID returns the unique identifier for this source type.
func (mfs *MediaFileSrc) UID() string {
	return "MEDIA"
}

// Delete removes a media file from the database.
func (mfs *MediaFileSrc) Delete(ctx context.Context, uid string) error {
	qID, err := mfs.getIdQ(uid)
	if err != nil {
		return fmt.Errorf("failed to get id query: %w", err)
	}
	if _, err := mfs.facade.GetCollection().Deleter().Filter(qID).DeleteOne(ctx); err != nil {
		return fmt.Errorf("failed to delete media file from db: %w", err)
	}
	return nil
}

// Exists checks if a media file exists in the database.
func (mfs *MediaFileSrc) Exists(ctx context.Context, uid string) (bool, error) {
	qID, err := mfs.getIdQ(uid)
	if err != nil {
		return false, fmt.Errorf("failed to get id query: %w", err)
	}
	n, err := mfs.facade.GetCollection().Finder().Filter(qID).Count(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to check if media file exists: %w", err)
	}
	return n > 0, nil
}

// getIdQ converts a UID string to a MongoDB query filter.
func (mfs *MediaFileSrc) getIdQ(uid string) (bson.M, error) {
	oid, err := bson.ObjectIDFromHex(uid)
	if err != nil {
		return nil, fmt.Errorf("failed to convert uid to object id: %w", err)
	}
	return bsonx.Id(oid), nil
}

// NewMediaFileSrc creates a new MediaFileSrc instance.
func NewMediaFileSrc(facade ftypes.IFacade[types.MediaFileDoc], streamWorkerPool stream.IWorkerPool) *MediaFileSrc {
	return &MediaFileSrc{
		facade:           facade,
		streamWorkerPool: streamWorkerPool,
	}
}

// ===
//
// MediaFile represents a single media file in the filesystem.
type MediaFile struct {
	fs.Inode
	media            *types.MediaFileDoc
	streamWorkerPool stream.IWorkerPool
}

var _ IFile = (*MediaFile)(nil)

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

// Name returns the display name of the media file.
func (mf *MediaFile) Name() string {
	return mf.media.UName
}

// UID returns the unique identifier of the media file.
func (mf *MediaFile) UID() string {
	return mf.media.ID.Hex()
}

// Size returns the size of the media file in bytes.
func (mf *MediaFile) Size() uint64 {
	return uint64(mf.media.Meta.FileSize)
}

// Mtime returns the modification time of the media file as a Unix timestamp.
func (mf *MediaFile) Mtime() uint64 {
	return uint64(mf.media.CreatedAt.Unix())
}

// Atime returns the access time of the media file as a Unix timestamp.
func (mf *MediaFile) Atime() uint64 {
	return uint64(mf.media.UpdatedAt.Unix())
}

// Ctime returns the creation time of the media file as a Unix timestamp.
func (mf *MediaFile) Ctime() uint64 {
	return uint64(mf.media.CreatedAt.Unix())
}

// Ext returns the file extension with the '.' prefix.
func (mf *MediaFile) Ext() string {
	return types.GetExtensionFromMimeType(mf.media.Meta.MimeType)
}
func (mf *MediaFile) getLogger(fn string) *logrus.Entry {
	return log.GetLogger(log.FuseModule).WithField("func", fmt.Sprintf("%T.%s", mf, fn))
}

// ===

// MediaFileHandle handles read operations on a media file.
type MediaFileHandle struct {
	media            *types.MediaFileDoc
	streamWorkerPool stream.IWorkerPool
	ctx              context.Context
	cancel           context.CancelFunc
}

var _ fs.FileReader = (*MediaFileHandle)(nil)
var _ fs.FileReleaser = (*MediaFileHandle)(nil)

// Read reads data from the file at the specified offset
func (mfh *MediaFileHandle) Read(ctx context.Context, dest []byte, off int64) (fuse.ReadResult, syscall.Errno) {
	ll := mfh.getLogger("Read")
	ll.Debugf("Read request: offset=%d, size=%d, fileSize=%d", off, len(dest), mfh.media.Meta.FileSize)

	// Check if offset is beyond file size
	if off >= mfh.media.Meta.FileSize {
		ll.Debug("EOF: offset beyond file size")
		return fuse.ReadResultData(nil), 0
	}

	// Calculate how much to read
	toRead := int64(len(dest))
	if off+toRead > mfh.media.Meta.FileSize {
		toRead = mfh.media.Meta.FileSize - off
	}

	// Create a new streamer for this read operation with the correct offset
	// This allows seeking to any position in the file
	end := off + toRead - 1
	if end >= mfh.media.Meta.FileSize {
		end = mfh.media.Meta.FileSize - 1
	}

	// Create a context that is canceled when either:
	// 1. The request context (ctx) is canceled (user interrupt, FUSE connection close)
	// 2. The file handle context (mfh.ctx) is canceled (file is closed)
	// This ensures stream operations are canceled in both cases.
	// The goroutine exits when any of the contexts are canceled, preventing leaks.
	streamCtx, streamCancel := context.WithCancel(ctx)
	go func() {
		select {
		case <-mfh.ctx.Done():
			// File handle was closed, cancel the stream operation
			streamCancel()
		case <-ctx.Done():
			// Request was canceled, streamCancel will be called by defer
			// This case ensures the goroutine doesn't block if ctx is canceled
		case <-streamCtx.Done():
			// Stream context already canceled (e.g., Read returned normally)
		}
	}()
	defer streamCancel() // Ensure stream context is canceled when Read returns

	// Use the combined context for stream operations
	streamer, err := stream.NewStreamer(streamCtx, mfh.streamWorkerPool, mfh.media.MessageID, off, end)
	if err != nil {
		ll.WithError(err).Error("Failed to create streamer")
		return fuse.ReadResultData(nil), syscall.EIO
	}

	// Read the data
	data := make([]byte, toRead)
	totalRead := int64(0)
	for totalRead < toRead {
		n, err := streamer.Read(data[totalRead:])
		if err != nil && err != io.EOF {
			ll.WithError(err).Error("Failed to read from streamer")
			return nil, syscall.EIO
		}
		if n == 0 {
			break
		}
		totalRead += int64(n)
		if err == io.EOF {
			break
		}
	}

	// Trim to actual read size
	if totalRead < toRead {
		data = data[:totalRead]
	}
	ll.Debugf("Read %d bytes", totalRead)

	return fuse.ReadResultData(data), 0
}

// Release is called when the file is closed/released
func (mfh *MediaFileHandle) Release(ctx context.Context) syscall.Errno {
	ll := mfh.getLogger("Release")
	ll.Debug("File handle released, canceling context")
	if mfh.cancel != nil {
		mfh.cancel()
	}
	return 0
}

func (mfh *MediaFileHandle) getLogger(fn string) *logrus.Entry {
	return log.GetLogger(log.FuseModule).WithField("func", fmt.Sprintf("%T.%s", mfh, fn))
}
