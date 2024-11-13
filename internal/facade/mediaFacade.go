package facade

import (
	"context"

	"github.com/amirdaaee/TGMon/internal/db"
	"github.com/amirdaaee/TGMon/internal/errs"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type mediaFacade struct {
	baseFacade[*db.MediaFileDoc]
	mongo db.Mongo
}

func (f *mediaFacade) getDatastore() *db.DataStore[*db.MediaFileDoc] {
	return f.mongo.GetMediaDatastore()
}
func (f *mediaFacade) Create(ctx context.Context, doc *db.MediaFileDoc, cl *mongo.Client) (*db.MediaFileDoc, errs.IMongoErr) {
	ds := f.getDatastore()
	newDoc, err := f.baseCreate(ctx, doc, cl, ds)
	return newDoc, err
}
func (f *mediaFacade) Read(ctx context.Context, filter *primitive.D, cl *mongo.Client) ([]*db.MediaFileDoc, errs.IMongoErr) {
	ds := f.getDatastore()
	docs, err := f.baseRead(ctx, filter, cl, ds)
	return docs, err
}
func (f *mediaFacade) Update(ctx context.Context, filter *primitive.D, doc *db.MediaFileDoc, cl *mongo.Client) (*db.MediaFileDoc, errs.IMongoErr) {
	ds := f.getDatastore()
	newDoc, err := f.baseUpdate(ctx, filter, doc, cl, ds)
	if err != nil {
		return nil, err
	}
	return newDoc, err
}
func (f *mediaFacade) Delete(ctx context.Context, filter *primitive.D, cl *mongo.Client) error {
	ds := f.getDatastore()
	return f.baseDelete(ctx, filter, cl, ds)
}
func NewMediaFacade(mongo *db.Mongo) *mediaFacade {
	return &mediaFacade{
		baseFacade: baseFacade[*db.MediaFileDoc]{name: "media"},
		mongo:      *mongo,
	}
}
