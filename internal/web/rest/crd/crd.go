package crd

import (
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type ICreateHandler[T any] interface {
	BindCreateRequest(g *gin.Context) (*T, error)
	MarshalCreateResponse(g *gin.Context, v *T) (any, error)
}
type IReadHandler[T any] interface {
	BindReadRequest(g *gin.Context) (bson.ObjectID, error)
	MarshalReadResponse(g *gin.Context, v *T) (any, error)
}
type IListHandler interface {
	HandleList(g *gin.Context) (any, error)
}
type IDeleteHandler interface {
	BindDeleteRequest(g *gin.Context) (bson.ObjectID, error)
}
