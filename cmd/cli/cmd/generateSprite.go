/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"

	ccmd "github.com/amirdaaee/TGMon/cmd"
	"github.com/amirdaaee/TGMon/internal/db"
	"github.com/amirdaaee/TGMon/internal/helper"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"go.mongodb.org/mongo-driver/bson"
)

// generateSprite represents the updateThumbnail command
var generateSprite = &cobra.Command{
	Use:   "generateSprite",
	Short: "",
	Long:  "",
	Run: func(cmd *cobra.Command, args []string) {
		genSprite()
	},
}

func init() {
	rootCmd.AddCommand(generateSprite)
}

func genSprite() {
	mongo := ccmd.GetMongoDB()
	medMongo := mongo.GetMediaMongo()
	ctx := context.TODO()
	mongoCl, err := medMongo.GetClient()
	if err != nil {
		logrus.WithError(err).Fatal("can not create mongo client")
	}
	defer mongoCl.Disconnect(ctx)
	mongoMedColl := medMongo.IMng.GetCollection(mongoCl)
	filter := bson.M{
		"$or": []bson.M{
			{"Vtt": bson.M{"$exists": false}},
			{"Vtt": ""},
			{"Sprite": bson.M{"$exists": false}},
			{"Sprite": ""},
		},
	}
	cursor, err := mongoMedColl.Find(ctx, filter)
	if err != nil {
		logrus.WithError(err).Fatal("can not get media docs")
	}
	var jobI []db.JobDoc
	for cursor.Next(context.TODO()) {
		var mediaDoc db.MediaFileDoc
		err := cursor.Decode(&mediaDoc)
		if err != nil {
			logrus.WithError(err).Error("can not decode doc")
			continue
		}
		j := db.JobDoc{MediaID: mediaDoc.ID, Type: db.SPRITEJobType}
		jobI = append(jobI, j)
		logrus.WithField("media", j.MediaID).Info("added")
	}
	if len(jobI) > 0 {
		if err := helper.AddJob(ctx, mongo, jobI); err != nil {
			logrus.WithError(err).Fatal("can not write jobs doc")
		} else {
			logrus.Info("finish")
		}
	} else {
		logrus.Info("nothing to add")
	}
}
