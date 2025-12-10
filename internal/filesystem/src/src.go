package src

import (
	"context"

	"github.com/hanwen/go-fuse/v2/fs"
)

// ISrc defines the interface for data sources that provide files to the filesystem.
// Each source can list files, look up files by UID, delete files, and check file existence.
type ISrc interface {
	UID() string
	List(ctx context.Context) ([]IFile, error)
	Lookup(ctx context.Context, uid string) (IFile, error)
	Delete(ctx context.Context, uid string) error
	Exists(ctx context.Context, uid string) (bool, error)
}

// IFile defines the interface for files in the filesystem.
// It extends FUSE node interfaces and provides file metadata methods
// such as name, size, and timestamps.
type IFile interface {
	fs.NodeOpener
	fs.NodeGetattrer
	fs.InodeEmbedder
	// Name returns the display name of the file.
	Name() string
	// Ext returns the file extension with the '.' prefix.
	Ext() string
	// UID returns the unique identifier of the file.
	UID() string
	// Size returns the size of the file in bytes.
	Size() uint64
	// Mtime returns the modification time as a Unix timestamp.
	Mtime() uint64
	// Atime returns the access time as a Unix timestamp.
	Atime() uint64
	// Ctime returns the creation time as a Unix timestamp.
	Ctime() uint64
}
