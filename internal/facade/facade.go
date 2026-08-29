package facade

import (
	"context"
	"errors"
	"fmt"

	ftypes "github.com/amirdaaee/TGMon/internal/facade/types"
	"github.com/amirdaaee/TGMon/internal/log"
	"github.com/amirdaaee/TGMon/internal/repository"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.uber.org/zap"
)

// BaseFacade provides a generic implementation of IFacade for type T.
type BaseFacade[T any] struct {
	store repository.Store[T]
	crd   ftypes.ICrud[T]
}

var _ ftypes.IFacade[any] = (*BaseFacade[any])(nil)

// CreateOne creates a document after running pre-create hooks. Post-create hooks run in a goroutine; errors are logged but not returned.
func (f *BaseFacade[T]) CreateOne(ctx context.Context, doc *T) (*T, error) {
	ll := f.getLogger("CreateOne")
	ll.Info("Creating document")
	if doc == nil {
		return nil, fmt.Errorf("document is nil")
	}
	if err := f.crd.PreCreate(ctx, doc); err != nil {
		return nil, fmt.Errorf("error pre-creating hook: %w", err)
	}
	if err := f.store.Insert(ctx, doc); err != nil {
		return nil, fmt.Errorf("error creating document: %w", err)
	}
	// PostCreate runs in a goroutine; errors are logged but not returned.
	postCtx := context.Background()
	go func() {
		if err := f.crd.PostCreate(postCtx, doc); err != nil {
			ll.With(zap.Error(err)).Error("error in post-creating hook")
		} else {
			ll.Info("document post-creating hook completed")
		}
	}()
	return doc, nil
}

// DeleteByID deletes a document by ID after running pre-delete hooks. Post-delete hooks run in a goroutine; errors are logged but not returned.
func (f *BaseFacade[T]) DeleteByID(ctx context.Context, id bson.ObjectID) (*T, error) {
	ll := f.getLogger("DeleteByID")
	ll.Info("Deleting document")
	doc, err := f.store.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ftypes.ErrNoDocumentsFound
		}
		return nil, fmt.Errorf("error finding document to delete: %w", err)
	}
	if err := f.crd.PreDelete(ctx, doc); err != nil {
		return nil, fmt.Errorf("error pre-deleting hook: %w", err)
	}
	if err := f.store.DeleteByID(ctx, id); err != nil {
		return nil, fmt.Errorf("error deleting document: %w", err)
	}
	postCtx := context.Background()
	go func() {
		if err := f.crd.PostDelete(postCtx, doc); err != nil {
			ll.With(zap.Error(err)).Error("error in post-deleting hook")
		} else {
			ll.Info("document post-deleting hook completed")
		}
	}()
	return doc, nil
}

// FindByID returns the document with the given ID.
func (f *BaseFacade[T]) FindByID(ctx context.Context, id bson.ObjectID) (*T, error) {
	doc, err := f.store.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ftypes.ErrNoDocumentsFound
		}
		return nil, err
	}
	return doc, nil
}

// GetCRD returns the underlying CRUD implementation for type T.
func (f *BaseFacade[T]) GetCRD() ftypes.ICrud[T] {
	return f.crd
}

func (f *BaseFacade[T]) getLogger(fn string) *zap.Logger {
	return log.Named(log.FacadeModule, fmt.Sprintf("%T.%s", f, fn))
}

// NewFacade returns a new BaseFacade for the given store and CRD implementation.
func NewFacade[T any](store repository.Store[T], crd ftypes.ICrud[T]) ftypes.IFacade[T] {
	return &BaseFacade[T]{store: store, crd: crd}
}
