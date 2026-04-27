package crd

import (
	"bytes"
	"context"
	"fmt"

	"github.com/amirdaaee/TGMon/internal/db"
	mngo "github.com/amirdaaee/TGMon/internal/db/mongo"
	ftypes "github.com/amirdaaee/TGMon/internal/facade/types"
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

var _ ftypes.ICrud[types.JobReqDoc] = (*JobReqCrud)(nil)

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
func NewJobReqCrud(container db.IDbContainer) ftypes.ICrud[types.JobReqDoc] {
	return &JobReqCrud{container: container}
}

// ===
// ===

// JobResCrud implements ICrud for JobResDoc, providing CRUD hooks and collection access.
type JobResCrud struct {
	container db.IDbContainer
	jReqFac   ftypes.IFacade[types.JobReqDoc]
}

var _ ftypes.ICrud[types.JobResDoc] = (*JobResCrud)(nil)

// PreCreate processes the job result and updates the related media document. Returns an error if the document is nil or processing fails.
func (crd *JobResCrud) PreCreate(ctx context.Context, doc *types.JobResDoc) error {
	if doc == nil {
		return fmt.Errorf("JobResDoc is nil")
	}
	jobReq, err := crd.getJobRequest(ctx, doc)
	if err != nil {
		return fmt.Errorf("failed to get corresponding job request: %w", err)
	}
	if err := crd.processJobResult(ctx, doc, jobReq); err != nil {
		return fmt.Errorf("failed to process job result: %w", err)
	}
	return nil
}

// PostCreate deletes the related job request after creating a job result. Logs errors but does not return them.
func (crd *JobResCrud) PostCreate(ctx context.Context, doc *types.JobResDoc) error {
	ll := crd.getLogger("PostCreate")
	if doc == nil {
		return fmt.Errorf("JobResDoc is nil")
	}
	if _, err := crd.jReqFac.DeleteOne(ctx, crd.getJobReqQ(doc)); err != nil {
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
func (crd *JobResCrud) getJobReqQ(doc *types.JobResDoc) bson.D {
	q := query.Id(doc.JobReqID)
	return q
}

// getJobRequest retrieves the related JobReqDoc for the given JobResDoc. Returns an error if not found or multiple found.
func (crd *JobResCrud) getJobRequest(ctx context.Context, doc *types.JobResDoc) (*types.JobReqDoc, error) {
	jobReqD, err := crd.jReqFac.GetCollection().Finder().Filter(crd.getJobReqQ(doc)).Find(ctx)
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

// processJobResult processes the job result, stores the result in Minio, and returns the update field for the media document.
func (crd *JobResCrud) processJobResult(ctx context.Context, doc *types.JobResDoc, jobReq *types.JobReqDoc) error {
	ll := crd.getLogger("processJobResult").WithField("mediaID", jobReq.MediaID.Hex()).WithField("jobType", jobReq.Type)
	jp, err := NewResultProcessor(jobReq, doc, crd.container)
	if err != nil {
		return fmt.Errorf("failed to create result processor: %w", err)
	}
	if err := jp.Validate(); err != nil {
		return fmt.Errorf("failed to validate job result: %w", err)
	}
	files, updates, err := jp.Process(ctx)
	if err != nil {
		return fmt.Errorf("failed to process job result: %w", err)
	}
	mnio := crd.container.GetMinioContainer().GetMinioClient()
	for fname, data := range files {
		if err := mnio.FileAdd(ctx, fname, data); err != nil {
			return fmt.Errorf("failed to add file to minio: %w", err)
		}
	}
	ll.Info("minio files added")
	// ...
	mngoColl := crd.container.GetMongoContainer().GetMediaFileCollection()
	for _, q := range updates {
		if _, err := mngoColl.Updater().Filter(query.Id(jobReq.MediaID)).Updates(q).UpdateOne(ctx); err != nil {
			return fmt.Errorf("failed to update media doc: %w", err)
		}
	}
	ll.Info("mongo doc updated")
	return nil

}

// NewJobResCrud creates a new JobResCrud with the provided database container.
func NewJobResCrud(container db.IDbContainer, jobReqFacade ftypes.IFacade[types.JobReqDoc]) ftypes.ICrud[types.JobResDoc] {
	return &JobResCrud{container: container, jReqFac: jobReqFacade}
}

// ===
// ===
type TaskResultProcessor interface {
	Validate() error
	Process(ctx context.Context) (map[string][]byte, []bson.D, error)
}
type baseResultProcessor struct {
	jobReqDoc *types.JobReqDoc
	jobResDoc *types.JobResDoc
	container db.IDbContainer
}
type ThumbnailResultProcessor struct {
	baseResultProcessor
}

var _ TaskResultProcessor = (*ThumbnailResultProcessor)(nil)

func (p *ThumbnailResultProcessor) Validate() error {
	if p.jobReqDoc.Type != types.THUMBNAILJobType {
		return fmt.Errorf("job type is not thumbnail")
	}
	if p.jobResDoc.Thumbnail == nil {
		return fmt.Errorf("thumbnail is nil")
	}
	return nil
}

func (p *ThumbnailResultProcessor) Process(ctx context.Context) (map[string][]byte, []bson.D, error) {
	fname := generateSuffixedFileName(p.jobReqDoc, ".jpeg")
	files := map[string][]byte{fname: p.jobResDoc.Thumbnail}
	MongoUpdates := []bson.D{update.Set(types.MediaFileDoc__ThumbnailField, fname)}
	return files, MongoUpdates, nil
}

// ---
const VTT_SPRITE_NAME_PLACEHOLDER = "__NAME__"

type SpriteResultProcessor struct {
	baseResultProcessor
}

var _ TaskResultProcessor = (*SpriteResultProcessor)(nil)

func (p *SpriteResultProcessor) Validate() error {
	if p.jobReqDoc.Type != types.SPRITEJobType {
		return fmt.Errorf("job type is not sprite")
	}
	if p.jobResDoc.Sprite == nil {
		return fmt.Errorf("sprite is nil")
	}
	if p.jobResDoc.Vtt == nil {
		return fmt.Errorf("vtt is nil")
	}
	return nil
}

func (p *SpriteResultProcessor) Process(ctx context.Context) (map[string][]byte, []bson.D, error) {
	fname := generateSuffixedFileName(p.jobReqDoc, "")
	spriteFname := fmt.Sprintf("%s.jpeg", fname)
	vttFname := fmt.Sprintf("%s.vtt", fname)
	vtt := bytes.ReplaceAll(p.jobResDoc.Vtt, []byte(VTT_SPRITE_NAME_PLACEHOLDER), []byte(spriteFname))
	files := map[string][]byte{spriteFname: p.jobResDoc.Sprite, vttFname: vtt}
	MongoUpdates := []bson.D{update.Set(types.MediaFileDoc__SpriteField, spriteFname), update.Set(types.MediaFileDoc__VttField, vttFname)}
	return files, MongoUpdates, nil
}

// ---
type EmbeddingResultProcessor struct {
	baseResultProcessor
}

var _ TaskResultProcessor = (*EmbeddingResultProcessor)(nil)

func (p *EmbeddingResultProcessor) Validate() error {
	if p.jobReqDoc.Type != types.EmbeddingJobType {
		return fmt.Errorf("job type is not embedding")
	}
	return nil
}
func (p *EmbeddingResultProcessor) Process(ctx context.Context) (map[string][]byte, []bson.D, error) {
	MongoUpdates := []bson.D{update.Set(types.MediaFileDoc__HasHashField, true)}
	return nil, MongoUpdates, nil
}

// ---
type TranscriptionResultProcessor struct {
	baseResultProcessor
}

var _ TaskResultProcessor = (*TranscriptionResultProcessor)(nil)

func (p *TranscriptionResultProcessor) Validate() error {
	if p.jobReqDoc.Type != types.TranscriptionJobType {
		return fmt.Errorf("job type is not transcription")
	}
	if p.jobResDoc.Transcription == nil {
		return fmt.Errorf("transcription is nil")
	}
	return nil
}
func (p *TranscriptionResultProcessor) Process(ctx context.Context) (map[string][]byte, []bson.D, error) {
	fname := generateSuffixedFileName(p.jobReqDoc, ".srt")
	files := map[string][]byte{fname: p.jobResDoc.Transcription}
	MongoUpdates := []bson.D{update.Set(types.MediaFileDoc__SrtField, fname)}
	return files, MongoUpdates, nil
}

// ---
func NewResultProcessor(jobReqDoc *types.JobReqDoc, jobResDoc *types.JobResDoc, container db.IDbContainer) (TaskResultProcessor, error) {
	baseProcessor := baseResultProcessor{jobReqDoc: jobReqDoc, jobResDoc: jobResDoc, container: container}
	switch jobReqDoc.Type {
	case types.THUMBNAILJobType:
		return &ThumbnailResultProcessor{baseProcessor}, nil
	case types.SPRITEJobType:
		return &SpriteResultProcessor{baseProcessor}, nil
	case types.EmbeddingJobType:
		return &EmbeddingResultProcessor{baseProcessor}, nil
	case types.TranscriptionJobType:
		return &TranscriptionResultProcessor{baseProcessor}, nil
	default:
		return nil, fmt.Errorf("unknown job type: %s", jobReqDoc.Type)
	}
}

// ===
// ===
// generateFileName generates a file name for the job result based on the JobResDoc and JobReqDoc.
func generateSuffixedFileName(jobReq *types.JobReqDoc, ext string) string {
	v := fmt.Sprintf("%s_%s", jobReq.MediaID.Hex(), jobReq.Type)
	if ext != "" {
		v = fmt.Sprintf("%s%s", v, ext)
	}
	return v
}
