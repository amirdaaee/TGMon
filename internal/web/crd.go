package web

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/amirdaaee/TGMon/internal/facade"
	"github.com/gin-gonic/gin"
)

// Package api provides generic API handler logic for CRUD operations using Gin and MongoDB.

type CRDApiHandler[T any] struct {
	hndler any
	fac    facade.IFacade[T]
	name   string
}

// ApiHandler provides CRUD handlers and route registration for a resource type T.
func (a *CRDApiHandler[T]) HandleCreate(g *gin.Context) {
	// HandleCreate handles HTTP POST requests to create a new resource.
	handler, ok := a.hndler.(ICreateApiHandler[T])
	if !ok {
		g.Error(NewHttpError(fmt.Errorf("handler is not a ICreateApiHandler"), http.StatusInternalServerError)) //nolint:golint,errcheck
		return
	}
	doc, err := handler.BindCreateRequest(g)
	if err != nil {
		g.Error(NewHttpError(err, http.StatusBadRequest)) //nolint:golint,errcheck
		return
	}
	res, err := a.fac.CreateOne(g.Request.Context(), doc)
	if err != nil {
		g.Error(NewHttpError(err, http.StatusInternalServerError)) //nolint:golint,errcheck
		return
	}
	h, err := handler.MarshalCreateResponse(g, res)
	if err != nil {
		g.Error(NewHttpError(err, http.StatusInternalServerError)) //nolint:golint,errcheck
		return
	}
	g.JSON(http.StatusOK, h)
}
func (a *CRDApiHandler[T]) HandleRead(g *gin.Context) {
	handler, ok := a.hndler.(IReadApiHandler[T])
	if !ok {
		g.Error(NewHttpError(fmt.Errorf("handler is not a IReadApiHandler"), http.StatusInternalServerError)) //nolint:golint,errcheck
		return
	}
	q, err := handler.BindReadRequest(g)
	if err != nil {
		g.Error(NewHttpError(err, http.StatusBadRequest)) //nolint:golint,errcheck
		return
	}
	res, err := a.fac.GetCRD().GetCollection().Finder().Filter(q).FindOne(g.Request.Context())
	if err != nil {
		if errors.Is(err, facade.ErrNoDocumentsFound) || errors.Is(err, facade.ErrMultipleDocumentsFound) {
			g.Error(NewHttpError(err, http.StatusBadRequest)) //nolint:golint,errcheck
			return
		}
		g.Error(NewHttpError(err, http.StatusInternalServerError)) //nolint:golint,errcheck
		return
	}
	h, err := handler.MarshalReadResponse(g, res)
	if err != nil {
		g.Error(NewHttpError(err, http.StatusInternalServerError)) //nolint:golint,errcheck
		return
	}
	g.JSON(http.StatusOK, h)
}
func (a *CRDApiHandler[T]) HandleList(g *gin.Context) {
	// HandleRead handles HTTP GET requests to read resources.
	handler, ok := a.hndler.(IListApiHandler[T])
	if !ok {
		g.Error(NewHttpError(fmt.Errorf("handler is not a IListApiHandler"), http.StatusInternalServerError)) //nolint:golint,errcheck
		return
	}
	req, err := handler.BindListRequest(g, a.fac.GetCollection().Finder())
	if err != nil {
		g.Error(NewHttpError(err, http.StatusBadRequest)) //nolint:golint,errcheck
		return
	}
	res, err := req.Find(g.Request.Context())
	if err != nil {
		g.Error(NewHttpError(err, http.StatusInternalServerError)) //nolint:golint,errcheck
		return
	}
	h, err := handler.MarshalListResponse(g, res)
	if err != nil {
		g.Error(NewHttpError(err, http.StatusInternalServerError)) //nolint:golint,errcheck
		return
	}
	g.JSON(http.StatusOK, h)
}
func (a *CRDApiHandler[T]) HandleDelete(g *gin.Context) {
	// HandleDelete handles HTTP DELETE requests to delete a resource.
	handler, ok := a.hndler.(IDeleteApiHandler[T])
	if !ok {
		g.Error(NewHttpError(fmt.Errorf("handler is not a IDeleteApiHandler"), http.StatusInternalServerError)) //nolint:golint,errcheck
		return
	}
	q, err := handler.BindDeleteRequest(g)
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
func (a *CRDApiHandler[T]) RegisterRoutes(r *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	// RegisterRoutes registers CRUD routes for the resource on the given router group.
	apiG := r.Group(fmt.Sprintf("/%s", a.name))
	if _, ok := a.hndler.(ICreateApiHandler[T]); ok {
		apiG.POST("/", authMiddleware, a.HandleCreate)
	}
	if _, ok := a.hndler.(IListApiHandler[T]); ok {
		apiG.GET("/", authMiddleware, a.HandleList)
	}
	if _, ok := a.hndler.(IDeleteApiHandler[T]); ok {
		apiG.DELETE("/:id", authMiddleware, a.HandleDelete)
	}
	if _, ok := a.hndler.(IReadApiHandler[T]); ok {
		apiG.GET("/:id", authMiddleware, a.HandleRead)
	}
}
func NewCRDApiHandler[T any](hndler any, fac facade.IFacade[T], name string) *CRDApiHandler[T] {
	// NewApiHandler creates a new ApiHandler for the given handler, manager, and resource name.
	return &CRDApiHandler[T]{
		hndler: hndler,
		fac:    fac,
		name:   name,
	}
}
