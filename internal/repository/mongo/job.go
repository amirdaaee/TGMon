package mongo

import (
	"context"

	"github.com/amirdaaee/TGMon/internal/repository"
	"github.com/amirdaaee/TGMon/internal/types"
	"github.com/chenmingyong0423/go-mongox/v2/bsonx"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// JobReqRepo is the MongoDB implementation of repository.JobReqRepository.
type JobReqRepo struct {
	collectionStore[types.JobReqDoc]
}

var _ repository.JobReqRepository = (*JobReqRepo)(nil)

// NewJobReqRepo returns a job-request repository backed by the given client.
func NewJobReqRepo(c *Client) *JobReqRepo {
	return &JobReqRepo{collectionStore: newCollectionStore[types.JobReqDoc](c, jobReqCollectionName)}
}

func (r *JobReqRepo) List(ctx context.Context) ([]*types.JobReqDoc, error) {
	return r.coll.Finder().Find(ctx)
}

func (r *JobReqRepo) DeleteByMediaID(ctx context.Context, mediaID bson.ObjectID) (int64, error) {
	res, err := r.coll.Deleter().Filter(bsonx.NewD().Add(types.JobReqDoc__MediaIDField, mediaID).Build()).DeleteMany(ctx)
	if err != nil {
		return 0, err
	}
	if res == nil {
		return 0, nil
	}
	return res.DeletedCount, nil
}

// JobResRepo is the MongoDB implementation of repository.JobResRepository.
type JobResRepo struct {
	collectionStore[types.JobResDoc]
}

var _ repository.JobResRepository = (*JobResRepo)(nil)

// NewJobResRepo returns a job-result repository backed by the given client.
func NewJobResRepo(c *Client) *JobResRepo {
	return &JobResRepo{collectionStore: newCollectionStore[types.JobResDoc](c, jobResCollectionName)}
}
