package app

import (
	"context"
	"fmt"
	"net/http"

	"github.com/amirdaaee/TGMon/internal/bot"
	"github.com/amirdaaee/TGMon/internal/config"
	"github.com/amirdaaee/TGMon/internal/db"
	"github.com/amirdaaee/TGMon/internal/db/minio"
	"github.com/amirdaaee/TGMon/internal/db/mongo"
	"github.com/amirdaaee/TGMon/internal/facade"
	"github.com/amirdaaee/TGMon/internal/filesystem"
	"github.com/amirdaaee/TGMon/internal/stash"
	"github.com/amirdaaee/TGMon/internal/stream"
	"github.com/amirdaaee/TGMon/internal/tlg"
	"github.com/amirdaaee/TGMon/internal/types"
	tgmonTypes "github.com/amirdaaee/TGMon/internal/types"
	"github.com/amirdaaee/TGMon/internal/web"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/hanwen/go-fuse/v2/fuse"
	"github.com/mazrean/kessoku"
	realMinio "github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

func NewWebServer(cfg *config.ConfigType, g *gin.Engine, hndl *web.HandlerContainer) *http.Server {
	hCfg := cfg.HttpConfig
	web.RegisterRoutes(g, hndl, cfg.HttpConfig.ApiToken, cfg.HttpConfig.Swagger)
	return &http.Server{
		Addr:    hCfg.ListenAddr,
		Handler: g,
	}
}
func NewFuzeServer(cfg *config.ConfigType, mediaFacade facade.IFacade[tgmonTypes.MediaFileDoc], wp stream.IWorkerPool) (*fuse.Server, error) {
	fCfg := cfg.FuseConfig
	mountDir := fCfg.MediaDir
	opts := &filesystem.MountOptions{
		AllowOther: fCfg.AllowOther,
		Debug:      fCfg.Debug,
	}
	server, err := filesystem.MountWithOptions(mountDir, mediaFacade, wp, opts)
	if err != nil {
		return nil, fmt.Errorf("can not mount filesystem: %w", err)
	}
	return server, nil
}

// ... Web
func NewGinEngine(cfg *config.ConfigType, hndlr *web.HandlerContainer) *gin.Engine {
	hCfg := cfg.HttpConfig
	g := gin.Default()
	coresCfg := cors.DefaultConfig()
	if len(hCfg.CoresAllowed) > 0 {
		coresCfg.AllowOrigins = hCfg.CoresAllowed
	} else {
		coresCfg.AllowAllOrigins = true
	}
	coresCfg.AddAllowHeaders("Authorization")
	g.Use(cors.New(coresCfg))
	return g
}

func NewWebHandler(cfg *config.ConfigType, dbCnt db.IDbContainer, mediafacade facade.IFacade[tgmonTypes.MediaFileDoc], jobReqFacade facade.IFacade[types.JobReqDoc], jobResFacade facade.IFacade[tgmonTypes.JobResDoc], wp stream.IWorkerPool, stshCl *stash.StashQlClient) *web.HandlerContainer {
	hCfg := cfg.HttpConfig
	sCfg := config.Config().StashRedirectorConfig

	streamHandler := web.NewStreamHandler(dbCnt, mediafacade, wp)
	mediaHandler := web.MediaHandler{DBContainer: dbCnt}
	jobReqHandler := web.JobReqHandler{}
	jobResHandler := web.JobResHandler{}
	infoHandler := web.InfoApiHandler{
		MediaFacade: mediafacade,
	}
	loginHandler := web.LoginApiHandler{
		UserName: hCfg.UserName,
		UserPass: hCfg.UserPass,
		Token:    hCfg.ApiToken,
	}
	sessionHandler := web.SessionApiHandler{
		Token: hCfg.ApiToken,
	}
	randomMediaHandler := web.RandomMediaApiHandler{
		MediaFacade: mediafacade,
	}
	hndlrs := web.HandlerContainer{
		MediaHandler:       web.NewCRDApiHandler(&mediaHandler, mediafacade, "media"),
		JobReqHandler:      web.NewCRDApiHandler(&jobReqHandler, jobReqFacade, "jobReq"),
		JobResHandler:      web.NewCRDApiHandler(&jobResHandler, jobResFacade, "jobRes"),
		InfoHandler:        web.NewApiHandler(&infoHandler, "info"),
		LoginHandler:       web.NewApiHandler(&loginHandler, "auth/login"),
		SessionHandler:     web.NewApiHandler(&sessionHandler, "auth/session"),
		RandomMediaHandler: web.NewApiHandler(&randomMediaHandler, "media/random"),
		StreamHandler:      streamHandler,
	}
	if sCfg.Enabled {
		stachCl := stash.NewStashQlClient(sCfg.StashEndpoint, sCfg.StashApiKey)
		stashVTTRedirectorHandler := web.StashVTTRedirectorApiHandler{
			MinioUrl:    sCfg.MinioUrl,
			StashCl:     stachCl,
			MediaFacade: mediafacade,
		}
		stashCoverRedirectorHandler := web.StashCoverRedirectorApiHandler{
			StashVTTRedirectorApiHandler: stashVTTRedirectorHandler,
		}
		hndlrs.StashVTTRedirectorHandler = web.NewApiHandler(&stashVTTRedirectorHandler, "")
		hndlrs.StashCoverRedirectorHandler = web.NewApiHandler(&stashCoverRedirectorHandler, "")
	}
	return &hndlrs
}

// ... Stash
func NewStashQlClient(cfg *config.ConfigType) *stash.StashQlClient {
	return stash.NewStashQlClient(cfg.StashRedirectorConfig.StashEndpoint, cfg.StashRedirectorConfig.StashApiKey)
}

// ... Database
func NewDbContainer(cfg *config.ConfigType) (db.IDbContainer, error) {
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

// ... Facades
func NewMediaFacade(dbContainer db.IDbContainer, workerContainer stream.IWorkerPool) facade.IFacade[tgmonTypes.MediaFileDoc] {
	cfg := config.Config()
	return facade.NewFacade(facade.NewMediaCrud(dbContainer, workerContainer, cfg.RuntimeConfig.KeepDupFiles))
}

func NewJobReqFacade(dbContainer db.IDbContainer) facade.IFacade[tgmonTypes.JobReqDoc] {
	return facade.NewFacade(facade.NewJobReqCrud(dbContainer))
}

func NewJobResFacade(dbContainer db.IDbContainer) facade.IFacade[tgmonTypes.JobResDoc] {
	return facade.NewFacade(facade.NewJobResCrud(dbContainer))
}

var FacadeProviderSet = kessoku.Set(
	kessoku.Provide(NewMediaFacade),
	kessoku.Provide(NewJobReqFacade),
	kessoku.Provide(NewJobResFacade),
)

// ... Telegram
func NewTgSessionConfig(cfg *config.ConfigType) *tlg.SessionConfig {
	return &tlg.SessionConfig{
		SocksProxy: cfg.TelegramConfig.TGSocksProxy,
		SessionDir: cfg.TelegramConfig.SessionDir,
		AppID:      cfg.TelegramConfig.AppID,
		AppHash:    cfg.TelegramConfig.AppHash,
	}
}

func NewTgClient(cfg *config.ConfigType, sessCfg *tlg.SessionConfig) tlg.IClient {
	tgClient := tlg.NewTgClient(sessCfg, cfg.TelegramConfig.BotToken)
	return tgClient
}
func NewTgWorkerPool(cfg *config.ConfigType, sessCfg *tlg.SessionConfig) (stream.IWorkerPool, error) {
	wp, err := stream.NewWorkerPool(cfg.TelegramConfig.WorkerTokens, sessCfg, cfg.TelegramConfig.ChannelID, cfg.TelegramConfig.WorkerCacheRoot)
	if err != nil {
		return nil, fmt.Errorf("can not create worker pool: %w", err)
	}
	return wp, nil
}

var TgProviderSet = kessoku.Set(
	kessoku.Provide(NewTgSessionConfig),
	kessoku.Provide(NewTgClient),
	kessoku.Provide(NewTgWorkerPool),
)

// ... Bot
func NewBot(tgClient tlg.IClient, mediafacade facade.IFacade[tgmonTypes.MediaFileDoc], wp stream.IWorkerPool) (*bot.Bot, error) {
	tgBot, err := bot.NewBot(tgClient)
	if err != nil {
		return nil, fmt.Errorf("can not create bot: %w", err)
	}
	hndler, err := bot.NewHandler(mediafacade, config.Config().TelegramConfig.ChannelID, wp)
	if err != nil {
		return nil, fmt.Errorf("can not create bot handler: %w", err)
	}
	hndler.Register(tgBot)
	return tgBot, nil
}
