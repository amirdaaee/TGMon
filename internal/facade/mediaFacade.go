package facade

import (
	"context"
	"fmt"

	"github.com/amirdaaee/TGMon/internal/db"
	"github.com/amirdaaee/TGMon/internal/errs"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type FullMediaData struct {
	doc   *db.MediaFileDoc
	thumb []byte
}
type mediaFacade struct {
	baseFacade[*db.MediaFileDoc]
	minio *db.MinioClient
}

func (f *mediaFacade) Create(ctx context.Context, data *FullMediaData, cl *mongo.Client) (*db.MediaFileDoc, error) {
	newDoc, err := f.baseCreate(ctx, data.doc, cl)
	go func() {
		ll := f.getLogger("create:side-effect")
		innerCtx := context.Background()
		innerCl, err := f.mongo.GetClient()
		if err != nil {
			ll.WithError(err).Error("can not get mongo client")
			return
		}
		defer innerCl.Disconnect(innerCtx)
		// ...
		if err := createMediaJob(innerCtx, *newDoc, f.mongo, innerCl, db.THUMBNAILJobType); err != nil {
			ll.WithError(err).Error("can not create sprite job")
		}
		// ...
		if data.thumb != nil {
			if err := updateMediaMinioFiles(innerCtx, newDoc, f.minio, f.mongo, innerCl, &mediaMinioFile{thumbData: data.thumb}); err != nil {
				ll.WithError(err).Error("can not process thumbnail")
			}
		}
	}()
	return newDoc, err
}
func (f *mediaFacade) Read(ctx context.Context, filter *primitive.D, cl *mongo.Client) ([]*db.MediaFileDoc, error) {
	docs, err := f.baseRead(ctx, filter, cl)
	return docs, err
}
func (f *mediaFacade) Delete(ctx context.Context, filter *primitive.D, cl *mongo.Client) error {
	docs, err := f.baseRead(ctx, filter, cl)
	doc := docs[0]
	if err != nil {
		return err
	}
	if err := f.baseDelete(ctx, filter, cl); err != nil {
		return err
	}
	go func() {
		ll := f.getLogger("delete:side-effect")
		innerCtx := context.Background()
		innerCl, err := f.mongo.GetClient()
		if err != nil {
			ll.WithError(err).Error("can not get mongo client")
			return
		}
		defer innerCl.Disconnect(innerCtx)
		// ...
		if err := deleteMediaAllJobs(innerCtx, doc, f.mongo, innerCl); err != nil {
			ll.WithError(err).Error("can not delete jobs of doc")
		}
		// ...
		deleteMediaAllMinioFiles(innerCtx, doc, f.minio)
	}()
	return nil
}

func NewMediaFacade(mongo *db.Mongo, minio *db.MinioClient) *mediaFacade {
	return &mediaFacade{
		baseFacade: baseFacade[*db.MediaFileDoc]{
			name:   "media",
			mongo:  mongo,
			dsName: db.MEDIA_DS,
		},
		minio: minio,
	}
}

// ...
func createMediaJob(ctx context.Context, doc db.MediaFileDoc, mongo *db.Mongo, cl *mongo.Client, jType db.JobType) error {
	jobDoc := db.JobDoc{
		MediaID: doc.GetID(),
		Type:    jType,
	}
	jobDs := mongo.GetJobDatastore()
	if _, err := jobDs.Create(ctx, &jobDoc, cl); err != nil {
		return err
	}
	return nil
}
func deleteMediaAllJobs(ctx context.Context, doc *db.MediaFileDoc, mongo *db.Mongo, cl *mongo.Client) error {
	ll := logrus.WithField("func", "deleteMediaAllJobs")
	jobFilter := db.JobDoc{
		MediaID: doc.GetID(),
	}
	jobDs := mongo.GetJobDatastore()
	jobFilterD, err := jobDs.MarshalOmitEmpty(&jobFilter)
	if err != nil {
		return fmt.Errorf("can not create filter: %s", err)
	}
	if err := jobDs.DeleteMany(ctx, jobFilterD, cl); err != nil {
		if errs.IsErr(err, errs.MongoObjectNotfound{}) {
			ll.Info("no job found for media")
		} else {
			return fmt.Errorf("can not delete job objects")
		}
	}
	return nil
}

type mediaMinioFile struct {
	thumbData  []byte
	vttData    []byte
	spriteData []byte
}

func updateMediaMinioFiles(ctx context.Context, doc *db.MediaFileDoc, minio *db.MinioClient, mongo *db.Mongo, cl *mongo.Client, data *mediaMinioFile) error {
	ll := logrus.WithField("func", "updateMediaMinioFiles")
	updatedMedia := doc
	if data.thumbData != nil {
		fName := uuid.NewString() + ".jpeg"
		if err := minio.FileAdd(fName, data.thumbData, ctx); err != nil {
			ll.WithError(err).Error("can not add new thumbnail to minio")
		} else {
			updatedMedia.Thumbnail = fName
		}
	}
	if data.vttData != nil {
		fName := uuid.NewString() + ".vtt"
		if err := minio.FileAdd(fName, data.vttData, ctx); err != nil {
			ll.WithError(err).Error("can not add new vtt to minio")
		} else {
			updatedMedia.Vtt = fName
		}
	}
	if data.spriteData != nil {
		fName := uuid.NewString() + ".jpeg"
		if err := minio.FileAdd(fName, data.spriteData, ctx); err != nil {
			ll.WithError(err).Error("can not add new sprite to minio")
		} else {
			updatedMedia.Sprite = fName
		}
	}
	changed := false
	changed = changed || updatedMedia.Thumbnail != doc.Thumbnail
	changed = changed || updatedMedia.Vtt != doc.Vtt
	changed = changed || updatedMedia.Sprite != doc.Sprite

	if changed {
		mediaDs := mongo.GetMediaDatastore()
		filter := mediaDs.GetIDFilter(doc.GetID())
		_, err := mediaDs.Replace(ctx, filter, updatedMedia, cl)
		if err != nil {
			return fmt.Errorf("can not update media doc: %s", err)
		}
		if updatedMedia.Thumbnail != doc.Thumbnail {
			if err := _rmMinioFile(ctx, minio, doc.Thumbnail); err != nil {
				ll.WithError(err).Error("can not remove old thumbnail from minio")
			}
		}
		if updatedMedia.Vtt != doc.Vtt {
			if err := _rmMinioFile(ctx, minio, doc.Vtt); err != nil {
				ll.WithError(err).Error("can not remove old Vtt from minio")
			}
		}
		if updatedMedia.Sprite != doc.Sprite {
			if err := _rmMinioFile(ctx, minio, doc.Sprite); err != nil {
				ll.WithError(err).Error("can not remove old Sprite from minio")
			}
		}
	} else {
		ll.Warn("nothing to update")
	}
	return nil
}
func deleteMediaAllMinioFiles(ctx context.Context, doc *db.MediaFileDoc, minio *db.MinioClient) {
	ll := logrus.WithField("func", "deleteMediaAllMinioFiles")
	if doc.Thumbnail != "" {
		if err := _rmMinioFile(ctx, minio, doc.Thumbnail); err != nil {
			ll.WithError(err).Error("can not remove thumbnail from minio")
		}
	}
	if doc.Vtt != "" {
		if err := _rmMinioFile(ctx, minio, doc.Vtt); err != nil {
			ll.WithError(err).Error("can not remove vtt from minio")
		}
	}
	if doc.Sprite != "" {
		if err := _rmMinioFile(ctx, minio, doc.Sprite); err != nil {
			ll.WithError(err).Error("can not remove sprite from minio")
		}
	}
}
func _rmMinioFile(ctx context.Context, minio *db.MinioClient, fname string) error {
	return minio.FileRm(fname, ctx)
}
