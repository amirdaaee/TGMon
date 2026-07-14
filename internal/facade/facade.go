package facade

import (
	"context"
	"fmt"

	mngo "github.com/amirdaaee/TGMon/internal/db/mongo"
	ftypes "github.com/amirdaaee/TGMon/internal/facade/types"
	"github.com/amirdaaee/TGMon/internal/log"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.uber.org/zap"
)

// BaseFacade provides a generic implementation of IFacade for type T.
type BaseFacade[T any] struct {
	crd ftypes.ICrud[T]
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
	if _, err := f.GetCollection().Creator().InsertOne(ctx, doc); err != nil {
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

// DeleteOne deletes a single document matching the filter after running pre-delete hooks. Post-delete hooks run in a goroutine; errors are logged but not returned.
func (f *BaseFacade[T]) DeleteOne(ctx context.Context, filter bson.D) (*T, error) {
	ll := f.getLogger("DeleteOne")
	ll.Info("Deleting document")
	fnd := f.GetCollection().Finder().Filter(filter)
	c, err := fnd.Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("error counting existing documents: %w", err)
	}
	if c == 0 {
		return nil, ftypes.ErrNoDocumentsFound
	} else if c > 1 {
		return nil, ftypes.ErrMultipleDocumentsFound
	}
	doc, err := fnd.FindOne(ctx)
	if err != nil {
		return nil, fmt.Errorf("error finding document to delete: %w", err)
	}
	if err := f.crd.PreDelete(ctx, doc); err != nil {
		return nil, fmt.Errorf("error pre-deleting hook: %w", err)
	}
	if _, err = f.GetCollection().Deleter().Filter(filter).DeleteOne(ctx); err != nil {
		return nil, fmt.Errorf("error deleting document: %w", err)
	}
	// PostDelete runs in a goroutine; errors are logged but not returned.
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

// GetCRD returns the underlying CRUD implementation for type T.
func (f *BaseFacade[T]) GetCRD() ftypes.ICrud[T] {
	return f.crd
}

// GetCollection returns the collection for type T from the underlying CRUD implementation.
func (f *BaseFacade[T]) GetCollection() mngo.ICollection[T] {
	return f.crd.GetCollection()
}

// getLogger returns a logrus.Entry for the given function name, tagged with the struct type.
func (f *BaseFacade[T]) getLogger(fn string) *zap.Logger {
	return log.Named(log.FacadeModule, fmt.Sprintf("%T.%s", f, fn))
}

// NewFacade returns a new BaseFacade for the given CRD implementation.
func NewFacade[T any](crd ftypes.ICrud[T]) ftypes.IFacade[T] {
	return &BaseFacade[T]{crd: crd}
}
