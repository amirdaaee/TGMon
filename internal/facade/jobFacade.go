package facade

import (
	"context"

	"github.com/amirdaaee/TGMon/internal/db"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type jobFacade struct {
	baseFacade[*db.JobDoc]
}

func (f *jobFacade) Create(ctx context.Context, doc *db.JobDoc, cl *mongo.Client) (*db.JobDoc, error) {
	newDoc, err := f.baseCreate(ctx, doc, cl)
	return newDoc, err
}
func (f *jobFacade) Read(ctx context.Context, filter *primitive.D, cl *mongo.Client) ([]*db.JobDoc, error) {
	docs, err := f.baseRead(ctx, filter, cl)
	return docs, err
}
func (f *jobFacade) Update(ctx context.Context, filter *primitive.D, doc *db.JobDoc, cl *mongo.Client) (*db.JobDoc, error) {
	newDoc, err := f.baseUpdate(ctx, filter, doc, cl)
	return newDoc, err
}
func (f *jobFacade) Delete(ctx context.Context, filter *primitive.D, cl *mongo.Client) error {
	return f.baseDelete(ctx, filter, cl)
}

func NewJobFacade(mongo *db.Mongo) *jobFacade {
	return &jobFacade{
		baseFacade: baseFacade[*db.JobDoc]{
			name:   "job",
			mongo:  *mongo,
			dsName: db.JOB_DS,
		},
	}
}
