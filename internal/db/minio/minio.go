// Package db provides database abstraction layers for MinIO object storage and MongoDB.
// It includes client managers for thread-safe operations and backward-compatible global functions.
package minio

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/minio/minio-go/v7"
)

// IMinioCl defines the interface for MinIO client operations.
// This interface wraps the essential MinIO operations needed for object storage management.
// It is designed to be mockable for testing purposes.
//
//go:generate mockgen -source=minio.go -destination=../../../mocks/db/minio/minio.go -package=mocks
type IMinioCl interface {
	// BucketExists checks if a bucket with the given name exists.
	// Returns true if the bucket exists, false otherwise, and any error encountered.
	BucketExists(ctx context.Context, bucketName string) (bool, error)

	// MakeBucket creates a new bucket with the specified name and options.
	// Returns an error if the bucket creation fails.
	MakeBucket(ctx context.Context, bucketName string, opts minio.MakeBucketOptions) (err error)

	// PutObject uploads an object to the specified bucket.
	// Returns upload information and any error encountered during the upload.
	PutObject(ctx context.Context, bucketName string, objectName string, reader io.Reader, objectSize int64, opts minio.PutObjectOptions) (info minio.UploadInfo, err error)

	// RemoveObject deletes an object from the specified bucket.
	// Returns an error if the deletion fails.
	RemoveObject(ctx context.Context, bucketName string, objectName string, opts minio.RemoveObjectOptions) error
}

// IMinioClient defines the high-level interface for MinIO client operations.
// This interface provides simplified methods for common file operations on a specific bucket.
// It abstracts away the underlying MinIO complexity and provides a more user-friendly API.
//
//go:generate mockgen -source=minio.go -destination=../../../mocks/db/minio/minio.go -package=mocks
type IMinioClient interface {
	// CreateBucket creates the configured bucket if it doesn't already exist.
	// This is typically called during initialization to ensure the bucket is available.
	CreateBucket(ctx context.Context) error

	// FileAdd uploads binary data to the bucket with the specified filename.
	// The data is provided as a byte slice.
	FileAdd(ctx context.Context, fileName string, data []byte) error

	// FileAddStr uploads string data to the bucket with the specified filename.
	// The data is provided as a string and will be converted to bytes internally.
	FileAddStr(ctx context.Context, fileName string, data string) error

	// FileRm removes a file from the bucket.
	// The removal is forced, meaning it will delete the file even if it has versioning enabled.
	FileRm(ctx context.Context, fileName string) error
}

// MinioClient implements the IMinioClient interface and provides high-level file operations
// for a specific MinIO bucket. It wraps the lower-level IMinioCl interface to provide
// a more convenient API for common operations.
type MinioClient struct {
	IMinioCl
	bucket string
}

// MinioConfig holds the configuration parameters needed to connect to a MinIO server.
// All fields are required for proper MinIO client initialization.
type MinioConfig struct {
	// MinioEndpoint is the MinIO server endpoint (e.g., "localhost:9000" or "minio.example.com")
	MinioEndpoint string

	// MinioAccessKey is the access key for MinIO authentication
	MinioAccessKey string

	// MinioSecretKey is the secret key for MinIO authentication
	MinioSecretKey string

	// MinioBucket is the name of the bucket to use for operations
	MinioBucket string

	// MinioSecure indicates whether to use HTTPS (true) or HTTP (false) for connections
	MinioSecure bool
}

// CreateBucket creates the configured bucket if it doesn't already exist.
// This method is idempotent - if the bucket already exists, it returns nil without error.
// It first checks for bucket existence and only attempts creation if the bucket doesn't exist.
func (cl *MinioClient) CreateBucket(ctx context.Context) error {
	exists, err := cl.BucketExists(ctx, cl.bucket)
	if err != nil {
		return fmt.Errorf("failed to check bucket existence for bucket '%s': %w", cl.bucket, err)
	}
	if exists {
		return nil
	}

	if err := cl.MakeBucket(ctx, cl.bucket, minio.MakeBucketOptions{}); err != nil {
		return fmt.Errorf("failed to create bucket '%s': %w", cl.bucket, err)
	}
	return nil
}

// FileAdd uploads binary data to the configured bucket with the specified filename.
// The data is provided as a byte slice and uploaded using the most efficient method available.
func (cl *MinioClient) FileAdd(ctx context.Context, fileName string, data []byte) error {
	reader := bytes.NewReader(data)
	return cl.fileAddAnything(ctx, fileName, reader, reader.Size())
}

// FileAddStr uploads string data to the configured bucket with the specified filename.
// The string data is converted to bytes internally and uploaded efficiently.
func (cl *MinioClient) FileAddStr(ctx context.Context, fileName string, data string) error {
	reader := strings.NewReader(data)
	return cl.fileAddAnything(ctx, fileName, reader, reader.Size())
}

// fileAddAnything is a private helper method that handles the actual upload logic
// for both binary and string data. It accepts any io.Reader and the size of the data.
func (cl *MinioClient) fileAddAnything(ctx context.Context, fileName string, r io.Reader, s int64) error {
	_, err := cl.PutObject(ctx, cl.bucket, fileName, r, s, minio.PutObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to upload file '%s' to bucket '%s': %w", fileName, cl.bucket, err)
	}
	return nil
}

// FileRm removes a file from the configured bucket.
// The removal is forced, meaning versioned objects will be permanently deleted.
// Returns an error if the file cannot be removed.
func (cl *MinioClient) FileRm(ctx context.Context, fileName string) error {
	err := cl.RemoveObject(ctx, cl.bucket, fileName, minio.RemoveObjectOptions{ForceDelete: true})
	if err != nil {
		return fmt.Errorf("failed to remove file '%s' from bucket '%s': %w", fileName, cl.bucket, err)
	}
	return nil
}

var _ IMinioClient = (*MinioClient)(nil)

// NewMinioClient creates a new MinioClient instance with the specified low-level client and bucket name.
// This constructor function wraps a low-level IMinioCl implementation to provide high-level file operations
// for a specific bucket. The returned client will operate exclusively on the specified bucket.
//
// Parameters:
//   - iCl: The low-level MinIO client interface implementation that handles the actual MinIO operations
//   - bucketName: The name of the bucket that this client will operate on for all file operations
//
// Returns:
//   - *MinioClient: A configured high-level MinIO client ready for file operations
func NewMinioClient(iCl IMinioCl, bucketName string) *MinioClient {
	return &MinioClient{
		IMinioCl: iCl,
		bucket:   bucketName,
	}
}
