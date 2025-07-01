package db

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/sirupsen/logrus"
)

//go:generate mockgen -source=minio.go -destination=../../mocks/db/minio.go -package=mocks
type IMinioCl interface {
	BucketExists(ctx context.Context, bucketName string) (bool, error)
	MakeBucket(ctx context.Context, bucketName string, opts minio.MakeBucketOptions) (err error)
	PutObject(ctx context.Context, bucketName string, objectName string, reader io.Reader, objectSize int64, opts minio.PutObjectOptions) (info minio.UploadInfo, err error)
	RemoveObject(ctx context.Context, bucketName string, objectName string, opts minio.RemoveObjectOptions) error
}

//go:generate mockgen -source=minio.go -destination=../../mocks/db/minio.go -package=mocks
type IMinioClient interface {
	CreateBucket(ctx context.Context) error
	FileAdd(ctx context.Context, fileName string, data []byte) error
	FileAddStr(ctx context.Context, fileName string, data string) error
	FileRm(ctx context.Context, fileName string) error
}
type MinioClient struct {
	IMinioCl
	bucket string
}
type MinioConfig struct {
	MinioEndpoint  string
	MinioAccessKey string
	MinioSecretKey string
	MinioBucket    string
	MinioSecure    bool
}

func (cl *MinioClient) CreateBucket(ctx context.Context) error {
	if exists, err := cl.BucketExists(ctx, cl.bucket); err != nil {
		return fmt.Errorf("can not check bucket existance: %s", err)
	} else if exists {
		return nil
	}
	if err := cl.MakeBucket(ctx, cl.bucket, minio.MakeBucketOptions{}); err != nil {
		return fmt.Errorf("can not create bucket: %s", err)
	}
	return nil
}
func (cl *MinioClient) FileAdd(ctx context.Context, fileName string, data []byte) error {
	reader := bytes.NewReader(data)
	return cl.fileAddAnything(ctx, fileName, reader, reader.Size())
}
func (cl *MinioClient) FileAddStr(ctx context.Context, fileName string, data string) error {
	reader := strings.NewReader(data)
	return cl.fileAddAnything(ctx, fileName, reader, reader.Size())
}
func (cl *MinioClient) fileAddAnything(ctx context.Context, fileName string, r io.Reader, s int64) error {
	_, err := cl.PutObject(ctx, cl.bucket, fileName, r, s, minio.PutObjectOptions{})
	if err != nil {
		return err
	}
	return nil
}
func (cl *MinioClient) FileRm(ctx context.Context, fileName string) error {
	err := cl.RemoveObject(ctx, cl.bucket, fileName, minio.RemoveObjectOptions{ForceDelete: true})
	if err != nil {
		return fmt.Errorf("error removing object: %s", err)
	}
	return nil
}

// ...
var minioCl *MinioClient
var minioClLock = sync.RWMutex{}

func InitMinioClient(minioCfg *MinioConfig, factory func(string, *minio.Options) (IMinioCl, error), skipAssertBucket bool) {
	minioClLock.Lock()
	defer minioClLock.Unlock()
	if factory == nil {
		factory = newMinioCl
	}
	minioClient, err := factory(minioCfg.MinioEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(minioCfg.MinioAccessKey, minioCfg.MinioSecretKey, ""),
		Secure: minioCfg.MinioSecure,
	})
	if err != nil {
		logrus.Fatalf("failed to init minio client: %s", err)
	}
	cl := MinioClient{IMinioCl: minioClient, bucket: minioCfg.MinioBucket}
	if !skipAssertBucket {
		if err := cl.CreateBucket(context.TODO()); err != nil {
			logrus.Fatalf("failed to create bucket: %s", err)
		}
	}
	minioCl = &cl
}
func newMinioCl(endpoint string, opts *minio.Options) (IMinioCl, error) {
	return minio.New(endpoint, opts)
}

func GetMinioClient() *MinioClient {
	minioClLock.RLock()
	defer minioClLock.RUnlock()
	if minioCl == nil {
		logrus.Fatal("minio client not initialized")
	}
	return minioCl
}
