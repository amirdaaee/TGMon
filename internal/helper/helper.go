package helper

import (
	"context"
	"fmt"

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
