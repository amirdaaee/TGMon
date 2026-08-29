package mongo

import (
	"context"
	"errors"

	"github.com/amirdaaee/TGMon/internal/repository"
	"github.com/chenmingyong0423/go-mongox/v2"
	"github.com/chenmingyong0423/go-mongox/v2/builder/query"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type collectionStore[T any] struct {
	coll *mongox.Collection[T]
}

func newCollectionStore[T any](c *Client, name string) collectionStore[T] {
	return collectionStore[T]{coll: newCollection[T](c, name)}
}

func (s collectionStore[T]) Insert(ctx context.Context, doc *T) error {
	_, err := s.coll.Creator().InsertOne(ctx, doc)
	return err
}

func (s collectionStore[T]) FindByID(ctx context.Context, id bson.ObjectID) (*T, error) {
	doc, err := s.coll.Finder().Filter(query.Id(id)).FindOne(ctx)
	return doc, mapNotFound(err)
}

func (s collectionStore[T]) DeleteByID(ctx context.Context, id bson.ObjectID) error {
	res, err := s.coll.Deleter().Filter(query.Id(id)).DeleteOne(ctx)
	if err != nil {
		return err
	}
	if res != nil && res.DeletedCount == 0 {
		return repository.ErrNotFound
	}
	return nil
}

func mapNotFound(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, mongo.ErrNoDocuments) {
		return repository.ErrNotFound
	}
	return err
}
