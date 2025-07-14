// Package facade provides generic CRUD and facade logic for TGMon entities.
package facade

import (
	"context"
	"fmt"

	mngo "github.com/amirdaaee/TGMon/internal/db/mongo"
	"github.com/amirdaaee/TGMon/internal/log"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// ICrud defines hooks and collection access for CRUD operations on type T.
//
//go:generate mockgen -source=facade.go -destination=../../mocks/facade/facade.go -package=mocks
type ICrud[T any] interface {
	PreCreate(ctx context.Context, doc *T) error
	PostCreate(ctx context.Context, doc *T) error // errors in post handlers won't affect main transaction (see docs)
	PreDelete(ctx context.Context, doc *T) error
	PostDelete(ctx context.Context, doc *T) error // errors in post handlers won't affect main transaction (see docs)
	GetCollection() mngo.ICollection[T]
}

// IFacade defines the main facade interface for CRUD operations on type T.
//
//go:generate mockgen -source=facade.go -destination=../../mocks/facade/facade.go -package=mocks
type IFacade[T any] interface {
	CreateOne(ctx context.Context, doc *T) (*T, error)
	DeleteOne(ctx context.Context, filter bson.D) (*T, error)
	Read(ctx context.Context, filter bson.D) ([]*T, error)
	GetCRD() ICrud[T]
}

// BaseFacade provides a generic implementation of IFacade for type T.
type BaseFacade[T any] struct {
	crd ICrud[T]
}

var _ IFacade[any] = (*BaseFacade[any])(nil)

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
	if _, err := f.getCollection().Creator().InsertOne(ctx, doc); err != nil {
		return nil, fmt.Errorf("error creating document: %w", err)
	}
	// PostCreate runs in a goroutine; errors are logged but not returned.
	go func() {
		if err := f.crd.PostCreate(ctx, doc); err != nil {
			ll.WithError(err).Error("error in post-creating hook")
		} else {
			ll.Info("document post-creating hook completed")
		}
	}()
	return doc, nil
}

// Read returns documents matching the filter from the collection.
func (f *BaseFacade[T]) Read(ctx context.Context, filter bson.D) ([]*T, error) {
	ll := f.getLogger("Read")
	ll.Info("Reading documents")
	docs, err := f.getCollection().Finder().Filter(filter).Find(ctx)
	if err != nil {
		return nil, fmt.Errorf("error reading documents: %w", err)
	}
	return docs, nil
}

// DeleteOne deletes a single document matching the filter after running pre-delete hooks. Post-delete hooks run in a goroutine; errors are logged but not returned.
func (f *BaseFacade[T]) DeleteOne(ctx context.Context, filter bson.D) (*T, error) {
	ll := f.getLogger("DeleteOne")
	ll.Info("Deleting document")
	fnd := f.getCollection().Finder().Filter(filter)
	c, err := fnd.Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("error counting existing documents: %w", err)
	}
	if c == 0 {
		return nil, fmt.Errorf("no documents found to delete")
	} else if c > 1 {
		return nil, fmt.Errorf("multiple documents found to delete")
	}
	doc, err := fnd.FindOne(ctx)
	if err != nil {
		return nil, fmt.Errorf("error finding document to delete: %w", err)
	}
	if err := f.crd.PreDelete(ctx, doc); err != nil {
		return nil, fmt.Errorf("error pre-deleting hook: %w", err)
	}
	if _, err = f.getCollection().Deleter().Filter(filter).DeleteOne(ctx); err != nil {
		return nil, fmt.Errorf("error deleting document: %w", err)
	}
	// PostDelete runs in a goroutine; errors are logged but not returned.
	go func() {
		if err := f.crd.PostDelete(ctx, doc); err != nil {
			ll.WithError(err).Error("error in post-deleting hook")
		} else {
			ll.Info("document post-deleting hook completed")
		}
	}()
	return doc, nil
}

// GetCRD returns the underlying CRUD implementation for type T.
func (f *BaseFacade[T]) GetCRD() ICrud[T] {
	return f.crd
}

// getLogger returns a logrus.Entry for the given function name, tagged with the struct type.
func (f *BaseFacade[T]) getLogger(fn string) *logrus.Entry {
	return log.GetLogger(log.FacadeModule).WithField("func", fmt.Sprintf("%T.%s", f, fn))
}

// getCollection returns the collection for type T from the underlying CRUD implementation.
func (f *BaseFacade[T]) getCollection() mngo.ICollection[T] {
	return f.crd.GetCollection()
}

// NewFacade returns a new BaseFacade for the given CRD implementation.
func NewFacade[T any](crd ICrud[T]) IFacade[T] {
	return &BaseFacade[T]{crd: crd}
}
