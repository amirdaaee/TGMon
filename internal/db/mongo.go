package db

import (
	"context"
	"reflect"
	"sync"

	"github.com/amirdaaee/TGMon/internal/errs"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type IMongoClient interface {
	Disconnect(context.Context) error
	Database(name string, opts ...*options.DatabaseOptions) *mongo.Database
}
type IMongoCollection interface {
	Aggregate(ctx context.Context, pipeline interface{}, opts ...*options.AggregateOptions) (*mongo.Cursor, error)
	BulkWrite(ctx context.Context, models []mongo.WriteModel, opts ...*options.BulkWriteOptions) (*mongo.BulkWriteResult, error)
	CountDocuments(ctx context.Context, filter interface{}, opts ...*options.CountOptions) (int64, error)
	Database() *mongo.Database
	DeleteMany(ctx context.Context, filter interface{}, opts ...*options.DeleteOptions) (*mongo.DeleteResult, error)
	DeleteOne(ctx context.Context, filter interface{}, opts ...*options.DeleteOptions) (*mongo.DeleteResult, error)
	Distinct(ctx context.Context, fieldName string, filter interface{}, opts ...*options.DistinctOptions) ([]interface{}, error)
	Drop(ctx context.Context) error
	EstimatedDocumentCount(ctx context.Context, opts ...*options.EstimatedDocumentCountOptions) (int64, error)
	Find(ctx context.Context, filter interface{}, opts ...*options.FindOptions) (cur *mongo.Cursor, err error)
	FindOne(ctx context.Context, filter interface{}, opts ...*options.FindOneOptions) *mongo.SingleResult
	FindOneAndDelete(ctx context.Context, filter interface{}, opts ...*options.FindOneAndDeleteOptions) *mongo.SingleResult
	FindOneAndReplace(ctx context.Context, filter interface{}, replacement interface{}, opts ...*options.FindOneAndReplaceOptions) *mongo.SingleResult
	FindOneAndUpdate(ctx context.Context, filter interface{}, update interface{}, opts ...*options.FindOneAndUpdateOptions) *mongo.SingleResult
	Indexes() mongo.IndexView
	InsertMany(ctx context.Context, documents []interface{}, opts ...*options.InsertManyOptions) (*mongo.InsertManyResult, error)
	InsertOne(ctx context.Context, document interface{}, opts ...*options.InsertOneOptions) (*mongo.InsertOneResult, error)
	Name() string
	ReplaceOne(ctx context.Context, filter interface{}, replacement interface{}, opts ...*options.ReplaceOptions) (*mongo.UpdateResult, error)
	SearchIndexes() mongo.SearchIndexView
	UpdateByID(ctx context.Context, id interface{}, update interface{}, opts ...*options.UpdateOptions) (*mongo.UpdateResult, error)
	UpdateMany(ctx context.Context, filter interface{}, update interface{}, opts ...*options.UpdateOptions) (*mongo.UpdateResult, error)
	UpdateOne(ctx context.Context, filter interface{}, update interface{}, opts ...*options.UpdateOptions) (*mongo.UpdateResult, error)
	Watch(ctx context.Context, pipeline interface{}, opts ...*options.ChangeStreamOptions) (*mongo.ChangeStream, error)
}
type IMongo interface {
	GetCollection(cl IMongoClient) IMongoCollection
	GetClient() (IMongoClient, error)
}
type Mongo struct {
	IMng                IMongo
	DBUri               string
	DBName              string
	MediaCollectionName string
	JobCollectionName   string
}

func (m *Mongo) GetClient() (IMongoClient, error) {
	serverAPI := options.ServerAPI(options.ServerAPIVersion1)
	opts := options.Client().ApplyURI(m.DBUri).SetServerAPIOptions(serverAPI)
	cl, err := mongo.Connect(context.TODO(), opts)
	if err != nil {
		return nil, err
	}
	return cl, nil
}
func (m *Mongo) assertClient(cl IMongoClient) (IMongoClient, func(context.Context) error, error) {
	if cl != nil {
		return cl, func(context.Context) error { return nil }, nil
	}
	cl, err := m.GetClient()
	if err != nil {
		return nil, nil, err
	}
	return cl, cl.Disconnect, nil
}
func (m *Mongo) DocAdd(ctx context.Context, doc interface{}, cl IMongoClient) (*mongo.InsertOneResult, error) {
	cl, disc, err := m.assertClient(cl)
	if err != nil {
		return nil, err
	}
	defer disc(ctx)
	return m.IMng.GetCollection(cl).InsertOne(ctx, doc)
}
func (m *Mongo) DocAddMany(ctx context.Context, doc []interface{}, cl IMongoClient) (*mongo.InsertManyResult, error) {
	cl, disc, err := m.assertClient(cl)
	if err != nil {
		return nil, err
	}
	defer disc(ctx)
	return m.IMng.GetCollection(cl).InsertMany(ctx, doc)
}
func (m *Mongo) DocGetById(ctx context.Context, docID string, result interface{}, cl IMongoClient) error {
	cl, disc, err := m.assertClient(cl)
	if err != nil {
		return err
	}
	defer disc(ctx)
	filter, err := FilterById(docID)
	if err != nil {
		return err
	}
	return m.IMng.GetCollection(cl).FindOne(ctx, filter).Decode(result)
}
func (m *Mongo) DocDelById(ctx context.Context, docID string, cl IMongoClient) error {
	cl, disc, err := m.assertClient(cl)
	if err != nil {
		return err
	}
	defer disc(ctx)
	filter, err := FilterById(docID)
	if err != nil {
		return err
	}
	_, err = m.IMng.GetCollection(cl).DeleteOne(ctx, filter)
	return err
}
func (m *Mongo) DocGetAll(ctx context.Context, result interface{}, cl IMongoClient, opts ...*options.FindOptions) error {
	cl, disc, err := m.assertClient(cl)
	if err != nil {
		return err
	}
	defer disc(ctx)
	cur_, err := m.IMng.GetCollection(cl).Find(ctx, bson.D{}, opts...)
	if err != nil {
		return err
	}
	if err = cur_.All(ctx, result); err != nil {
		return err
	}
	return nil
}
func (m *Mongo) DocGetNeighbour(ctx context.Context, mediaDoc MediaFileDoc, cl IMongoClient) (*MediaFileDoc, *MediaFileDoc, error) {
	ll := logrus.WithField("module", "DocGetNeighbour").WithField("target", mediaDoc.ID)
	cl, disc, err := m.assertClient(cl)
	if err != nil {
		return nil, nil, err
	}
	defer disc(ctx)
	collection := m.IMng.GetCollection(cl)
	// ...
	prevOpts := options.FindOne().SetSort(bson.D{{Key: "DateAdded", Value: -1}, {Key: "FileID", Value: 1}})
	nextOpts := options.FindOne().SetSort(bson.D{{Key: "DateAdded", Value: 1}, {Key: "FileID", Value: -1}})
	prevFilter := bson.M{"DateAdded": bson.M{"$lt": mediaDoc.DateAdded}}
	nextFilter := bson.M{"DateAdded": bson.M{"$gt": mediaDoc.DateAdded}}
	wg := sync.WaitGroup{}
	wg.Add(2)
	var nextDoc, prevDoc MediaFileDoc
	go func() {
		defer wg.Done()
		if err := collection.FindOne(ctx, prevFilter, prevOpts).Decode(&prevDoc); err != nil && err != mongo.ErrNoDocuments {
			ll.WithError(err).Error("error getting previous doc")
		}
	}()
	go func() {
		defer wg.Done()
		if err := collection.FindOne(ctx, nextFilter, nextOpts).Decode(&nextDoc); err != nil && err != mongo.ErrNoDocuments {
			ll.WithError(err).Error("error getting next doc")
		}
	}()
	wg.Wait()
	return &prevDoc, &nextDoc, nil
}

func (m *Mongo) GetMediaMongo() *Mongo {
	mng := *m
	mng.IMng = &mediaMongo{&mng}
	return &mng
}
func (m *Mongo) GetJobMongo() *Mongo {
	mng := *m
	mng.IMng = &jobMongo{&mng}
	return &mng
}

// ....
type mediaMongo struct {
	*Mongo
}
type jobMongo struct {
	*Mongo
}

func (m *mediaMongo) GetCollection(cl IMongoClient) IMongoCollection {
	return cl.Database(m.DBName).Collection(m.MediaCollectionName)
}
func (m *jobMongo) GetCollection(cl IMongoClient) IMongoCollection {
	return cl.Database(m.DBName).Collection(m.JobCollectionName)
}

// ...
func FilterById(docID string) (*bson.D, error) {
	docId, err := primitive.ObjectIDFromHex(docID)
	if err != nil {
		return nil, err
	}
	return &bson.D{{Key: "_id", Value: docId}}, err
}

// ---
type IDataStore[T IMongoDoc] interface {
	GetCollection(cl IMongoClient) IMongoCollection
	Create(ctx context.Context, doc T, cl IMongoClient) (T, errs.IMongoErr)
	Find(ctx context.Context, filter *primitive.D, cl IMongoClient) (T, errs.IMongoErr)
	FindMany(ctx context.Context, filter *primitive.D, cl IMongoClient) ([]T, errs.IMongoErr)
	Delete(ctx context.Context, filter *primitive.D, cl IMongoClient) errs.IMongoErr
	DeleteMany(ctx context.Context, filter *primitive.D, cl IMongoClient) errs.IMongoErr
	Replace(ctx context.Context, filter *primitive.D, doc T, cl IMongoClient) (T, errs.IMongoErr)
}

type DataStore[T IMongoDoc] struct {
	dbName            string
	collectionName    string
	collectionFactory func(IMongoClient) IMongoCollection
}

func (m *DataStore[T]) GetCollection(cl IMongoClient) IMongoCollection {
	if m.collectionFactory != nil {
		return m.collectionFactory(cl)
	}
	return cl.Database(m.dbName).Collection(m.collectionName)
}
func (m *DataStore[T]) Create(ctx context.Context, doc T, cl IMongoClient) (T, errs.IMongoErr) {
	res, err := m.GetCollection(cl).InsertOne(ctx, doc)
	if err != nil {
		return doc, errs.NewMongoOpErr(err)
	}
	id := res.InsertedID.(primitive.ObjectID)
	doc.SetID(id)
	return doc, nil
}
func (m *DataStore[T]) Find(ctx context.Context, filter *primitive.D, cl IMongoClient) (T, errs.IMongoErr) {
	docs, err := m.FindMany(ctx, filter, cl)
	if err != nil {
		return *new(T), errs.NewMongoOpErr(err)
	}
	if len(docs) == 0 {
		return *new(T), errs.NewMongoObjectNotfound(*filter)
	}
	if len(docs) > 1 {
		return *new(T), errs.NewMongoMultipleObjectfound(*filter)
	}
	return docs[0], nil
}
func (m *DataStore[T]) FindMany(ctx context.Context, filter *primitive.D, cl IMongoClient) ([]T, errs.IMongoErr) {
	cursor, err := m.GetCollection(cl).Find(ctx, filter)
	if err != nil {
		return nil, errs.NewMongoOpErr(err)
	}

	res := []T{}
	for cursor.Next(ctx) {
		var r T
		err := cursor.Decode(&r)
		if err != nil {
			return res, errs.NewMongoUnMarshalErr(err)
		}
		res = append(res, r)
	}
	return res, nil
}
func (m *DataStore[T]) Delete(ctx context.Context, filter *primitive.D, cl IMongoClient) errs.IMongoErr {
	if res, err := m.GetCollection(cl).DeleteOne(ctx, filter); err != nil {
		return errs.NewMongoOpErr(err)
	} else if res.DeletedCount == 0 {
		return errs.NewMongoObjectNotfound(*filter)
	}
	return nil
}
func (m *DataStore[T]) DeleteMany(ctx context.Context, filter *primitive.D, cl IMongoClient) errs.IMongoErr {
	if res, err := m.GetCollection(cl).DeleteMany(ctx, filter); err != nil {
		return errs.NewMongoOpErr(err)
	} else if res.DeletedCount == 0 {
		return errs.NewMongoObjectNotfound(*filter)
	}
	return nil
}
func (m *DataStore[T]) Replace(ctx context.Context, filter *primitive.D, doc T, cl IMongoClient) (T, errs.IMongoErr) {
	res, err := m.GetCollection(cl).ReplaceOne(ctx, filter, doc)
	if err != nil {
		return doc, err
	}
	if res.MatchedCount == 0 {
		return doc, errs.NewMongoObjectNotfound(*filter)
	}
	return doc, nil
}
func (m *DataStore[T]) WithCollectionFactory(factory func(IMongoClient) IMongoCollection) IDataStore[T] {
	m.collectionFactory = factory
	return m
}

func NewDatastore[T IMongoDoc](dbName string, collectionName string) IDataStore[T] {
	return &DataStore[T]{
		dbName:         dbName,
		collectionName: collectionName,
	}
}

// ...
type DatastoreEnum int

const (
	MEDIA_DS DatastoreEnum = iota
	JOB_DS
)

// ...
func GetIDFilter(id primitive.ObjectID) *primitive.D {
	return &bson.D{{Key: "_id", Value: id}}
}

func MarshalOmitEmpty(doc IMongoDoc) (*primitive.D, errs.IMongoErr) {
	var filteredData = make(map[string]interface{})
	vInd := reflect.ValueOf(doc)
	v := reflect.Indirect(vInd)
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		if !field.IsZero() {
			filteredData[v.Type().Field(i).Tag.Get("bson")] = field.Interface()
		}
	}
	marsh, err := bson.Marshal(filteredData)
	if err != nil {
		return nil, errs.NewMongoMarshalErr(err)
	}
	unmarsh := new(bson.D)
	if err := bson.Unmarshal(marsh, unmarsh); err != nil {
		return nil, errs.NewMongoUnMarshalErr(err)
	}
	return unmarsh, nil
}
