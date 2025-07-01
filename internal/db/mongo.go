package db

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/amirdaaee/TGMon/internal/log"
	"github.com/amirdaaee/TGMon/internal/types"
	"github.com/chenmingyong0423/go-mongox/v2"
	"github.com/chenmingyong0423/go-mongox/v2/aggregator"
	"github.com/chenmingyong0423/go-mongox/v2/creator"
	"github.com/chenmingyong0423/go-mongox/v2/deleter"
	"github.com/chenmingyong0423/go-mongox/v2/finder"
	"github.com/chenmingyong0423/go-mongox/v2/updater"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
)

//go:generate mockgen -source=mongo.go -destination=../../mocks/db/mongo.go -package=mocks
type IClient interface {
	Disconnect(context.Context) error
	NewDatabase(string) IDatabase
}
type Client struct {
	xCl *mongox.Client
}

func (c *Client) Disconnect(ctx context.Context) error {
	return c.xCl.Disconnect(ctx)
}
func (c *Client) NewDatabase(name string) IDatabase {
	return &Database{xDb: c.xCl.NewDatabase(name)}
}

var _ IClient = (*Client)(nil)

//go:generate mockgen -source=mongo.go -destination=../../mocks/db/mongo.go -package=mocks
type IDatabase interface {
}
type Database struct {
	xDb *mongox.Database
}

var _ IDatabase = (*Database)(nil)

// ICollection defines the interface for MongoDB collection operations.
// It provides access to various MongoDB operations through specialized sub-interfaces
// from the go-mongox library, along with additional utility methods.
// The interface is generic and works with any type that implements IMongoDoc.
//
//go:generate mockgen -source=mongo.go -destination=../../mocks/db/mongo.go -package=mocks
type ICollection[T types.IMongoDoc] interface {
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
	// Distinct retrieves distinct values for a specified field across the collection.
	// Returns a slice of unique string values for the given field key.
	Distinct(context.Context, string) ([]string, error)
}

// Collection implements the ICollection interface and provides MongoDB collection operations.
// It wraps a go-mongox Collection to provide a consistent interface for database operations.
// The struct is generic and can work with any document type that implements IMongoDoc.
type Collection[T types.IMongoDoc] struct {
	xColl *mongox.Collection[T]
}

// Compile-time check to ensure Collection implements ICollection
var _ ICollection[types.IMongoDoc] = (*Collection[types.IMongoDoc])(nil)

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

// Distinct retrieves all unique values for a specified field across the collection.
// This is useful for finding all possible values of a particular field without duplicates.
//
// Parameters:
//   - ctx: Context for the operation
//   - key: The field name to find distinct values for
//
// Returns a slice of unique string values and any error encountered.
func (c *Collection[T]) Distinct(ctx context.Context, key string) ([]string, error) {
	res := c.Finder().Distinct(ctx, key)
	if err := res.Err(); err != nil {
		return nil, fmt.Errorf("error calling mongo distinct: %w", err)
	}
	resData := []string{}
	if err := res.Decode(&resData); err != nil {
		return nil, fmt.Errorf("error decoding distinct result: %w", err)
	}
	return resData, nil
}

// DBase represents a MongoDB database connection handler.
// It encapsulates connection details and provides basic connectivity operations.
// This struct manages the lifecycle of a MongoDB client connection and provides
// methods for establishing, testing, and closing database connections.
type DBase struct {
	uri         string
	name        string
	client      IClient
	db          IDatabase
	pingTimeout time.Duration
}

// Close gracefully disconnects the MongoDB client using the provided context.
// This method allows the caller to control the timeout and cancellation behavior
// of the disconnect operation. If the client is nil, this method is a no-op.
//
// Parameters:
//   - ctx: Context to control the disconnect timeout and cancellation
//
// Returns an error if the disconnect operation fails.
func (d *DBase) Close(ctx context.Context) error {
	if d.client == nil {
		return nil
	}
	return d.client.Disconnect(ctx)
}

// CloseWithTimeout gracefully disconnects the MongoDB client with a specified timeout.
// This is a convenience method that creates a timeout context internally and calls Close.
// It's useful when you want to specify a disconnect timeout without managing the context yourself.
//
// Parameters:
//   - timeout: Maximum duration to wait for the disconnect operation to complete
//
// Returns an error if the disconnect operation fails or times out.
func (d *DBase) CloseWithTimeout(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return d.Close(ctx)
}

// DatastoreType represents the type of datastore available in the system.
// This enum-like type is used to identify and retrieve specific datastores
// by their logical type rather than their generic parameter type.
type DatastoreType string

const (
	// MediaDatastore represents the datastore for media file documents
	MediaDatastore DatastoreType = "media"

	// JobReqDatastore represents the datastore for job request documents
	JobReqDatastore DatastoreType = "jobreq"

	// JobResDatastore represents the datastore for job result documents
	JobResDatastore DatastoreType = "jobres"
)

