package facade

import (
	"context"
	"fmt"

	"github.com/amirdaaee/TGMon/internal/db"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type JobFacade struct {
	baseFacade[*db.JobDoc]
}

// create new job if not exist and omit creation if doesn't
func (f *JobFacade) Create(ctx context.Context, doc *db.JobDoc, cl *mongo.Client) (*db.JobDoc, error) {
	ll := f.getLogger("create")
	// check for exist
	filter := doc
	filter.SetID(primitive.NilObjectID)
	filterD, err := f.jobDS.MarshalOmitEmpty(filter)
	if err != nil {
		return nil, fmt.Errorf("can not marshal filter to find duplicates: %s", err)
	}
	res, err := f.baseRead(ctx, filterD, cl)
	if err != nil {
		return nil, fmt.Errorf("can not list jobs to find duplicates: %s", err)
	}
	if len(res) != 0 {
		ll.Warn("job already exists")
		return nil, nil
	}
	// ...
	newDoc, err := f.baseCreate(ctx, doc, cl)
	return newDoc, err
}
func (f *JobFacade) Read(ctx context.Context, filter *primitive.D, cl *mongo.Client) ([]*db.JobDoc, error) {
	docs, err := f.baseRead(ctx, filter, cl)
	return docs, err
}
func (f *JobFacade) Delete(ctx context.Context, filter *primitive.D, cl *mongo.Client) error {
	return f.baseDelete(ctx, filter, cl)
}

// update media based on job result and delete job itself
//
// job is kept only if provided data are not constistant
func (f *JobFacade) Done(ctx context.Context, id primitive.ObjectID, cl *mongo.Client, data *mediaMinioFile) error {
	ll := f.getLogger("done")
	ds := f.jobDS
	jobDoc, err := ds.Find(ctx, ds.GetIDFilter(id), cl)
	if err != nil {
		return fmt.Errorf("can not get job doc: %s", err)
	}
	if jobDoc.Type == db.THUMBNAILJobType {
		if data.thumbData == nil {
			return fmt.Errorf("thumbnail is empty")
		}
		data.vttData = nil
		data.spriteData = nil
	}
	if jobDoc.Type == db.SPRITEJobType {
		if data.vttData == nil || data.spriteData == nil {
			return fmt.Errorf("vtt or sprite is empty")
		}
		data.thumbData = nil
	}
	// ...
	// anyway job should be deleted from this point on
	go deleteJob(ds.GetIDFilter(id), f.mongo, f.jobDS)
	// ...
	mediaDoc, err := f.mediaDS.Find(ctx, ds.GetIDFilter(jobDoc.MediaID), cl)
	if err != nil {
		ll.WithError(err).Error("error getting corresponding media")
		return nil
	}
	// ...
	if err := updateMediaMinioFiles(ctx, mediaDoc, f.minio, f.mediaDS, cl, data); err != nil {
		ll.WithError(err).Error("error updating media files")
		return nil
	}
	return nil
}

func NewJobFacade(mongo db.IMongo, minio db.IMinioClient, jobDS db.IDataStore[*db.JobDoc], mediaDS db.IDataStore[*db.MediaFileDoc]) *JobFacade {
	return &JobFacade{
		baseFacade: baseFacade[*db.JobDoc]{
			name:    "job",
			mongo:   mongo,
			dsName:  db.JOB_DS,
			jobDS:   jobDS,
			mediaDS: mediaDS,
			minio:   minio,
		},
	}
}

// ...
func deleteJob(filter *primitive.D, monog db.IMongo, jobDS db.IDataStore[*db.JobDoc]) {
	ll := logrus.WithField("func", "deleteJob")
	ctx := context.Background()
	cl, err := monog.GetClient()
	if err != nil {
		ll.WithError(err).Error("")
		return
	}
	defer cl.Disconnect(ctx)
	if err := jobDS.Delete(ctx, filter, cl); err != nil {
		ll.WithError(err).Error("can not delete job doc")
		return
	}
}
