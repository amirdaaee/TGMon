package web

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/amirdaaee/TGMon/internal/bot"
	"github.com/amirdaaee/TGMon/internal/db"
	"github.com/amirdaaee/TGMon/internal/ffmpeg"
	"github.com/amirdaaee/TGMon/internal/helper"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	mongoD "go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func streamHandlerFactory(wp *bot.WorkerPool, mongo *db.Mongo, chunckSize int64, profileFile string) func(g *gin.Context) {
	medMongo := mongo.GetMediaMongo()
	return func(g *gin.Context) {
		var media streamReq
		if err := g.ShouldBindUri(&media); err != nil {
			g.AbortWithError(http.StatusBadRequest, err)
			return
		}
		// media.ID = strings.TrimSuffix(media.ID, ".m3u8")
		media.ID = strings.TrimSuffix(media.ID, ".mp4")
		if strings.Contains(media.ID, ".") {
			g.AbortWithStatus(http.StatusNotFound)
			return
		}
		err := steam(g, media, wp, medMongo, chunckSize, profileFile)
		if err != nil {
			g.AbortWithError(http.StatusBadRequest, err)
			return
		}
	}
}
func listMediaHandlerFactory(mongo *db.Mongo) func(g *gin.Context) {
	medMongo := mongo.GetMediaMongo()
	return func(g *gin.Context) {
		ll := logrus.WithField("handler", "listMediaHandler")
		var listReq mediaListReq
		if err := g.ShouldBind(&listReq); err != nil {
			g.AbortWithError(http.StatusBadRequest, err)
			return
		}
		cl_, err := mongo.GetClient()
		if err != nil {
			ll.WithError(err).Error("error get client")
			g.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		defer cl_.Disconnect(g)
		// ...
		coll_ := medMongo.IMng.GetCollection(cl_)
		count_, err := coll_.CountDocuments(g, bson.D{})
		if err != nil {
			ll.WithError(err).Error("error count")
			g.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		opts := options.Find().SetSort(bson.D{{Key: "DateAdded", Value: -1}, {Key: "FileID", Value: 1}})
		if listReq.PageSize > 0 {
			opts = opts.SetLimit(int64(listReq.PageSize))
			opts = opts.SetSkip(int64(listReq.PageSize * (listReq.Page - 1)))
		}
		mediaList := []db.MediaFileDoc{}
		if err := medMongo.DocGetAll(g, &mediaList, cl_, opts); err != nil {
			ll.WithError(err).Error("error get docs")
			g.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		response := mediaListRes{
			Media: mediaList,
			Total: count_,
		}
		g.JSON(http.StatusOK, response)
	}
}
func infoMediaHandlerFactory(mongo *db.Mongo) func(g *gin.Context) {
	medMongo := mongo.GetMediaMongo()
	return func(g *gin.Context) {
		var mediaReq streamReq
		if err := g.ShouldBindUri(&mediaReq); err != nil {
			g.AbortWithError(http.StatusBadRequest, err)
			return
		}
		// ....
		cl_, err := mongo.GetClient()
		if err != nil {
			g.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		defer cl_.Disconnect(g)
		// ...
		var resData mediaInfoRes
		var mediaDoc db.MediaFileDoc
		if err := medMongo.DocGetById(g, mediaReq.ID, &mediaDoc, cl_); err != nil {
			g.AbortWithError(http.StatusBadRequest, err)
			return
		}
		resData.Media = mediaDoc
		// ...
		backMediaDoc, nextMediaDoc, err := medMongo.DocGetNeighbour(g, mediaDoc, cl_)
		if err != nil {
			logrus.WithError(err).Error("error getting Neighbour docs")
		} else {
			if backMediaDoc != nil {
				resData.Back = *backMediaDoc
			}
			if nextMediaDoc != nil {
				resData.Next = *nextMediaDoc
			}
		}
		// ...
		g.JSON(http.StatusOK, resData)
	}
}
func deleteMediaHandlerFactory(wp *bot.WorkerPool, mongo *db.Mongo, minio *db.MinioClient) func(g *gin.Context) {
	medMongo := mongo.GetMediaMongo()
	return func(g *gin.Context) {
		var mediaReq streamReq
		if err := g.ShouldBindUri(&mediaReq); err != nil {
			g.AbortWithError(http.StatusBadRequest, err)
			return
		}
		cl_, err := mongo.GetClient()
		if err != nil {
			g.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		defer cl_.Disconnect(g)
		// ...
		ctx := context.Background()
		var mediaDoc db.MediaFileDoc
		if err := medMongo.DocGetById(ctx, mediaReq.ID, &mediaDoc, cl_); err != nil {
			g.AbortWithError(http.StatusBadRequest, err)
			return
		}
		go wp.GetNextWorker().DeleteMessages([]int{mediaDoc.MessageID})
		go minio.FileRm(mediaDoc.Thumbnail, ctx)
		if err := medMongo.DocDelById(g, mediaDoc.ID, cl_); err != nil {
			g.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		g.JSON(http.StatusOK, "")
	}
}
func createThumbnailHandlerFactory(mongo *db.Mongo, minio *db.MinioClient, ffimage string, server string) func(g *gin.Context) {
	medMongo := mongo.GetMediaMongo()
	return func(g *gin.Context) {
		var thumbReq thumbnailReq
		if err := g.ShouldBindJSON(&thumbReq); err != nil {
			g.AbortWithError(http.StatusBadRequest, err)
			return
		}
		// ...
		go func() {
			ctx := context.Background()
			l1 := logrus.WithField("module", "thumbnail_gen")
			mongoCl, err := mongo.GetClient()
			if err != nil {
				l1.WithError(err).Error("can not connect to db")
				return
			}
			defer mongoCl.Disconnect(g)
			ffContainer, err := ffmpeg.NewFFmpegContainer(ffimage)
			if err != nil {
				l1.WithError(err).Error("can not create ffmpeg container")
				return
			}
			defer ffContainer.Close()
			for _, m := range thumbReq.MediaIDs {
				l2 := l1.WithField("media", m)
				doc := new(db.MediaFileDoc)
				if err := medMongo.DocGetById(ctx, m, doc, mongoCl); err != nil {
					l2.WithError(err).Error("error getting media from db")
					continue
				}
				timeAt := int(doc.Duration * 0.1)
				data, err := ffmpeg.GenThumnail(ffContainer, fmt.Sprintf("%s/stream/%s", server, m), timeAt)
				if err != nil {
					l2.WithError(err).Error("can not generate thumbnail")
					continue
				}
				filename := uuid.NewString() + ".jpeg"
				if err := minio.FileAdd(filename, data, ctx); err != nil {
					l2.WithError(err).Error("can not add new thumbnail to minio")
					continue
				}
				// ...
				updateDoc := doc
				oldThumb := updateDoc.Thumbnail
				updateDoc.Thumbnail = filename
				_filter, _ := db.FilterById(updateDoc.ID)
				updateDoc.ID = ""
				if _, err := medMongo.IMng.GetCollection(mongoCl).ReplaceOne(ctx, _filter, updateDoc); err != nil {
					l2.WithError(err).Error("can not replace mongo record")
					continue
				}
				if oldThumb != "" {
					if err := minio.FileRm(oldThumb, ctx); err != nil {
						l2.WithError(err).Warn("can not remove old thumbnail")
					}
				}
				l2.Info("updated")
			}
		}()
		g.JSON(http.StatusOK, "")
	}
}
func loginHandlerFactory(username string, password string, sessToken string) func(g *gin.Context) {
	return func(g *gin.Context) {
		var cred loginReq
		if err := g.ShouldBind(&cred); err != nil {
			g.JSON(http.StatusBadRequest, gin.H{"msg": err.Error()})
			return
		}
		if cred.Username == username && cred.Password == password {
			g.JSON(http.StatusOK, map[string]string{"token": sessToken})
			return
		}
		g.AbortWithStatus(http.StatusUnauthorized)
	}
}
func sessionHandlerFactory(sessToken string) func(g *gin.Context) {
	return func(g *gin.Context) { g.JSON(http.StatusOK, map[string]string{"token": sessToken}) }
}

// ----
func listJobsHandlerFactory(mongo *db.Mongo) func(g *gin.Context) {
	medMongo := mongo.GetJobMongo()
	return func(g *gin.Context) {
		ll := logrus.WithField("handler", "listJobsHandler")
		cl_, err := mongo.GetClient()
		if err != nil {
			ll.WithError(err).Error("error get client")
			g.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		defer cl_.Disconnect(g)
		// ...
		jobList := []db.JobDoc{}
		if err := medMongo.DocGetAll(g, &jobList, cl_); err != nil {
			ll.WithError(err).Error("error get docs")
			g.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		response := listJobRes{
			Job: jobList,
		}
		g.JSON(http.StatusOK, response)
	}
}
func createJobsHandlerFactory(mongo *db.Mongo) func(g *gin.Context) {
	medMongo := mongo.GetJobMongo()
	return func(g *gin.Context) {
		ll := logrus.WithField("handler", "createJobsHandler")
		var crJobReq createJobReq
		if err := g.ShouldBind(&crJobReq); err != nil {
			g.JSON(http.StatusBadRequest, gin.H{"msg": err.Error()})
			return
		}
		cl_, err := mongo.GetClient()
		if err != nil {
			ll.WithError(err).Error("error get client")
			g.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		go func() {
			ctx := context.Background()
			defer cl_.Disconnect(ctx)
			for _, j := range crJobReq.Job {
				if _, err := medMongo.DocAdd(ctx, j, cl_); err != nil {
					ll.WithError(err).Error("can not add job to db")
				}
			}
		}()

		g.AbortWithStatus(http.StatusOK)
	}
}
func putJobResultHandlerFactory(mongo *db.Mongo, minio *db.MinioClient) func(g *gin.Context) {
	jobMongo := mongo.GetJobMongo()
	medMongo := mongo.GetMediaMongo()
	return func(g *gin.Context) {
		ll := logrus.WithField("handler", "putJobResult")
		var jobResReq putJobResultReq
		if err := g.ShouldBindUri(&jobResReq); err != nil {
			g.AbortWithError(http.StatusBadRequest, err)
			return
		}
		cl_, err := mongo.GetClient()
		if err != nil {
			ll.WithError(err).Error("error get client")
			g.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		defer cl_.Disconnect(g)
		var jobDoc db.JobDoc
		var medDoc db.MediaFileDoc
		if err := jobMongo.DocGetById(g, jobResReq.ID, &jobDoc, cl_); err != nil {
			var stat int
			if err == mongoD.ErrNoDocuments {
				stat = http.StatusBadRequest
			} else {
				ll.WithError(err).Error("error job get doc")
				stat = http.StatusInternalServerError
			}
			g.AbortWithError(stat, err)
			return
		}
		if err := medMongo.DocGetById(g, jobDoc.MediaID, &medDoc, cl_); err != nil {
			var stat int
			if err == mongoD.ErrNoDocuments {
				stat = http.StatusBadRequest
				if err := jobMongo.DocDelById(g, jobDoc.ID, cl_); err != nil {
					ll.WithError(err).Error("can not remove job from mongo")
				}
			} else {
				ll.WithError(err).Error("error get media doc")
				stat = http.StatusInternalServerError
			}
			g.AbortWithError(stat, err)
			return
		}
		if jobResReq.Status == -1 {
			logrus.WithField("job", jobDoc).Warn("not success")
		} else {
			switch jobDoc.Type {
			case db.THUMBNAILJobType:
				file, err := g.FormFile("file")
				if err != nil {
					ll.WithError(err).Error("can not get form file")
					g.AbortWithError(http.StatusInternalServerError, err)
					return
				}
				f, err := file.Open()
				if err != nil {
					ll.WithError(err).Error("can not open form file")
					g.AbortWithError(http.StatusInternalServerError, err)
					return
				}
				data, err := io.ReadAll(f)
				if err != nil {
					ll.WithError(err).Error("can not read form file")
					g.AbortWithError(http.StatusInternalServerError, err)
					return
				}
				if err := helper.UpdateMediaThumbnail(g, medMongo, minio, data, &medDoc, cl_); err != nil {
					ll.WithError(err).Error("can not replace thumbnail in db")
					g.AbortWithError(http.StatusInternalServerError, err)
					return
				}
			}
		}
		if err := jobMongo.DocDelById(g, jobDoc.ID, cl_); err != nil {
			ll.WithError(err).Error("can not remove job from db")
			g.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		g.AbortWithStatus(http.StatusOK)
	}
}
