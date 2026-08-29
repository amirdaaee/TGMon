package repository

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// Store is the persistence kernel used by the generic facade.
//
//go:generate mockgen -source=store.go -destination=../../mocks/repository/store.go -package=mocks_repository
type Store[T any] interface {
	Insert(ctx context.Context, doc *T) error
	FindByID(ctx context.Context, id bson.ObjectID) (*T, error)
	DeleteByID(ctx context.Context, id bson.ObjectID) error
}
