package worker

import (
	"context"
	"fmt"

	"github.com/amirdaaee/TGMon/internal/repository"
	"github.com/gotd/td/tg"
	"go.uber.org/zap"
)

//go:generate mockgen -source=cache.go -destination=../../mocks/worker/cache.go -package=mocks_worker
type ITGDocCache interface {
	GetOrSet(ctx context.Context, WorkerID int64, messageID int, fn func() (*tg.Document, error)) (*tg.Document, error)
}
type tgDocCache struct {
	workerRepo repository.WorkerMediaRepository
	ll         *zap.Logger
}

var _ ITGDocCache = (*tgDocCache)(nil)

func (c *tgDocCache) GetOrSet(ctx context.Context, WorkerID int64, messageID int, fn func() (*tg.Document, error)) (*tg.Document, error) {
	doc, err := c.workerRepo.GetTelegramDoc(ctx, WorkerID, messageID)
	if err != nil {
		c.ll.With(zap.Error(err)).Error("error getting telegram doc from db")
	} else {
		return doc, nil
	}
	doc, err = fn()
	if err != nil {
		return nil, fmt.Errorf("error getting live telegram doc: %w", err)
	}
	if err := c.workerRepo.SetTelegramDoc(ctx, WorkerID, messageID, doc); err != nil {
		c.ll.With(zap.Error(err)).Error("error setting telegram doc to db")
	}
	return doc, nil
}
