package crd

import (
	"github.com/chenmingyong0423/go-mongox/v2/finder"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type ICreateHandler[T any] interface {
	BindCreateRequest(g *gin.Context) (*T, error)
	MarshalCreateResponse(g *gin.Context, v *T) (any, error)
}
type IReadHandler[T any] interface {
	BindReadRequest(g *gin.Context) (bson.D, error)
	MarshalReadResponse(g *gin.Context, v *T) (any, error)
}
type IListHandler[T any] interface {
	BindListRequest(g *gin.Context, fnd finder.IFinder[T]) (finder.IFinder[T], error)
	MarshalListResponse(g *gin.Context, v []*T) (any, error)
}
type IDeleteHandler[T any] interface {
	BindDeleteRequest(g *gin.Context) (bson.D, error)
}
