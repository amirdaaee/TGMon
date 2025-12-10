package src

import (
	"context"

	"github.com/hanwen/go-fuse/v2/fs"
)

type ISrc interface {
	UID() string
	List(ctx context.Context) ([]IFile, error)
	Lookup(ctx context.Context, uid string) (IFile, error)
	Delete(ctx context.Context, uid string) error
	Exists(ctx context.Context, uid string) (bool, error)
}

type IFile interface {
	fs.NodeOpener
	fs.NodeGetattrer
	fs.InodeEmbedder
	Name() string
	Ext() string // extension of the file, with prefix '.'
	UID() string
	Size() uint64
	Mtime() uint64
	Atime() uint64
	Ctime() uint64
}
