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
	"github.com/amirdaaee/TGMon/internal/log"
	"github.com/amirdaaee/TGMon/internal/types"
	"github.com/chenmingyong0423/go-mongox/v2/builder/query"
	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
	mnio "github.com/minio/minio-go/v7"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type SrtFileSrc struct {
	facade      ftypes.IFacade[types.MediaFileDoc]
	minioClient minio.IMinioClient
}

var _ ISrc = (*SrtFileSrc)(nil)

func (mfs *SrtFileSrc) List(ctx context.Context) ([]IFile, error) {
	q := query.NewBuilder().Nor(query.Eq(types.MediaFileDoc__SrtField, ""), query.Exists(types.MediaFileDoc__SrtField, false)).Build()
	docs, err := mfs.facade.GetCollection().Finder().Filter(q).Sort(bson.D{{Key: "_id", Value: 1}}).Find(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list media files from db: %w", err)
	}
	files := make([]IFile, 0, len(docs))
	for _, doc := range docs {
		files = append(files, &SrtFile{orgMedia: doc, minioClient: mfs.minioClient})
	}
	return files, nil
}
func (mfs *SrtFileSrc) Lookup(ctx context.Context, uid string) (IFile, error) {
	oid, err := bson.ObjectIDFromHex(strings.TrimPrefix(uid, "srt-"))
	if err != nil {
		return nil, fmt.Errorf("failed to convert uid to object id: %w", err)
	}
	doc, err := mfs.facade.GetCollection().Finder().Filter(bson.D{{Key: "_id", Value: oid}}).FindOne(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup media file from db: %w", err)
	}
	return &SrtFile{orgMedia: doc, minioClient: mfs.minioClient}, nil
}

func NewSrtSrc(facade ftypes.IFacade[types.MediaFileDoc], minioClient minio.IMinioClient) *SrtFileSrc {
	return &SrtFileSrc{
		facade:      facade,
		minioClient: minioClient,
	}
}

// ===
//
// SrtFile represents a single srt file in the filesystem
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
		ll.WithError(err).Error("failed to get file info")
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
	ll.Debugf("Opening srt file: %s (flags: %d)", mf.orgMedia.ID.Hex(), flags)
	// Only allow read operations
	if flags&fuse.O_ANYWRITE != 0 {
		return nil, 0, syscall.EACCES
	}

	obj, err := mf.minioClient.FileGet(ctx, mf.orgMedia.Srt)
	if err != nil {
		ll.WithError(err).Error("failed to get file object")
		return nil, 0, syscall.EIO
	}
	info, err := mf.info(ctx)
	if err != nil {
		ll.WithError(err).Error("failed to get file info")
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

func (mf *SrtFile) Name() string {
	return mf.orgMedia.UName
}
func (mf *SrtFile) UID() string {
	return fmt.Sprintf("srt-%s", mf.orgMedia.ID.Hex())
}

func (mf *SrtFile) Size() uint64 {
	info, err := mf.info(context.Background())
	if err != nil {
		return 0
	}
	return uint64(info.Size)
}
func (mf *SrtFile) Mtime() uint64 {
	return uint64(mf.orgMedia.CreatedAt.Unix())
}
func (mf *SrtFile) Atime() uint64 {
	info, err := mf.info(context.Background())
	if err != nil {
		return 0
	}
	return uint64(info.LastModified.Unix())
}
func (mf *SrtFile) Ctime() uint64 {
	return uint64(mf.orgMedia.CreatedAt.Unix())
}
func (mf *SrtFile) Ext() string {
	return ".srt"
}
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
func (mf *SrtFile) getLogger(fn string) *logrus.Entry {
	return log.GetLogger(log.FuseModule).WithField("func", fmt.Sprintf("%T.%s", mf, fn))
}

// ===

// MediaFileHandle handles read operations on a media file
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
	ll.Debugf("Read request: offset=%d, size=%d, fileSize=%d", off, len(dest), mfh.info.Size)
	// Check if offset is beyond file size
	if off >= mfh.info.Size {
		ll.Debug("EOF: offset beyond file size")
		return fuse.ReadResultData(nil), 0
	}

	if _, err := mfh.obj.Seek(off, io.SeekStart); err != nil {
		ll.WithError(err).Error("failed to seek file data")
		return nil, syscall.EIO
	}

	if _, err := mfh.obj.Read(dest); err != nil && err != io.EOF {
		ll.WithError(err).Error("failed to read file data")
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

func (mfh *SrtFileHandle) getLogger(fn string) *logrus.Entry {
	return log.GetLogger(log.FuseModule).WithField("func", fmt.Sprintf("%T.%s", mfh, fn))
}
