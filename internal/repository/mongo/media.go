package mongo

import (
	"context"
	"errors"
	"math/rand"

	"github.com/amirdaaee/TGMon/internal/repository"
	"github.com/amirdaaee/TGMon/internal/types"
	"github.com/chenmingyong0423/go-mongox/v2/bsonx"
	"github.com/chenmingyong0423/go-mongox/v2/builder/query"
	"github.com/chenmingyong0423/go-mongox/v2/builder/update"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// MediaFileRepo is the MongoDB implementation of repository.MediaFileRepository.
type MediaFileRepo struct {
	collectionStore[types.MediaFileDoc]
}

var _ repository.MediaFileRepository = (*MediaFileRepo)(nil)

// NewMediaFileRepo returns a media file repository backed by the given client.
func NewMediaFileRepo(c *Client) *MediaFileRepo {
	return &MediaFileRepo{collectionStore: newCollectionStore[types.MediaFileDoc](c, filesCollectionName)}
}

func (r *MediaFileRepo) FindByUName(ctx context.Context, uname string) (*types.MediaFileDoc, error) {
	doc, err := r.coll.Finder().Filter(query.Eq(types.MediaFileDoc__UnameField, uname)).FindOne(ctx)
	return doc, mapNotFound(err)
}

func (r *MediaFileRepo) Count(ctx context.Context) (int64, error) {
	return r.coll.Finder().Count(ctx)
}

func (r *MediaFileRepo) CountByFileID(ctx context.Context, fileID int64) (int64, error) {
	return r.coll.Finder().Filter(bsonx.NewD().Add(types.MediaFileDoc__FileIDField, fileID).Build()).Count(ctx)
}

func (r *MediaFileRepo) ListByID(ctx context.Context) ([]*types.MediaFileDoc, error) {
	return r.coll.Finder().Sort(bson.D{{Key: "_id", Value: 1}}).Find(ctx)
}

func (r *MediaFileRepo) ListPage(ctx context.Context, page, pageSize int) ([]*types.MediaFileDoc, error) {
	return r.coll.Finder().
		Sort(bson.D{{Key: createdAtField, Value: -1}}).
		Skip(int64(pageSize) * int64(page)).
		Limit(int64(pageSize)).
		Find(ctx)
}

func (r *MediaFileRepo) FindNeighbors(ctx context.Context, doc *types.MediaFileDoc) (*bson.ObjectID, *bson.ObjectID, error) {
	prevID, err := r.findNeighbor(ctx, doc, query.NewBuilder().Lt, -1)
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return nil, nil, err
	}
	nextID, err := r.findNeighbor(ctx, doc, query.NewBuilder().Gt, 1)
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return nil, nil, err
	}
	return prevID, nextID, nil
}

func (r *MediaFileRepo) FindRandom(ctx context.Context) (*types.MediaFileDoc, error) {
	total, err := r.Count(ctx)
	if err != nil {
		return nil, err
	}
	if total == 0 {
		return nil, repository.ErrNotFound
	}
	n := rand.Int63n(total)
	docs, err := r.coll.Finder().Skip(n).Limit(1).Find(ctx)
	if err != nil {
		return nil, err
	}
	if len(docs) == 0 {
		return nil, repository.ErrNotFound
	}
	return docs[0], nil
}

func (r *MediaFileRepo) SetThumbnail(ctx context.Context, id bson.ObjectID, name string) error {
	return r.setFields(ctx, id, update.Set(types.MediaFileDoc__ThumbnailField, name))
}

func (r *MediaFileRepo) SetUName(ctx context.Context, id bson.ObjectID, name string) error {
	return r.setFields(ctx, id, update.Set(types.MediaFileDoc__UnameField, name))
}

func (r *MediaFileRepo) SetHasHash(ctx context.Context, id bson.ObjectID, has bool) error {
	return r.setFields(ctx, id, update.Set(types.MediaFileDoc__HasHashField, has))
}

func (r *MediaFileRepo) SetSpriteAndVtt(ctx context.Context, id bson.ObjectID, sprite, vtt string) error {
	if err := r.setFields(ctx, id, update.Set(types.MediaFileDoc__SpriteField, sprite)); err != nil {
		return err
	}
	return r.setFields(ctx, id, update.Set(types.MediaFileDoc__VttField, vtt))
}

func (r *MediaFileRepo) SetSrt(ctx context.Context, id bson.ObjectID, srt string) error {
	return r.setFields(ctx, id, update.Set(types.MediaFileDoc__SrtField, srt))
}

func (r *MediaFileRepo) findNeighbor(ctx context.Context, doc *types.MediaFileDoc, qFactory func(string, any) *query.Builder, sort int) (*bson.ObjectID, error) {
	filter := qFactory(createdAtField, doc.CreatedAt).Build()
	srt := bsonx.NewD().Add(createdAtField, sort).Build()
	found, err := r.coll.Finder().Filter(filter).Sort(srt).FindOne(ctx)
	if err != nil {
		return nil, mapNotFound(err)
	}
	return &found.ID, nil
}

func (r *MediaFileRepo) setFields(ctx context.Context, id bson.ObjectID, updates bson.D) error {
	_, err := r.coll.Updater().Filter(query.Id(id)).Updates(updates).UpdateOne(ctx)
	return err
}
