package main

import (
	"context"

	"github.com/amirdaaee/TGMon/cmd"
	"github.com/amirdaaee/TGMon/internal/helper"
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
	mongo := cmd.GetMongoDB().GetMediaMongo()
	minio, err := cmd.GetMinioDB()
	if err != nil {
		logrus.WithError(err).Fatal("can not create minio client")
	}
	// ...
	for {
		doc := <-notifer.DocNotifier.Chan
		ctx := context.Background()
		go func() {
			if err := helper.AddMedia(ctx, mongo, minio, doc, wp); err != nil {
				logrus.WithError(err).Error("error adding media")
			}
		}()
	}
}
