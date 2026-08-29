package types

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// IFacade defines the main facade interface for CRUD operations on type T.
//
//go:generate mockgen -source=facade.go -destination=../../../mocks/facade/types/facade.go -package=mocks_facade_types
type IFacade[T any] interface {
	CreateOne(ctx context.Context, doc *T) (*T, error)
	DeleteByID(ctx context.Context, id bson.ObjectID) (*T, error)
	FindByID(ctx context.Context, id bson.ObjectID) (*T, error)
	GetCRD() ICrud[T]
}
