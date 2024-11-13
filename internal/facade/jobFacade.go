package facade

import (
	"context"

	"github.com/amirdaaee/TGMon/internal/db"
	"github.com/amirdaaee/TGMon/internal/errs"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type jobFacade struct {
	baseFacade[*db.JobDoc]
	mongo db.Mongo
}

func (f *jobFacade) getDatastore() *db.DataStore[*db.JobDoc] {
	return f.mongo.GetJobDatastore()
}
func (f *jobFacade) Create(ctx context.Context, doc *db.JobDoc, cl *mongo.Client) (*db.JobDoc, errs.IMongoErr) {
	ds := f.getDatastore()
	newDoc, err := f.baseCreate(ctx, doc, cl, ds)
	return newDoc, err
}
func (f *jobFacade) Read(ctx context.Context, filter *primitive.D, cl *mongo.Client) ([]*db.JobDoc, errs.IMongoErr) {
	ds := f.getDatastore()
	docs, err := f.baseRead(ctx, filter, cl, ds)
	return docs, err
}
func (f *jobFacade) Update(ctx context.Context, filter *primitive.D, doc *db.JobDoc, cl *mongo.Client) (*db.JobDoc, errs.IMongoErr) {
	ds := f.getDatastore()
	newDoc, err := f.baseUpdate(ctx, filter, doc, cl, ds)
	return newDoc, err
}
func (f *jobFacade) Delete(ctx context.Context, filter *primitive.D, cl *mongo.Client) error {
	ds := f.getDatastore()
	return f.baseDelete(ctx, filter, cl, ds)
}

func NewJobFacade(mongo *db.Mongo) *jobFacade {
	return &jobFacade{
		baseFacade: baseFacade[*db.JobDoc]{name: "job"},
		mongo:      *mongo,
	}
}
