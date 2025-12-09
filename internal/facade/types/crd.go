package types

import (
	"context"

	mngo "github.com/amirdaaee/TGMon/internal/db/mongo"
)

// ICrud defines hooks and collection access for CRUD operations on type T.
//
//go:generate mockgen -source=crd.go -destination=../../../mocks/facade/types/crd.go -package=mocks
type ICrud[T any] interface {
	PreCreate(ctx context.Context, doc *T) error
	PostCreate(ctx context.Context, doc *T) error // errors in post handlers won't affect main transaction (see docs)
	PreDelete(ctx context.Context, doc *T) error
	PostDelete(ctx context.Context, doc *T) error // errors in post handlers won't affect main transaction (see docs)
	GetCollection() mngo.ICollection[T]
}
