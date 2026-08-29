package repository

import (
	"context"

	"github.com/amirdaaee/TGMon/internal/types"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// MediaFileRepository persists media file documents.
//
//go:generate mockgen -source=media.go -destination=../../mocks/repository/media.go -package=mocks_repository
type MediaFileRepository interface {
	Store[types.MediaFileDoc]
	FindByUName(ctx context.Context, uname string) (*types.MediaFileDoc, error)
	Count(ctx context.Context) (int64, error)
	CountByFileID(ctx context.Context, fileID int64) (int64, error)
	ListByID(ctx context.Context) ([]*types.MediaFileDoc, error)
	ListPage(ctx context.Context, page int, pageSize int) ([]*types.MediaFileDoc, error)
	FindNeighbors(ctx context.Context, doc *types.MediaFileDoc) (*bson.ObjectID, *bson.ObjectID, error)
	FindRandom(ctx context.Context) (*types.MediaFileDoc, error)
	SetThumbnail(ctx context.Context, id bson.ObjectID, name string) error
	SetUName(ctx context.Context, id bson.ObjectID, name string) error
	SetHasHash(ctx context.Context, id bson.ObjectID, has bool) error
	SetSpriteAndVtt(ctx context.Context, id bson.ObjectID, sprite string, vtt string) error
	SetSrt(ctx context.Context, id bson.ObjectID, srt string) error
}
