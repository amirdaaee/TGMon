package filesystem

import (
	"context"
	"fmt"
	"io"
	"sync"
	"syscall"
	"time"

	"github.com/amirdaaee/TGMon/internal/db"
	"github.com/amirdaaee/TGMon/internal/log"
	"github.com/amirdaaee/TGMon/internal/stream"
	"github.com/amirdaaee/TGMon/internal/types"
	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
	"github.com/sirupsen/logrus"
)

// MediaFS is the root filesystem node that lists all media files
type MediaFS struct {
	fs.Inode
	dbContainer     db.IDbContainer
	streamContainer stream.IWorkerContainer
	mediaCache      map[string]*types.MediaFileDoc
	cacheMutex      sync.RWMutex
	cacheExpiry     time.Time
	cacheTTL        time.Duration
}

var _ fs.NodeOnAdder = (*MediaFS)(nil)
var _ fs.NodeReaddirer = (*MediaFS)(nil)
var _ fs.NodeLookuper = (*MediaFS)(nil)

// NewMediaFS creates a new MediaFS filesystem
func NewMediaFS(dbContainer db.IDbContainer, streamContainer stream.IWorkerContainer) *MediaFS {
	return &MediaFS{
		dbContainer:     dbContainer,
		streamContainer: streamContainer,
		mediaCache:      make(map[string]*types.MediaFileDoc),
		cacheTTL:        30 * time.Second, // Cache media list for 30 seconds
	}
}

// OnAdd is called when the filesystem is mounted
func (mfs *MediaFS) OnAdd(ctx context.Context) {
	mfs.getLogger("OnAdd").Info("MediaFS mounted")
}

// Readdir lists all media files in the root directory
func (mfs *MediaFS) Readdir(ctx context.Context) (fs.DirStream, syscall.Errno) {
	ll := mfs.getLogger("Readdir")
	ll.Debug("Reading directory")

	// Get all media files from database
	mediaFiles, err := mfs.getMediaFiles(ctx)
	if err != nil {
		ll.WithError(err).Error("Failed to get media files")
		return nil, syscall.EIO
	}

	// Create directory entries
	entries := make([]fuse.DirEntry, 0, len(mediaFiles))
	for _, media := range mediaFiles {
		filename := mfs.getFilename(media)
		entries = append(entries, fuse.DirEntry{
			Name: filename,
			Mode: fuse.S_IFREG | 0444, // Regular file, read-only
		})
	}

	ll.Debugf("Returning %d entries", len(entries))
	return fs.NewListDirStream(entries), 0
}

// Lookup finds a file by name and returns a file node
func (mfs *MediaFS) Lookup(ctx context.Context, name string, out *fuse.EntryOut) (*fs.Inode, syscall.Errno) {
	ll := mfs.getLogger("Lookup")
	ll.Debugf("Looking up file: %s", name)

	// Get all media files
	mediaFiles, err := mfs.getMediaFiles(ctx)
	if err != nil {
		ll.WithError(err).Error("Failed to get media files")
		return nil, syscall.EIO
	}

	// Find the media file by name
	var media *types.MediaFileDoc
	for _, m := range mediaFiles {
		if mfs.getFilename(m) == name {
			media = m
			break
		}
	}

	if media == nil {
		ll.Debugf("File not found: %s", name)
		return nil, syscall.ENOENT
	}

	// Create file node
	fileNode := &MediaFile{
		media:           media,
		dbContainer:     mfs.dbContainer,
		streamContainer: mfs.streamContainer,
	}

	// Set entry attributes
	out.Mode = fuse.S_IFREG | 0444
	out.Size = uint64(media.Meta.FileSize)
	out.Mtime = uint64(media.CreatedAt.Unix())
	out.Atime = uint64(media.UpdatedAt.Unix())
	out.Ctime = uint64(media.CreatedAt.Unix())

	stable := fs.StableAttr{
		Mode: fuse.S_IFREG,
		Ino:  uint64(media.ID.Timestamp().Unix()),
	}

	ll.Debugf("Found file: %s (size: %d)", name, media.Meta.FileSize)
	return mfs.NewInode(ctx, fileNode, stable), 0
}

