// Package facade provides CRUD logic for media file documents.
package facade

import (
	"context"
	"fmt"
	"time"

	"github.com/amirdaaee/TGMon/internal/db"
	"github.com/amirdaaee/TGMon/internal/db/minio"
	mngo "github.com/amirdaaee/TGMon/internal/db/mongo"
	"github.com/amirdaaee/TGMon/internal/log"
	"github.com/amirdaaee/TGMon/internal/types"
	"github.com/chenmingyong0423/go-mongox/v2/bsonx"
	"github.com/sirupsen/logrus"
)

// MediaCrud implements ICrud for MediaFileDoc, providing CRUD hooks and collection access.
type MediaCrud struct {
	container db.IDbContainer
	jReqFac   IFacade[types.JobReqDoc]
}

var _ ICrud[types.MediaFileDoc] = (*MediaCrud)(nil)

// PreCreate checks for duplicates before creating a MediaFileDoc. Returns an error if the document is nil or a duplicate is found.
func (crd *MediaCrud) PreCreate(ctx context.Context, doc *types.MediaFileDoc) error {
	if doc == nil {
		return fmt.Errorf("MediaFileDoc is nil")
	}
	// TODO: duplicated check (stub)
	return nil
}

// PostCreate creates a sprite job request after creating a media file. Returns an error if the document is nil or job creation fails.
func (crd *MediaCrud) PostCreate(ctx context.Context, doc *types.MediaFileDoc) error {
	if doc == nil {
		return fmt.Errorf("MediaFileDoc is nil")
	}
	_, err := crd.jReqFac.CreateOne(ctx, &types.JobReqDoc{
		Type:    types.SPRITEJobType,
		MediaID: doc.ID,
	})
	return err
}

// PreDelete is a pre-delete hook for MediaFileDoc. No-op in this implementation.
func (crd *MediaCrud) PreDelete(ctx context.Context, doc *types.MediaFileDoc) error {
	return nil
}

// PostDelete deletes orphaned jobs and files after deleting a media file. Retries file deletion up to 3 times. Logs errors but does not return them.
func (crd *MediaCrud) PostDelete(ctx context.Context, doc *types.MediaFileDoc) error {
	ll := crd.getLogger("PostDelete")
	if doc == nil {
		return fmt.Errorf("MediaFileDoc is nil")
	}
	q := bsonx.NewD().Add(types.JobReqDoc__MediaIDField, doc.ID).Build()
	if dl, err := crd.jReqFac.GetCRD().GetCollection().Deleter().Filter(q).DeleteMany(ctx); err != nil {
		ll.WithError(err).Error("failed to delete orphaned jobs")
	} else if dl.DeletedCount > 0 {
		ll.Infof("deleted %d orphaned jobs", dl.DeletedCount)
	}
	for _, fn := range []string{doc.Vtt, doc.Thumbnail} {
		if fn != "" {
			var lastErr error
			for i := 0; i < 3; i++ {
				if err := crd.getMinioClient().FileRm(ctx, fn); err != nil {
					lastErr = err
					time.Sleep(100 * time.Millisecond)
				} else {
					lastErr = nil
					break
				}
			}
			if lastErr != nil {
				ll.WithError(lastErr).Error("failed to remove orphaned file after retries")
			}
		}
	}
	return nil
}

// GetCollection returns the MediaFile collection from the database container.
func (crd *MediaCrud) GetCollection() mngo.ICollection[types.MediaFileDoc] {
	return crd.container.GetMongoContainer().GetMediaFileCollection()
}

// getMinioClient returns the Minio client from the database container.
func (crd *MediaCrud) getMinioClient() minio.IMinioClient {
	return crd.container.GetMinioContainer().GetMinioClient()
}

// getLogger returns a logrus.Entry for the given function name, tagged with the struct type.
func (crd *MediaCrud) getLogger(fn string) *logrus.Entry {
	return log.GetLogger(log.FacadeModule).WithField("func", fmt.Sprintf("%T.%s", crd, fn))
}

// NewMediaCrud creates a new MediaCrud with the provided database container.
func NewMediaCrud(container db.IDbContainer) ICrud[types.MediaFileDoc] {
	return &MediaCrud{container: container, jReqFac: GetFacade[types.JobReqDoc](container)}
}

func init() {
	RegisterCRD(NewMediaCrud)
}
