package src

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"syscall"

	"github.com/amirdaaee/TGMon/internal/db/minio"
	ftypes "github.com/amirdaaee/TGMon/internal/facade/types"
	"github.com/amirdaaee/TGMon/internal/filesystem/cache"
	"github.com/amirdaaee/TGMon/internal/log"
	"github.com/amirdaaee/TGMon/internal/types"
	"github.com/chenmingyong0423/go-mongox/v2/bsonx"
	"github.com/chenmingyong0423/go-mongox/v2/builder/update"
	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
	mnio "github.com/minio/minio-go/v7"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.uber.org/zap"
)

// SrtFileSrc is a source implementation that provides access to SRT subtitle files
// stored in MinIO and referenced in the database.
type SrtFileSrc struct {
	facade      ftypes.IFacade[types.MediaFileDoc]
	minioClient minio.IMinioClient
	cache       *cache.DBCache[string, *types.MediaFileDoc]
}

var _ ISrc = (*SrtFileSrc)(nil)

// List returns all SRT files from the database that have an associated SRT file.
func (mfs *SrtFileSrc) List(ctx context.Context) ([]IFile, error) {
	docs, err := mfs.cache.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list media files from cache: %w", err)
	}
	files := make([]IFile, 0, len(docs))
	for _, doc := range docs {
		if doc.Srt == "" {
			continue
		}
		files = append(files, &SrtFile{orgMedia: doc, minioClient: mfs.minioClient})
	}
	return files, nil
}

// Lookup finds an SRT file by its UID.
// The UID should be in the format "srt-{mediaFileID}".
func (mfs *SrtFileSrc) Lookup(ctx context.Context, uid string) (IFile, error) {
	doc, err := mfs.cache.Find(ctx, strings.TrimPrefix(uid, "srt-"))
	if err != nil {
		return nil, fmt.Errorf("failed to find media file from cache: %w", err)
	}
	if doc.Srt == "" {
		return nil, fmt.Errorf("media doesn't have srt")
	}
	return &SrtFile{orgMedia: doc, minioClient: mfs.minioClient}, nil
}

// UID returns the unique identifier for this source type.
func (mfs *SrtFileSrc) UID() string {
	return "SRT"
}

// Delete removes the SRT file reference from the database by clearing the SRT field. cache get invalidated in facade machinary
func (mfs *SrtFileSrc) Delete(ctx context.Context, uid string) error {
	qID, err := mfs.getIdQ(uid)
	if err != nil {
		return fmt.Errorf("failed to get id query: %w", err)
	}
	if _, err := mfs.facade.GetCollection().Updater().Filter(qID).Updates([]bson.D{update.Set(types.MediaFileDoc__SrtField, "")}).UpdateOne(ctx); err != nil {
		return fmt.Errorf("failed to delete srt file refference in db: %w", err)
	}
	mfs.cache.Invalidate(ctx)
	return nil
}

// getIdQ converts a UID string to a MongoDB query filter.
// The UID should be in the format "srt-{mediaFileID}".
func (mfs *SrtFileSrc) getIdQ(uid string) (bson.M, error) {
	oid, err := bson.ObjectIDFromHex(strings.TrimPrefix(uid, "srt-"))
	if err != nil {
		return nil, fmt.Errorf("failed to convert uid to object id: %w", err)
	}
	return bsonx.Id(oid), nil
}

// NewSrtSrc creates a new SrtFileSrc instance.
func NewSrtSrc(facade ftypes.IFacade[types.MediaFileDoc], minioClient minio.IMinioClient, cache *cache.DBCache[string, *types.MediaFileDoc]) *SrtFileSrc {
	return &SrtFileSrc{
		facade:      facade,
		minioClient: minioClient,
		cache:       cache,
	}
}

// ===
//
// SrtFile represents a single SRT subtitle file in the filesystem.
type SrtFile struct {
	fs.Inode
	orgMedia    *types.MediaFileDoc
	minioClient minio.IMinioClient
	_info       *mnio.ObjectInfo
	mu          sync.Mutex
}

var _ IFile = (*SrtFile)(nil)

// Getattr returns file attributes
func (mf *SrtFile) Getattr(ctx context.Context, f fs.FileHandle, out *fuse.AttrOut) syscall.Errno {
	ll := mf.getLogger("Getattr")
	info, err := mf.info(ctx)
	if err != nil {
		ll.With(zap.Error(err)).Error("failed to get file info")
		return syscall.EIO
	}
	out.Mode = fuse.S_IFREG | 0444
	out.Size = uint64(info.Size)
	out.Mtime = uint64(mf.orgMedia.CreatedAt.Unix())
	out.Atime = uint64(info.LastModified.Unix())
	out.Ctime = uint64(mf.orgMedia.CreatedAt.Unix())
	return 0
}

