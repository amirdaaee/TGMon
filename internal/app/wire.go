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
	"github.com/amirdaaee/TGMon/internal/facade/crd"
	ftypes "github.com/amirdaaee/TGMon/internal/facade/types"
	"github.com/amirdaaee/TGMon/internal/filesystem"
	fsSrc "github.com/amirdaaee/TGMon/internal/filesystem/src"
	"github.com/amirdaaee/TGMon/internal/stash"
	"github.com/amirdaaee/TGMon/internal/stream"
	"github.com/amirdaaee/TGMon/internal/tlg"
	tgmonTypes "github.com/amirdaaee/TGMon/internal/types"
	"github.com/amirdaaee/TGMon/internal/web"
	wApi "github.com/amirdaaee/TGMon/internal/web/api"
	waHndlr "github.com/amirdaaee/TGMon/internal/web/api/handler"
	wRest "github.com/amirdaaee/TGMon/internal/web/rest"
	wrCrd "github.com/amirdaaee/TGMon/internal/web/rest/crd"
	wStream "github.com/amirdaaee/TGMon/internal/web/stream"
	wtypes "github.com/amirdaaee/TGMon/internal/web/types"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/google/wire"
	"github.com/hanwen/go-fuse/v2/fuse"
	realMinio "github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

func NewFuseSrcs(mediafacade ftypes.IFacade[tgmonTypes.MediaFileDoc], wp stream.IWorkerPool, dbC db.IDbContainer) []fsSrc.ISrc {
	return []fsSrc.ISrc{fsSrc.NewMediaFileSrc(mediafacade, wp), fsSrc.NewSrtSrc(mediafacade, dbC.GetMinioContainer().GetMinioClient())}
}
func NewFuzeServer(ctx context.Context, cfg *config.ConfigType, srcs []fsSrc.ISrc, dbContainer db.IDbContainer) (*fuse.Server, error) {
	fCfg := cfg.FuseConfig
	mountDir := fCfg.MediaDir
	opts := &filesystem.MountOptions{
		AllowOther: fCfg.AllowOther,
		Debug:      fCfg.Debug,
	}
	server, err := filesystem.MountWithOptions(ctx, mountDir, srcs, dbContainer, opts)
	if err != nil {
		return nil, fmt.Errorf("can not mount filesystem: %w", err)
	}
	return server, nil
}

var FuseProviderSet = wire.NewSet(
	NewFuseSrcs,
	NewFuzeServer,
)

