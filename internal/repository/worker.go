package repository

import (
	"context"

	"github.com/gotd/td/tg"
)

//go:generate mockgen -source=worker.go -destination=../../mocks/repository/worker.go -package=mocks_repository
type WorkerMediaRepository interface {
	GetTelegramDoc(ctx context.Context, WorkerID int64, messageID int) (*tg.Document, error)
	SetTelegramDoc(ctx context.Context, WorkerID int64, messageID int, telegramDoc *tg.Document) error
}