// Open opens the file for reading
func (mf *SrtFile) Open(ctx context.Context, flags uint32) (fs.FileHandle, uint32, syscall.Errno) {
	ll := mf.getLogger("Open")
	ll.Sugar().Debugf("Opening srt file: %s (flags: %d)", mf.orgMedia.ID.Hex(), flags)
	// Only allow read operations
	if flags&fuse.O_ANYWRITE != 0 {
		return nil, 0, syscall.EACCES
	}

	obj, err := mf.minioClient.FileGet(ctx, mf.orgMedia.Srt)
	if err != nil {
		ll.With(zap.Error(err)).Error("failed to get file object")
		return nil, 0, syscall.EIO
	}
	info, err := mf.info(ctx)
	if err != nil {
		ll.With(zap.Error(err)).Error("failed to get file info")
		return nil, 0, syscall.EIO
	}

	// Create a cancelable context for this file handle
	// This context will be canceled when the file is closed
	fileCtx, cancel := context.WithCancel(ctx)

	fileHandle := &SrtFileHandle{
		orgMedia: mf.orgMedia,
		obj:      obj,
		info:     info,
		ctx:      fileCtx,
		cancel:   cancel,
	}

	return fileHandle, fuse.FOPEN_KEEP_CACHE, 0
}

// Name returns the display name of the SRT file.
func (mf *SrtFile) Name() string {
	return mf.orgMedia.UName
}

// UID returns the unique identifier of the SRT file in the format "srt-{mediaFileID}".
func (mf *SrtFile) UID() string {
	return fmt.Sprintf("srt-%s", mf.orgMedia.ID.Hex())
}

// Size returns the size of the SRT file in bytes.
func (mf *SrtFile) Size() uint64 {
	info, err := mf.info(context.Background())
	if err != nil {
		return 0
	}
	return uint64(info.Size)
}

// Mtime returns the modification time of the SRT file as a Unix timestamp.
func (mf *SrtFile) Mtime() uint64 {
	return uint64(mf.orgMedia.CreatedAt.Unix())
}

// Atime returns the access time of the SRT file as a Unix timestamp.
func (mf *SrtFile) Atime() uint64 {
	info, err := mf.info(context.Background())
	if err != nil {
		return 0
	}
	return uint64(info.LastModified.Unix())
}

// Ctime returns the creation time of the SRT file as a Unix timestamp.
func (mf *SrtFile) Ctime() uint64 {
	return uint64(mf.orgMedia.CreatedAt.Unix())
}

// Ext returns the file extension, which is always ".srt" for SRT files.
func (mf *SrtFile) Ext() string {
	return ".srt"
}

// info retrieves the object information from MinIO, caching it for subsequent calls.
func (mf *SrtFile) info(ctx context.Context) (*mnio.ObjectInfo, error) {
	mf.mu.Lock()
	defer mf.mu.Unlock()
	if mf._info == nil {
		info, err := mf.minioClient.FileInfo(ctx, mf.orgMedia.Srt)
		if err != nil {
			return nil, fmt.Errorf("failed to get file info: %w", err)
		}
		mf._info = info
	}
	return mf._info, nil
}
func (mf *SrtFile) getLogger(fn string) *zap.Logger {
	return log.Named(log.FuseModule, fmt.Sprintf("%T.%s", mf, fn))
}

// ===

// SrtFileHandle handles read operations on an SRT file.
type SrtFileHandle struct {
	orgMedia *types.MediaFileDoc
	obj      *mnio.Object
	info     *mnio.ObjectInfo
	ctx      context.Context
	cancel   context.CancelFunc
}

var _ fs.FileReader = (*SrtFileHandle)(nil)
var _ fs.FileReleaser = (*SrtFileHandle)(nil)

// Read reads data from the file at the specified offset
func (mfh *SrtFileHandle) Read(ctx context.Context, dest []byte, off int64) (fuse.ReadResult, syscall.Errno) {
	ll := mfh.getLogger("Read")
	ll.Sugar().Debugf("Read request: offset=%d, size=%d, fileSize=%d", off, len(dest), mfh.info.Size)
	// Check if offset is beyond file size
	if off >= mfh.info.Size {
		ll.Debug("EOF: offset beyond file size")
		return fuse.ReadResultData(nil), 0
	}

	if _, err := mfh.obj.Seek(off, io.SeekStart); err != nil {
		ll.With(zap.Error(err)).Error("failed to seek file data")
		return nil, syscall.EIO
	}

	if _, err := mfh.obj.Read(dest); err != nil && err != io.EOF {
		ll.With(zap.Error(err)).Error("failed to read file data")
		return nil, syscall.EIO
	}
	return fuse.ReadResultData(dest), 0
}

// Release is called when the file is closed/released
func (mfh *SrtFileHandle) Release(ctx context.Context) syscall.Errno {
	ll := mfh.getLogger("Release")
	ll.Debug("File handle released, canceling context")
	if mfh.cancel != nil {
		mfh.cancel()
	}
	return 0
}

func (mfh *SrtFileHandle) getLogger(fn string) *zap.Logger {
	return log.Named(log.FuseModule, fmt.Sprintf("%T.%s", mfh, fn))
}
