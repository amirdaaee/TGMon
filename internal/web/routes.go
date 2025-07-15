package web

import (
	"github.com/amirdaaee/TGMon/internal/types"
	"github.com/gin-gonic/gin"
)

type HandlerContainer struct {
	MediaHandler  *ApiHandler[types.MediaFileDoc]
	JobReqHandler *ApiHandler[types.JobReqDoc]
	JobResHandler *ApiHandler[types.JobResDoc]
}

func RegisterRoutes(r *gin.Engine, streamHandler *Streamhandler, hndlrs HandlerContainer, apiToken string) {
	webRoot := r.Group("/", errMiddleware())
	webRoot.Match([]string{"HEAD", "GET"}, "/stream/:mediaID", streamHandler.Stream)
	authMiddleware := apiAuthMiddleware(apiToken)
	apiRoot := webRoot.Group("api/")
	hndlrs.MediaHandler.RegisterRoutes(apiRoot, authMiddleware)
	hndlrs.JobReqHandler.RegisterRoutes(apiRoot, authMiddleware)
	hndlrs.JobResHandler.RegisterRoutes(apiRoot, authMiddleware)
}
