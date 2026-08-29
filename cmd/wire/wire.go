package wire

import (
	"github.com/amirdaaee/TGMon/internal/log"
	"go.uber.org/dig"
	"go.uber.org/zap"
)

func mustProvide(c *dig.Container, name string, ctor any) {
	if err := c.Provide(ctor); err != nil {
		log.GetLogger(log.CmdModule).With(zap.Error(err), zap.String("provider", name)).Fatal("wire: register provider")
	}
}
func GetProvider() *dig.Container {
	c := dig.New()
	mustProvide(c, "Config", ProvideConfig)
	mustProvide(c, "FuseSrcs", ProvideFuseSrcs)
	mustProvide(c, "WebServer", ProvideWebServer)
	mustProvide(c, "GinEngine", ProvideGinEngine)
	mustProvide(c, "WebHandler", ProvideWebHandler)
	mustProvide(c, "StashQlClient", ProvideStashQlClient)
	mustProvide(c, "MongoClient", ProvideMongoClient)
	mustProvide(c, "ObjectStore", ProvideObjectStore)
	mustProvide(c, "MediaFileRepo", ProvideMediaFileRepo)
	mustProvide(c, "JobReqRepo", ProvideJobReqRepo)
	mustProvide(c, "JobResRepo", ProvideJobResRepo)
	mustProvide(c, "FuseStateRepo", ProvideFuseStateRepo)
	mustProvide(c, "MediaFacade", ProvideMediaFacade)
	mustProvide(c, "JobReqFacade", ProvideJobReqFacade)
	mustProvide(c, "JobResFacade", ProvideJobResFacade)
	mustProvide(c, "TgSessionConfig", ProvideTgSessionConfig)
	mustProvide(c, "TgClient", ProvideTgClient)
	mustProvide(c, "TgWorkerPool", ProvideTgWorkerPool)
	mustProvide(c, "FuzeCache", ProvideFuzeCache)
	mustProvide(c, "Bot", ProvideBot)
	return c
}
