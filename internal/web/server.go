package web

import (
	"github.com/amirdaaee/TGMon/config"
	"github.com/gin-gonic/gin"
)

func Start() {
	r := gin.Default()
	setupRoutes(r)
	r.Run(config.Config().ListenURL)
}
