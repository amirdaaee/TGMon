package db

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Mongo struct {
	DBUri          string
	DBName         string
	CollectionName string
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

func (m *Mongo) GetFileCollection(cl *mongo.Client) *mongo.Collection {
	return cl.Database(m.DBName).Collection(m.CollectionName)
}

// ...
func (m *Mongo) DocAdd(ctx context.Context, doc interface{}, cl *mongo.Client) (*mongo.InsertOneResult, error) {
	cl, disc, err := m.assertClient(cl)
	if err != nil {
		return nil, err
	}
	defer disc(ctx)
	return m.GetFileCollection(cl).InsertOne(ctx, doc)
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
	return m.GetFileCollection(cl).FindOne(ctx, filter).Decode(result)
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
	_, err = m.GetFileCollection(cl).DeleteOne(ctx, filter)
	return err
}
func (m *Mongo) DocGetAll(ctx context.Context, collection *mongo.Collection, result interface{}, cl *mongo.Client, opts ...*options.FindOptions) error {
	cl, disc, err := m.assertClient(cl)
	if err != nil {
		return err
	}
	defer disc(ctx)
	cur_, err := m.GetFileCollection(cl).Find(ctx, bson.D{}, opts...)
	if err != nil {
		return err
	}
	if err = cur_.All(ctx, result); err != nil {
		return err
	}
	return nil
}

// ...
func FilterById(docID string) (*bson.D, error) {
	docId, err := primitive.ObjectIDFromHex(docID)
	if err != nil {
		return nil, err
	}
	return &bson.D{{Key: "_id", Value: docId}}, err
}
