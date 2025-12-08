package handler

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	ftypes "github.com/amirdaaee/TGMon/internal/facade/types"
	"github.com/amirdaaee/TGMon/internal/stash"
	"github.com/amirdaaee/TGMon/internal/types"
	wtypes "github.com/amirdaaee/TGMon/internal/web/types"
	"github.com/chenmingyong0423/go-mongox/v2/builder/query"
	"github.com/gin-gonic/gin"
)

type StashVTTRedirectorApiHandler struct {
	MinioUrl    string
	StashCl     *stash.StashQlClient
	MediaFacade ftypes.IFacade[types.MediaFileDoc]
}
type StashCoverRedirectorApiHandler struct {
	StashVTTRedirectorApiHandler
}

var _ IGetApiHandler = (*StashVTTRedirectorApiHandler)(nil)
var _ IGetApiHandler = (*StashCoverRedirectorApiHandler)(nil)
var _ wtypes.RootRegisterer = (*StashVTTRedirectorApiHandler)(nil)
var _ wtypes.RootRegisterer = (*StashCoverRedirectorApiHandler)(nil)

// ===
type idURIType struct {
	ID string `uri:"id" binding:"required"`
}

func (h *StashVTTRedirectorApiHandler) Get(g *gin.Context) {
	var id idURIType
	if err := g.ShouldBindUri(&id); err != nil {
		g.Error(wtypes.NewHttpError(err, http.StatusBadRequest)) //nolint:golint,errcheck
		return
	}
	osHashSplt := strings.Split(id.ID, "_")
	scene, err := h.StashCl.FindSceneByHash(g.Request.Context(), osHashSplt[0])
	if err != nil {
		g.Error(wtypes.NewHttpError(err, http.StatusNotFound)) //nolint:golint,errcheck
		return
	}
	media, err := h.getMediaByScene(g.Request.Context(), scene)
	if err != nil {
		g.Error(wtypes.NewHttpError(err, http.StatusNotFound)) //nolint:golint,errcheck
		return
	}
	if media != nil && media.Vtt != "" {
		g.Redirect(http.StatusPermanentRedirect, fmt.Sprintf("%s/%s", h.MinioUrl, media.Vtt))

	} else {
		g.Error(wtypes.NewHttpError(errors.New("no vtt file found"), http.StatusNotFound)) //nolint:golint,errcheck
	}
}
func (h *StashVTTRedirectorApiHandler) AuthGet() bool {
	return false
}
func (h *StashVTTRedirectorApiHandler) RelativePathGet() string {
	return "/scene/:id"
}

func (h *StashVTTRedirectorApiHandler) RegisterToRoot() bool {
	return true
}

func (h *StashVTTRedirectorApiHandler) getMediaByScene(ctx context.Context, scene *stash.Scene) (*types.MediaFileDoc, error) {
	file := scene.Files[0]
	fname := file.Basename
	if idx := strings.LastIndex(fname, "."); idx != -1 {
		fname = fname[:idx]
	}
	media, err := h.MediaFacade.GetCollection().Finder().Filter(query.Eq(types.MediaFileDoc__UnameField, fname)).FindOne(ctx)
	if err != nil {
		return nil, fmt.Errorf("can not query media by uname (%s): %w", fname, err)
	}
	return media, nil
}

// ===
func (h *StashCoverRedirectorApiHandler) Get(g *gin.Context) {
	var id idURIType
	if err := g.ShouldBindUri(&id); err != nil {
		g.Error(wtypes.NewHttpError(err, http.StatusBadRequest)) //nolint:golint,errcheck
		return
	}
	scene, err := h.StashCl.FindSceneById(g.Request.Context(), id.ID)
	if err != nil {
		g.Error(wtypes.NewHttpError(err, http.StatusNotFound)) //nolint:golint,errcheck
		return
	}
	media, err := h.getMediaByScene(g.Request.Context(), scene)
	if err != nil {
		g.Error(wtypes.NewHttpError(err, http.StatusNotFound)) //nolint:golint,errcheck
		return
	}
	if media != nil {
		g.Redirect(http.StatusPermanentRedirect, fmt.Sprintf("%s/%s", h.MinioUrl, media.Thumbnail))

	} else {
		g.Error(wtypes.NewHttpError(err, http.StatusNotFound)) //nolint:golint,errcheck
	}
}
func (h *StashCoverRedirectorApiHandler) RelativePathGet() string {
	return "/scene/:id/screenshot"
}
func (h *StashCoverRedirectorApiHandler) RegisterToRoot() bool {
	return true
}
