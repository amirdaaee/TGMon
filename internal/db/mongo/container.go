package mongo

import (
	"context"
	"fmt"

	"github.com/amirdaaee/TGMon/internal/types"
	"github.com/chenmingyong0423/go-mongox/v2"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
)

// IMongoContainer defines the interface for accessing MongoDB client, database, and collections.
// It provides methods to retrieve the MongoDB client, database, and specific collections used in the application.
//
//go:generate mockgen -source=container.go -destination=../../../mocks/db/mongo/container.go -package=mocks
type IMongoContainer interface {
	GetMongoClient() IMongoClient
	GetMongoDb() IDatabase
	GetJobReqCollection() ICollection[types.JobReqDoc]
	GetJobResCollection() ICollection[types.JobResDoc]
	GetMediaFileCollection() ICollection[types.MediaFileDoc]
}

// MongoContainer implements the IMongoContainer interface and holds references to the MongoDB client, database, and helper structs.
type MongoContainer struct {
	cl          *mongo.Client
	mongoClient *MongoClient
	db          *Database
}

// GetMongoClient returns the MongoDB client used for database operations.
func (c *MongoContainer) GetMongoClient() IMongoClient {
	return c.mongoClient
}

// GetMongoDb returns the database instance used for MongoDB operations.
func (c *MongoContainer) GetMongoDb() IDatabase {
	return c.db
}

// GetJobReqCollection returns the collection for job request documents.
func (c *MongoContainer) GetJobReqCollection() ICollection[types.JobReqDoc] {
	xCol := mongox.NewCollection[types.JobReqDoc](c.db.Database, string(JOBREQ_COLLECTION_NAME))
	return &Collection[types.JobReqDoc]{xColl: xCol}
}

// GetJobResCollection returns the collection for job result documents.
func (c *MongoContainer) GetJobResCollection() ICollection[types.JobResDoc] {
	xCol := mongox.NewCollection[types.JobResDoc](c.db.Database, string(JOBRES_COLLECTION_NAME))
	return &Collection[types.JobResDoc]{xColl: xCol}
}

// GetMediaFileCollection returns the collection for media file documents.
func (c *MongoContainer) GetMediaFileCollection() ICollection[types.MediaFileDoc] {
	xCol := mongox.NewCollection[types.MediaFileDoc](c.db.Database, string(FILE_COLLECTION_NAME))
	return &Collection[types.MediaFileDoc]{xColl: xCol}
}

var _ IMongoContainer = (*MongoContainer)(nil)

// MongoContainerConfig holds configuration for connecting to a MongoDB instance.
type MongoContainerConfig struct {
	// Endpoint is the MongoDB server URI
	Endpoint string
	// DbName is the name of the database to use
	DbName string
}

// Validate checks if the MongoContainerConfig has all required fields set.
func (c *MongoContainerConfig) Validate() error {
	if c.Endpoint == "" {
		return fmt.Errorf("mongo endpoint is required")
	}
	if c.DbName == "" {
		return fmt.Errorf("mongo database name is required")
	}
	return nil
}

// NewMongoContainer creates and initializes a new MongoDB container.
// It establishes a connection to MongoDB, optionally pings the server, and prepares the client, database, and collections.
//
// Parameters:
//   - ctx: Context for controlling connection and ping operations
//   - config: MongoContainerConfig with endpoint and database name
//   - ping: Whether to ping the server after connecting
//
// Returns an IMongoContainer and an error if connection or ping fails.
func NewMongoContainer(ctx context.Context, config MongoContainerConfig, ping bool) (IMongoContainer, error) {
	cl, err := mongo.Connect(options.Client().ApplyURI(config.Endpoint))
	if err != nil {
		return nil, fmt.Errorf("error creating mongo client: %w", err)
	}
	// ...
	if ping {
		if err := cl.Ping(ctx, readpref.Primary()); err != nil {
			// Ensure client is disconnected if ping fails to prevent resource leak
			if disconnectErr := cl.Disconnect(ctx); disconnectErr != nil {
				logrus.Warnf("Failed to disconnect client after ping failure: %v", disconnectErr)
			}
			return nil, fmt.Errorf("error pinging mongo: %w", err)
		}
	}

	// ...
	mCl := MongoClient{
		xCl: mongox.NewClient(cl, &mongox.Config{}),
	}
	return &MongoContainer{
		cl:          cl,
		mongoClient: &mCl,
		db:          mCl.NewDatabase(config.DbName).(*Database),
	}, nil
}
