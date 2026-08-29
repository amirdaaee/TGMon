package crd

import (
	"bytes"
	"context"
	"fmt"

	ftypes "github.com/amirdaaee/TGMon/internal/facade/types"
	"github.com/amirdaaee/TGMon/internal/filesystem/cache"
	"github.com/amirdaaee/TGMon/internal/log"
	"github.com/amirdaaee/TGMon/internal/repository"
	"github.com/amirdaaee/TGMon/internal/types"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.uber.org/zap"
)

// JobReqCrud implements ICrud for JobReqDoc, providing CRUD hooks.
type JobReqCrud struct{}

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

// NewJobReqCrud creates a new JobReqCrud.
func NewJobReqCrud() ftypes.ICrud[types.JobReqDoc] {
	return &JobReqCrud{}
}

// ===
// ===

// JobResCrud implements ICrud for JobResDoc, providing CRUD hooks.
type JobResCrud struct {
	jobReqs repository.JobReqRepository
	media   repository.MediaFileRepository
	objects repository.ObjectStore
	jReqFac ftypes.IFacade[types.JobReqDoc]
	fsCache *cache.DBCache[string, *types.MediaFileDoc]
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
	if _, err := crd.jReqFac.DeleteByID(ctx, doc.JobReqID); err != nil {
		ll.With(zap.Error(err)).Error("failed to delete job req")
	}
	crd.fsCache.Invalidate(ctx)
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

func (crd *JobResCrud) getLogger(fn string) *zap.Logger {
	return log.Named(log.FacadeModule, fmt.Sprintf("%T.%s", crd, fn))
}

func (crd *JobResCrud) getJobRequest(ctx context.Context, doc *types.JobResDoc) (*types.JobReqDoc, error) {
	return crd.jobReqs.FindByID(ctx, doc.JobReqID)
}

func (crd *JobResCrud) processJobResult(ctx context.Context, doc *types.JobResDoc, jobReq *types.JobReqDoc) error {
	ll := crd.getLogger("processJobResult").With(zap.String("mediaID", jobReq.MediaID.Hex()), zap.String("jobType", string(jobReq.Type)))
	jp, err := NewResultProcessor(jobReq, doc)
	if err != nil {
		return fmt.Errorf("failed to create result processor: %w", err)
	}
	if err := jp.Validate(); err != nil {
		return fmt.Errorf("failed to validate job result: %w", err)
	}
	files, apply, err := jp.Process(ctx)
	if err != nil {
		return fmt.Errorf("failed to process job result: %w", err)
	}
	for fname, data := range files {
		if err := crd.objects.Put(ctx, fname, data); err != nil {
			return fmt.Errorf("failed to add file to object store: %w", err)
		}
	}
	ll.Info("object store files added")
	if err := apply(ctx, crd.media, jobReq.MediaID); err != nil {
		return fmt.Errorf("failed to update media doc: %w", err)
	}
	ll.Info("mongo doc updated")
	return nil
}

// NewJobResCrud creates a new JobResCrud with the provided repositories.
func NewJobResCrud(jobReqs repository.JobReqRepository, media repository.MediaFileRepository, objects repository.ObjectStore, jobReqFacade ftypes.IFacade[types.JobReqDoc], fsCache *cache.DBCache[string, *types.MediaFileDoc]) ftypes.ICrud[types.JobResDoc] {
	return &JobResCrud{jobReqs: jobReqs, media: media, objects: objects, jReqFac: jobReqFacade, fsCache: fsCache}
}

// ===
// ===
type mediaApplyFunc func(ctx context.Context, media repository.MediaFileRepository, mediaID bson.ObjectID) error

type TaskResultProcessor interface {
	Validate() error
	Process(ctx context.Context) (map[string][]byte, mediaApplyFunc, error)
}

type baseResultProcessor struct {
	jobReqDoc *types.JobReqDoc
	jobResDoc *types.JobResDoc
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

func (p *ThumbnailResultProcessor) Process(ctx context.Context) (map[string][]byte, mediaApplyFunc, error) {
	fname := generateSuffixedFileName(p.jobReqDoc, ".jpeg")
	files := map[string][]byte{fname: p.jobResDoc.Thumbnail}
	return files, func(ctx context.Context, media repository.MediaFileRepository, mediaID bson.ObjectID) error {
		return media.SetThumbnail(ctx, mediaID, fname)
	}, nil
}

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

func (p *SpriteResultProcessor) Process(ctx context.Context) (map[string][]byte, mediaApplyFunc, error) {
	fname := generateSuffixedFileName(p.jobReqDoc, "")
	spriteFname := fmt.Sprintf("%s.jpeg", fname)
	vttFname := fmt.Sprintf("%s.vtt", fname)
	vtt := bytes.ReplaceAll(p.jobResDoc.Vtt, []byte(VTT_SPRITE_NAME_PLACEHOLDER), []byte(spriteFname))
	files := map[string][]byte{spriteFname: p.jobResDoc.Sprite, vttFname: vtt}
	return files, func(ctx context.Context, media repository.MediaFileRepository, mediaID bson.ObjectID) error {
		return media.SetSpriteAndVtt(ctx, mediaID, spriteFname, vttFname)
	}, nil
}

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

func (p *EmbeddingResultProcessor) Process(ctx context.Context) (map[string][]byte, mediaApplyFunc, error) {
	return nil, func(ctx context.Context, media repository.MediaFileRepository, mediaID bson.ObjectID) error {
		return media.SetHasHash(ctx, mediaID, true)
	}, nil
}

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

func (p *TranscriptionResultProcessor) Process(ctx context.Context) (map[string][]byte, mediaApplyFunc, error) {
	fname := generateSuffixedFileName(p.jobReqDoc, ".srt")
	files := map[string][]byte{fname: p.jobResDoc.Transcription}
	return files, func(ctx context.Context, media repository.MediaFileRepository, mediaID bson.ObjectID) error {
		return media.SetSrt(ctx, mediaID, fname)
	}, nil
}

func NewResultProcessor(jobReqDoc *types.JobReqDoc, jobResDoc *types.JobResDoc) (TaskResultProcessor, error) {
	baseProcessor := baseResultProcessor{jobReqDoc: jobReqDoc, jobResDoc: jobResDoc}
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

func generateSuffixedFileName(jobReq *types.JobReqDoc, ext string) string {
	v := fmt.Sprintf("%s_%s", jobReq.MediaID.Hex(), jobReq.Type)
	if ext != "" {
		v = fmt.Sprintf("%s%s", v, ext)
	}
	return v
}
