/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"

	ccmd "github.com/amirdaaee/TGMon/cmd"
	"github.com/amirdaaee/TGMon/internal/bot"
	"github.com/amirdaaee/TGMon/internal/db"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// updateThumbnailCmd represents the updateThumbnail command
var updateThumbnailCmd = &cobra.Command{
	Use:   "updateThumbnail",
	Short: "",
	Long:  "",
	Run: func(cmd *cobra.Command, args []string) {
		updateThumbnail()
	},
}

func init() {
	rootCmd.AddCommand(updateThumbnailCmd)
}

func updateThumbnail() {
	wp, err := ccmd.GetWorkerPool()
	if err != nil {
		logrus.WithError(err).Fatal("can not start workers")
	}
	mongo := ccmd.GetMongoDB()
	minio, err := ccmd.GetMinioDB()
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
	mongoColl := mongo.GetMediaMongo().IMng.GetCollection(mongoCl)
	mediaDocList := []db.MediaFileDoc{}
	if err := mongo.DocGetAll(ctx, &mediaDocList, mongoCl); err != nil {
		logrus.WithError(err).Fatal("error getting current records")
	}
	msgIdList := []int{}
	mediaDocListUpdate := []db.MediaFileDoc{}
	for _, MedDoc := range mediaDocList {
		if MedDoc.Thumbnail == "" {
			msgIdList = append(msgIdList, MedDoc.MessageID)
			mediaDocListUpdate = append(mediaDocListUpdate, MedDoc)
		}
	}
	if len(msgIdList) == 0 {
		logrus.Info("no document with empty thumbnail")
		logrus.Exit(0)
	}
	// ...
	allMsgs, err := wp.GetNextWorker().GetMessages(msgIdList, ctx)
	if err != nil {
		logrus.WithError(err).Fatal("can not get messages")
	}
	// ...
	for c, medDoc := range mediaDocListUpdate {
		doc := bot.Document{}
		doc.FromMessage(allMsgs.Messages[c])
		thumb, err := wp.GetNextWorker().GetThumbnail(&doc, ctx)
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
