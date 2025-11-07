/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/amirdaaee/TGMon/internal/config"
	"github.com/amirdaaee/TGMon/internal/db"
	"github.com/amirdaaee/TGMon/internal/db/minio"
	"github.com/amirdaaee/TGMon/internal/db/mongo"
	"github.com/amirdaaee/TGMon/internal/facade"
	"github.com/amirdaaee/TGMon/internal/log"
	"github.com/amirdaaee/TGMon/internal/stream"
	"github.com/amirdaaee/TGMon/internal/tlg"
	"github.com/amirdaaee/TGMon/internal/types"
	realMinio "github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "TGMon",
	Short: "Telegram media manager",
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func buildDbContainer() (db.IDbContainer, error) {
	cfg := config.Config()
	ctx := context.TODO()
	mongoContainer, err := mongo.NewMongoContainer(ctx, mongo.MongoContainerConfig{Endpoint: cfg.MongoDBConfig.Uri, DbName: cfg.MongoDBConfig.DBName}, true)
	if err != nil {
		return nil, fmt.Errorf("can not create mongo container: %w", err)
	}
	minioContainer, err := minio.NewMinioContainer(ctx, minio.MinioContainerConfig{
		Endpoint: cfg.MinioConfig.Endpoint,
		Opts: &realMinio.Options{
			Creds:  credentials.NewStaticV4(cfg.MinioConfig.AccessKey, cfg.MinioConfig.SecretKey, ""),
			Secure: cfg.MinioConfig.Secure,
		},
		Bucket: cfg.MinioConfig.Bucket,
	}, true)
	if err != nil {
		return nil, fmt.Errorf("can not create minio container: %w", err)
	}
	dbContainer := db.NewDbContainer(mongoContainer, minioContainer)
	return dbContainer, nil
}
func buildSessionConfig() *tlg.SessionConfig {
	cfg := config.Config()
	return &tlg.SessionConfig{
		SocksProxy: cfg.TelegramConfig.TGSocksProxy,
		SessionDir: cfg.TelegramConfig.SessionDir,
		AppID:      cfg.TelegramConfig.AppID,
		AppHash:    cfg.TelegramConfig.AppHash,
	}
}
func buildTgClient() (tlg.IClient, error) {
	cfg := config.Config()
	tgClient := tlg.NewTgClient(buildSessionConfig(), cfg.TelegramConfig.BotToken)
	return tgClient, nil
}
func buildWorkerPool() (stream.IWorkerPool, error) {
	cfg := config.Config()
	wp, err := stream.NewWorkerPool(cfg.TelegramConfig.WorkerTokens, buildSessionConfig(), cfg.TelegramConfig.ChannelID, cfg.TelegramConfig.WorkerCacheRoot)
	if err != nil {
		return nil, fmt.Errorf("can not create worker pool: %w", err)
	}
	return wp, nil
}
func buildMediaFacade(dbContainer db.IDbContainer, workerContainer stream.IWorkerPool) facade.IFacade[types.MediaFileDoc] {
	cfg := config.Config()
	return facade.NewFacade(facade.NewMediaCrud(dbContainer, workerContainer, cfg.RuntimeConfig.KeepDupFiles))
}
func buildJobReqFacade(dbContainer db.IDbContainer) facade.IFacade[types.JobReqDoc] {
	return facade.NewFacade(facade.NewJobReqCrud(dbContainer))
}
func buildJobResFacade(dbContainer db.IDbContainer) facade.IFacade[types.JobResDoc] {
	return facade.NewFacade(facade.NewJobResCrud(dbContainer))
}
func setupLogger() {
	cfg := config.Config()
	log.Setup(cfg.RuntimeConfig.LogLevel)
}
