package repository

import (
	"context"

	"github.com/amirdaaee/TGMon/internal/types"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// JobReqRepository persists job request documents.
//
//go:generate mockgen -source=job.go -destination=../../mocks/repository/job.go -package=mocks_repository
type JobReqRepository interface {
	Store[types.JobReqDoc]
	List(ctx context.Context) ([]*types.JobReqDoc, error)
	DeleteByMediaID(ctx context.Context, mediaID bson.ObjectID) (int64, error)
}

// JobResRepository persists job result documents.
type JobResRepository interface {
	Store[types.JobResDoc]
}
