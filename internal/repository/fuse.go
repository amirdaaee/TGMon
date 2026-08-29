package repository

import (
	"context"

	"github.com/amirdaaee/TGMon/internal/types"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// FuseStateRepository persists FUSE rename-map state.
//
//go:generate mockgen -source=fuse.go -destination=../../mocks/repository/fuse.go -package=mocks_repository
type FuseStateRepository interface {
	Insert(ctx context.Context, doc *types.FuseStateDoc) error
	DeleteByID(ctx context.Context, id bson.ObjectID) error
	SetRename(ctx context.Context, id bson.ObjectID, rename string) error
	List(ctx context.Context) ([]*types.FuseStateDoc, error)
}
