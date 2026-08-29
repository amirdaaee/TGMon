package minio

import (
	"context"
	"io"

	"github.com/minio/minio-go/v7"
)

// ObjectAPI is the subset of the MinIO SDK used by Store. It exists so tests can stub the SDK.
//
//go:generate mockgen -source=client.go -destination=../../../mocks/repository/minio/client.go -package=mocks_repository_minio
type ObjectAPI interface {
	BucketExists(ctx context.Context, bucketName string) (bool, error)
	MakeBucket(ctx context.Context, bucketName string, opts minio.MakeBucketOptions) error
	PutObject(ctx context.Context, bucketName string, objectName string, reader io.Reader, objectSize int64, opts minio.PutObjectOptions) (minio.UploadInfo, error)
	RemoveObject(ctx context.Context, bucketName string, objectName string, opts minio.RemoveObjectOptions) error
	GetObject(ctx context.Context, bucketName string, objectName string, opts minio.GetObjectOptions) (*minio.Object, error)
	StatObject(ctx context.Context, bucketName string, objectName string, opts minio.StatObjectOptions) (minio.ObjectInfo, error)
}

var _ ObjectAPI = (*minio.Client)(nil)
