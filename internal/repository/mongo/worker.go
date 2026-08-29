package mongo

import (
	"context"
	"fmt"

	"github.com/amirdaaee/TGMon/internal/repository"
	"github.com/amirdaaee/TGMon/internal/types"
	"github.com/chenmingyong0423/go-mongox/v2/bsonx"
	"github.com/chenmingyong0423/go-mongox/v2/builder/update"
	"github.com/gotd/td/bin"
	"github.com/gotd/td/tg"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// WorkerRepo is the MongoDB implementation of repository.IWorkerRepository.
type WorkerRepo struct {
	collectionStore[types.WorkerMediaDoc]
}

var _ repository.WorkerMediaRepository = (*WorkerRepo)(nil)

func (r *WorkerRepo) GetTelegramDoc(ctx context.Context, WorkerID int64, messageID int) (*tg.Document, error) {
	doc, err := r.coll.Finder().Filter(workerMessageFilter(WorkerID, messageID)).FindOne(ctx)
	if err != nil {
		return nil, mapNotFound(err)
	}
	return decodeTelegramDoc(doc.TelegramDoc)
}

func (r *WorkerRepo) SetTelegramDoc(ctx context.Context, WorkerID int64, messageID int, telegramDoc *tg.Document) error {
	raw, err := encodeTelegramDoc(telegramDoc)
	if err != nil {
		return err
	}
	_, err = r.coll.Updater().
		Filter(workerMessageFilter(WorkerID, messageID)).
		Updates(update.Set(types.WorkerMediaDoc__TelegramDocField, raw)).
		Upsert(ctx)
	return err
}

func encodeTelegramDoc(doc *tg.Document) ([]byte, error) {
	if doc == nil {
		return nil, fmt.Errorf("telegram doc is nil")
	}
	var buf bin.Buffer
	if err := doc.Encode(&buf); err != nil {
		return nil, fmt.Errorf("error encoding telegram doc: %w", err)
	}
	return buf.Buf, nil
}

func decodeTelegramDoc(raw []byte) (*tg.Document, error) {
	if len(raw) == 0 {
		return nil, fmt.Errorf("telegram doc is empty")
	}
	doc := tg.Document{}
	if err := doc.Decode(&bin.Buffer{Buf: raw}); err != nil {
		return nil, fmt.Errorf("error decoding telegram doc: %w", err)
	}
	return &doc, nil
}

func workerMessageFilter(workerID int64, messageID int) bson.D {
	return bsonx.NewD().
		Add(types.WorkerMediaDoc__WorkerIDField, workerID).
		Add(types.WorkerMediaDoc__MessageIDField, messageID).
		Build()
}

// NewWorkerRepo returns a worker media repository backed by the given client.
func NewWorkerRepo(c *Client) *WorkerRepo {
	return &WorkerRepo{collectionStore: newCollectionStore[types.WorkerMediaDoc](c, workerMediaCollectionName)}
}
