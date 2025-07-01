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

//go:generate mockgen -source=mongo.go -destination=../../mocks/db/mongo.go -package=mocks
type ICollection[T types.IMongoDoc] interface {
	Aggregator() aggregator.IAggregator[T]
	Creator() creator.ICreator[T]
	Deleter() deleter.IDeleter[T]
	Finder() finder.IFinder[T]
	Updater() updater.IUpdater[T]
	// ...
	Distinct(context.Context, string) ([]string, error)
}
type Collection[T types.IMongoDoc] struct {
	xColl *mongox.Collection[T]
}

var _ ICollection[types.IMongoDoc] = (*Collection[types.IMongoDoc])(nil)

func (c *Collection[T]) Aggregator() aggregator.IAggregator[T] {
	return c.xColl.Aggregator()
}
func (c *Collection[T]) Creator() creator.ICreator[T] {
	return c.xColl.Creator()
}
func (c *Collection[T]) Deleter() deleter.IDeleter[T] {
	return c.xColl.Deleter()
}
func (c *Collection[T]) Finder() finder.IFinder[T] {
	return c.xColl.Finder()
}
func (c *Collection[T]) Updater() updater.IUpdater[T] {
	return c.xColl.Updater()
}
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

// ---
// DBase represents a MongoDB database connection handler.
// It encapsulates connection details and provides basic connectivity operations.
type DBase struct {
	uri         string
	name        string
	client      *mongo.Client
	pingTimeout time.Duration
}

// NewClient establishes a new MongoDB client connection and verifies it.
// Returns:
// - *mongo.Client: Connected MongoDB client instance
// - error: If connection or ping fails
func (d *DBase) NewClient() (*mongo.Client, error) {
	client, err := mongo.Connect(options.Client().ApplyURI(d.uri))
	if err != nil {
		return nil, fmt.Errorf("error connecting to mongo: %w", err)
	}
	ctx, ctxCancel := context.WithTimeout(context.Background(), d.pingTimeout)
	defer ctxCancel()
	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		return nil, fmt.Errorf("error pinging mongo: %w", err)
	}
	return client, nil
}

// Close gracefully disconnects the MongoDB client.
// Returns error if disconnection fails.
func (d *DBase) Close() error {
	if d.client == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return d.client.Disconnect(ctx)
}

// ---
type DatastoreEnum int

const (
	MEDIA_DS DatastoreEnum = iota
	JOB_REQ_DS
	JOB_RES_DS
)

type Datastore[T types.IMongoDoc] struct {
	baseDB         *DBase
	collectionName string
	Collection     *mongox.Collection[T]
}

var DsLock = sync.RWMutex{}
var mediads *Datastore[*types.MediaFileDoc]
var jobreqds *Datastore[*types.JobReqDoc]
var jobresds *Datastore[*types.JobResDoc]

func InitDatastores(dbase *DBase) {
	DsLock.Lock()
	defer DsLock.Unlock()
	mediads = &Datastore[*types.MediaFileDoc]{
		baseDB:         dbase,
		collectionName: "files",
	}
	jobreqds = &Datastore[*types.JobReqDoc]{
		baseDB:         dbase,
		collectionName: "jobs",
	}
	jobresds = &Datastore[*types.JobResDoc]{
		baseDB:         dbase,
		collectionName: "jobres",
	}
}
func GetDatastore[T types.IMongoDoc]() *Datastore[T] {
	DsLock.RLock()
	defer DsLock.RUnlock()
	if mediads == nil || jobreqds == nil || jobresds == nil {
		logrus.Fatal("datastores not initialized")
	}
	var zeroT T
	var an any
	switch any(zeroT).(type) {
	case *types.MediaFileDoc:
		an = any(mediads)
	case *types.JobReqDoc:
		an = any(jobreqds)
	case *types.JobResDoc:
		an = any(jobresds)
	default:
		logrus.Fatalf("unknown datastore type %T", any(*new(T)))
		return nil
	}
	res, ok := an.(*Datastore[T])
	if !ok {
		logrus.Fatalf("failed to cast datastore to %T", any(*new(T)))
		return nil
	}
	return res
}

func NewDatabase(uri string, name string, pingTimeout time.Duration) (*DBase, error) {
	d := DBase{
		uri:         uri,
		name:        name,
		pingTimeout: pingTimeout,
	}
	dCl, err := d.NewClient()
	if err != nil {
		return nil, err
	}
	d.client = dCl
	return &d, nil
}

// ...