// getMediaFiles retrieves all media files from the database, with caching
func (mfs *MediaFS) getMediaFiles(ctx context.Context) ([]*types.MediaFileDoc, error) {
	mfs.cacheMutex.RLock()
	// Check if cache is still valid
	if time.Now().Before(mfs.cacheExpiry) && len(mfs.mediaCache) > 0 {
		// Return cached data
		mediaFiles := make([]*types.MediaFileDoc, 0, len(mfs.mediaCache))
		for _, media := range mfs.mediaCache {
			mediaFiles = append(mediaFiles, media)
		}
		mfs.cacheMutex.RUnlock()
		return mediaFiles, nil
	}
	mfs.cacheMutex.RUnlock()

	// Cache expired or empty, fetch from database
	mfs.cacheMutex.Lock()
	defer mfs.cacheMutex.Unlock()

	// Double-check after acquiring write lock
	if time.Now().Before(mfs.cacheExpiry) && len(mfs.mediaCache) > 0 {
		mediaFiles := make([]*types.MediaFileDoc, 0, len(mfs.mediaCache))
		for _, media := range mfs.mediaCache {
			mediaFiles = append(mediaFiles, media)
		}
		return mediaFiles, nil
	}

	// Fetch from database
	collection := mfs.dbContainer.GetMongoContainer().GetMediaFileCollection()
	mediaFiles, err := collection.Finder().Find(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to find media files: %w", err)
	}

	// Update cache
	mfs.mediaCache = make(map[string]*types.MediaFileDoc)
	for _, media := range mediaFiles {
		filename := mfs.getFilename(media)
		mfs.mediaCache[filename] = media
	}
	mfs.cacheExpiry = time.Now().Add(mfs.cacheTTL)

	return mediaFiles, nil
}

// getFilename returns the filename for a media file
func (mfs *MediaFS) getFilename(media *types.MediaFileDoc) string {
	if media.Meta.FileName != "" {
		return media.Meta.FileName
	}
	// Use ID as filename with appropriate extension based on mime type
	ext := mfs.getExtensionFromMimeType(media.Meta.MimeType)
	return fmt.Sprintf("%s%s", media.ID.Hex(), ext)
}

// getExtensionFromMimeType returns a file extension based on mime type
func (mfs *MediaFS) getExtensionFromMimeType(mimeType string) string {
	switch mimeType {
	case "video/mp4":
		return ".mp4"
	case "video/webm":
		return ".webm"
	case "video/x-matroska":
		return ".mkv"
	case "video/quicktime":
		return ".mov"
	case "audio/mpeg":
		return ".mp3"
	case "audio/ogg":
		return ".ogg"
	case "audio/webm":
		return ".weba"
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/gif":
		return ".gif"
	default:
		return ".bin"
	}
}

func (mfs *MediaFS) getLogger(fn string) *logrus.Entry {
	return log.GetLogger(log.WebModule).WithField("func", fmt.Sprintf("%T.%s", mfs, fn))
}

// MediaFile represents a single media file in the filesystem
type MediaFile struct {
	fs.Inode
	media           *types.MediaFileDoc
	dbContainer     db.IDbContainer
	streamContainer stream.IWorkerContainer
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

	fileHandle := &MediaFileHandle{
		media:           mf.media,
		streamContainer: mf.streamContainer,
	}

	return fileHandle, fuse.FOPEN_KEEP_CACHE, 0
}

func (mf *MediaFile) getLogger(fn string) *logrus.Entry {
	return log.GetLogger(log.WebModule).WithField("func", fmt.Sprintf("%T.%s", mf, fn))
}

// MediaFileHandle handles read operations on a media file
type MediaFileHandle struct {
	media           *types.MediaFileDoc
	streamContainer stream.IWorkerContainer
}

var _ fs.FileReader = (*MediaFileHandle)(nil)

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

	streamer, err := mfh.streamContainer.GetWorkerPool().Stream(ctx, mfh.media.MessageID, off, end)
	if err != nil {
		ll.WithError(err).Error("Failed to create streamer")
		return nil, syscall.EIO
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

func (mfh *MediaFileHandle) getLogger(fn string) *logrus.Entry {
	return log.GetLogger(log.WebModule).WithField("func", fmt.Sprintf("%T.%s", mfh, fn))
}

// Mount mounts the media filesystem at the specified mount point
func Mount(mountPoint string, dbContainer db.IDbContainer, streamContainer stream.IWorkerContainer) (*fuse.Server, error) {
	ll := log.GetLogger(log.WebModule).WithField("func", "Mount")
	ll.Infof("Mounting filesystem at: %s", mountPoint)

	// Create root filesystem
	root := NewMediaFS(dbContainer, streamContainer)

	// Create FUSE server
	opts := &fs.Options{}
	opts.Debug = false
	opts.AllowOther = false

	server, err := fs.Mount(mountPoint, root, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to mount filesystem: %w", err)
	}

	ll.Info("Filesystem mounted successfully")
	return server, nil
}

// Unmount unmounts the filesystem
func Unmount(mountPoint string) error {
	ll := log.GetLogger(log.WebModule).WithField("func", "Unmount")
	ll.Infof("Unmounting filesystem at: %s", mountPoint)

	// Try to unmount using fusermount
	if err := syscall.Unmount(mountPoint, 0); err != nil {
		// If that fails, try with MNT_FORCE
		if err := syscall.Unmount(mountPoint, syscall.MNT_FORCE); err != nil {
			return fmt.Errorf("failed to unmount: %w", err)
		}
	}

	ll.Info("Filesystem unmounted successfully")
	return nil
}
