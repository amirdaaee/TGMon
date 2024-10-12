package helper

import (
	"context"
	"fmt"
	"strings"

	"github.com/amirdaaee/TGMon/internal/db"
	"github.com/google/uuid"
	mongoD "go.mongodb.org/mongo-driver/mongo"
)

func UpdateMediaThumbnail(ctx context.Context, mongo *db.Mongo, minio *db.MinioClient, data []byte, doc *db.MediaFileDoc, cl_ *mongoD.Client) error {
	filename := uuid.NewString() + ".jpeg"
	if err := minio.FileAdd(filename, data, ctx); err != nil {
		return fmt.Errorf("error addign file to minio: %s", err)
	}
	updateDoc := doc
	oldThumb := updateDoc.Thumbnail
	updateDoc.Thumbnail = filename
	_filter, _ := db.FilterById(updateDoc.ID)
	updateDoc.ID = ""
	if _, err := mongo.IMng.GetCollection(cl_).ReplaceOne(ctx, _filter, updateDoc); err != nil {
		return fmt.Errorf("can not replace mongo record: %s", err)
	}
	if oldThumb != "" {
		minio.FileRm(oldThumb, ctx)
	}
	return nil
}
func UpdateMediaVtt(ctx context.Context, mongo *db.Mongo, minio *db.MinioClient, image []byte, vtt []byte, doc *db.MediaFileDoc, cl_ *mongoD.Client) error {
	u := uuid.NewString()
	imageName := u + ".jpeg"
	vttName := u + ".vtt"
	if err := minio.FileAdd(imageName, image, ctx); err != nil {
		return fmt.Errorf("error addign image file to minio: %s", err)
	}
	vttStr := strings.ReplaceAll(string(vtt), "__NAME__", imageName)
	if err := minio.FileAddStr(vttName, vttStr, ctx); err != nil {
		return fmt.Errorf("error addign vtt file to minio: %s", err)
	}
	updateDoc := doc
	oldVtt := updateDoc.VttFileName
	oldImg := updateDoc.VttImageName
	updateDoc.VttFileName = vttName
	updateDoc.VttImageName = imageName
	_filter, _ := db.FilterById(updateDoc.ID)
	updateDoc.ID = ""
	if _, err := mongo.IMng.GetCollection(cl_).ReplaceOne(ctx, _filter, updateDoc); err != nil {
		return fmt.Errorf("can not replace mongo record: %s", err)
	}
	if oldVtt != "" {
		minio.FileRm(oldVtt, ctx)
	}
	if oldImg != "" {
		minio.FileRm(oldImg, ctx)
	}
	return nil
}
