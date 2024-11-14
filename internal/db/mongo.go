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

type iMongo interface {
	GetCollection(cl *mongo.Client) *mongo.Collection
}
type Mongo struct {
	IMng                iMongo
	DBUri               string
	DBName              string
	MediaCollectionName string
	JobCollectionName   string
}

func (m *Mongo) GetClient() (*mongo.Client, error) {
	serverAPI := options.ServerAPI(options.ServerAPIVersion1)
	opts := options.Client().ApplyURI(m.DBUri).SetServerAPIOptions(serverAPI)
	cl, err := mongo.Connect(context.TODO(), opts)
	if err != nil {
		return nil, err
	}
	return cl, nil
}

func (m *Mongo) assertClient(cl *mongo.Client) (*mongo.Client, func(context.Context) error, error) {
	if cl != nil {
		return cl, func(context.Context) error { return nil }, nil
	}
	cl, err := m.GetClient()
	if err != nil {
		return nil, nil, err
	}
	return cl, cl.Disconnect, nil
}
func (m *Mongo) DocAdd(ctx context.Context, doc interface{}, cl *mongo.Client) (*mongo.InsertOneResult, error) {
	cl, disc, err := m.assertClient(cl)
	if err != nil {
		return nil, err
	}
	defer disc(ctx)
	return m.IMng.GetCollection(cl).InsertOne(ctx, doc)
}
func (m *Mongo) DocAddMany(ctx context.Context, doc []interface{}, cl *mongo.Client) (*mongo.InsertManyResult, error) {
	cl, disc, err := m.assertClient(cl)
	if err != nil {
		return nil, err
	}
	defer disc(ctx)
	return m.IMng.GetCollection(cl).InsertMany(ctx, doc)
}
func (m *Mongo) DocGetById(ctx context.Context, docID string, result interface{}, cl *mongo.Client) error {
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
func (m *Mongo) DocDelById(ctx context.Context, docID string, cl *mongo.Client) error {
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
func (m *Mongo) DocGetAll(ctx context.Context, result interface{}, cl *mongo.Client, opts ...*options.FindOptions) error {
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
func (m *Mongo) DocGetNeighbour(ctx context.Context, mediaDoc MediaFileDoc, cl *mongo.Client) (*MediaFileDoc, *MediaFileDoc, error) {
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

func (m *mediaMongo) GetCollection(cl *mongo.Client) *mongo.Collection {
	return cl.Database(m.DBName).Collection(m.MediaCollectionName)
}
func (m *jobMongo) GetCollection(cl *mongo.Client) *mongo.Collection {
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
	GetCollection(cl *mongo.Client) *mongo.Collection
	Create(ctx context.Context, doc T, cl *mongo.Client) (T, errs.IMongoErr)
	List(ctx context.Context, filter *primitive.D, cl *mongo.Client) ([]T, errs.IMongoErr)
	Delete(ctx context.Context, filter *primitive.D, cl *mongo.Client) errs.IMongoErr
	DeleteMany(ctx context.Context, filter *primitive.D, cl *mongo.Client) errs.IMongoErr
	Find(ctx context.Context, filter *primitive.D, cl *mongo.Client) (T, errs.IMongoErr)
	Replace(ctx context.Context, filter *primitive.D, doc T, cl *mongo.Client) (T, errs.IMongoErr)
	MarshalOmitEmpty(doc T) (*primitive.D, errs.IMongoErr)
	GetIDFilter(id primitive.ObjectID) *primitive.D
}

type DataStore[T IMongoDoc] struct {
	DB         *Mongo
	collection string
}

func (m *DataStore[T]) GetCollection(cl *mongo.Client) *mongo.Collection {
	return cl.Database(m.DB.DBName).Collection(m.collection)
}
func (m *DataStore[T]) Create(ctx context.Context, doc T, cl *mongo.Client) (T, errs.IMongoErr) {
	res, err := m.GetCollection(cl).InsertOne(ctx, doc)
	if err != nil {
		return doc, errs.NewMongoOpErr(err)
	}
	id := res.InsertedID.(primitive.ObjectID)
	doc.SetID(id)
	return doc, nil
}

func (m *DataStore[T]) List(ctx context.Context, filter *primitive.D, cl *mongo.Client) ([]T, errs.IMongoErr) {
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
func (m *DataStore[T]) Delete(ctx context.Context, filter *primitive.D, cl *mongo.Client) errs.IMongoErr {
	if res, err := m.GetCollection(cl).DeleteOne(ctx, filter); err != nil {
		return errs.NewMongoOpErr(err)
	} else if res.DeletedCount == 0 {
		return errs.NewMongoObjectNotfound(*filter)
	}
	return nil
}
func (m *DataStore[T]) DeleteMany(ctx context.Context, filter *primitive.D, cl *mongo.Client) errs.IMongoErr {
	if res, err := m.GetCollection(cl).DeleteMany(ctx, filter); err != nil {
		return errs.NewMongoOpErr(err)
	} else if res.DeletedCount == 0 {
		return errs.NewMongoObjectNotfound(*filter)
	}
	return nil
}
func (m *DataStore[T]) Find(ctx context.Context, filter *primitive.D, cl *mongo.Client) (T, errs.IMongoErr) {
	docs, err := m.List(ctx, filter, cl)
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
func (m *DataStore[T]) Replace(ctx context.Context, filter *primitive.D, doc T, cl *mongo.Client) (T, errs.IMongoErr) {
	res, err := m.GetCollection(cl).ReplaceOne(ctx, filter, doc)
	if err != nil {
		return doc, err
	}
	if res.MatchedCount == 0 {
		return doc, errs.NewMongoObjectNotfound(*filter)
	}
	return doc, nil
}
func (m *DataStore[T]) GetIDFilter(id primitive.ObjectID) *primitive.D {
	return &bson.D{{Key: "_id", Value: id}}
}
func (m *DataStore[T]) MarshalOmitEmpty(doc T) (*primitive.D, errs.IMongoErr) {
	var filteredData = make(map[string]interface{})
	v := reflect.ValueOf(doc)
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
func (DB *Mongo) GetMediaDatastore() *DataStore[*MediaFileDoc] {
	return &DataStore[*MediaFileDoc]{
		DB:         DB,
		collection: DB.MediaCollectionName,
	}
}
func (DB *Mongo) GetJobDatastore() *DataStore[*JobDoc] {
	return &DataStore[*JobDoc]{
		DB:         DB,
		collection: DB.JobCollectionName,
	}
}

type DatastoreEnum int

const (
	MEDIA_DS DatastoreEnum = iota
	JOB_DS
)

func (DB *Mongo) GetDatastore(name DatastoreEnum) IDataStore[IMongoDoc] {
	ds := new(DataStore[IMongoDoc])
	switch name {
	case MEDIA_DS:
		ds = (*DataStore[IMongoDoc])(DB.GetMediaDatastore())
	case JOB_DS:
		ds = (*DataStore[IMongoDoc])(DB.GetJobDatastore())
	}
	return ds
}

// ...
func FilterByID(id primitive.ObjectID) bson.D {
	return bson.D{{Key: "_id", Value: id}}
}
