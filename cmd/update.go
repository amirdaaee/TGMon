package cmd

import (
	"context"

	"github.com/amirdaaee/TGMon/config"
	"github.com/amirdaaee/TGMon/internal/bot"
	"github.com/amirdaaee/TGMon/internal/db"
	"github.com/gotd/td/tg"
	"go.mongodb.org/mongo-driver/bson"
	"go.uber.org/zap"
)

func UpdateMeta() {
	ll := zap.L()
	_, err := bot.StartWorkers(ll)
	if err != nil {
		panpan(err)
	}
	coll_, cl_, err := db.GetFileCollection()
	if err != nil {
		panpan(err)
	}
	ctx := context.Background()
	defer cl_.Disconnect(ctx)
	cur_, err := coll_.Find(ctx, bson.D{})
	if err != nil {
		panpan(err)
	}
	var mediaList []db.MediaFileDoc
	if err = cur_.All(ctx, &mediaList); err != nil {
		panpan(err)
	}
	if mediaList == nil {
		mediaList = []db.MediaFileDoc{}
		ll.Sugar().Warn("empty media list")
	}
	// ...
	msgsIds := []tg.InputMessageClass{}
	for _, m := range mediaList {
		msgsIds = append(msgsIds, &tg.InputMessageID{ID: m.MessageID})
	}

	worker := bot.GetNextWorker()
	msgsCls, err := worker.Client.CreateContext().GetMessages(config.Config().ChannelID, msgsIds)
	if err != nil {
		panpan(err)
	}
	for c, m := range msgsCls {
		switch msg := m.(type) {
		case *tg.Message:
			updateDoc := mediaList[c]
			bot.FillDocMetadata(msg.Media.(*tg.MessageMediaDocument).Document.(*tg.Document), &updateDoc)
			if err != nil {
				panpan(err)
			}
			_filter, _ := db.FilterById(updateDoc.ID)
			updateDoc.ID = ""
			if _, err := coll_.ReplaceOne(ctx, _filter, updateDoc); err != nil {
				ll.Sugar().With("filename", mediaList[c].FileName).Error(err)
			}
			ll.Sugar().With("filename", mediaList[c].FileName).Info("updated")
		case *tg.MessageEmpty:
			ll.Sugar().With("filename", mediaList[c].FileName).Info("file is removed")
			db.DelDocById(ctx, coll_, mediaList[c].ID)
		}
	}
}
