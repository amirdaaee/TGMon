/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"fmt"

	ccmd "github.com/amirdaaee/TGMon/cmd"
	"github.com/amirdaaee/TGMon/internal/db"
	"github.com/amirdaaee/TGMon/internal/ffmpeg"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// updateThumbnailCmd represents the updateThumbnail command
var generateThumbnailCmd = &cobra.Command{
	Use:   "generateThumbnail",
	Short: "",
	Long:  "",
	Run:   generateThumbnail,
}

func init() {
	rootCmd.AddCommand(generateThumbnailCmd)
	generateThumbnailCmd.Flags().StringArray("mediaID", []string{}, "media list to generate thumbnail for")
	generateThumbnailCmd.Flags().String("server", "http://localhost:8080", "address of running media server")
	generateThumbnailCmd.Flags().String("image", "linuxserver/ffmpeg", "ffmpeg docker image")
	generateThumbnailCmd.Flags().Float64("time", 0.1, "frame to extract as thumbnail")
}

func generateThumbnail(cmd *cobra.Command, args []string) {
	mediaID, err := cmd.Flags().GetStringArray("mediaID")
	if err != nil {
		logrus.WithError(err).Fatal("can not get mediaID arg")
	}
	server, err := cmd.Flags().GetString("server")
	if err != nil {
		logrus.WithError(err).Fatal("can not get server arg")
	}
	ffimage, err := cmd.Flags().GetString("image")
	if err != nil {
		logrus.WithError(err).Fatal("can not get image arg")
	}
	time, err := cmd.Flags().GetFloat64("time")
	if err != nil {
		logrus.WithError(err).Fatal("can not get time arg")
	}
	ctx := context.Background()
	mongo := ccmd.GetMongoDB()
	mongoCl, err := mongo.GetClient()
	if err != nil {
		logrus.WithError(err).Fatal("can not create mongo client")
	}
	defer mongoCl.Disconnect(ctx)
	minio, err := ccmd.GetMinioDB()
	if err != nil {
		logrus.WithError(err).Fatal("can not create minio client")
	}
	ffContainer, err := ffmpeg.NewFFmpegContainer(ffimage)
	if err != nil {
		logrus.WithError(err).Fatal("can not create ffmpeg container")
	}
	defer ffContainer.Close()
	for _, m := range mediaID {
		ll := logrus.WithField("media", m)
		doc := new(db.MediaFileDoc)
		if err := mongo.DocGetById(ctx, m, doc, mongoCl); err != nil {
			ll.WithError(err).Error("error getting media from db")
			continue
		}
		var timeAt int
		if time < 1 {
			timeAt = int(doc.Duration * time)
		} else {
			timeAt = int(time)
		}
		data, err := ffmpeg.GenThumnail(ffContainer, fmt.Sprintf("%s/stream/%s", server, m), timeAt)
		if err != nil {
			ll.WithError(err).Error("can not generate thumbnail")
			continue
		}
		filename := uuid.NewString() + ".jpeg"
		if err := minio.FileAdd(filename, data, ctx); err != nil {
			ll.WithError(err).Error("can not add new thumbnail to minio")
			continue
		}
		// ...
		updateDoc := doc
		oldThumb := updateDoc.Thumbnail
		updateDoc.Thumbnail = filename
		_filter, _ := db.FilterById(updateDoc.ID)
		updateDoc.ID = ""
		if _, err := mongo.GetMediaMongo().IMng.GetCollection(mongoCl).ReplaceOne(ctx, _filter, updateDoc); err != nil {
			ll.WithError(err).Error("can not replace mongo record")
			continue
		}
		if oldThumb != "" {
			if err := minio.FileRm(oldThumb, ctx); err != nil {
				ll.WithError(err).Warn("can not remove old thumbnail")
			}
		}
		ll.Info("updated")
	}
}
