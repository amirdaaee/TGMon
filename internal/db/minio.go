// Package db provides database abstraction layers for MinIO object storage and MongoDB.
// It includes client managers for thread-safe operations and backward-compatible global functions.
package db

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/amirdaaee/TGMon/internal/log"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/sirupsen/logrus"
)

// IMinioCl defines the interface for MinIO client operations.
// This interface wraps the essential MinIO operations needed for object storage management.
// It is designed to be mockable for testing purposes.
//
//go:generate mockgen -source=minio.go -destination=../../mocks/db/minio.go -package=mocks
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
//go:generate mockgen -source=minio.go -destination=../../mocks/db/minio.go -package=mocks
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

// MinioClientRegistry manages MinioClient instances in a thread-safe manner.
// It provides safe concurrent access to MinIO client operations and handles
// client lifecycle management. This is the recommended approach for managing
// MinIO clients in production applications.
type MinioClientRegistry struct {
	mu     sync.RWMutex
	client IMinioClient
	x      IMongoClient
}

// NewMinioClientRegistry creates a new MinioClientManager instance.
// The manager starts uninitialized and requires a call to InitMinioClient
// before it can be used to serve client requests.
func NewMinioClientRegistry() *MinioClientRegistry {
	return &MinioClientRegistry{}
}

// InitMinioClient initializes the managed MinIO client with the provided configuration.
// This method is thread-safe and will replace any existing client instance.
//
// Parameters:
//   - ctx: Context for the initialization operation and bucket creation
//   - minioCfg: Configuration parameters for the MinIO connection
//   - factory: Optional factory function for creating IMinioCl instances (uses default if nil)
//   - skipAssertBucket: If true, skips bucket creation during initialization
//
// Returns an error if client initialization or bucket creation fails.
func (m *MinioClientRegistry) InitMinioClient(
	ctx context.Context,
	minioCfg *MinioConfig,
	skipAssertBucket bool,
	minioClFactory func(string, *minio.Options) (IMinioCl, error),
	minioClientFactory func(IMinioCl, string) IMinioClient,
) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.client != nil {
		m.getLogger("InitMinioClient").Warn("client already initialized. re-initializing...")
	}
	if minioClFactory == nil {
		minioClFactory = defaultMinioClFactory
	}

	minioClient, err := minioClFactory(minioCfg.MinioEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(minioCfg.MinioAccessKey, minioCfg.MinioSecretKey, ""),
		Secure: minioCfg.MinioSecure,
	})
	if err != nil {
		return fmt.Errorf("failed to initialize minio client: %w", err)
	}
	// ...
	if minioClientFactory == nil {
		minioClientFactory = defaultMinioClientFactory
	}
	cl := minioClientFactory(minioClient, minioCfg.MinioBucket)
	if !skipAssertBucket {
		if err := cl.CreateBucket(ctx); err != nil {
			return fmt.Errorf("failed to create bucket during initialization: %w", err)
		}
	}

	m.client = cl
	return nil
}

// GetMinioClient returns the managed MinIO client instance.
// This method is thread-safe and can be called concurrently.
// Returns an error if the client has not been initialized.
func (m *MinioClientRegistry) GetMinioClient() IMinioClient {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.client == nil {
		m.getLogger("GetMinioClient").Fatal("minio client not initialized")
	}
	return m.client
}
func (m *MinioClientRegistry) getLogger(fn string) *logrus.Entry {
	return log.GetLogger(log.DBModule).WithField("fn", fmt.Sprintf("%T.%s", m, fn))
}

// defaultMinioClFactory is the default factory function for creating MinIO client instances.
// It creates a new minio.Client with the provided endpoint and options.
func defaultMinioClFactory(endpoint string, opts *minio.Options) (IMinioCl, error) {
	return minio.New(endpoint, opts)
}
func defaultMinioClientFactory(minioCl IMinioCl, bucket string) IMinioClient {
	return &MinioClient{IMinioCl: minioCl, bucket: bucket}
}

// Global instance for backward compatibility (deprecated)
// TODO: Remove this and use dependency injection instead
var DefaultMinioRegistry = NewMinioClientRegistry()
