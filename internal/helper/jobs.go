package helper

import (
	"context"
	"fmt"
	"sync"

	"github.com/amirdaaee/TGMon/internal/db"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	mongoD "go.mongodb.org/mongo-driver/mongo"
)

func AddJob(ctx context.Context, mongo *db.Mongo, jobs []db.JobDoc) error {
	cl_, err := mongo.GetClient()
	if err != nil {
		return fmt.Errorf("can not get mongo client: %s", err)
	}
	defer cl_.Disconnect(ctx)
	jobMongo := mongo.GetJobMongo()
	coll_ := jobMongo.IMng.GetCollection(cl_)
	wp := sync.WaitGroup{}
	for _, jb := range jobs {
		wp.Add(1)
		go func(j db.JobDoc) {
			defer wp.Done()
			ll2 := logrus.WithField("record", j)
			jCopy := j
			jCopy.SetID(primitive.NilObjectID)
			filter, err := bson.Marshal(jCopy)
			if err != nil {
				ll2.WithError(err).Error("can not create lookup filter")
				return
			}
			res := coll_.FindOne(ctx, filter)
			if res.Err() == nil {
				ll2.Warn("record already exists")
				return
			} else if res.Err() != mongoD.ErrNoDocuments {
				ll2.WithError(res.Err()).Warn("error lookup job record")
				return
			}
			if _, err := jobMongo.DocAdd(ctx, j, cl_); err != nil {
				ll2.WithError(err).Error("can not add job to db")
				return
			}
		}(jb)
	}
	wp.Wait()
	return nil
}
func RmMediaJob(ctx context.Context, mongo *db.Mongo, mediaID string) error {
	jobMongo := mongo.GetJobMongo()
	cl, err := mongo.GetClient()
	if err != nil {
		return fmt.Errorf("can not get mongo client: %s", err)
	}
	coll := jobMongo.IMng.GetCollection(cl)
	qMB := bson.M{"MediaID": mediaID}
	if _, err := coll.DeleteMany(ctx, qMB); err != nil {
		return fmt.Errorf("can not delete job records: %s", err)
	}
	return nil
}
