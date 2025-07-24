package mongo

import (
	"github.com/chenmingyong0423/go-mongox/v2"
	"github.com/chenmingyong0423/go-mongox/v2/aggregator"
	"github.com/chenmingyong0423/go-mongox/v2/creator"
	"github.com/chenmingyong0423/go-mongox/v2/deleter"
	"github.com/chenmingyong0423/go-mongox/v2/finder"
	"github.com/chenmingyong0423/go-mongox/v2/updater"
)

type CollectionNameType string

const (
	FILE_COLLECTION_NAME   CollectionNameType = "files"
	JOBREQ_COLLECTION_NAME CollectionNameType = "job"
	JOBRES_COLLECTION_NAME CollectionNameType = "jobres"
)

// ICollection defines the interface for MongoDB collection operations.
// It provides access to various MongoDB operations through specialized sub-interfaces
// from the go-mongox library, along with additional utility methods.
// The interface is generic and works with any type that implements IMongoDoc.
//
//go:generate mockgen -source=collection.go -destination=../../../mocks/db/mongo/collection.go -package=mocks
type ICollection[T any] interface {
	// Aggregator returns an aggregator for building and executing aggregation pipelines.
	Aggregator() aggregator.IAggregator[T]

	// Creator returns a creator for inserting documents into the collection.
	Creator() creator.ICreator[T]

	// Deleter returns a deleter for removing documents from the collection.
	Deleter() deleter.IDeleter[T]

	// Finder returns a finder for querying and retrieving documents from the collection.
	Finder() finder.IFinder[T]

	// Updater returns an updater for modifying documents in the collection.
	Updater() updater.IUpdater[T]
}

// Collection implements the ICollection interface and provides MongoDB collection operations.
// It wraps a go-mongox Collection to provide a consistent interface for database operations.
// The struct is generic and can work with any document type that implements IMongoDoc.
type Collection[T any] struct {
	xColl *mongox.Collection[T]
}

// Compile-time check to ensure Collection implements ICollection
var _ ICollection[any] = (*Collection[any])(nil)

// Aggregator returns an aggregator instance for building and executing aggregation pipelines.
// This provides access to MongoDB's aggregation framework through the go-mongox library.
func (c *Collection[T]) Aggregator() aggregator.IAggregator[T] {
	return c.xColl.Aggregator()
}

// Creator returns a creator instance for inserting new documents into the collection.
// This provides methods for single and bulk document insertion operations.
func (c *Collection[T]) Creator() creator.ICreator[T] {
	return c.xColl.Creator()
}

// Deleter returns a deleter instance for removing documents from the collection.
// This provides methods for single and bulk document deletion operations.
func (c *Collection[T]) Deleter() deleter.IDeleter[T] {
	return c.xColl.Deleter()
}

// Finder returns a finder instance for querying and retrieving documents from the collection.
// This provides methods for finding single documents, multiple documents, and building complex queries.
func (c *Collection[T]) Finder() finder.IFinder[T] {
	return c.xColl.Finder()
}

// Updater returns an updater instance for modifying documents in the collection.
// This provides methods for single and bulk document update operations.
func (c *Collection[T]) Updater() updater.IUpdater[T] {
	return c.xColl.Updater()
}
