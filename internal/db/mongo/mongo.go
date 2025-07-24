package mongo

import (
	"context"

	"github.com/chenmingyong0423/go-mongox/v2"
)

// IMongoClient defines the interface for MongoDB client operations.
// This interface provides methods for managing MongoDB client connections
// and creating database instances. It is designed to be mockable for testing purposes.
//
//go:generate mockgen -source=mongo.go -destination=../../../mocks/db/mongo/mongo.go -package=mocks
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
	return &Database{Database: c.xCl.NewDatabase(name)}
}

var _ IMongoClient = (*MongoClient)(nil)

// IDatabase defines the interface for MongoDB database operations.
// This interface provides access to database-level functionality and is designed
// to be extensible for future database operations.
//
//go:generate mockgen -source=mongo.go -destination=../../../mocks/db/mongo/mongo.go -package=mocks
type IDatabase interface {
	// Future database-level operations can be added here
}

// Database implements the IDatabase interface and wraps a go-mongox database.
// It provides the foundation for database-level operations.
type Database struct {
	*mongox.Database
}

var _ IDatabase = (*Database)(nil)
