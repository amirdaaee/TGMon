package web

import (
	"net/http"
	"strings"

	"github.com/amirdaaee/TGMon/config"
	"github.com/amirdaaee/TGMon/internal/bot"
	"github.com/amirdaaee/TGMon/internal/db"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type streamReq struct {
	ID string `uri:"mediaID" binding:"required"`
}
type mediaListReq struct {
	Page     int `form:"page"`
	PageSize int `form:"page_size"`
}

type mediaListRes struct {
	Media []db.MediaFileDoc
	Total int64
}
type loginReq struct {
	Username string
	Password string
}

func streamHandler(g *gin.Context) {
	var media streamReq
	if err := g.ShouldBindUri(&media); err != nil {
		g.JSON(http.StatusBadRequest, gin.H{"msg": err.Error()})
		return
	}
	media.ID = strings.TrimSuffix(media.ID, ".m3u8") //for android
	if strings.Contains(media.ID, ".") {
		g.AbortWithStatus(http.StatusNotFound)
		return
	}
	steam(g, media)
}
func listMediaHandler(g *gin.Context) {
	var listReq mediaListReq
	if err := g.ShouldBind(&listReq); err != nil {
		g.JSON(http.StatusBadRequest, gin.H{"msg": err.Error()})
		return
	}
	coll_, cl_, err := db.GetFileCollection()
	if err != nil {
		errResp(g, err)
		return
	}
	defer cl_.Disconnect(g)
	count_, err := coll_.CountDocuments(g, bson.D{})
	if err != nil {
		errResp(g, err)
		return
	}
	opts := options.Find().SetSort(bson.D{{Key: "DateAdded", Value: -1}})
	if listReq.PageSize > 0 {
		opts = opts.SetLimit(int64(listReq.PageSize))
		opts = opts.SetSkip(int64(listReq.PageSize * (listReq.Page - 1)))
	}
	cur_, err := coll_.Find(g, bson.D{}, opts)
	if err != nil {
		errResp(g, err)
		return
	}
	var mediaList []db.MediaFileDoc
	if err = cur_.All(g, &mediaList); err != nil {
		errResp(g, err)
		return
	}
	if mediaList == nil {
		mediaList = []db.MediaFileDoc{}
	}
	response := mediaListRes{
		Media: mediaList,
		Total: count_,
	}
	g.JSON(http.StatusOK, response)
}
func deleteMediaHandler(g *gin.Context) {
	var mediaReq streamReq
	if err := g.ShouldBindUri(&mediaReq); err != nil {
		g.JSON(400, gin.H{"msg": err.Error()})
		return
	}
	coll_, cl_, err := db.GetFileCollection()
	if err != nil {
		errResp(g, err)
		return
	}
	defer cl_.Disconnect(g)
	var mediaDoc db.MediaFileDoc
	if err := db.GetDocById(g, coll_, mediaReq.ID, &mediaDoc); err != nil {
		errResp(g, err)
		return
	}
	worker := bot.GetNextWorker()
	bot.DeleteMessage(worker, mediaDoc.MessageID)
	// if err := bot.DeleteMessage(worker, mediaDoc.MessageID); err != nil {
	// 	errResp(g, err)
	// 	return
	// }
	if err := db.DelDocById(g, coll_, mediaDoc.ID); err != nil {
		errResp(g, err)
		return
	}
	g.JSON(http.StatusOK, "")
}
func loginHandler(g *gin.Context) {
	var cred loginReq
	if err := g.ShouldBind(&cred); err != nil {
		g.JSON(http.StatusBadRequest, gin.H{"msg": err.Error()})
		return
	}
	if cred.Username == config.Config().UserName && cred.Password == config.Config().UserPass {
		g.JSON(http.StatusOK, map[string]string{"token": config.Config().UserToken})
		return
	}
	g.AbortWithStatus(http.StatusBadRequest)
}
func sessionHandler(g *gin.Context) {
	g.JSON(http.StatusOK, map[string]string{"token": config.Config().UserToken})
}
func errResp(g *gin.Context, err error) {
	g.JSON(400, gin.H{"msg": err.Error()})
}
