package crd

import (
	"context"
	"fmt"
	"time"

	ftypes "github.com/amirdaaee/TGMon/internal/facade/types"
	"github.com/amirdaaee/TGMon/internal/filesystem/cache"
	"github.com/amirdaaee/TGMon/internal/log"
	"github.com/amirdaaee/TGMon/internal/repository"
	"github.com/amirdaaee/TGMon/internal/types"
	"github.com/amirdaaee/TGMon/internal/worker"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// MediaCrud implements ICrud for MediaFileDoc, providing CRUD hooks.
type MediaCrud struct {
	media           repository.MediaFileRepository
	objects         repository.ObjectStore
	jobReqs         repository.JobReqRepository
	jReqFac         ftypes.IFacade[types.JobReqDoc]
	workerContainer worker.IWorkerPool
	keepDup         bool
	fsCache         *cache.DBCache[string, *types.MediaFileDoc]
	mediaMeta       repository.MediaExtendedMetaRepository
}

var _ ftypes.ICrud[types.MediaFileDoc] = (*MediaCrud)(nil)

// PreCreate checks for duplicates before creating a MediaFileDoc. Returns an error if the document is nil or a duplicate is found.
func (crd *MediaCrud) PreCreate(ctx context.Context, doc *types.MediaFileDoc) error {
	ll := crd.getLogger("PreCreate")
	if doc == nil {
		return fmt.Errorf("MediaFileDoc is nil")
	}
	if crd.keepDup {
		return nil
	}
	n, err := crd.media.CountByFileID(ctx, doc.Meta.FileID)
	if err != nil {
		ll.With(zap.Error(err)).Error("failed to check for duplicates")
	} else if n > 0 {
		return fmt.Errorf("%w: %d", ftypes.ErrFileAlreadyExists, doc.Meta.FileID)
	}
	return nil
}

// PostCreate creates a sprite job request after creating a media file. Returns an error if the document is nil or job creation fails.
func (crd *MediaCrud) PostCreate(ctx context.Context, doc *types.MediaFileDoc) error {
	ll := crd.getLogger("PostCreate")
	if doc == nil {
		return fmt.Errorf("MediaFileDoc is nil")
	}
	newCtx := context.TODO()
	go func() {
		setInitialJobs(newCtx, crd, doc, ll)
		setMediaThumbnail(newCtx, crd, doc, ll)
		setUName(newCtx, crd, doc, ll)
		crd.fsCache.Invalidate(newCtx)
	}()
	crd.fsCache.Invalidate(ctx)
	return nil
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
	if deleted, err := crd.jobReqs.DeleteByMediaID(ctx, doc.ID); err != nil {
		ll.With(zap.Error(err)).Error("failed to delete orphaned jobs")
	} else if deleted > 0 {
		ll.Sugar().Debugf("deleted %d orphaned jobs", deleted)
	}
	if err := crd.mediaMeta.DeleteByMediaFileID(ctx, doc.ID); err != nil {
		ll.With(zap.Error(err)).Error("failed to delete orphaned media meta")
	}
	for _, fn := range []string{doc.Vtt, doc.Thumbnail, doc.Srt, doc.Sprite} {
		if fn != "" {
			var lastErr error
			for i := 0; i < 3; i++ {
				if err := crd.objects.Delete(ctx, fn); err != nil {
					lastErr = err
					time.Sleep(100 * time.Millisecond)
				} else {
					lastErr = nil
					break
				}
			}
			if lastErr != nil {
				ll.With(zap.Error(lastErr)).Error("failed to remove orphaned file after retries")
			}
		}
	}
	crd.fsCache.Invalidate(ctx)
	return nil
}

func (crd *MediaCrud) getLogger(fn string) *zap.Logger {
	return log.Named(log.FacadeModule, fmt.Sprintf("%T.%s", crd, fn))
}

// NewMediaCrud creates a new MediaCrud with the provided repositories.
func NewMediaCrud(media repository.MediaFileRepository, objects repository.ObjectStore, jobReqs repository.JobReqRepository, workerContainer worker.IWorkerPool, keepDup bool, jobReqFacade ftypes.IFacade[types.JobReqDoc], fsCache *cache.DBCache[string, *types.MediaFileDoc], mediaMeta repository.MediaExtendedMetaRepository) ftypes.ICrud[types.MediaFileDoc] {
	return &MediaCrud{
		media:           media,
		objects:         objects,
		jobReqs:         jobReqs,
		jReqFac:         jobReqFacade,
		workerContainer: workerContainer,
		keepDup:         keepDup,
		fsCache:         fsCache,
		mediaMeta:       mediaMeta,
	}
}

func setMediaThumbnail(ctx context.Context, crd *MediaCrud, doc *types.MediaFileDoc, ll *zap.Logger) {
	wrkr := crd.workerContainer.GetNextWorker()
	if wrkr == nil {
		ll.Error("no available worker")
		return
	}
	thumb, err := wrkr.GetThumbnail(ctx, doc.MessageID)
	if err != nil {
		ll.With(zap.Error(err)).Error("failed to set initial thumbnail")
		return
	}
	fname := fmt.Sprintf("%s.jpg", uuid.NewString())
	if err := crd.objects.Put(ctx, fname, thumb); err != nil {
		ll.With(zap.Error(err)).Error("failed to add thumbnail to object store")
		return
	}
	if err := crd.media.SetThumbnail(ctx, doc.ID, fname); err != nil {
		ll.With(zap.Error(err)).Error("failed to update thumbnail in db")
		return
	}
	ll.Info("initial thumbnail set")
}

func setInitialJobs(ctx context.Context, crd *MediaCrud, doc *types.MediaFileDoc, ll *zap.Logger) {
	for _, jobType := range []types.JobTypeEnum{types.SPRITEJobType, types.THUMBNAILJobType, types.EmbeddingJobType, types.TranscriptionJobType} {
		if _, err := crd.jReqFac.CreateOne(ctx, &types.JobReqDoc{
			Type:    jobType,
			MediaID: doc.ID,
		}); err != nil {
			ll.With(zap.Error(err), zap.String("jobType", string(jobType))).Error("failed to create job")
		} else {
			ll.Sugar().Debugf("%s job created", jobType)
		}
	}
}

func setUName(ctx context.Context, crd *MediaCrud, doc *types.MediaFileDoc, ll *zap.Logger) {
	if err := crd.media.SetUName(ctx, doc.ID, doc.Name()); err != nil {
		ll.With(zap.Error(err)).Error("failed to set uname")
		return
	}
	ll.Info("uname set")
}
