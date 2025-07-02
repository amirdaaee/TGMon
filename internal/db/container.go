package db

import (
	"github.com/amirdaaee/TGMon/internal/db/minio"
	"github.com/amirdaaee/TGMon/internal/db/mongo"
)

//go:generate mockgen -source=container.go -destination=../../mocks/db/container.go -package=mocks
type IDbContainer interface {
	GetMongoContainer() mongo.IMongoContainer
	GetMinioContainer() minio.IMinioContainer
}

type DbContainer struct {
	mongoContainer mongo.IMongoContainer
	minioContainer minio.IMinioContainer
}

var _ IDbContainer = (*DbContainer)(nil)

func (c *DbContainer) GetMongoContainer() mongo.IMongoContainer {
	return c.mongoContainer
}

func (c *DbContainer) GetMinioContainer() minio.IMinioContainer {
	return c.minioContainer
}
func NewDbContainer(mongoContainer mongo.IMongoContainer, minioContainer minio.IMinioContainer) IDbContainer {
	return &DbContainer{
		mongoContainer: mongoContainer,
		minioContainer: minioContainer,
	}
}
