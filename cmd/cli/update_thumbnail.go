package main

import (
	"context"

	"github.com/amirdaaee/TGMon/cmd"
	"github.com/amirdaaee/TGMon/internal/bot"
	"github.com/amirdaaee/TGMon/internal/db"
	"github.com/google/uuid"
	"github.com/gotd/td/tg"
	"github.com/sirupsen/logrus"
)

func updateThumbnail() {
	wp, err := cmd.GetWorkerPool()
	if err != nil {
		logrus.WithError(err).Fatal("can not start workers")
	}
	mongo := cmd.GetMongoDB()
	minio, err := cmd.GetMinioDB()
	if err != nil {
		logrus.WithError(err).Fatal("can not create minio client")
	}
	// ...
	ctx := context.TODO()
	mongoCl, err := mongo.GetClient()
	if err != nil {
		logrus.WithError(err).Fatal("can not create mongo client")
	}
	defer mongoCl.Disconnect(ctx)
	mongoColl := mongo.GetFileCollection(mongoCl)
	mediaDocList := []db.MediaFileDoc{}
	if err := mongo.DocGetAll(ctx, mongoColl, &mediaDocList, mongoCl); err != nil {
		logrus.WithError(err).Fatal("error getting current records")
	}
	mediaDocInputMsgList := []tg.InputMessageClass{}
	for _, MedDoc := range mediaDocList {
		if MedDoc.Thumbnail == "" {
			mediaDocInputMsgList = append(mediaDocInputMsgList, &tg.InputMessageID{ID: MedDoc.MessageID})
		}
	}
	if len(mediaDocInputMsgList) == 0 {
		logrus.Info("no document with empty thumbnail")
		logrus.Exit(0)
	}
	// ...
	w := wp.GetNextWorker()
	chatList, err := w.Client.API().ChannelsGetChannels(ctx, []tg.InputChannelClass{&tg.InputChannel{ChannelID: w.TargetChannelId}})
	if err != nil {
		logrus.WithError(err).Fatal("can not get channel")
	}
	var channel tg.InputChannelClass
	for _, cht := range chatList.GetChats() {
		if cht.GetID() == w.TargetChannelId {
			if chn, ok := cht.(*tg.Channel); !ok {
				logrus.Fatal("target channel is not a channel!")
			} else {
				channel = chn.AsInput()
				break
			}
		}
	}
	if channel == nil {
		logrus.Fatal("target channel not found!")
	}
	logrus.WithField("channel", channel).Debug("found channel")
	// ...
	allMsgsCls, err := w.Client.API().ChannelsGetMessages(ctx, &tg.ChannelsGetMessagesRequest{Channel: channel, ID: mediaDocInputMsgList})
	if err != nil {
		logrus.WithError(err).Fatal("can not get messages!")
	}
	allMsgs, ok := allMsgsCls.(*tg.MessagesChannelMessages)
	if !ok {
		logrus.Fatalf("allMsgsCls is %T!", allMsgsCls)
	}
	// ...
	for c, medDoc := range mediaDocList {
		doc := bot.Document{}
		doc.FromMessage(allMsgs.Messages[c])
		thumb, err := w.GetThumbnail(&doc, ctx)
		if err != nil {
			logrus.WithError(err).Warn("can not get thumbnail")
			continue
		}
		updateDoc := medDoc
		oldThumb := updateDoc.Thumbnail
		filename := uuid.NewString() + ".jpeg"
		if err := minio.FileAdd(filename, thumb, ctx); err != nil {
			logrus.WithError(err).Warn("error storing thumbnail")
			continue
		}
		updateDoc.Thumbnail = filename
		_filter, _ := db.FilterById(updateDoc.ID)
		updateDoc.ID = ""
		if _, err := mongoColl.ReplaceOne(ctx, _filter, updateDoc); err != nil {
			logrus.WithError(err).Warn("can not replace mongo record")
			continue
		}
		if oldThumb != "" {
			if err := minio.FileRm(oldThumb, ctx); err != nil {
				logrus.WithError(err).Warn("can not remove old thumbnail")
			}
		}
		logrus.Info("updated")
	}
}
