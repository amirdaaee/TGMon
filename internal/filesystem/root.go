package filesystem

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"sort"
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
	dbContainer      db.IDbContainer
	streamWorkerPool stream.IWorkerPool
	mediaCache       map[string]*types.MediaFileDoc
	cacheMutex       sync.RWMutex
	cacheExpiry      time.Time
	cacheTTL         time.Duration
}

var _ fs.NodeOnAdder = (*MediaFS)(nil)
var _ fs.NodeReaddirer = (*MediaFS)(nil)
var _ fs.NodeLookuper = (*MediaFS)(nil)
var _ fs.NodeGetattrer = (*MediaFS)(nil)
var _ fs.NodeOpendirer = (*MediaFS)(nil)

// OnAdd is called when the filesystem is mounted
func (mfs *MediaFS) OnAdd(ctx context.Context) {
	mfs.getLogger("OnAdd").Info("MediaFS mounted")
}

// Opendir opens a directory for reading
func (mfs *MediaFS) Opendir(ctx context.Context) syscall.Errno {
	mfs.getLogger("Opendir").Debug("Opening directory")
	return 0
}

// Getattr returns directory attributes for the root directory
func (mfs *MediaFS) Getattr(ctx context.Context, f fs.FileHandle, out *fuse.AttrOut) syscall.Errno {
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
func (mfs *MediaFS) Readdir(ctx context.Context) (fs.DirStream, syscall.Errno) {
	ll := mfs.getLogger("Readdir")
	ll.Debug("Reading directory")

	// Check if context is already canceled
	if ctx.Err() != nil {
		ll.Debug("Context canceled before reading directory")
		return nil, syscall.EINTR
	}

	// Create a context with timeout for database operations
	// Increased timeout for large directories (1000+ files)
	// This prevents the filesystem from hanging if the database is slow
	queryCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Get all media files from database
	mediaFiles, err := mfs.getMediaFiles(queryCtx)
	if err != nil {
		// Check if error is due to context cancellation or timeout
		if ctx.Err() != nil || queryCtx.Err() != nil {
			ll.WithError(err).Debug("Context canceled or timed out during getMediaFiles")
			// Return empty directory instead of error to prevent I/O errors
			// This allows the container to continue working even if DB is temporarily unavailable
			return fs.NewListDirStream([]fuse.DirEntry{}), 0
		}
		ll.WithError(err).Warn("Failed to get media files, returning empty directory")
		// Return empty directory instead of error to prevent I/O errors
		// This is safer for container access - they can retry later
		return fs.NewListDirStream([]fuse.DirEntry{}), 0
	}

	// For large directories, optimize memory allocation
	// Pre-allocate slice with exact capacity to avoid reallocations
	entries := make([]fuse.DirEntry, 0, len(mediaFiles))

	// Create directory entries directly (avoid intermediate struct for better performance)
	for _, media := range mediaFiles {
		filename := mfs.getFilename(media)
		// Set Ino to match what we use in Lookup - use hash of ObjectID for uniqueness
		ino := mfs.getInodeNumber(media.ID)
		entries = append(entries, fuse.DirEntry{
			Name: filename,
			Mode: fuse.S_IFREG | 0444, // Regular file, read-only
			Ino:  ino,
		})
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
func (mfs *MediaFS) Lookup(ctx context.Context, name string, out *fuse.EntryOut) (*fs.Inode, syscall.Errno) {
	ll := mfs.getLogger("Lookup")
	ll.Debugf("Looking up file: %s", name)

	// Check if context is already canceled
	if ctx.Err() != nil {
		ll.Debug("Context canceled before lookup")
		return nil, syscall.EINTR
	}

	// Create a context with timeout for database operations
	queryCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Get all media files
	mediaFiles, err := mfs.getMediaFiles(queryCtx)
	if err != nil {
		// Check if error is due to context cancellation or timeout
		if ctx.Err() != nil || queryCtx.Err() != nil {
			ll.WithError(err).Debug("Context canceled or timed out during getMediaFiles in Lookup")
			return nil, syscall.EINTR
		}
		ll.WithError(err).Warn("Failed to get media files in Lookup")
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
		media:            media,
		dbContainer:      mfs.dbContainer,
		streamWorkerPool: mfs.streamWorkerPool,
	}

	// Set entry attributes
	out.Mode = fuse.S_IFREG | 0444
	out.Size = uint64(media.Meta.FileSize)
	out.Mtime = uint64(media.CreatedAt.Unix())
	out.Atime = uint64(media.UpdatedAt.Unix())
	out.Ctime = uint64(media.CreatedAt.Unix())

	stable := fs.StableAttr{
		Mode: fuse.S_IFREG,
		Ino:  mfs.getInodeNumber(media.ID),
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
	for _, media := range mediaFiles[:100] {
		filename := mfs.getFilename(media)
		mfs.mediaCache[filename] = media
	}
	mfs.cacheExpiry = time.Now().Add(mfs.cacheTTL)

	return mediaFiles, nil
}

// getInodeNumber generates a unique inode number from an ObjectID
// Uses SHA256 hash to ensure uniqueness even if timestamps collide
func (mfs *MediaFS) getInodeNumber(id interface{}) uint64 {
	// Convert ObjectID to bytes for hashing
	var idBytes []byte
	switch v := id.(type) {
	case fmt.Stringer:
		idBytes = []byte(v.String())
	default:
		idBytes = []byte(fmt.Sprintf("%v", id))
	}

	// Use first 8 bytes of SHA256 hash as inode number
	hash := sha256.Sum256(idBytes)
	// Convert first 8 bytes to uint64, ensuring it's non-zero
	ino := uint64(hash[0])<<56 | uint64(hash[1])<<48 | uint64(hash[2])<<40 | uint64(hash[3])<<32 |
		uint64(hash[4])<<24 | uint64(hash[5])<<16 | uint64(hash[6])<<8 | uint64(hash[7])

	// Ensure inode is never 0 (0 is reserved)
	if ino == 0 {
		ino = 1
	}
	return ino
}

// getFilename returns the filename for a media file
func (mfs *MediaFS) getFilename(media *types.MediaFileDoc) string {
	ext := mfs.getExtensionFromMimeType(media.Meta.MimeType)
	if media.Meta.FileName != "" {
		return fmt.Sprintf("%s-%s%s", media.Meta.FileName, media.ID.Hex(), ext)
	}
	// Use ID as filename with appropriate extension based on mime type
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

// NewMediaFS creates a new MediaFS filesystem
func NewMediaFS(dbContainer db.IDbContainer, streamWorkerPool stream.IWorkerPool) *MediaFS {
	return &MediaFS{
		dbContainer:      dbContainer,
		streamWorkerPool: streamWorkerPool,
		mediaCache:       make(map[string]*types.MediaFileDoc),
		cacheTTL:         30 * time.Second, // Cache media list for 30 seconds
	}
}

// Mount mounts the media filesystem at the specified mount point
func Mount(mountPoint string, dbContainer db.IDbContainer, streamWorkerPool stream.IWorkerPool) (*fuse.Server, error) {
	return MountWithOptions(mountPoint, dbContainer, streamWorkerPool, &MountOptions{})
}

// MountOptions configures the filesystem mount behavior
type MountOptions struct {
	// AllowOther allows other users (including containers) to access the filesystem
	// Note: This requires /etc/fuse.conf to have "user_allow_other" enabled
	AllowOther bool
	// Debug enables FUSE debug logging
	Debug bool
}

// MountWithOptions mounts the media filesystem with custom options
func MountWithOptions(mountPoint string, dbContainer db.IDbContainer, streamWorkerPool stream.IWorkerPool, opts *MountOptions) (*fuse.Server, error) {
	ll := log.GetLogger(log.WebModule).WithField("func", "Mount")
	ll.Infof("Mounting filesystem at: %s", mountPoint)

	if opts == nil {
		opts = &MountOptions{}
	}

	// ensure mount point exists with proper permissions
	// Use 0755 for normal, 0777 if allow-other is enabled (needed for container access)
	mountPerms := os.FileMode(0755)
	if opts.AllowOther {
		mountPerms = 0777
	}
	if err := os.MkdirAll(mountPoint, mountPerms); err != nil {
		return nil, fmt.Errorf("failed to create mount point: %w", err)
	}

	// Create root filesystem
	root := NewMediaFS(dbContainer, streamWorkerPool)

	// Create FUSE server
	fuseOpts := &fs.Options{}
	fuseOpts.Debug = opts.Debug
	fuseOpts.AllowOther = opts.AllowOther
	// Set timeouts for better performance and stability
	// These help with container access by caching attributes and entries
	// Longer timeouts reduce database load when containers scan directories frequently
	attrTimeout := 5 * time.Second
	entryTimeout := 5 * time.Second
	fuseOpts.AttrTimeout = &attrTimeout
	fuseOpts.EntryTimeout = &entryTimeout
	// NegativeTimeout of 0 means don't cache failed lookups (safer)
	zeroTimeout := time.Duration(0)
	fuseOpts.NegativeTimeout = &zeroTimeout
	// Set MaxBackground to handle concurrent requests from containers
	// Higher value is needed for large directory scans (1000+ files)
	// This allows more concurrent FUSE operations without blocking
	// Default is typically 12, increasing to 128 helps with large directories and concurrent access
	fuseOpts.MaxBackground = 128

	if opts.AllowOther {
		ll.Info("AllowOther enabled - filesystem will be accessible to other users/containers")
	}

	server, err := fs.Mount(mountPoint, root, fuseOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to mount filesystem: %w", err)
	}

	// After successful mount, try to set permissions for allow-other access
	if opts.AllowOther {
		if err := os.Chmod(mountPoint, 0777); err != nil {
			ll.WithError(err).Warn("Failed to set mount point permissions to 0777 (this may be normal)")
		}
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
		ll.WithError(err).Error("failed to unmount filesystem using fusermount. trying with MNT_FORCE")
		if err := syscall.Unmount(mountPoint, syscall.MNT_FORCE); err != nil {
			return fmt.Errorf("failed to unmount: %w", err)
		}
	}

	ll.Info("Filesystem unmounted successfully")
	return nil
}
