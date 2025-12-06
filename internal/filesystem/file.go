package filesystem

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"
	"syscall"
	"unicode"

	"github.com/amirdaaee/TGMon/internal/log"
	"github.com/amirdaaee/TGMon/internal/stream"
	"github.com/amirdaaee/TGMon/internal/types"
	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// MediaFile represents a single media file in the filesystem
type MediaFile struct {
	fs.Inode
	media            *types.MediaFileDoc
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
	return log.GetLogger(log.FuseModule).WithField("func", fmt.Sprintf("%T.%s", mf, fn))
}

// ...
func getFilename(media *types.MediaFileDoc) string {
	return sanitizeFilename(media.NameExt())
}

func getInodeNumber(id bson.ObjectID) uint64 {
	// Convert ObjectID to bytes for hashing
	idBytes := []byte(id.Hex())

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

// sanitizeFilename makes a filename safe for use in filesystems by replacing
// unsafe characters with underscores. This handles slashes, colons, and other
// characters that are problematic in filenames.
func sanitizeFilename(filename string) string {
	var builder strings.Builder
	builder.Grow(len(filename) * 2) // Pre-allocate space for potential encoding

	for _, r := range filename {
		// Replace unsafe characters with underscore
		// Unsafe characters: / \ : * ? " < > | and control characters
		if r == '/' || r == '\\' || r == ':' || r == '*' || r == '?' ||
			r == '"' || r == '<' || r == '>' || r == '|' ||
			unicode.IsControl(r) {
			builder.WriteRune('_')
		} else {
			builder.WriteRune(r)
		}
	}

	result := builder.String()
	// Remove leading/trailing spaces and dots (problematic on Windows)
	result = strings.Trim(result, " .")
	// If the result is empty after trimming, use a default name
	if result == "" {
		result = "file"
	}
	return result
}
