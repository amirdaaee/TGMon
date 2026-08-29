package types

import (
	"context"
)

// ICrud defines hooks for CRUD operations on type T.
//
//go:generate mockgen -source=crd.go -destination=../../../mocks/facade/types/crd.go -package=mocks_facade_types
type ICrud[T any] interface {
	PreCreate(ctx context.Context, doc *T) error
	PostCreate(ctx context.Context, doc *T) error // errors in post handlers won't affect main transaction (see docs)
	PreDelete(ctx context.Context, doc *T) error
	PostDelete(ctx context.Context, doc *T) error // errors in post handlers won't affect main transaction (see docs)
}
