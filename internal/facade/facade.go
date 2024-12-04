package facade

import (
	"context"

	"github.com/amirdaaee/TGMon/internal/db"
	"github.com/amirdaaee/TGMon/internal/errs"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type baseFacade[T db.IMongoDoc] struct {
	name    string
	mongo   db.IMongo
	dsName  db.DatastoreEnum
	jobDS   db.IDataStore[*db.JobDoc]
	mediaDS db.IDataStore[*db.MediaFileDoc]
	minio   db.IMinioClient
}

func (f *baseFacade[T]) getLogger(fn string) *logrus.Entry {
	return logrus.WithField("facade", f.name).WithField("func", fn)
}
func (f *baseFacade[T]) getDatastore() db.IDataStore[T] {
	ds := new(db.IDataStore[T])
	switch f.dsName {
	case db.JOB_DS:
		*ds = f.jobDS.(db.IDataStore[T])
	case db.MEDIA_DS:
		*ds = f.mediaDS.(db.IDataStore[T])
	default:
		logrus.Panicf("unknown ds %d", f.dsName)
	}
	return *ds
}
func (f *baseFacade[T]) baseCreate(ctx context.Context, doc T, cl db.IMongoClient) (T, errs.IMongoErr) {
	ds := f.getDatastore()
	newDoc, err := ds.Create(ctx, doc, cl)
	if err != nil {
		return *new(T), err
	}
	return newDoc, err
}
func (f *baseFacade[T]) baseRead(ctx context.Context, filter *primitive.M, cl db.IMongoClient) ([]T, errs.IMongoErr) {
	ds := f.getDatastore()
	docs, err := ds.FindMany(ctx, filter, cl)
	if err != nil {
		return nil, err
	}
	return docs, err
}

//	func (f *baseFacade[T]) baseUpdate(ctx context.Context, filter *primitive.M, doc T, cl db.IMongoClient) (T, errs.IMongoErr) {
//		ds := f.getDatastore()
//		newDoc, err := ds.Replace(ctx, filter, doc, cl)
//		if err != nil {
//			return *new(T), err
//		}
//		return newDoc, err
//	}
func (f *baseFacade[T]) baseDelete(ctx context.Context, filter *primitive.M, cl db.IMongoClient) errs.IMongoErr {
	ds := f.getDatastore()
	return ds.Delete(ctx, filter, cl)
}
