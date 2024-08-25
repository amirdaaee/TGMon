package cmd

import (
	"context"

	"github.com/amirdaaee/TGMon/config"
	"github.com/amirdaaee/TGMon/internal/bot"
	"github.com/amirdaaee/TGMon/internal/db"
	"github.com/gotd/td/tg"
	"go.uber.org/zap"
)

func UpdateThumb() {
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
	mediaList := []db.MediaFileDoc{}
	if err := db.GetDocAll(ctx, coll_, &mediaList); err != nil {
		panpan(err)
	}
	// ...
	minioCl, err := db.NewMinioClient()
	if err != nil {
		panpan(err)
	}
	// ...
	msgsIds := []tg.InputMessageClass{}
	for _, m := range mediaList {
		msgsIds = append(msgsIds, &tg.InputMessageID{ID: m.MessageID})
	}
	// ...
	worker := bot.GetNextWorker()
	msgsCls, err := worker.Client.CreateContext().GetMessages(config.Config().ChannelID, msgsIds)
	if err != nil {
		panpan(err)
	}
	for c, m := range msgsCls {
		switch msg := m.(type) {
		case *tg.Message:
			media, ok := msg.Media.(*tg.MessageMediaDocument)
			if !ok {
				ll.Sugar().Warnf("media type (%T) is not tg.MessageMediaDocument", msg.Media)
				continue
			}
			document, ok := media.Document.AsNotEmpty()
			if !ok {
				ll.Sugar().Warnf("unexpected type %T", media)
				continue
			}
			updateDoc := mediaList[c]
			if updateDoc.Thumbnail != "" {
				if err := minioCl.RmFile(updateDoc.Thumbnail, ctx); err != nil {
					ll.Sugar().Warn("can not remove old thumbnail: %s", err)
				}
				updateDoc.Thumbnail = ""
			}
			file, err := bot.StoreThumbnail(ctx, document, minioCl)
			if err != nil {
				ll.Sugar().Error(err)
				continue
			}
			updateDoc.Thumbnail = file
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
