package db

import (
	"context"
	"fmt"
	"sync"
	"time"

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

// IMongoClient defines the interface for MongoDB client operations.
// This interface provides methods for managing MongoDB client connections
// and creating database instances. It is designed to be mockable for testing purposes.
//
//go:generate mockgen -source=mongo.go -destination=../../mocks/db/mongo.go -package=mocks
type IMongoClient interface {
	// Disconnect gracefully closes the MongoDB client connection.
	// The context parameter controls the timeout and cancellation behavior.
	Disconnect(context.Context) error

	// NewDatabase creates a new database instance with the specified name.
	// Returns an IDatabase interface for database-level operations.
	NewDatabase(string) IDatabase
}

// MongoClient implements the IMongoClient interface and wraps a go-mongox client.
// It provides the connection management functionality for MongoDB operations.
type MongoClient struct {
	xCl *mongox.Client
}

// Disconnect gracefully closes the MongoDB client connection.
// This method delegates to the underlying go-mongox client's Disconnect method.
//
// Parameters:
//   - ctx: Context to control the disconnect timeout and cancellation
//
// Returns an error if the disconnect operation fails.
func (c *MongoClient) Disconnect(ctx context.Context) error {
	return c.xCl.Disconnect(ctx)
}

// NewDatabase creates a new database instance with the specified name.
// This method wraps the go-mongox client's NewDatabase functionality.
//
// Parameters:
//   - name: The name of the database to create
//
// Returns an IDatabase interface for database-level operations.
func (c *MongoClient) NewDatabase(name string) IDatabase {
	return &Database{xDb: c.xCl.NewDatabase(name)}
}

var _ IMongoClient = (*MongoClient)(nil)

// IDatabase defines the interface for MongoDB database operations.
// This interface provides access to database-level functionality and is designed
// to be extensible for future database operations.
//
//go:generate mockgen -source=mongo.go -destination=../../mocks/db/mongo.go -package=mocks
type IDatabase interface {
	// Future database-level operations can be added here
}

// Database implements the IDatabase interface and wraps a go-mongox database.
// It provides the foundation for database-level operations.
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
	client      IMongoClient
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

// CollectionRegistry manages all datastores in a thread-safe manner.
// It provides centralized access to different datastore instances and ensures
// proper initialization and lifecycle management. This manager follows the
// dependency injection pattern and is the recommended way to access datastores
// in production applications.
type CollectionRegistry struct {
	mu    sync.RWMutex
	dbase *DBase
}

// NewCollectionRegistry creates a new DatastoreManager instance with the provided database connection.
// The manager starts in an uninitialized state and requires a call to InitDatastores
// before it can serve datastore requests.
//
// Parameters:
//   - dbase: The database connection to use for all managed datastores
//
// Returns a new DatastoreManager instance.
func NewCollectionRegistry() *CollectionRegistry {
	return &CollectionRegistry{}
}

// defaultMongoClientFactory creates a new MongoDB client with the provided connection URI.
// This factory function establishes a connection to MongoDB, performs a ping test to verify
// connectivity, and returns a wrapped client instance. If the ping fails, it ensures
// proper cleanup by disconnecting the client to prevent resource leaks.
//
// Parameters:
//   - uri: MongoDB connection URI (e.g., "mongodb://localhost:27017")
//   - pingTimeout: Maximum duration to wait for the ping operation to complete
//
// Returns a new IMongoClient instance and any error encountered during connection or ping.
func defaultMongoClientFactory(uri string, pingTimeout time.Duration) (IMongoClient, error) {
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
	return &MongoClient{xCl: xCl}, nil
}

// defaultMongoDatabaseFactory creates a new database instance using the provided client.
// This factory function wraps the client's NewDatabase method and provides a consistent
// interface for database creation. The current implementation cannot fail and always
// returns nil for the error.
//
// Parameters:
//   - client: The MongoDB client to use for database creation
//   - name: The name of the database to create
//
// Returns a new IDatabase instance and nil error.
func defaultMongoDatabaseFactory(client IMongoClient, name string) (IDatabase, error) {
	return client.NewDatabase(name), nil
}

// defaultMongoCollectionFactory creates a new collection instance for the specified document type.
// This factory function creates a go-mongox collection and wraps it in a Collection struct
// that implements the ICollection interface. The function is generic and works with any
// type that implements IMongoDoc.
//
// Parameters:
//   - db: The database instance to create the collection in
//   - name: The name of the collection to create
//
// Returns a new ICollection instance for the specified document type.
// defaultMongoCollectionFactory creates a new collection instance for the specified document type.
// This factory function creates a go-mongox collection and wraps it in a Collection struct
// that implements the ICollection interface. The function is generic and works with any
// type that implements IMongoDoc.
//
// Parameters:
//   - db: The database instance to create the collection in
//   - name: The name of the collection to create
//
// Returns a new ICollection instance for the specified document type.
func defaultMongoCollectionFactory[T types.IMongoDoc](db IDatabase, name string) ICollection[T] {
	xCol := mongox.NewCollection[T](db.(*mongox.Database), name)
	return &Collection[T]{xColl: xCol}
}

var DefaultCollectionRegistry = NewCollectionRegistry()

// InitDatastores initializes all managed datastores with their respective collection configurations.
// This method is thread-safe and idempotent - calling it multiple times will not cause issues.
// It sets up datastores for media files, job requests, and job results with their
// corresponding MongoDB collection names.
//
// Returns an error if initialization fails, though the current implementation
// cannot fail and always returns nil.
func InitDatastoreRegistry(
	reg *CollectionRegistry,
	uri string,
	dbName string,
	pingTimeout time.Duration,
	mongoClFactory func(uri string, pingTimeout time.Duration) (IMongoClient, error),
	mongoDbFactory func(client IMongoClient, name string) (IDatabase, error),
	collectionFactory func(db IDatabase, name string) any,
) error {
	if reg == nil {
		reg = DefaultCollectionRegistry
	}
	reg.mu.Lock()
	defer reg.mu.Unlock()
	if reg.dbase != nil {
		logrus.Warn("datastores already initialized. re-initializing...")
	}

	reg.dbase = &DBase{
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
	xCl, err := mongoClFactory(reg.dbase.uri, reg.dbase.pingTimeout)
	if err != nil {
		return fmt.Errorf("failed to create mongo client: %w", err)
	}
	reg.dbase.client = xCl
	// ...
	xDb, err := mongoDbFactory(reg.dbase.client, reg.dbase.name)
	if err != nil {
		return fmt.Errorf("failed to create mongo database: %w", err)
	}
	reg.dbase.db = xDb
	return nil
}

// GetCollection returns a MongoDB collection for the specified document type from the given registry.
// This function determines the collection name based on the type of the document and uses the provided
// collectionFactory to create the collection instance. If the registry is nil, it uses the default registry.
// If the collectionFactory is nil, it uses the defaultMongoCollectionFactory.
//
// Parameters:
//   - reg: The collection registry to use (if nil, uses DefaultCollectionRegistry)
//   - collectionFactory: Optional factory function to create the collection (if nil, uses default)
//
// Returns an ICollection instance for the specified document type. Panics if the registry is not initialized
// or if the document type is unknown.
func GetCollection[T types.IMongoDoc](reg *CollectionRegistry, collectionFactory func(db IDatabase, name string) ICollection[T]) ICollection[T] {
	if reg == nil {
		reg = DefaultCollectionRegistry
	}
	reg.mu.RLock()
	defer reg.mu.RUnlock()
	if reg.dbase == nil {
		logrus.Fatal("datastores not initialized")
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
		logrus.Fatalf("unknown datastore type %T", any(*new(T)))
	}
	return collectionFactory(reg.dbase.db, collName)
}
