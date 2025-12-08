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
	hndler any // ICreateHandler[T] | IReadHandler[T] | IListHandler[T] | IDeleteHandler[T]
	fac    ftypes.IFacade[T]
	name   string
}

var _ wtypes.Registereable = (*RestHandler[any])(nil)

// ApiHandler provides CRUD handlers and route registration for a resource type T.
func (a *RestHandler[T]) HandleCreate(g *gin.Context) {
	// HandleCreate handles HTTP POST requests to create a new resource.
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
	q, err := handler.BindReadRequest(g)
	if err != nil {
		g.Error(wtypes.NewHttpError(err, http.StatusBadRequest)) //nolint:golint,errcheck
		return
	}
	res, err := a.fac.GetCRD().GetCollection().Finder().Filter(q).FindOne(g.Request.Context())
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
	// HandleRead handles HTTP GET requests to read resources.
	handler, ok := a.hndler.(crd.IListHandler[T])
	if !ok {
		g.Error(wtypes.NewHttpError(fmt.Errorf("handler is not a IListApiHandler"), http.StatusInternalServerError)) //nolint:golint,errcheck
		return
	}
	req, err := handler.BindListRequest(g, a.fac.GetCollection().Finder())
	if err != nil {
		g.Error(wtypes.NewHttpError(err, http.StatusBadRequest)) //nolint:golint,errcheck
		return
	}
	res, err := req.Find(g.Request.Context())
	if err != nil {
		g.Error(wtypes.NewHttpError(err, http.StatusInternalServerError)) //nolint:golint,errcheck
		return
	}
	h, err := handler.MarshalListResponse(g, res)
	if err != nil {
		g.Error(wtypes.NewHttpError(err, http.StatusInternalServerError)) //nolint:golint,errcheck
		return
	}
	g.JSON(http.StatusOK, h)
}
func (a *RestHandler[T]) HandleDelete(g *gin.Context) {
	// HandleDelete handles HTTP DELETE requests to delete a resource.
	handler, ok := a.hndler.(crd.IDeleteHandler[T])
	if !ok {
		g.Error(wtypes.NewHttpError(fmt.Errorf("handler is not a IDeleteApiHandler"), http.StatusInternalServerError)) //nolint:golint,errcheck
		return
	}
	q, err := handler.BindDeleteRequest(g)
	if err != nil {
		g.Error(wtypes.NewHttpError(err, http.StatusBadRequest)) //nolint:golint,errcheck
		return
	}
	if _, err := a.fac.DeleteOne(g.Request.Context(), q); err != nil {
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
	// RegisterRoutes registers CRUD routes for the resource on the given router group.
	registered := false
	apiG := r.Group(fmt.Sprintf("/%s", a.name))
	if _, ok := a.hndler.(crd.ICreateHandler[T]); ok {
		apiG.POST("/", authMiddleware, a.HandleCreate)
		registered = true
	}
	if _, ok := a.hndler.(crd.IListHandler[T]); ok {
		apiG.GET("/", authMiddleware, a.HandleList)
		registered = true
	}
	if _, ok := a.hndler.(crd.IDeleteHandler[T]); ok {
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
	// NewApiHandler creates a new ApiHandler for the given handler, manager, and resource name.
	return &RestHandler[T]{
		hndler: hndler,
		fac:    fac,
		name:   name,
	}
}
