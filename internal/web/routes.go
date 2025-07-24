package web

import (
	docs "github.com/amirdaaee/TGMon/docs"
	"github.com/amirdaaee/TGMon/internal/types"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

type HandlerContainer struct {
	MediaHandler       *CRDApiHandler[types.MediaFileDoc]
	JobReqHandler      *CRDApiHandler[types.JobReqDoc]
	JobResHandler      *CRDApiHandler[types.JobResDoc]
	InfoHandler        *ApiHandler
	LoginHandler       *ApiHandler
	SessionHandler     *ApiHandler
	RandomMediaHandler *ApiHandler
}

func RegisterRoutes(r *gin.Engine, streamHandler *Streamhandler, hndlrs HandlerContainer, apiToken string, swag bool) {
	webRoot := r.Group("/", errMiddleware())
	if swag {
		docs.SwaggerInfo.Title = "Tgmon API"
		r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}
	webRoot.Match([]string{"HEAD", "GET"}, "/stream/:mediaID", streamHandler.Stream)
	authMiddleware := apiAuthMiddleware(apiToken)
	apiRoot := webRoot.Group("api/")
	hndlrs.MediaHandler.RegisterRoutes(apiRoot, authMiddleware)
	hndlrs.JobReqHandler.RegisterRoutes(apiRoot, authMiddleware)
	hndlrs.JobResHandler.RegisterRoutes(apiRoot, authMiddleware)
	hndlrs.InfoHandler.RegisterRoutes(apiRoot, authMiddleware)
	hndlrs.LoginHandler.RegisterRoutes(apiRoot, authMiddleware)
	hndlrs.SessionHandler.RegisterRoutes(apiRoot, authMiddleware)
	hndlrs.RandomMediaHandler.RegisterRoutes(apiRoot, authMiddleware)
}
