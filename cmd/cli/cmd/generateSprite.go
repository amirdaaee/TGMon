/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"

	ccmd "github.com/amirdaaee/TGMon/cmd"
	"github.com/amirdaaee/TGMon/internal/db"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"go.mongodb.org/mongo-driver/bson"
	mongoD "go.mongodb.org/mongo-driver/mongo"
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
	medMongo := ccmd.GetMongoDB().GetMediaMongo()
	jobMongo := ccmd.GetMongoDB().GetJobMongo()
	// ...
	ctx := context.TODO()
	mongoCl, err := medMongo.GetClient()
	if err != nil {
		logrus.WithError(err).Fatal("can not create mongo client")
	}
	defer mongoCl.Disconnect(ctx)
	mongoMedColl := medMongo.IMng.GetCollection(mongoCl)
	mongoJobColl := jobMongo.IMng.GetCollection(mongoCl)
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
	var jobI []interface{}
	for cursor.Next(context.TODO()) {
		var mediaDoc db.MediaFileDoc
		err := cursor.Decode(&mediaDoc)
		if err != nil {
			logrus.WithError(err).Error("can not decode doc")
			continue
		}
		j := db.JobDoc{MediaID: mediaDoc.ID, Type: db.SPRITEJobType}
		filter, err := bson.Marshal(j)
		if err != nil {
			logrus.WithError(err).Error("can not create job lookup filter")
			continue
		}
		jobRes := mongoJobColl.FindOne(ctx, filter)
		if jobRes.Err() == nil {
			continue
		} else if jobRes.Err() != mongoD.ErrNoDocuments {
			logrus.WithError(err).Error("rror lookup job record")
			continue
		}
		jobI = append(jobI, j)
		logrus.WithField("media", j.MediaID).Info("added")
	}
	if len(jobI) > 0 {
		if _, err := mongoJobColl.InsertMany(ctx, jobI); err != nil {
			logrus.WithError(err).Error("can not write jobs doc")
		} else {
			logrus.Info("finish")
		}
	} else {
		logrus.Info("nothing to add")
	}

}
