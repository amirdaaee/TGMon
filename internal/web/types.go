package web

import "github.com/amirdaaee/TGMon/internal/db"

type streamReq struct {
	ID string `uri:"mediaID" binding:"required"`
}
type mediaListReq struct {
	Page     int `form:"page"`
	PageSize int `form:"page_size"`
}
type thumbnailReq struct {
	MediaIDs []string `json:"media_ids" binding:"required"`
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
