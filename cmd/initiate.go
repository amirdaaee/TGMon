package cmd

import (
	"github.com/amirdaaee/TGMon/config"
	"github.com/amirdaaee/TGMon/internal/bot"
	"github.com/amirdaaee/TGMon/internal/db"
	"github.com/sirupsen/logrus"
)

func init() {
	llStr := config.Config().LogLevel
	ll, err := logrus.ParseLevel(llStr)
	if err != nil {
		logrus.WithError(err).Error("can not parse log level. default to warning")
		logrus.SetLevel(logrus.WarnLevel)
	} else {
		logrus.SetLevel(ll)
	}
}
func GetWorkerPool() (*bot.WorkerPool, error) {
	cfg := config.Config()
	botCfg := bot.SessionConfig{
		SocksProxy: cfg.TGSocksProxy,
		SessionDir: cfg.SessionDir,
		AppID:      cfg.AppID,
		AppHash:    cfg.AppHash,
		ChannelId:  cfg.ChannelID,
	}
	return bot.NewWorkerPool(cfg.WorkerTokens, &botCfg)
}
func GetMasterBot() (*bot.Notifier, error) {
	cfg := config.Config()
	botCfg := bot.SessionConfig{
		SocksProxy: cfg.TGSocksProxy,
		SessionDir: cfg.SessionDir,
		AppID:      cfg.AppID,
		AppHash:    cfg.AppHash,
		ChannelId:  cfg.ChannelID,
	}
	return bot.NewMaster(cfg.BotToken, &botCfg)
}
func GetMongoDB() *db.Mongo {
	cfg := config.Config()

	mongo := db.Mongo{
		DBUri:               cfg.MongoDBUri,
		DBName:              cfg.MongoDBName,
		MediaCollectionName: "files",
		JobCollectionName:   "jobs",
	}
	return &mongo
}
func GetMinioDB() (*db.MinioClient, error) {
	cfg := config.Config()
	MinioConfig := db.MinioConfig{
		MinioEndpoint:  cfg.MinioEndpoint,
		MinioAccessKey: cfg.MinioAccessKey,
		MinioSecretKey: cfg.MinioSecretKey,
		MinioBucket:    cfg.MinioBucket,
		MinioSecure:    cfg.MinioSecure,
	}
	return db.NewMinioClient(&MinioConfig)
}
