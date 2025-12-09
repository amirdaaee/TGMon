package types

import "github.com/gin-gonic/gin"

type RootRegisterer interface {
	RegisterToRoot() bool // RegisterToRoot returns true if the handler should be registered to the root group.
}
type Registereable interface {
	RootRegisterer
	RegisterRoutes(r *gin.RouterGroup, authMiddleware gin.HandlerFunc) error
}
