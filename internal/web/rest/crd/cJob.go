package crd

import (
	"fmt"

	"github.com/amirdaaee/TGMon/internal/repository"
	"github.com/amirdaaee/TGMon/internal/types"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// JobReqHandler implements IHandler for job request resources.
type JobReqHandler struct {
	jobs repository.JobReqRepository
}

// JobResHandler implements IHandler for job result resources.
type JobResHandler struct{}

var _ ICreateHandler[types.JobReqDoc] = (*JobReqHandler)(nil)
var _ IListHandler = (*JobReqHandler)(nil)
var _ IDeleteHandler = (*JobReqHandler)(nil)

var _ ICreateHandler[types.JobResDoc] = (*JobResHandler)(nil)

func NewJobReqHandler(jobs repository.JobReqRepository) *JobReqHandler {
	return &JobReqHandler{jobs: jobs}
}

// =====
// @Summary	Create job request
// @Tags		jobReq
// @Produce	json
// @Param		data	body		types.JobReqDoc	true	"Job Request Data"
// @Success	200	{object}	types.JobReqDoc
// @Router		/api/jobReq/ [post]
// @Security	ApiKeyAuth
func (h *JobReqHandler) BindCreateRequest(g *gin.Context) (*types.JobReqDoc, error) {
	var v types.JobReqDoc
	if err := g.ShouldBindJSON(&v); err != nil {
		return nil, err
	}
	return &v, nil
}
func (h *JobReqHandler) MarshalCreateResponse(g *gin.Context, v *types.JobReqDoc) (any, error) {
	return v, nil
}

// @Summary	List job requests
// @Tags		jobReq
// @Produce	json
// @Success	200	{array}	types.JobReqDoc
// @Router		/api/jobReq/ [get]
// @Security	ApiKeyAuth
func (h *JobReqHandler) HandleList(g *gin.Context) (any, error) {
	docs, err := h.jobs.List(g.Request.Context())
	if err != nil {
		return nil, fmt.Errorf("error listing job requests: %w", err)
	}
	res := make([]*types.JobReqDoc, len(docs))
	for i, doc := range docs {
		_v := types.JobReqDoc(*doc)
		res[i] = &_v
	}
	return JobReqListResType(res), nil
}

// @Summary	Delete job request
// @Tags		jobReq
// @Produce	json
// @Param		id	path		string	true	"Job Request ID"
// @Success	200	{string}	string	"OK"
// @Router		/api/jobReq/{id}/ [delete]
// @Security	ApiKeyAuth
func (h *JobReqHandler) BindDeleteRequest(g *gin.Context) (bson.ObjectID, error) {
	var qID JobReqDelReqType
	if err := g.ShouldBindUri(&qID); err != nil {
		return bson.ObjectID{}, err
	}
	idObj, err := bson.ObjectIDFromHex(qID.ID)
	if err != nil {
		return bson.ObjectID{}, fmt.Errorf("invalid id: %w", err)
	}
	return idObj, nil
}

// =====
//
//	@Summary	Create job response
//	@Tags		jobRes
//	@Accept		json
//	@Produce	json
//	@Param		data	body		types.JobResDoc	true	"Job Response Data"
//	@Success	200		{object}	types.JobResDoc
//	@Router		/api/jobRes/ [post]
//	@Security	ApiKeyAuth
func (h *JobResHandler) BindCreateRequest(g *gin.Context) (*types.JobResDoc, error) {
	var v types.JobResDoc
	if err := g.ShouldBindJSON(&v); err != nil {
		return nil, err
	}
	return &v, nil
}
func (h *JobResHandler) MarshalCreateResponse(g *gin.Context, v *types.JobResDoc) (any, error) {
	return v, nil
}
