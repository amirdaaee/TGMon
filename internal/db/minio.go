package db

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MinioClient struct {
	*minio.Client
	bucket string
}
type MinioConfig struct {
	MinioEndpoint  string
	MinioAccessKey string
	MinioSecretKey string
	MinioBucket    string
	MinioSecure    bool
}

func NewMinioClient(minioCfg *MinioConfig) (*MinioClient, error) {
	minioClient, err := minio.New(minioCfg.MinioEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(minioCfg.MinioAccessKey, minioCfg.MinioSecretKey, ""),
		Secure: minioCfg.MinioSecure,
	})
	if err != nil {
		return nil, err
	}
	cl := MinioClient{Client: minioClient, bucket: minioCfg.MinioBucket}
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
func (cl *MinioClient) FileAdd(fileName string, data []byte, ctx context.Context) error {
	reader := bytes.NewReader(data)
	_, err := cl.PutObject(ctx, cl.bucket, fileName, reader, reader.Size(), minio.PutObjectOptions{})
	if err != nil {
		return err
	}
	return nil
}
func (cl *MinioClient) FileAddStr(fileName string, data string, ctx context.Context) error {
	reader := strings.NewReader(data)
	_, err := cl.PutObject(ctx, cl.bucket, fileName, reader, reader.Size(), minio.PutObjectOptions{})
	if err != nil {
		return err
	}
	return nil
}
func (cl *MinioClient) FileRm(fileName string, ctx context.Context) error {
	err := cl.RemoveObject(ctx, cl.bucket, fileName, minio.RemoveObjectOptions{ForceDelete: true})
	if err != nil {
		return fmt.Errorf("error removing object: %s", err)
	}
	return nil
}
