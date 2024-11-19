package facade

import (
	"context"
	"fmt"

	"github.com/amirdaaee/TGMon/internal/db"
	"github.com/amirdaaee/TGMon/internal/errs"
	"github.com/google/uuid"
	"github.com/onsi/ginkgo/v2"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type FullMediaData struct {
	doc   *db.MediaFileDoc
	thumb []byte
}

func NewFullMediaData(doc *db.MediaFileDoc, thumb []byte) *FullMediaData {
	return &FullMediaData{
		doc:   doc,
		thumb: thumb,
	}
}

type MediaFacade struct {
	baseFacade[*db.MediaFileDoc]
}

// create new media doc
// + set thumbnail
// + generate sprite job
func (f *MediaFacade) Create(ctx context.Context, data *FullMediaData, cl *mongo.Client) (*db.MediaFileDoc, error) {
	newDoc, err := f.baseCreate(ctx, data.doc, cl)
	if err != nil {
		return nil, err
	}
	go func() {
		defer ginkgo.GinkgoRecover()
		ll := f.getLogger("create:side-effect")
		innerCtx := context.Background()
		innerCl, err := f.mongo.GetClient()
		if err != nil {
			ll.WithError(err).Error("can not get mongo client")
			return
		}
		defer innerCl.Disconnect(innerCtx)
		// ...
		if err := createMediaJob(innerCtx, *newDoc, f.jobDS, innerCl, db.SPRITEJobType); err != nil {
			ll.WithError(err).Error("can not create sprite job")
		}
		// ...
		if data.thumb != nil {
			if err := updateMediaMinioFiles(innerCtx, newDoc, f.minio, f.mediaDS, innerCl, &MediaMinioFile{thumbData: data.thumb}); err != nil {
				ll.WithError(err).Error("can not process thumbnail")
			}
		}
	}()
	return newDoc, err
}
func (f *MediaFacade) Read(ctx context.Context, filter *primitive.D, cl *mongo.Client) ([]*db.MediaFileDoc, error) {
	docs, err := f.baseRead(ctx, filter, cl)
	return docs, err
}

// delete new media doc
// + delete minio files
// + delete all related jobs
func (f *MediaFacade) Delete(ctx context.Context, filter *primitive.D, cl *mongo.Client) error {
	doc, err := f.mediaDS.Find(ctx, filter, cl)
	if err != nil {
		return err
	}
	if err := f.baseDelete(ctx, filter, cl); err != nil {
		return err
	}
	go func() {
		defer ginkgo.GinkgoRecover()
		ll := f.getLogger("delete:side-effect")
		innerCtx := context.Background()
		innerCl, err := f.mongo.GetClient()
		if err != nil {
			ll.WithError(err).Error("can not get mongo client")
			return
		}
		defer innerCl.Disconnect(innerCtx)
		// ...
		if err := deleteMediaAllJobs(innerCtx, doc, f.jobDS, innerCl); err != nil {
			ll.WithError(err).Error("can not delete jobs of doc")
		}
		// ...
		deleteMediaAllMinioFiles(innerCtx, doc, f.minio)
	}()
	return nil
}

func NewMediaFacade(mongo db.IMongo, minio db.IMinioClient, jobDS db.IDataStore[*db.JobDoc], mediaDS db.IDataStore[*db.MediaFileDoc]) *MediaFacade {
	return &MediaFacade{
		baseFacade: baseFacade[*db.MediaFileDoc]{
			name:    "media",
			mongo:   mongo,
			dsName:  db.MEDIA_DS,
			jobDS:   jobDS,
			mediaDS: mediaDS,
			minio:   minio,
		},
	}
}

// ...
func createMediaJob(ctx context.Context, doc db.MediaFileDoc, jobDs db.IDataStore[*db.JobDoc], cl *mongo.Client, jType db.JobType) error {
	jobDoc := db.JobDoc{
		MediaID: doc.GetID(),
		Type:    jType,
	}
	if _, err := jobDs.Create(ctx, &jobDoc, cl); err != nil {
		return err
	}
	return nil
}
func deleteMediaAllJobs(ctx context.Context, doc *db.MediaFileDoc, jobDs db.IDataStore[*db.JobDoc], cl *mongo.Client) error {
	ll := logrus.WithField("func", "deleteMediaAllJobs")
	jobFilter := db.JobDoc{
		MediaID: doc.GetID(),
	}
	jobFilterD, err := db.MarshalOmitEmpty(&jobFilter)
	if err != nil {
		return fmt.Errorf("can not create filter: %s", err)
	}
	if err := jobDs.DeleteMany(ctx, jobFilterD, cl); err != nil {
		if errs.IsErr(err, errs.MongoObjectNotfound{}) {
			ll.Info("no job found for media")
		} else {
			return fmt.Errorf("can not delete job objects: %s", err)
		}
	}
	return nil
}

type MediaMinioFile struct {
	thumbData  []byte
	vttData    []byte
	spriteData []byte
}

// add new files to minio, update media doc with new files, remove old files from minio
func updateMediaMinioFiles(ctx context.Context, doc *db.MediaFileDoc, minio db.IMinioClient, mediaDs db.IDataStore[*db.MediaFileDoc], cl *mongo.Client, data *MediaMinioFile) error {
	ll := logrus.WithField("func", "updateMediaMinioFiles")
	updatedMedia := *doc
	if data.thumbData != nil {
		fName := uuid.NewString() + ".jpeg"
		if err := minio.FileAdd(ctx, fName, data.thumbData); err != nil {
			ll.WithError(err).Error("can not add new thumbnail to minio")
		} else {
			updatedMedia.Thumbnail = fName
		}
	}
	if data.vttData != nil {
		fName := uuid.NewString() + ".vtt"
		if err := minio.FileAdd(ctx, fName, data.vttData); err != nil {
			ll.WithError(err).Error("can not add new vtt to minio")
		} else {
			updatedMedia.Vtt = fName
		}
	}
	if data.spriteData != nil {
		fName := uuid.NewString() + ".jpeg"
		if err := minio.FileAdd(ctx, fName, data.spriteData); err != nil {
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
		filter := db.GetIDFilter(doc.GetID())
		_, err := mediaDs.Replace(ctx, filter, &updatedMedia, cl)
		if err != nil {
			return fmt.Errorf("can not update media doc: %s", err)
		}
		if doc.Thumbnail != "" && updatedMedia.Thumbnail != doc.Thumbnail {
			if err := _rmMinioFile(ctx, minio, doc.Thumbnail); err != nil {
				ll.WithError(err).Error("can not remove old thumbnail from minio")
			}
		}
		if doc.Vtt != "" && updatedMedia.Vtt != doc.Vtt {
			if err := _rmMinioFile(ctx, minio, doc.Vtt); err != nil {
				ll.WithError(err).Error("can not remove old Vtt from minio")
			}
		}
		if doc.Sprite != "" && updatedMedia.Sprite != doc.Sprite {
			if err := _rmMinioFile(ctx, minio, doc.Sprite); err != nil {
				ll.WithError(err).Error("can not remove old Sprite from minio")
			}
		}
	} else {
		ll.Warn("nothing to update")
	}
	return nil
}
func deleteMediaAllMinioFiles(ctx context.Context, doc *db.MediaFileDoc, minio db.IMinioClient) {
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
func _rmMinioFile(ctx context.Context, minio db.IMinioClient, fname string) error {
	return minio.FileRm(ctx, fname)
}
