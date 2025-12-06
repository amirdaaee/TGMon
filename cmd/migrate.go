/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"

	"github.com/amirdaaee/TGMon/internal/app"
	"github.com/amirdaaee/TGMon/internal/config"
	"github.com/amirdaaee/TGMon/internal/types"
	"github.com/chenmingyong0423/go-mongox/v2/builder/query"
	"github.com/chenmingyong0423/go-mongox/v2/builder/update"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// migrateCmd represents the migrate command
var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Run migration on DB",
	Run: func(cmd *cobra.Command, args []string) {
		cfg := config.Config()
		dbContainer, err := app.NewDbContainer(cfg)
		if err != nil {
			logrus.WithError(err).Fatal("can not initialize db container")
		}
		facade := app.NewMediaFacade(dbContainer, nil)
		ctx := context.TODO()
		// ... fill uname
		ll := logrus.WithField("step", "fill uname")
		filter := query.NewBuilder().Or(query.Eq(types.MediaFileDoc__UnameField, ""), query.Exists(types.MediaFileDoc__UnameField, false)).Build()
		docs, err := facade.GetCollection().Finder().Filter(filter).Find(ctx)
		if err != nil {
			ll.WithError(err).Fatal("can not find media files")
		}
		ll.Infof("media files found: %d", len(docs))
		collection := facade.GetCollection()
		for _, doc := range docs {
			if _, err := collection.Updater().Filter(query.Id(doc.ID)).Updates(update.Set(types.MediaFileDoc__UnameField, doc.Name())).UpdateOne(ctx); err != nil {
				ll.WithError(err).Error("can not update media file")
			}
			ll.Infof("updated media file: %s", doc.ID.Hex())
		}
	},
}

func init() {
	rootCmd.AddCommand(migrateCmd)
}
