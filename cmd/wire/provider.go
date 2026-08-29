package wire

import (
	"context"
	"fmt"
	"net/http"

	"github.com/amirdaaee/TGMon/internal/bot"
	"github.com/amirdaaee/TGMon/internal/config"
	"github.com/amirdaaee/TGMon/internal/facade"
	"github.com/amirdaaee/TGMon/internal/facade/crd"
	ftypes "github.com/amirdaaee/TGMon/internal/facade/types"
	"github.com/amirdaaee/TGMon/internal/filesystem/cache"
	fsSrc "github.com/amirdaaee/TGMon/internal/filesystem/src"
	"github.com/amirdaaee/TGMon/internal/repository"
	repominio "github.com/amirdaaee/TGMon/internal/repository/minio"
	repomongo "github.com/amirdaaee/TGMon/internal/repository/mongo"
	"github.com/amirdaaee/TGMon/internal/stash"
	"github.com/amirdaaee/TGMon/internal/stream"
	"github.com/amirdaaee/TGMon/internal/tlg"
	"github.com/amirdaaee/TGMon/internal/types"
	"github.com/amirdaaee/TGMon/internal/web"
	wApi "github.com/amirdaaee/TGMon/internal/web/api"
	waHndlr "github.com/amirdaaee/TGMon/internal/web/api/handler"
	wRest "github.com/amirdaaee/TGMon/internal/web/rest"
	wrCrd "github.com/amirdaaee/TGMon/internal/web/rest/crd"
	wStream "github.com/amirdaaee/TGMon/internal/web/stream"
	wtypes "github.com/amirdaaee/TGMon/internal/web/types"
	"github.com/amirdaaee/TGMon/internal/worker"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	realMinio "github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

func ProvideConfig() *config.ConfigType {
	return config.Config()
}
func ProvideFuzeCache(media repository.MediaFileRepository) *cache.DBCache[string, *types.MediaFileDoc] {
	return cache.NewDBCacher(func(ctx context.Context) (map[string]*types.MediaFileDoc, error) {
		docs, err := media.ListByID(ctx)
		if err != nil {
			return nil, err
		}
		docsMap := make(map[string]*types.MediaFileDoc)
		for _, doc := range docs {
			docsMap[doc.ID.Hex()] = doc
		}
		return docsMap, nil
	})
}
func ProvideFuseSrcs(cfg *config.ConfigType, mediafacade ftypes.IFacade[types.MediaFileDoc], wp worker.IWorkerPool, media repository.MediaFileRepository, objects repository.ObjectStore, fsCache *cache.DBCache[string, *types.MediaFileDoc]) []fsSrc.ISrc {
	return []fsSrc.ISrc{fsSrc.NewMediaFileSrc(mediafacade, wp, fsCache, &stream.StreamConfig{
		StreamBufferCount: cfg.StreamConfig.StreamBufferCount,
		StreamConcurrency: cfg.StreamConfig.StreamConcurrency,
		StreamMaxRetries:  cfg.StreamConfig.StreamMaxRetries,
		StreamTimeoutSec:  cfg.StreamConfig.StreamTimeoutSec,
	}), fsSrc.NewSrtSrc(media, objects, fsCache)}
}

// ... Web
func ProvideWebServer(cfg *config.ConfigType, g *gin.Engine, hndl []wtypes.Registereable) *http.Server {
	hCfg := cfg.HttpConfig
	web.RegisterRoutes(g, hndl, cfg.HttpConfig.ApiToken, cfg.HttpConfig.Swagger)
	return &http.Server{
		Addr:    hCfg.ListenAddr,
		Handler: g,
	}
}
func ProvideGinEngine(cfg *config.ConfigType, hndlr []wtypes.Registereable) *gin.Engine {
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

func ProvideWebHandler(cfg *config.ConfigType, mediafacade ftypes.IFacade[types.MediaFileDoc], jobReqFacade ftypes.IFacade[types.JobReqDoc], jobResFacade ftypes.IFacade[types.JobResDoc], media repository.MediaFileRepository, jobReqs repository.JobReqRepository, wp worker.IWorkerPool, stshCl *stash.StashQlClient) []wtypes.Registereable {
	hCfg := cfg.HttpConfig
	stshCfg := cfg.StashRedirectorConfig
	strmConfig := cfg.StreamConfig
	streamHandler := wStream.NewStreamHandler(mediafacade, wp, &stream.StreamConfig{
		StreamBufferCount: strmConfig.StreamBufferCount,
		StreamConcurrency: strmConfig.StreamConcurrency,
		StreamMaxRetries:  strmConfig.StreamMaxRetries,
		StreamTimeoutSec:  strmConfig.StreamTimeoutSec,
	})
	mediaHandler := wrCrd.NewMediaHandler(media)
	jobReqHandler := wrCrd.NewJobReqHandler(jobReqs)
	jobResHandler := &wrCrd.JobResHandler{}
	infoHandler := &waHndlr.MediaInfoApiHandler{
		Media: media,
	}
	loginHandler := &waHndlr.LoginApiHandler{
		UserName: hCfg.UserName,
		UserPass: hCfg.UserPass,
		Token:    hCfg.ApiToken,
	}
	sessionHandler := &waHndlr.SessionApiHandler{
		Token: hCfg.ApiToken,
	}
	randomMediaHandler := waHndlr.NewRandomMediaApiHandler(media)
	result := []wtypes.Registereable{
		streamHandler,
		wRest.NewCRDApiHandler(mediaHandler, mediafacade, "media"),
		wRest.NewCRDApiHandler(jobReqHandler, jobReqFacade, "jobReq"),
		wRest.NewCRDApiHandler(jobResHandler, jobResFacade, "jobRes"),
		wApi.NewApiHandler(infoHandler, "info"),
		wApi.NewApiHandler(loginHandler, "auth/login"),
		wApi.NewApiHandler(sessionHandler, "auth/session"),
		wApi.NewApiHandler(randomMediaHandler, "media/random"),
	}
	if stshCfg.Enabled {
		stashVTTRedirectorHandler := waHndlr.StashVTTRedirectorApiHandler{
			MinioUrl: stshCfg.MinioUrl,
			StashCl:  stshCl,
			Media:    media,
		}
		stashCoverRedirectorHandler := waHndlr.StashCoverRedirectorApiHandler{
			StashVTTRedirectorApiHandler: stashVTTRedirectorHandler,
		}
		result = append(result, wApi.NewApiHandler(&stashVTTRedirectorHandler, ""))
		result = append(result, wApi.NewApiHandler(&stashCoverRedirectorHandler, ""))
	}
	return result
}

// ... Stash
func ProvideStashQlClient(cfg *config.ConfigType) *stash.StashQlClient {
	return stash.NewStashQlClient(cfg.StashRedirectorConfig.StashEndpoint, cfg.StashRedirectorConfig.StashApiKey)
}

// ... Database
func ProvideMongoClient(cfg *config.ConfigType) (*repomongo.Client, error) {
	cl, err := repomongo.Connect(context.TODO(), repomongo.Config{Endpoint: cfg.MongoDBConfig.Uri, DBName: cfg.MongoDBConfig.DBName}, true)
	if err != nil {
		return nil, fmt.Errorf("can not connect to mongo: %w", err)
	}
	return cl, nil
}

func ProvideObjectStore(cfg *config.ConfigType) (repository.ObjectStore, error) {
	st, err := repominio.Connect(context.TODO(), repominio.Config{
		Endpoint: cfg.MinioConfig.Endpoint,
		Opts: &realMinio.Options{
			Creds:  credentials.NewStaticV4(cfg.MinioConfig.AccessKey, cfg.MinioConfig.SecretKey, ""),
			Secure: cfg.MinioConfig.Secure,
		},
		Bucket: cfg.MinioConfig.Bucket,
	}, true)
	if err != nil {
		return nil, fmt.Errorf("can not connect to minio: %w", err)
	}
	return st, nil
}

func ProvideMediaFileRepo(cl *repomongo.Client) repository.MediaFileRepository {
	return repomongo.NewMediaFileRepo(cl)
}

func ProvideJobReqRepo(cl *repomongo.Client) repository.JobReqRepository {
	return repomongo.NewJobReqRepo(cl)
}

func ProvideJobResRepo(cl *repomongo.Client) repository.JobResRepository {
	return repomongo.NewJobResRepo(cl)
}

func ProvideFuseStateRepo(cl *repomongo.Client) repository.FuseStateRepository {
	return repomongo.NewFuseStateRepo(cl)
}
func ProvideWorkerRepo(cl *repomongo.Client) repository.WorkerMediaRepository {
	return repomongo.NewWorkerRepo(cl)
}

// ... Facades
func ProvideMediaFacade(media repository.MediaFileRepository, objects repository.ObjectStore, jobReqs repository.JobReqRepository, workerContainer worker.IWorkerPool, jobReqFacade ftypes.IFacade[types.JobReqDoc], fsCache *cache.DBCache[string, *types.MediaFileDoc]) ftypes.IFacade[types.MediaFileDoc] {
	cfg := config.Config()
	return facade.NewFacade(media, crd.NewMediaCrud(media, objects, jobReqs, workerContainer, cfg.RuntimeConfig.KeepDupFiles, jobReqFacade, fsCache))
}

func ProvideJobReqFacade(jobReqs repository.JobReqRepository) ftypes.IFacade[types.JobReqDoc] {
	return facade.NewFacade(jobReqs, crd.NewJobReqCrud())
}

func ProvideJobResFacade(jobRes repository.JobResRepository, jobReqs repository.JobReqRepository, media repository.MediaFileRepository, objects repository.ObjectStore, jobReqFacade ftypes.IFacade[types.JobReqDoc], fsCache *cache.DBCache[string, *types.MediaFileDoc]) ftypes.IFacade[types.JobResDoc] {
	return facade.NewFacade(jobRes, crd.NewJobResCrud(jobReqs, media, objects, jobReqFacade, fsCache))
}

// ... Telegram
func ProvideTgSessionConfig(cfg *config.ConfigType) *tlg.SessionConfig {
	return &tlg.SessionConfig{
		SocksProxy: cfg.TelegramConfig.TGSocksProxy,
		SessionDir: cfg.TelegramConfig.SessionDir,
		AppID:      cfg.TelegramConfig.AppID,
		AppHash:    cfg.TelegramConfig.AppHash,
	}
}

func ProvideTgClient(cfg *config.ConfigType, sessCfg *tlg.SessionConfig) tlg.IClient {
	tgClient := tlg.NewTgClient(sessCfg, cfg.TelegramConfig.BotToken)
	return tgClient
}
func ProvideTgWorkerPool(cfg *config.ConfigType, sessCfg *tlg.SessionConfig, workerRepo repository.WorkerMediaRepository) (worker.IWorkerPool, error) {
	wp, err := worker.NewWorkerPool(cfg.TelegramConfig.WorkerTokens, sessCfg, cfg.TelegramConfig.ChannelID, workerRepo, &stream.StreamConfig{
		StreamBufferCount: cfg.StreamConfig.StreamBufferCount,
		StreamConcurrency: cfg.StreamConfig.StreamConcurrency,
		StreamMaxRetries:  cfg.StreamConfig.StreamMaxRetries,
		StreamTimeoutSec:  cfg.StreamConfig.StreamTimeoutSec,
	})
	if err != nil {
		return nil, fmt.Errorf("can not create worker pool: %w", err)
	}
	return wp, nil
}

// ... Bot
func ProvideBot(tgClient tlg.IClient, mediafacade ftypes.IFacade[types.MediaFileDoc], wp worker.IWorkerPool) (*bot.Bot, error) {
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