// DatastoreManager manages all datastores in a thread-safe manner.
// It provides centralized access to different datastore instances and ensures
// proper initialization and lifecycle management. This manager follows the
// dependency injection pattern and is the recommended way to access datastores
// in production applications.
type DatastoreManager struct {
	mu     sync.RWMutex
	baseDB *DBase
}

// NewDatastoreManager creates a new DatastoreManager instance with the provided database connection.
// The manager starts in an uninitialized state and requires a call to InitDatastores
// before it can serve datastore requests.
//
// Parameters:
//   - dbase: The database connection to use for all managed datastores
//
// Returns a new DatastoreManager instance.
func NewDatastoreManager() *DatastoreManager {
	return &DatastoreManager{}
}

// InitDatastores initializes all managed datastores with their respective collection configurations.
// This method is thread-safe and idempotent - calling it multiple times will not cause issues.
// It sets up datastores for media files, job requests, and job results with their
// corresponding MongoDB collection names.
//
// Returns an error if initialization fails, though the current implementation
// cannot fail and always returns nil.
func (dm *DatastoreManager) InitDatastores(
	uri string,
	dbName string,
	pingTimeout time.Duration,
	mongoClFactory func(uri string, pingTimeout time.Duration) (IClient, error),
	mongoDbFactory func(client IClient, name string) (IDatabase, error),
) error {
	dm.mu.Lock()
	defer dm.mu.Unlock()
	if dm.baseDB != nil {
		dm.getLogger("InitDatastores").Warn("datastores already initialized. re-initializing...")
	}
	dm.baseDB = &DBase{
		uri:         uri,
		name:        dbName,
		pingTimeout: pingTimeout,
	}
	if mongoClFactory == nil {
		mongoClFactory = defaultMongoClientFactory
	}
	if mongoDbFactory == nil {
		mongoDbFactory = defaultMongoDatabaseFactory
	}
	// ...
	xCl, err := mongoClFactory(dm.baseDB.uri, dm.baseDB.pingTimeout)
	if err != nil {
		return fmt.Errorf("failed to create mongo client: %w", err)
	}
	dm.baseDB.client = xCl
	// ...
	xDb, err := mongoDbFactory(dm.baseDB.client, dm.baseDB.name)
	if err != nil {
		return fmt.Errorf("failed to create mongo database: %w", err)
	}
	dm.baseDB.db = xDb
	return nil
}

func (dm *DatastoreManager) getLogger(fn string) *logrus.Entry {
	return log.GetLogger(log.DBModule).WithField("fn", fmt.Sprintf("%T.%s", dm, fn))
}

func defaultMongoClientFactory(uri string, pingTimeout time.Duration) (IClient, error) {
	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		return nil, fmt.Errorf("error connecting to mongo: %w", err)
	}

	ctx, ctxCancel := context.WithTimeout(context.Background(), pingTimeout)
	defer ctxCancel()

	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		// Ensure client is disconnected if ping fails to prevent resource leak
		if disconnectErr := client.Disconnect(context.Background()); disconnectErr != nil {
			logrus.Warnf("Failed to disconnect client after ping failure: %v", disconnectErr)
		}
		return nil, fmt.Errorf("error pinging mongo: %w", err)
	}
	xCl := mongox.NewClient(client, &mongox.Config{})
	return &Client{xCl: xCl}, nil
}
func defaultMongoDatabaseFactory(client IClient, name string) (IDatabase, error) {
	return client.NewDatabase(name), nil
}
func defaultMongoCollectionFactory[T types.IMongoDoc](db IDatabase, name string) (ICollection[T], error) {
	xCol := mongox.NewCollection[T](db.(*mongox.Database), name)
	return &Collection[T]{xColl: xCol}, nil
}

var defaultDsManager = NewDatastoreManager()

func GetCollection[T types.IMongoDoc](dsMan *DatastoreManager, collectionFactory func(db IDatabase, name string) (ICollection[T], error)) (ICollection[T], error) {
	if dsMan == nil {
		dsMan = defaultDsManager
	}
	dsMan.mu.RLock()
	defer dsMan.mu.RUnlock()
	if dsMan.baseDB == nil {
		return nil, fmt.Errorf("datastores not initialized")
	}
	if collectionFactory == nil {
		collectionFactory = defaultMongoCollectionFactory[T]
	}
	var zeroT T
	var collName string
	switch any(zeroT).(type) {
	case *types.MediaFileDoc:
		collName = "files"
	case *types.JobReqDoc:
		collName = "jobs"
	case *types.JobResDoc:
		collName = "jobres"
	default:
		return nil, fmt.Errorf("unknown datastore type %T", any(*new(T)))
	}
	return collectionFactory(dsMan.baseDB.db, collName)
}
