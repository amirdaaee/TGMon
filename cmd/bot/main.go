package main

import (
	"context"
	"time"

	"github.com/amirdaaee/TGMon/cmd"
	"github.com/amirdaaee/TGMon/internal/bot"
	"github.com/amirdaaee/TGMon/internal/db"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

func main() {
	wp, err := cmd.GetWorkerPool()
	if err != nil {
		logrus.WithError(err).Fatal("can not start workers")
	}
	notifer, err := cmd.GetMasterBot()
	if err != nil {
		logrus.WithError(err).Fatal("can not start master bot")
	}
	mongo := cmd.GetMongoDB()
	minio, err := cmd.GetMinioDB()
	if err != nil {
		logrus.WithError(err).Fatal("can not create minio client")
	}
	// ...
	for {
		select {
		case doc := <-notifer.DocNotifier.Chan:
			go func(_doc *bot.Document) {
				ll := logrus.WithField("message-id", _doc.MessageID)
				ll.Debug("started processing")
				ctx := context.TODO()
				ll.Debug("getting metadata")
				docMeta := _doc.GetMetadata()
				ll.Debug("getting thumbnail")
				thmb := ""
				thmbData, err := wp.GetNextWorker().GetThumbnail(_doc, ctx)
				if err != nil {
					logrus.WithError(err).Warn("can not get thumbnail")
				} else {
					thmb = uuid.NewString() + ".jpeg"
					ll.Debug("storing thumbnail in minio")
					if err := minio.FileAdd(thmb, thmbData, ctx); err != nil {
						thmb = ""
						logrus.WithError(err).Warn("can not store thumbnail")
					}
				}
				dbDoc := db.MediaFileDoc{
					Location:  docMeta.Location,
					FileSize:  docMeta.FileSize,
					FileName:  docMeta.FileName,
					MimeType:  docMeta.MimeType,
					FileID:    docMeta.DocID,
					MessageID: _doc.MessageID,
					Duration:  docMeta.Duration,
					Thumbnail: thmb,
					DateAdded: time.Now().Unix(),
				}
				ll.Debug("adding to mongo")
				if _, err := mongo.DocAdd(ctx, dbDoc, nil); err != nil {
					logrus.WithError(err).Error("error adding to mongo")
				}
				ll.Debug("done processing")
			}(doc)

		}
	}
}
