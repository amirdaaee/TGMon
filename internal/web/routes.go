package web

import (
	"github.com/amirdaaee/TGMon/internal/types"
	"github.com/gin-gonic/gin"
)

type HandlerContainer struct {
	MediaHandler  *CRDApiHandler[types.MediaFileDoc]
	JobReqHandler *CRDApiHandler[types.JobReqDoc]
	JobResHandler *CRDApiHandler[types.JobResDoc]
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
