package minio

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/amirdaaee/TGMon/internal/repository"
	"github.com/minio/minio-go/v7"
)

// Config holds MinIO connection settings.
type Config struct {
	Endpoint string
	Opts     *minio.Options
	Bucket   string
}

// Store is a bucket-scoped MinIO ObjectStore.
type Store struct {
	api    ObjectAPI
	bucket string
}

var _ repository.ObjectStore = (*Store)(nil)

// Connect creates a MinIO client and optionally ensures the bucket exists.
func Connect(ctx context.Context, cfg Config, createBucket bool) (*Store, error) {
	if cfg.Endpoint == "" {
		return nil, fmt.Errorf("minio endpoint is required")
	}
	if cfg.Opts == nil {
		return nil, fmt.Errorf("minio options are required")
	}
	if cfg.Bucket == "" {
		return nil, fmt.Errorf("minio bucket is required")
	}
	cl, err := minio.New(cfg.Endpoint, cfg.Opts)
	if err != nil {
		return nil, fmt.Errorf("error creating minio client: %w", err)
	}
	st := NewStore(cl, cfg.Bucket)
	if createBucket {
		if err := st.EnsureBucket(ctx); err != nil {
			return nil, err
		}
	}
	return st, nil
}

// NewStore wraps an ObjectAPI for the given bucket. Used by tests.
func NewStore(api ObjectAPI, bucket string) *Store {
	return &Store{api: api, bucket: bucket}
}

func (s *Store) Put(ctx context.Context, name string, data []byte) error {
	reader := bytes.NewReader(data)
	_, err := s.api.PutObject(ctx, s.bucket, name, reader, reader.Size(), minio.PutObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to upload file '%s' to bucket '%s': %w", name, s.bucket, err)
	}
	return nil
}

func (s *Store) Get(ctx context.Context, name string) (io.ReadSeekCloser, error) {
	obj, err := s.api.GetObject(ctx, s.bucket, name, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get file '%s' from bucket '%s': %w", name, s.bucket, err)
	}
	return obj, nil
}

func (s *Store) Stat(ctx context.Context, name string) (*repository.ObjectInfo, error) {
	info, err := s.api.StatObject(ctx, s.bucket, name, minio.StatObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to stat file '%s' from bucket '%s': %w", name, s.bucket, err)
	}
	return &repository.ObjectInfo{Size: info.Size, LastModified: info.LastModified}, nil
}

func (s *Store) Delete(ctx context.Context, name string) error {
	err := s.api.RemoveObject(ctx, s.bucket, name, minio.RemoveObjectOptions{ForceDelete: true})
	if err != nil {
		return fmt.Errorf("failed to remove file '%s' from bucket '%s': %w", name, s.bucket, err)
	}
	return nil
}

func (s *Store) EnsureBucket(ctx context.Context) error {
	exists, err := s.api.BucketExists(ctx, s.bucket)
	if err != nil {
		return fmt.Errorf("failed to check bucket existence for bucket '%s': %w", s.bucket, err)
	}
	if exists {
		return nil
	}
	if err := s.api.MakeBucket(ctx, s.bucket, minio.MakeBucketOptions{}); err != nil {
		return fmt.Errorf("failed to create bucket '%s': %w", s.bucket, err)
	}
	return nil
}
