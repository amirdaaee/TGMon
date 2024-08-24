package db

import (
	"bytes"
	"context"
	"fmt"

	"github.com/amirdaaee/TGMon/config"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MinioClient struct {
	*minio.Client
	bucket string
}

func NewMinioClient() (*MinioClient, error) {
	minioClient, err := minio.New(config.Config().MinioEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(config.Config().MinioAccessKey, config.Config().MinioSecretKey, ""),
		Secure: config.Config().MinioSecure,
	})
	if err != nil {
		return nil, err
	}
	cl := MinioClient{Client: minioClient, bucket: config.Config().MinioBucket}
	if err := cl.CreateBucket(context.TODO()); err != nil {
		return nil, err
	}
	return &cl, err
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
func (cl *MinioClient) MinioAddFile(data []byte, ctx context.Context) (string, error) {
	fileName := fmt.Sprintf("%s.jpeg", uuid.NewString())
	reader := bytes.NewReader(data)
	_, err := cl.PutObject(ctx, cl.bucket, fileName, reader, reader.Size(), minio.PutObjectOptions{})
	if err != nil {
		return "", err
	}
	return fileName, nil
}
