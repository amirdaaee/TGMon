package types

import (
	"context"

	mngo "github.com/amirdaaee/TGMon/internal/db/mongo"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// IFacade defines the main facade interface for CRUD operations on type T.
//
//go:generate mockgen -source=facade.go -destination=../../../mocks/facade/types/facade.go -package=mocks_facade_types
type IFacade[T any] interface {
	CreateOne(ctx context.Context, doc *T) (*T, error)
	DeleteOne(ctx context.Context, filter bson.D) (*T, error)
	GetCRD() ICrud[T]
	GetCollection() mngo.ICollection[T]
}
