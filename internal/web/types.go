package web

import "github.com/amirdaaee/TGMon/internal/db"

type streamReq struct {
	ID string `uri:"mediaID" binding:"required"`
}
type mediaListReq struct {
	Page     int `form:"page"`
	PageSize int `form:"page_size"`
}
type loginReq struct {
	Username string
	Password string
}

type mediaListRes struct {
	Media []db.MediaFileDoc
	Total int64
}
type mediaInfoRes struct {
	Media db.MediaFileDoc
	Next  db.MediaFileDoc
	Back  db.MediaFileDoc
}

// ...
type createJobReq struct {
	Job []db.JobDoc `json:"job" binding:"required"`
}
type putJobResultReq struct {
	ID     string `uri:"jobID" binding:"required"`
	Status int    `uri:"status" binding:"required"`
}
type listJobRes struct {
	Job []db.JobDoc
}
