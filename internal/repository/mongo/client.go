package mongo

import (
	"context"
	"fmt"

	"github.com/amirdaaee/TGMon/internal/log"
	"github.com/chenmingyong0423/go-mongox/v2"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
	"go.uber.org/zap"
)

const (
	filesCollectionName       = "files"
	jobReqCollectionName      = "job"
	jobResCollectionName      = "jobres"
	fuseStateCollectionName   = "fuserename"
	workerMediaCollectionName = "workermedia"
	mediaExtCollectionName    = "mediaext"
	createdAtField            = "created_at"
)

// Config holds MongoDB connection settings.
type Config struct {
	Endpoint string
	DBName   string
}

// Client is a connected MongoDB client and database.
type Client struct {
	cl *mongo.Client
	db *mongox.Database
}

// Connect establishes a MongoDB connection and optionally pings the server.
func Connect(ctx context.Context, cfg Config, ping bool) (*Client, error) {
	if cfg.Endpoint == "" {
		return nil, fmt.Errorf("mongo endpoint is required")
	}
	if cfg.DBName == "" {
		return nil, fmt.Errorf("mongo database name is required")
	}
	cl, err := mongo.Connect(options.Client().ApplyURI(cfg.Endpoint))
	if err != nil {
		return nil, fmt.Errorf("error creating mongo client: %w", err)
	}
	if ping {
		if err := cl.Ping(ctx, readpref.Primary()); err != nil {
			if disconnectErr := cl.Disconnect(ctx); disconnectErr != nil {
				log.GetLogger(log.DBModule).With(zap.Error(disconnectErr)).Warn("Failed to disconnect client after ping failure")
			}
			return nil, fmt.Errorf("error pinging mongo: %w", err)
		}
	}
	xCl := mongox.NewClient(cl, &mongox.Config{})
	return &Client{
		cl: cl,
		db: xCl.NewDatabase(cfg.DBName),
	}, nil
}

func newCollection[T any](c *Client, name string) *mongox.Collection[T] {
	return mongox.NewCollection[T](c.db, name)
}
