package db

import (
	"context"

	"github.com/amirdaaee/TGMon/config"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func Client() (*mongo.Database, *mongo.Client, error) {
	serverAPI := options.ServerAPI(options.ServerAPIVersion1)
	opts := options.Client().ApplyURI(config.Config().MongoDBUri).SetServerAPIOptions(serverAPI)
	cl, err := mongo.Connect(context.TODO(), opts)
	if err != nil {
		return nil, nil, err
	}
	db := cl.Database(config.Config().MongoDBName)
	return db, cl, nil
}
func GetFileCollection() (*mongo.Collection, *mongo.Client, error) {
	db_, cl_, err := Client()
	if err != nil {
		return nil, nil, err
	}
	col_ := db_.Collection("files")
	return col_, cl_, nil
}
func AddDoc(ctx context.Context, collection *mongo.Collection, doc interface{}) (*mongo.InsertOneResult, error) {
	return collection.InsertOne(ctx, doc)
}
func FilterById(docID string) (*bson.D, error) {
	docId, err := primitive.ObjectIDFromHex(docID)
	if err != nil {
		return nil, err
	}
	return &bson.D{{Key: "_id", Value: docId}}, err
}
func GetDocById(ctx context.Context, collection *mongo.Collection, docID string, result interface{}) error {
	filter, err := FilterById(docID)
	if err != nil {
		return err
	}
	err = collection.FindOne(ctx, filter).Decode(result)
	return err
}
func DelDocById(ctx context.Context, collection *mongo.Collection, docID string) error {
	filter, err := FilterById(docID)
	if err != nil {
		return err
	}
	_, err = collection.DeleteOne(ctx, filter)
	return err
}
