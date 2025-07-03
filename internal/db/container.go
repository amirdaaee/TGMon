package db

import (
	"github.com/amirdaaee/TGMon/internal/db/minio"
	"github.com/amirdaaee/TGMon/internal/db/mongo"
)

// IDbContainer defines the interface for accessing MongoDB and MinIO containers.
// It provides methods to retrieve the MongoDB and MinIO container interfaces used in the application.
//
//go:generate mockgen -source=container.go -destination=../../mocks/db/container.go -package=mocks
type IDbContainer interface {
	GetMongoContainer() mongo.IMongoContainer
	GetMinioContainer() minio.IMinioContainer
}

// DbContainer implements the IDbContainer interface and holds references to the MongoDB and MinIO containers.
type DbContainer struct {
	mongoContainer mongo.IMongoContainer
	minioContainer minio.IMinioContainer
}

var _ IDbContainer = (*DbContainer)(nil)

// GetMongoContainer returns the MongoDB container interface.
func (c *DbContainer) GetMongoContainer() mongo.IMongoContainer {
	return c.mongoContainer
}

// GetMinioContainer returns the MinIO container interface.
func (c *DbContainer) GetMinioContainer() minio.IMinioContainer {
	return c.minioContainer
}

// NewDbContainer creates and initializes a new DbContainer with the provided MongoDB and MinIO containers.
// It returns an IDbContainer interface for accessing the containers.
func NewDbContainer(mongoContainer mongo.IMongoContainer, minioContainer minio.IMinioContainer) IDbContainer {
	return &DbContainer{
		mongoContainer: mongoContainer,
		minioContainer: minioContainer,
	}
}
