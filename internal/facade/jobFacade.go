// Package facade provides CRUD logic for job request and result documents.
package facade

import (
	"context"
	"fmt"

	"github.com/amirdaaee/TGMon/internal/db"
	mngo "github.com/amirdaaee/TGMon/internal/db/mongo"
	"github.com/amirdaaee/TGMon/internal/log"
	"github.com/amirdaaee/TGMon/internal/types"
	"github.com/chenmingyong0423/go-mongox/v2/builder/query"
	"github.com/chenmingyong0423/go-mongox/v2/builder/update"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// JobReqCrud implements ICrud for JobReqDoc, providing CRUD hooks and collection access.
type JobReqCrud struct {
	container db.IDbContainer
}

var _ ICrud[types.JobReqDoc] = (*JobReqCrud)(nil)

// PreCreate checks for duplicates before creating a JobReqDoc. Returns an error if the document is nil or a duplicate is found.
func (crd *JobReqCrud) PreCreate(ctx context.Context, doc *types.JobReqDoc) error {
	if doc == nil {
		return fmt.Errorf("JobReqDoc is nil")
	}
	// TODO: duplicated check (stub)
	return nil
}

// PostCreate is a post-create hook for JobReqDoc. No-op in this implementation.
func (crd *JobReqCrud) PostCreate(ctx context.Context, doc *types.JobReqDoc) error {
	return nil
}

// PreDelete is a pre-delete hook for JobReqDoc. No-op in this implementation.
func (crd *JobReqCrud) PreDelete(ctx context.Context, doc *types.JobReqDoc) error {
	return nil
}

// PostDelete is a post-delete hook for JobReqDoc. No-op in this implementation.
func (crd *JobReqCrud) PostDelete(ctx context.Context, doc *types.JobReqDoc) error {
	return nil
}

// GetCollection returns the JobReq collection from the database container.
func (crd *JobReqCrud) GetCollection() mngo.ICollection[types.JobReqDoc] {
	return crd.container.GetMongoContainer().GetJobReqCollection()
}

// NewJobReqCrud creates a new JobReqCrud with the provided database container.
func NewJobReqCrud(container db.IDbContainer) ICrud[types.JobReqDoc] {
	return &JobReqCrud{container: container}
}

// JobResCrud implements ICrud for JobResDoc, providing CRUD hooks and collection access.
type JobResCrud struct {
	container db.IDbContainer
	jReqFac   IFacade[types.JobReqDoc]
}

var _ ICrud[types.JobResDoc] = (*JobResCrud)(nil)

// PreCreate processes the job result and updates the related media document. Returns an error if the document is nil or processing fails.
func (crd *JobResCrud) PreCreate(ctx context.Context, doc *types.JobResDoc) error {
	if doc == nil {
		return fmt.Errorf("JobResDoc is nil")
	}
	jobReq, err := crd.getJobRequest(ctx, doc)
	if err != nil {
		return err
	}

	fileName := crd.generateFileName(doc, jobReq)

	updateField, err := crd.processJobResult(ctx, doc, jobReq, fileName)
	if err != nil {
		return err
	}

	return crd.updateMediaDocument(ctx, jobReq.MediaID, updateField)
}

// PostCreate deletes the related job request after creating a job result. Logs errors but does not return them.
func (crd *JobResCrud) PostCreate(ctx context.Context, doc *types.JobResDoc) error {
	ll := crd.getLogger("PostCreate")
	if doc == nil {
		return fmt.Errorf("JobResDoc is nil")
	}
	if _, err := crd.jReqFac.DeleteOne(ctx, getJobReqQ(doc)); err != nil {
		ll.WithError(err).Error("failed to delete job req")
	}
	return nil
}

// PreDelete is a pre-delete hook for JobResDoc. No-op in this implementation.
func (crd *JobResCrud) PreDelete(ctx context.Context, doc *types.JobResDoc) error {
	return nil
}

// PostDelete is a post-delete hook for JobResDoc. No-op in this implementation.
func (crd *JobResCrud) PostDelete(ctx context.Context, doc *types.JobResDoc) error {
	return nil
}

// GetCollection returns the JobRes collection from the database container.
func (crd *JobResCrud) GetCollection() mngo.ICollection[types.JobResDoc] {
	return crd.container.GetMongoContainer().GetJobResCollection()
}

// getLogger returns a logrus.Entry for the given function name, tagged with the struct type.
func (crd *JobResCrud) getLogger(fn string) *logrus.Entry {
	return log.GetLogger(log.FacadeModule).WithField("func", fmt.Sprintf("%T.%s", crd, fn))
}

// getJobReqQ constructs a BSON query for the JobReqID in the given JobResDoc.
func getJobReqQ(doc *types.JobResDoc) bson.D {
	q := query.Id(doc.JobReqID)
	return q
}

// getJobRequest retrieves the related JobReqDoc for the given JobResDoc. Returns an error if not found or multiple found.
func (crd *JobResCrud) getJobRequest(ctx context.Context, doc *types.JobResDoc) (*types.JobReqDoc, error) {
	jobReqD, err := crd.jReqFac.GetCollection().Finder().Filter(getJobReqQ(doc)).Find(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get job req doc: %w", err)
	}

	if len(jobReqD) == 0 {
		return nil, fmt.Errorf("job req doc not found")
	} else if len(jobReqD) > 1 {
		return nil, fmt.Errorf("multiple job req docs found")
	}

	return jobReqD[0], nil
}

// generateFileName generates a file name for the job result based on the JobResDoc and JobReqDoc.
func (crd *JobResCrud) generateFileName(doc *types.JobResDoc, jobReq *types.JobReqDoc) string {
	return fmt.Sprintf("%s_%s", doc.ID.Hex(), jobReq.Type)
}

// processJobResult processes the job result, stores the result in Minio, and returns the update field for the media document.
func (crd *JobResCrud) processJobResult(ctx context.Context, doc *types.JobResDoc, jobReq *types.JobReqDoc, fileName string) ([]bson.D, error) {
	mno := crd.container.GetMinioContainer().GetMinioClient()
	if doc.Thumbnail != nil {
		if err := mno.FileAdd(ctx, crd.thumbFileName(fileName), doc.Thumbnail); err != nil {
			return nil, fmt.Errorf("failed to add thumbnail file to minio: %w", err)
		}
	}
	if doc.Sprite != nil {
		if err := mno.FileAdd(ctx, crd.spriteFileName(fileName), doc.Sprite); err != nil {
			return nil, fmt.Errorf("failed to add sprite file to minio: %w", err)
		}
	}
	if doc.Vtt != nil {
		if err := mno.FileAdd(ctx, crd.vttFileName(fileName), doc.Vtt); err != nil {
			return nil, fmt.Errorf("failed to add vtt file to minio: %w", err)
		}
	}
	return crd.getUpdateField(jobReq.Type, fileName)
}

// getUpdateField returns the BSON update field for the given job type and file name.
func (crd *JobResCrud) getUpdateField(jobType types.JobTypeEnum, fileName string) ([]bson.D, error) {
	switch jobType {
	case types.THUMBNAILJobType:
		return []bson.D{update.Set(types.MediaFileDoc__ThumbnailField, crd.thumbFileName(fileName))}, nil
	case types.SPRITEJobType:
		return []bson.D{update.Set(types.MediaFileDoc__SpriteField, crd.spriteFileName(fileName)), update.Set(types.MediaFileDoc__VttField, crd.vttFileName(fileName))}, nil
	default:
		return nil, fmt.Errorf("unknown job type: %s", jobType)
	}
}

// updateMediaDocument updates the media document with the given media ID and update field.
func (crd *JobResCrud) updateMediaDocument(ctx context.Context, mediaID bson.ObjectID, updateFields []bson.D) error {
	for _, q := range updateFields {
		if _, err := crd.container.GetMongoContainer().GetMediaFileCollection().Updater().Filter(query.Id(mediaID)).Updates(q).UpdateOne(ctx); err != nil {
			return fmt.Errorf("failed to update media doc: %w", err)
		}
	}

	return nil
}
func (crd *JobResCrud) vttFileName(s string) string {
	return fmt.Sprintf("%s.vtt", s)
}
func (crd *JobResCrud) thumbFileName(s string) string {
	return fmt.Sprintf("%s.jpeg", s)
}
func (crd *JobResCrud) spriteFileName(s string) string {
	return fmt.Sprintf("%s.jpeg", s)
}

// NewJobResCrud creates a new JobResCrud with the provided database container.
func NewJobResCrud(container db.IDbContainer) ICrud[types.JobResDoc] {
	jobReqFacade := NewFacade(NewJobReqCrud(container))
	return &JobResCrud{container: container, jReqFac: jobReqFacade}
}