// ... Web
func NewWebServer(cfg *config.ConfigType, g *gin.Engine, hndl []wtypes.Registereable) *http.Server {
	hCfg := cfg.HttpConfig
	web.RegisterRoutes(g, hndl, cfg.HttpConfig.ApiToken, cfg.HttpConfig.Swagger)
	return &http.Server{
		Addr:    hCfg.ListenAddr,
		Handler: g,
	}
}
func NewGinEngine(cfg *config.ConfigType, hndlr []wtypes.Registereable) *gin.Engine {
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

func NewWebHandler(cfg *config.ConfigType, dbCnt db.IDbContainer, mediafacade ftypes.IFacade[tgmonTypes.MediaFileDoc], jobReqFacade ftypes.IFacade[tgmonTypes.JobReqDoc], jobResFacade ftypes.IFacade[tgmonTypes.JobResDoc], wp stream.IWorkerPool, stshCl *stash.StashQlClient) []wtypes.Registereable {
	hCfg := cfg.HttpConfig
	sCfg := config.Config().StashRedirectorConfig

	streamHandler := wStream.NewStreamHandler(dbCnt, mediafacade, wp)
	mediaHandler := wrCrd.MediaHandler{DBContainer: dbCnt}
	jobReqHandler := wrCrd.JobReqHandler{}
	jobResHandler := wrCrd.JobResHandler{}
	infoHandler := waHndlr.MediaInfoApiHandler{
		MediaFacade: mediafacade,
	}
	loginHandler := waHndlr.LoginApiHandler{
		UserName: hCfg.UserName,
		UserPass: hCfg.UserPass,
		Token:    hCfg.ApiToken,
	}
	sessionHandler := waHndlr.SessionApiHandler{
		Token: hCfg.ApiToken,
	}
	randomMediaHandler := waHndlr.RandomMediaApiHandler{
		MediaFacade: mediafacade,
	}
	result := []wtypes.Registereable{
		streamHandler,
		wRest.NewCRDApiHandler(&mediaHandler, mediafacade, "media"),
		wRest.NewCRDApiHandler(&jobReqHandler, jobReqFacade, "jobReq"),
		wRest.NewCRDApiHandler(&jobResHandler, jobResFacade, "jobRes"),
		wApi.NewApiHandler(&infoHandler, "info"),
		wApi.NewApiHandler(&loginHandler, "auth/login"),
		wApi.NewApiHandler(&sessionHandler, "auth/session"),
		wApi.NewApiHandler(&randomMediaHandler, "media/random"),
	}
	if sCfg.Enabled {
		stachCl := stash.NewStashQlClient(sCfg.StashEndpoint, sCfg.StashApiKey)
		stashVTTRedirectorHandler := waHndlr.StashVTTRedirectorApiHandler{
			MinioUrl:    sCfg.MinioUrl,
			StashCl:     stachCl,
			MediaFacade: mediafacade,
		}
		stashCoverRedirectorHandler := waHndlr.StashCoverRedirectorApiHandler{
			StashVTTRedirectorApiHandler: stashVTTRedirectorHandler,
		}
		result = append(result, wApi.NewApiHandler(&stashVTTRedirectorHandler, ""))
		result = append(result, wApi.NewApiHandler(&stashCoverRedirectorHandler, ""))
	}
	return result
}

var WebHandlerProviderSet = wire.NewSet(
	NewGinEngine, NewWebHandler, NewWebServer,
)

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
func NewMediaFacade(dbContainer db.IDbContainer, workerContainer stream.IWorkerPool, jobReqFacade ftypes.IFacade[tgmonTypes.JobReqDoc]) ftypes.IFacade[tgmonTypes.MediaFileDoc] {
	cfg := config.Config()
	return facade.NewFacade(crd.NewMediaCrud(dbContainer, workerContainer, cfg.RuntimeConfig.KeepDupFiles, jobReqFacade))
}

func NewJobReqFacade(dbContainer db.IDbContainer) ftypes.IFacade[tgmonTypes.JobReqDoc] {
	return facade.NewFacade(crd.NewJobReqCrud(dbContainer))
}

func NewJobResFacade(dbContainer db.IDbContainer, jobReqFacade ftypes.IFacade[tgmonTypes.JobReqDoc]) ftypes.IFacade[tgmonTypes.JobResDoc] {
	return facade.NewFacade(crd.NewJobResCrud(dbContainer, jobReqFacade))
}

var FacadeProviderSet = wire.NewSet(
	NewMediaFacade,
	NewJobReqFacade,
	NewJobResFacade,
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

var TgProviderSet = wire.NewSet(
	NewTgSessionConfig,
	NewTgClient,
	NewTgWorkerPool,
)

// ... Bot
func NewBot(tgClient tlg.IClient, mediafacade ftypes.IFacade[tgmonTypes.MediaFileDoc], wp stream.IWorkerPool) (*bot.Bot, error) {
	tgBot, err := bot.NewBot(tgClient)
	if err != nil {
		return nil, fmt.Errorf("can not create bot: %w", err)
	}
	hndler, err := bot.NewMediaHandler(mediafacade, config.Config().TelegramConfig.ChannelID, wp)
	if err != nil {
		return nil, fmt.Errorf("can not create bot handler: %w", err)
	}
	hndler.Register(tgBot)
	return tgBot, nil
}
