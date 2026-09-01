package handler

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type InfoGetResType struct {
	MediaCount int64
}

type LoginPostReqType struct {
	Username string `binding:"required"`
	Password string `binding:"required"`
}
type LoginPostResType struct {
	Token string
}

type RandomMediaGetResType struct {
	MediaID *bson.ObjectID
}

type MediaMetaIDReqType struct {
	ID string `uri:"id" binding:"required"`
}

type MediaMetaPutReqType struct {
	LastPlayedAt time.Time
	Checkpoint   uint64
	Score        int
	Likes        int
	IsFavorite   bool
}

type MediaMetaPatchReqType struct {
	LastPlayedAt *time.Time
	PlayCount    *int
	Checkpoint   *uint64
	Score        *int
	Likes        *int
	IsFavorite   *bool
}
