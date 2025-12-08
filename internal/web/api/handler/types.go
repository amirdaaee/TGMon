package handler

import "go.mongodb.org/mongo-driver/v2/bson"

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
