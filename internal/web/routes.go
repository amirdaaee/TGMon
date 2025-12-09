package web

import (
	docs "github.com/amirdaaee/TGMon/docs"
	"github.com/amirdaaee/TGMon/internal/log"
	wtypes "github.com/amirdaaee/TGMon/internal/web/types"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func RegisterRoutes(r *gin.Engine, hndlrs []wtypes.Registereable, apiToken string, swag bool) {
	ll := log.GetLogger(log.WebModule)
	authMiddleware := apiAuthMiddleware(apiToken)
	webRoot := r.Group("/", errMiddleware())
	apiRoot := webRoot.Group("api/")

	if swag {
		docs.SwaggerInfo.Title = "Tgmon API"
		r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}
	for _, hndlr := range hndlrs {
		r := apiRoot
		if hndlr.RegisterToRoot() {
			r = webRoot
		}
		if err := hndlr.RegisterRoutes(r, authMiddleware); err != nil {
			ll.WithError(err).Errorf("error registering routes for %+T", hndlr)
			continue
		}
	}
}
