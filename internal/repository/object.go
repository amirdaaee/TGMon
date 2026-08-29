package repository

import (
	"context"
	"io"
	"time"
)

// ObjectInfo is storage metadata for an object. It does not leak MinIO types.
type ObjectInfo struct {
	Size         int64
	LastModified time.Time
}

// ObjectStore is a bucket-scoped blob store.
//
//go:generate mockgen -source=object.go -destination=../../mocks/repository/object.go -package=mocks_repository
type ObjectStore interface {
	Put(ctx context.Context, name string, data []byte) error
	Get(ctx context.Context, name string) (io.ReadSeekCloser, error)
	Stat(ctx context.Context, name string) (*ObjectInfo, error)
	Delete(ctx context.Context, name string) error
}
