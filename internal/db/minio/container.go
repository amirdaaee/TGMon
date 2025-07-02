package minio

import (
	"context"
	"fmt"

	"github.com/minio/minio-go/v7"
)

//go:generate mockgen -source=container.go -destination=../../../mocks/db/minio/container.go -package=mocks

// IMinioContainer defines the interface for MinIO container operations.
// This interface provides a way to access configured MinIO clients through dependency injection.
// It encapsulates the MinIO client creation and configuration, providing a clean abstraction
// for MinIO operations within the application.
type IMinioContainer interface {
	// GetMinioClient returns the configured MinIO client instance.
	// This client is pre-configured with the appropriate bucket and connection settings.
	GetMinioClient() IMinioClient
}

// MinioContainer implements the IMinioContainer interface and manages MinIO client instances.
// It holds both the low-level MinIO client and the high-level wrapped client for convenience.
// This container pattern allows for proper dependency injection and easier testing.
type MinioContainer struct {
	// cl is the underlying MinIO client from the official MinIO Go SDK
	cl *minio.Client
	// minioClient is the wrapped high-level client that implements IMinioClient
	minioClient *MinioClient
}

// GetMinioClient returns the configured high-level MinIO client.
// This method provides access to the wrapped MinIO client that offers simplified
// file operations for the configured bucket.
func (c *MinioContainer) GetMinioClient() IMinioClient {
	return c.minioClient
}

var _ IMinioContainer = (*MinioContainer)(nil)

// MinioContainerConfig holds the configuration parameters needed to create a MinIO container.
// All fields are required for proper MinIO container initialization and client creation.
type MinioContainerConfig struct {
	// endpoint is the MinIO server endpoint (e.g., "localhost:9000" or "minio.example.com")
	endpoint string
	// opts contains MinIO connection options including credentials and SSL settings
	opts *minio.Options
	// bucket is the name of the bucket that the MinIO client will operate on
	bucket string
}

// NewMinioContainer creates a new MinIO container with the specified configuration.
// It initializes both the low-level MinIO client and the high-level wrapped client.
// If createBucket is true, it will attempt to create the specified bucket if it doesn't exist.
//
// Parameters:
//   - ctx: Context for the operation, used for bucket creation if enabled
//   - config: Configuration containing endpoint, options, and bucket name
//   - createBucket: Whether to create the bucket if it doesn't exist
//
// Returns:
//   - IMinioContainer: The configured container ready for use
//   - error: Any error encountered during client creation or bucket creation
func NewMinioContainer(ctx context.Context, config MinioContainerConfig, createBucket bool) (IMinioContainer, error) {
	cl, err := minio.New(config.endpoint, config.opts)
	if err != nil {
		return nil, fmt.Errorf("error creating minio client: %w", err)
	}
	mCl := NewMinioClient(cl, config.bucket)
	if createBucket {
		err = mCl.CreateBucket(ctx)
		if err != nil {
			return nil, fmt.Errorf("error creating minio bucket: %w", err)
		}
	}
	return &MinioContainer{
		cl:          cl,
		minioClient: mCl,
	}, nil
}
