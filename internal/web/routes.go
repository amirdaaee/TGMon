package web

import (
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.Engine, streamHandler *Streamhandler) {
	r.Match([]string{"HEAD", "GET"}, "/stream/:mediaID", streamHandler.Stream)
}
