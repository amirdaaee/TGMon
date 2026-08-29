package mongo

import (
	"context"

	"github.com/amirdaaee/TGMon/internal/repository"
	"github.com/amirdaaee/TGMon/internal/types"
	"github.com/chenmingyong0423/go-mongox/v2/builder/query"
	"github.com/chenmingyong0423/go-mongox/v2/builder/update"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// FuseStateRepo is the MongoDB implementation of repository.FuseStateRepository.
type FuseStateRepo struct {
	collectionStore[types.FuseStateDoc]
}

var _ repository.FuseStateRepository = (*FuseStateRepo)(nil)

// NewFuseStateRepo returns a FUSE state repository backed by the given client.
func NewFuseStateRepo(c *Client) *FuseStateRepo {
	return &FuseStateRepo{collectionStore: newCollectionStore[types.FuseStateDoc](c, fuseStateCollectionName)}
}

func (r *FuseStateRepo) SetRename(ctx context.Context, id bson.ObjectID, rename string) error {
	_, err := r.coll.Updater().Filter(query.Id(id)).Updates(update.Set(types.FuseStateDoc__RenameField, rename)).UpdateOne(ctx)
	return err
}

func (r *FuseStateRepo) List(ctx context.Context) ([]*types.FuseStateDoc, error) {
	return r.coll.Finder().Find(ctx)
}
