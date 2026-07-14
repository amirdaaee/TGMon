package wire

import (
	"github.com/amirdaaee/TGMon/internal/log"
	"go.uber.org/dig"
)

func mustProvide(c *dig.Container, name string, ctor any) {
	if err := c.Provide(ctor); err != nil {
		log.GetLogger(log.CmdModule).WithError(err).WithField("provider", name).Fatal("wire: register provider")
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
	mustProvide(c, "DbContainer", ProvideDbContainer)
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
