package web

import "github.com/amirdaaee/TGMon/internal/types"

type StreamMetaData struct {
	Start         int64
	End           int64
	ContentLength int64
	MimeType      string
	FileSize      int64
	Filename      string
}

type StreamReq struct {
	ID string `uri:"mediaID" binding:"required"`
}

// ===
type MediaListReqType struct {
	Page int `form:"page"`
}
type MediaDelReqType struct {
	ID string `uri:"id" binding:"required"`
}
type MediaListResType []*types.MediaFileDoc

// ===
type JobReqDelReqType struct {
	ID string `uri:"id" binding:"required"`
}
type JobReqListResType []*types.JobReqDoc

// ===
type JobResPostReqType *types.JobResDoc

type JobResListReqType []*types.JobResDoc

// type JobResPostResType []*types.JobResDoc
