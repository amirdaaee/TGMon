package repository

import (
	"context"

	"github.com/amirdaaee/TGMon/internal/types"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// MediaExtendedMetaRepository persists per-media client sidecar state.
//
//go:generate mockgen -source=mediaMeta.go -destination=../../mocks/repository/mediaMeta.go -package=mocks_repository
type MediaExtendedMetaRepository interface {
	GetOrCreateByMediaFileID(ctx context.Context, mediaFileID bson.ObjectID) (*types.MediaExtendedMeta, error)
	ReplaceByMediaFileID(ctx context.Context, mediaFileID bson.ObjectID, doc *types.MediaExtendedMeta) (*types.MediaExtendedMeta, error)
	PatchByMediaFileID(ctx context.Context, mediaFileID bson.ObjectID, patch types.MediaExtendedMetaPatch) (*types.MediaExtendedMeta, error)
	DeleteByMediaFileID(ctx context.Context, mediaFileID bson.ObjectID) error
}
