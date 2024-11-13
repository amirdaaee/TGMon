package facade

import (
	"context"

	"github.com/amirdaaee/TGMon/internal/db"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type mediaFacade struct {
	baseFacade[*db.MediaFileDoc]
}

func (f *mediaFacade) Create(ctx context.Context, doc *db.MediaFileDoc, cl *mongo.Client) (*db.MediaFileDoc, error) {
	newDoc, err := f.baseCreate(ctx, doc, cl)
	return newDoc, err
}
func (f *mediaFacade) Read(ctx context.Context, filter *primitive.D, cl *mongo.Client) ([]*db.MediaFileDoc, error) {
	docs, err := f.baseRead(ctx, filter, cl)
	return docs, err
}
func (f *mediaFacade) Update(ctx context.Context, filter *primitive.D, doc *db.MediaFileDoc, cl *mongo.Client) (*db.MediaFileDoc, error) {
	newDoc, err := f.baseUpdate(ctx, filter, doc, cl)
	if err != nil {
		return nil, err
	}
	return newDoc, err
}
func (f *mediaFacade) Delete(ctx context.Context, filter *primitive.D, cl *mongo.Client) error {
	return f.baseDelete(ctx, filter, cl)
}

func NewMediaFacade(mongo *db.Mongo) *mediaFacade {
	return &mediaFacade{
		baseFacade: baseFacade[*db.MediaFileDoc]{
			name:   "media",
			mongo:  *mongo,
			dsName: db.MEDIA_DS,
		},
	}
}
