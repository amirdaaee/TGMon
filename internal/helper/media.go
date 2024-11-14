package helper

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/amirdaaee/TGMon/internal/bot"
	"github.com/amirdaaee/TGMon/internal/db"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson/primitive"
	mongoD "go.mongodb.org/mongo-driver/mongo"
)

func AddMedia(ctx context.Context, mongo *db.Mongo, minio db.IMinioClient, doc *bot.Document, wp *bot.WorkerPool) error {
	ll := logrus.WithField("message-id", doc.MessageID)
	ll.Debug("started processing")
	docMeta := doc.GetMetadata()
	thmb := ""
	thmbData, err := wp.GetNextWorker().GetThumbnail(doc, ctx)
	if err != nil {
		logrus.WithError(err).Warn("can not get thumbnail")
	} else {
		thmb = uuid.NewString() + ".jpeg"
		if err := minio.FileAdd(thmb, thmbData, ctx); err != nil {
			thmb = ""
			logrus.WithError(err).Warn("can not store thumbnail")
		}
		ll.Debug("stored thumbnail in minio")
	}
	dbDoc := db.MediaFileDoc{
		Location:  docMeta.Location,
		FileSize:  docMeta.FileSize,
		FileName:  docMeta.FileName,
		MimeType:  docMeta.MimeType,
		FileID:    docMeta.DocID,
		MessageID: doc.MessageID,
		Duration:  docMeta.Duration,
		Thumbnail: thmb,
		DateAdded: time.Now().Unix(),
	}
	medMongo := mongo.GetMediaMongo()
	crRes, err := medMongo.DocAdd(ctx, dbDoc, nil)
	if err != nil {
		return fmt.Errorf("error adding to mongo: %s", err)
	}
	ll.Debug("done processing")
	crResID := crRes.InsertedID.(primitive.ObjectID)
	go func() {
		ctx := context.Background()
		if err := AddJob(ctx, mongo, []db.JobDoc{{MediaID: crResID, Type: db.SPRITEJobType}}); err != nil {
			logrus.WithError(err).Error("can not create sprite generation job")
		} else {
			logrus.Debug("created sprite generation job")
		}
	}()
	return nil
}
func RmMedia(ctx context.Context, mongo *db.Mongo, minio db.IMinioClient, docID string, wp *bot.WorkerPool) error {
	medMongo := mongo.GetMediaMongo()
	var mediaDoc db.MediaFileDoc
	if err := medMongo.DocGetById(ctx, docID, &mediaDoc, nil); err != nil {
		return fmt.Errorf("error get doc from db: %s", err)
	}
	if err := medMongo.DocDelById(ctx, mediaDoc.GetIDStr(), nil); err != nil {
		return fmt.Errorf("error remove doc from db: %s", err)
	}
	go func() {
		ctx := context.Background()
		if err := wp.GetNextWorker().DeleteMessages([]int{mediaDoc.MessageID}); err != nil {
			logrus.WithError(err).Error("error removing media message")
		}
		for _, m := range []string{mediaDoc.Thumbnail, mediaDoc.Sprite, mediaDoc.Vtt} {
			if m != "" {
				if err := minio.FileRm(m, ctx); err != nil {
					logrus.WithError(err).Errorf("error removing %s from media db", m)
				}
			}
		}
		if err := RmMediaJob(ctx, mongo, mediaDoc.GetIDStr()); err != nil {
			logrus.WithError(err).Error("error removing jobs from db")
		}
	}()

	return nil
}

func UpdateMediaThumbnail(ctx context.Context, mongo *db.Mongo, minio db.IMinioClient, data []byte, doc *db.MediaFileDoc, cl_ *mongoD.Client) error {
	filename := uuid.NewString() + ".jpeg"
	if err := minio.FileAdd(filename, data, ctx); err != nil {
		return fmt.Errorf("error adding file to minio: %s", err)
	}
	updateDoc := doc
	oldThumb := updateDoc.Thumbnail
	updateDoc.Thumbnail = filename
	_filter, _ := db.FilterById(updateDoc.GetIDStr())
	updateDoc.SetID(primitive.NilObjectID)
	if _, err := mongo.IMng.GetCollection(cl_).ReplaceOne(ctx, _filter, updateDoc); err != nil {
		return fmt.Errorf("can not replace mongo record: %s", err)
	}
	if oldThumb != "" {
		minio.FileRm(oldThumb, ctx)
	}
	return nil
}
func UpdateMediaVtt(ctx context.Context, mongo *db.Mongo, minio db.IMinioClient, image []byte, vtt []byte, doc *db.MediaFileDoc, cl_ *mongoD.Client) error {
	u := uuid.NewString()
	spriteName := u + ".jpeg"
	vttName := u + ".vtt"
	if err := minio.FileAdd(spriteName, image, ctx); err != nil {
		return fmt.Errorf("error addign image file to minio: %s", err)
	}
	vttStr := strings.ReplaceAll(string(vtt), "__NAME__", spriteName)
	if err := minio.FileAddStr(vttName, vttStr, ctx); err != nil {
		return fmt.Errorf("error addign vtt file to minio: %s", err)
	}
	updateDoc := doc
	oldVtt := updateDoc.Vtt
	oldSprite := updateDoc.Sprite
	updateDoc.Vtt = vttName
	updateDoc.Sprite = spriteName
	_filter, _ := db.FilterById(updateDoc.GetIDStr())
	updateDoc.SetID(primitive.NilObjectID)
	if _, err := mongo.IMng.GetCollection(cl_).ReplaceOne(ctx, _filter, updateDoc); err != nil {
		return fmt.Errorf("can not replace mongo record: %s", err)
	}
	if oldVtt != "" {
		minio.FileRm(oldVtt, ctx)
	}
	if oldSprite != "" {
		minio.FileRm(oldSprite, ctx)
	}
	return nil
}
