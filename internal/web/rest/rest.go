package rest

import (
	"errors"
	"fmt"
	"net/http"

	ftypes "github.com/amirdaaee/TGMon/internal/facade/types"
	crd "github.com/amirdaaee/TGMon/internal/web/rest/crd"
	wtypes "github.com/amirdaaee/TGMon/internal/web/types"
	"github.com/gin-gonic/gin"
)

type RestHandler[T any] struct {
	hndler any // ICreateHandler[T] | IReadHandler[T] | IListHandler | IDeleteHandler
	fac    ftypes.IFacade[T]
	name   string
}

var _ wtypes.Registereable = (*RestHandler[any])(nil)

// ApiHandler provides CRUD handlers and route registration for a resource type T.
func (a *RestHandler[T]) HandleCreate(g *gin.Context) {
	handler, ok := a.hndler.(crd.ICreateHandler[T])
	if !ok {
		g.Error(wtypes.NewHttpError(fmt.Errorf("handler is not a ICreateApiHandler"), http.StatusInternalServerError)) //nolint:golint,errcheck
		return
	}
	doc, err := handler.BindCreateRequest(g)
	if err != nil {
		g.Error(wtypes.NewHttpError(err, http.StatusBadRequest)) //nolint:golint,errcheck
		return
	}
	res, err := a.fac.CreateOne(g.Request.Context(), doc)
	if err != nil {
		g.Error(wtypes.NewHttpError(err, http.StatusInternalServerError)) //nolint:golint,errcheck
		return
	}
	h, err := handler.MarshalCreateResponse(g, res)
	if err != nil {
		g.Error(wtypes.NewHttpError(err, http.StatusInternalServerError)) //nolint:golint,errcheck
		return
	}
	g.JSON(http.StatusOK, h)
}
func (a *RestHandler[T]) HandleRead(g *gin.Context) {
	handler, ok := a.hndler.(crd.IReadHandler[T])
	if !ok {
		g.Error(wtypes.NewHttpError(fmt.Errorf("handler is not a IReadApiHandler"), http.StatusInternalServerError)) //nolint:golint,errcheck
		return
	}
	id, err := handler.BindReadRequest(g)
	if err != nil {
		g.Error(wtypes.NewHttpError(err, http.StatusBadRequest)) //nolint:golint,errcheck
		return
	}
	res, err := a.fac.FindByID(g.Request.Context(), id)
	if err != nil {
		if errors.Is(err, ftypes.ErrNoDocumentsFound) || errors.Is(err, ftypes.ErrMultipleDocumentsFound) {
			g.Error(wtypes.NewHttpError(err, http.StatusBadRequest)) //nolint:golint,errcheck
			return
		}
		g.Error(wtypes.NewHttpError(err, http.StatusInternalServerError)) //nolint:golint,errcheck
		return
	}
	h, err := handler.MarshalReadResponse(g, res)
	if err != nil {
		g.Error(wtypes.NewHttpError(err, http.StatusInternalServerError)) //nolint:golint,errcheck
		return
	}
	g.JSON(http.StatusOK, h)
}
func (a *RestHandler[T]) HandleList(g *gin.Context) {
	handler, ok := a.hndler.(crd.IListHandler)
	if !ok {
		g.Error(wtypes.NewHttpError(fmt.Errorf("handler is not a IListApiHandler"), http.StatusInternalServerError)) //nolint:golint,errcheck
		return
	}
	h, err := handler.HandleList(g)
	if err != nil {
		g.Error(wtypes.NewHttpError(err, http.StatusInternalServerError)) //nolint:golint,errcheck
		return
	}
	g.JSON(http.StatusOK, h)
}
func (a *RestHandler[T]) HandleDelete(g *gin.Context) {
	handler, ok := a.hndler.(crd.IDeleteHandler)
	if !ok {
		g.Error(wtypes.NewHttpError(fmt.Errorf("handler is not a IDeleteApiHandler"), http.StatusInternalServerError)) //nolint:golint,errcheck
		return
	}
	id, err := handler.BindDeleteRequest(g)
	if err != nil {
		g.Error(wtypes.NewHttpError(err, http.StatusBadRequest)) //nolint:golint,errcheck
		return
	}
	if _, err := a.fac.DeleteByID(g.Request.Context(), id); err != nil {
		if errors.Is(err, ftypes.ErrNoDocumentsFound) || errors.Is(err, ftypes.ErrMultipleDocumentsFound) {
			g.Error(wtypes.NewHttpError(err, http.StatusBadRequest)) //nolint:golint,errcheck
			return
		}
		g.Error(wtypes.NewHttpError(err, http.StatusInternalServerError)) //nolint:golint,errcheck
		return
	}
	g.AbortWithStatus(http.StatusOK)
}
func (a *RestHandler[T]) RegisterRoutes(r *gin.RouterGroup, authMiddleware gin.HandlerFunc) error {
	registered := false
	apiG := r.Group(fmt.Sprintf("/%s", a.name))
	if _, ok := a.hndler.(crd.ICreateHandler[T]); ok {
		apiG.POST("/", authMiddleware, a.HandleCreate)
		registered = true
	}
	if _, ok := a.hndler.(crd.IListHandler); ok {
		apiG.GET("/", authMiddleware, a.HandleList)
		registered = true
	}
	if _, ok := a.hndler.(crd.IDeleteHandler); ok {
		apiG.DELETE("/:id", authMiddleware, a.HandleDelete)
		registered = true
	}
	if _, ok := a.hndler.(crd.IReadHandler[T]); ok {
		apiG.GET("/:id", authMiddleware, a.HandleRead)
		registered = true
	}
	if !registered {
		return fmt.Errorf("no routes registered for %s", a.name)
	}
	return nil
}
func (a *RestHandler[T]) RegisterToRoot() bool {
	if v, ok := a.hndler.(wtypes.RootRegisterer); !ok {
		return false
	} else {
		return v.RegisterToRoot()
	}
}
func NewCRDApiHandler[T any](hndler any, fac ftypes.IFacade[T], name string) *RestHandler[T] {
	return &RestHandler[T]{
		hndler: hndler,
		fac:    fac,
		name:   name,
	}
}
