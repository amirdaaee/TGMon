package mongo

import (
	"context"

	"github.com/amirdaaee/TGMon/internal/repository"
	"github.com/amirdaaee/TGMon/internal/types"
	"github.com/chenmingyong0423/go-mongox/v2/bsonx"
	"github.com/chenmingyong0423/go-mongox/v2/builder/update"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// MediaExtendedMetaRepo is the MongoDB implementation of repository.MediaExtendedMetaRepository.
type MediaExtendedMetaRepo struct {
	collectionStore[types.MediaExtendedMeta]
}

var _ repository.MediaExtendedMetaRepository = (*MediaExtendedMetaRepo)(nil)

func (r *MediaExtendedMetaRepo) GetOrCreateByMediaFileID(ctx context.Context, mediaFileID bson.ObjectID) (*types.MediaExtendedMeta, error) {
	updates := update.NewBuilder().
		SetOnInsert(types.MediaExtendedMeta__MediaFileIDField, mediaFileID).
		Build()
	return r.upsertAndFind(ctx, mediaFileID, updates)
}

func (r *MediaExtendedMetaRepo) ReplaceByMediaFileID(ctx context.Context, mediaFileID bson.ObjectID, doc *types.MediaExtendedMeta) (*types.MediaExtendedMeta, error) {
	if doc == nil {
		doc = &types.MediaExtendedMeta{}
	}
	updates := update.NewBuilder().
		SetOnInsert(types.MediaExtendedMeta__MediaFileIDField, mediaFileID).
		Set(types.MediaExtendedMeta__LastPlayedAtField, doc.LastPlayedAt).
		Set(types.MediaExtendedMeta__CheckpointField, doc.Checkpoint).
		Set(types.MediaExtendedMeta__ScoreField, doc.Score).
		Set(types.MediaExtendedMeta__LikesField, doc.Likes).
		Set(types.MediaExtendedMeta__IsFavoriteField, doc.IsFavorite).
		Build()
	return r.upsertAndFind(ctx, mediaFileID, updates)
}

func (r *MediaExtendedMetaRepo) PatchByMediaFileID(ctx context.Context, mediaFileID bson.ObjectID, patch types.MediaExtendedMetaPatch) (*types.MediaExtendedMeta, error) {
	b := update.NewBuilder().SetOnInsert(types.MediaExtendedMeta__MediaFileIDField, mediaFileID)
	if patch.LastPlayedAt != nil {
		b.Set(types.MediaExtendedMeta__LastPlayedAtField, *patch.LastPlayedAt)
	}
	if patch.Checkpoint != nil {
		b.Set(types.MediaExtendedMeta__CheckpointField, *patch.Checkpoint)
	}
	if patch.Score != nil {
		b.Set(types.MediaExtendedMeta__ScoreField, *patch.Score)
	}
	if patch.Likes != nil {
		b.Set(types.MediaExtendedMeta__LikesField, *patch.Likes)
	}
	if patch.IsFavorite != nil {
		b.Set(types.MediaExtendedMeta__IsFavoriteField, *patch.IsFavorite)
	}
	if patch.PlayCount != nil {
		b.Set(types.MediaExtendedMeta__PlayCountField, *patch.PlayCount)
	}
	return r.upsertAndFind(ctx, mediaFileID, b.Build())
}

func (r *MediaExtendedMetaRepo) DeleteByMediaFileID(ctx context.Context, mediaFileID bson.ObjectID) error {
	_, err := r.coll.Deleter().Filter(mediaFileIDFilter(mediaFileID)).DeleteOne(ctx)
	return err
}

func (r *MediaExtendedMetaRepo) upsertAndFind(ctx context.Context, mediaFileID bson.ObjectID, updates bson.D) (*types.MediaExtendedMeta, error) {
	_, err := r.coll.Updater().
		Filter(mediaFileIDFilter(mediaFileID)).
		Updates(updates).
		Upsert(ctx)
	if err != nil {
		return nil, err
	}
	doc, err := r.coll.Finder().Filter(mediaFileIDFilter(mediaFileID)).FindOne(ctx)
	return doc, mapNotFound(err)
}

func mediaFileIDFilter(mediaFileID bson.ObjectID) bson.D {
	return bsonx.NewD().Add(types.MediaExtendedMeta__MediaFileIDField, mediaFileID).Build()
}

// NewMediaExtendedMetaRepo returns a media extended-meta repository backed by the given client.
func NewMediaExtendedMetaRepo(c *Client) *MediaExtendedMetaRepo {
	return &MediaExtendedMetaRepo{collectionStore: newCollectionStore[types.MediaExtendedMeta](c, mediaExtCollectionName)}
}
