package facade

import (
	"context"

	"github.com/amirdaaee/TGMon/internal/db"
	"github.com/amirdaaee/TGMon/internal/errs"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type baseFacade[T db.IMongoDoc] struct {
	name string
}

func (f *baseFacade[T]) getLogger(fn string) *logrus.Entry {
	return logrus.WithField("facade", f.name).WithField("func", fn)
}

func (f *baseFacade[T]) baseCreate(ctx context.Context, doc T, cl *mongo.Client, ds *db.DataStore[T]) (T, errs.IMongoErr) {
	newDoc, err := ds.Create(ctx, doc, cl)
	if err != nil {
		return *new(T), err
	}
	return newDoc, err
}
func (f *baseFacade[T]) baseRead(ctx context.Context, filter *primitive.D, cl *mongo.Client, ds *db.DataStore[T]) ([]T, errs.IMongoErr) {
	docs, err := ds.List(ctx, filter, cl)
	if err != nil {
		return nil, err
	}
	return docs, err
}
func (f *baseFacade[T]) baseUpdate(ctx context.Context, filter *primitive.D, doc T, cl *mongo.Client, ds *db.DataStore[T]) (T, errs.IMongoErr) {
	newDoc, err := ds.Replace(ctx, filter, doc, cl)
	if err != nil {
		return *new(T), err
	}
	return newDoc, err
}
func (f *baseFacade[T]) baseDelete(ctx context.Context, filter *primitive.D, cl *mongo.Client, ds *db.DataStore[T]) errs.IMongoErr {
	return ds.Delete(ctx, filter, cl)
}
