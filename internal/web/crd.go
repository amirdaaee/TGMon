package web

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/amirdaaee/TGMon/internal/facade"
	"github.com/gin-gonic/gin"
)

// Package api provides generic API handler logic for CRUD operations using Gin and MongoDB.

type ApiHandler[T any] struct {
	hndler IHandler[T]
	fac    facade.IFacade[T]
	name   string
}

// ApiHandler provides CRUD handlers and route registration for a resource type T.

func (a *ApiHandler[T]) HandleCreate(g *gin.Context) {
	// HandleCreate handles HTTP POST requests to create a new resource.
	doc, err := a.hndler.BindCreateRequest(g)
	if err != nil {
		g.Error(NewHttpError(err, http.StatusBadRequest)) //nolint:golint,errcheck
		return
	}
	res, err := a.fac.CreateOne(g.Request.Context(), doc)
	if err != nil {
		g.Error(NewHttpError(err, http.StatusInternalServerError)) //nolint:golint,errcheck
		return
	}
	g.JSON(http.StatusOK, a.hndler.MarshalCreateResponse(res))
}
func (a *ApiHandler[T]) HandleRead(g *gin.Context) {
	// HandleRead handles HTTP GET requests to read resources.
	req, err := a.hndler.BindListRequest(g, a.fac.GetCollection().Finder())
	if err != nil {
		g.Error(NewHttpError(err, http.StatusBadRequest)) //nolint:golint,errcheck
		return
	}
	res, err := req.Find(g.Request.Context())
	if err != nil {
		g.Error(NewHttpError(err, http.StatusInternalServerError)) //nolint:golint,errcheck
		return
	}
	g.JSON(http.StatusOK, a.hndler.MarshalListResponse(res))
}
func (a *ApiHandler[T]) HandleDelete(g *gin.Context) {
	// HandleDelete handles HTTP DELETE requests to delete a resource.
	q, err := a.hndler.BindDeleteRequest(g)
	if err != nil {
		g.Error(NewHttpError(err, http.StatusBadRequest)) //nolint:golint,errcheck
		return
	}
	if _, err := a.fac.DeleteOne(g.Request.Context(), q); err != nil {
		if errors.Is(err, facade.ErrNoDocumentsFound) || errors.Is(err, facade.ErrMultipleDocumentsFound) {
			g.Error(NewHttpError(err, http.StatusBadRequest)) //nolint:golint,errcheck
			return
		}
		g.Error(NewHttpError(err, http.StatusInternalServerError)) //nolint:golint,errcheck
		return
	}
	g.AbortWithStatus(http.StatusOK)
}
func (a *ApiHandler[T]) RegisterRoutes(r *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	// RegisterRoutes registers CRUD routes for the resource on the given router group.
	apiG := r.Group(fmt.Sprintf("/%s", a.name))
	apiG.GET("/", authMiddleware, a.HandleRead)
	apiG.POST("/", authMiddleware, a.HandleCreate)
	apiG.DELETE("/:id", authMiddleware, a.HandleDelete)
}
func NewApiHandler[T any](hndler IHandler[T], fac facade.IFacade[T], name string) *ApiHandler[T] {
	// NewApiHandler creates a new ApiHandler for the given handler, manager, and resource name.
	return &ApiHandler[T]{
		hndler: hndler,
		fac:    fac,
		name:   name,
	}
}
