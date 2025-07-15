package web

import (
	"fmt"

	"github.com/amirdaaee/TGMon/internal/types"
	"github.com/chenmingyong0423/go-mongox/v2/builder/query"
	"github.com/chenmingyong0423/go-mongox/v2/finder"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// Package api provides handler interfaces and implementations for API resource operations.

type ICRDApiHandler[T any] interface {
	BindCreateRequest(g *gin.Context) (*T, error)
	BindListRequest(g *gin.Context, fnd finder.IFinder[T]) (finder.IFinder[T], error)
	BindDeleteRequest(g *gin.Context) (bson.D, error)
	MarshalCreateResponse(*T) any
	MarshalListResponse([]*T) any

	HasCreate() bool
	HasGet() bool
	HasList() bool
	HasDelete() bool
}

// IHandler defines methods for binding requests and marshaling responses for a resource type T.

// MediaHandler implements IHandler for media resources.
type MediaHandler struct{}

// JobReqHandler implements IHandler for media resources.
type JobReqHandler struct{}

// JobResHandler implements IHandler for media resources.
type JobResHandler struct{}

var _ ICRDApiHandler[types.MediaFileDoc] = (*MediaHandler)(nil)
var _ ICRDApiHandler[types.JobReqDoc] = (*JobReqHandler)(nil)
var _ ICRDApiHandler[types.JobResDoc] = (*JobResHandler)(nil)

// =====
func (h *MediaHandler) BindCreateRequest(g *gin.Context) (*types.MediaFileDoc, error) {
	return nil, ErrNotImplemented
}
func (h *MediaHandler) BindListRequest(g *gin.Context, fnd finder.IFinder[types.MediaFileDoc]) (finder.IFinder[types.MediaFileDoc], error) {
	var v MediaListReqType
	const resultPerPage = 12
	if err := g.ShouldBindQuery(&v); err != nil {
		return nil, err
	}
	fnd = fnd.Sort(bson.D{{Key: "created_at", Value: -1}}).Skip(resultPerPage * int64(v.Page)).Limit(resultPerPage)
	return fnd, nil
}
func (h *MediaHandler) BindDeleteRequest(g *gin.Context) (bson.D, error) {
	var qID MediaDelReqType
	if err := g.ShouldBindUri(&qID); err != nil {
		return nil, err
	}
	idObj, err := bson.ObjectIDFromHex(qID.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid id: %w", err)
	}
	q := query.Id(idObj)
	return q, nil
}
func (h *MediaHandler) MarshalCreateResponse(v *types.MediaFileDoc) any {
	return nil
}
func (h *MediaHandler) MarshalListResponse(v []*types.MediaFileDoc) any {
	res := make([]*types.MediaFileDoc, len(v))
	for i, doc := range v {
		_v := types.MediaFileDoc(*doc)
		res[i] = &_v
	}
	return MediaListResType(res)
}
func (h *MediaHandler) HasCreate() bool {
	return false
}
func (h *MediaHandler) HasGet() bool {
	return true
}
func (h *MediaHandler) HasList() bool {
	return true
}
func (h *MediaHandler) HasDelete() bool {
	return true
}

// =====
func (h *JobReqHandler) BindCreateRequest(g *gin.Context) (*types.JobReqDoc, error) {
	return nil, ErrNotImplemented
}
func (h *JobReqHandler) BindListRequest(g *gin.Context, fnd finder.IFinder[types.JobReqDoc]) (finder.IFinder[types.JobReqDoc], error) {
	return fnd, nil
}
func (h *JobReqHandler) BindDeleteRequest(g *gin.Context) (bson.D, error) {
	var qID JobReqDelReqType
	if err := g.ShouldBindUri(&qID); err != nil {
		return nil, err
	}
	idObj, err := bson.ObjectIDFromHex(qID.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid id: %w", err)
	}
	q := query.Id(idObj)
	return q, nil
}
func (h *JobReqHandler) MarshalCreateResponse(v *types.JobReqDoc) any {
	return nil
}
func (h *JobReqHandler) MarshalListResponse(v []*types.JobReqDoc) any {
	res := make([]*types.JobReqDoc, len(v))
	for i, doc := range v {
		_v := types.JobReqDoc(*doc)
		res[i] = &_v
	}
	return JobReqListResType(res)
}
func (h *JobReqHandler) HasCreate() bool {
	return false
}
func (h *JobReqHandler) HasGet() bool {
	return false
}
func (h *JobReqHandler) HasList() bool {
	return true
}
func (h *JobReqHandler) HasDelete() bool {
	return true
}

// =====
func (h *JobResHandler) BindCreateRequest(g *gin.Context) (*types.JobResDoc, error) {
	var v JobResPostReqType
	if err := g.ShouldBindJSON(&v); err != nil {
		return nil, err
	}
	return v, nil
}
func (h *JobResHandler) BindListRequest(g *gin.Context, fnd finder.IFinder[types.JobResDoc]) (finder.IFinder[types.JobResDoc], error) {
	return nil, ErrNotImplemented
}
func (h *JobResHandler) BindDeleteRequest(g *gin.Context) (bson.D, error) {
	return nil, ErrNotImplemented
}
func (h *JobResHandler) MarshalCreateResponse(v *types.JobResDoc) any {
	return v
}
func (h *JobResHandler) MarshalListResponse(v []*types.JobResDoc) any {
	return nil
}
func (h *JobResHandler) HasCreate() bool {
	return true
}
func (h *JobResHandler) HasGet() bool {
	return false
}
func (h *JobResHandler) HasList() bool {
	return false
}
func (h *JobResHandler) HasDelete() bool {
	return false
}
