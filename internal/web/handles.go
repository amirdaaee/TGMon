package web

import (
	"context"
	"net/http"
	"strings"

	"github.com/amirdaaee/TGMon/internal/bot"
	"github.com/amirdaaee/TGMon/internal/db"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func streamHandlerFactory(wp *bot.WorkerPool, mongo *db.Mongo, chunckSize int64, profileFile string) func(g *gin.Context) {
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
		err := steam(g, media, wp, mongo, chunckSize, profileFile)
		if err != nil {
			g.AbortWithError(http.StatusBadRequest, err)
			return
		}
	}
}
func listMediaHandlerFactory(mongo *db.Mongo) func(g *gin.Context) {
	return func(g *gin.Context) {
		var listReq mediaListReq
		if err := g.ShouldBind(&listReq); err != nil {
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
		coll_ := mongo.GetFileCollection(cl_)
		count_, err := coll_.CountDocuments(g, bson.D{})
		if err != nil {
			g.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		opts := options.Find().SetSort(bson.D{{Key: "DateAdded", Value: -1}, {Key: "FileID", Value: 1}})
		if listReq.PageSize > 0 {
			opts = opts.SetLimit(int64(listReq.PageSize))
			opts = opts.SetSkip(int64(listReq.PageSize * (listReq.Page - 1)))
		}
		mediaList := []db.MediaFileDoc{}
		if err := mongo.DocGetAll(g, &mediaList, cl_, opts); err != nil {
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
		if err := mongo.DocGetById(g, mediaReq.ID, &mediaDoc, cl_); err != nil {
			g.AbortWithError(http.StatusBadRequest, err)
			return
		}
		resData.Media = mediaDoc
		// ...
		backMediaDoc, nextMediaDoc, err := mongo.DocGetNeighbour(g, mediaDoc, cl_)
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
		if err := mongo.DocGetById(ctx, mediaReq.ID, &mediaDoc, cl_); err != nil {
			g.AbortWithError(http.StatusBadRequest, err)
			return
		}
		go wp.GetNextWorker().DeleteMessages([]int{mediaDoc.MessageID})
		go minio.FileRm(mediaDoc.Thumbnail, ctx)
		if err := mongo.DocDelById(g, mediaDoc.ID, cl_); err != nil {
			g.AbortWithError(http.StatusInternalServerError, err)
			return
		}
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
