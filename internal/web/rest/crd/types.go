package crd

import (
	"github.com/amirdaaee/TGMon/internal/types"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type MediaReadReqType struct {
	ID string `uri:"id" binding:"required"`
}
type MediaWithMetaType struct {
	Media *types.MediaFileDoc
	Meta  *types.MediaExtendedMeta
}
type MediaReadResType struct {
	MediaWithMetaType
	PervID *bson.ObjectID `json:"pervID"`
	NextID *bson.ObjectID `json:"nextID"`
}
type MediaListReqType struct {
	Page int `form:"page"`
}
type MediaDelReqType struct {
	ID string `uri:"id" binding:"required"`
}
type MediaListResType struct {
	Media []*MediaWithMetaType
	Total int64
}

// ===
type JobReqDelReqType struct {
	ID string `uri:"id" binding:"required"`
}
type JobReqListResType []*types.JobReqDoc
