package bot

import (
	"context"

	"github.com/amirdaaee/TGMon/config"
	"github.com/amirdaaee/TGMon/internal/db"
	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/ext"
	"go.uber.org/zap"
)

func addMediaDB(ctx *ext.Context, u *ext.Update) error {
	l := zap.S()
	l.Debug("new message")
	chatId := u.EffectiveChat().GetID()
	effMsg := u.EffectiveMessage
	if chatId != config.Config().ChannelID {
		l.Debug("message not in channel")
		return dispatcher.EndGroups
	}
	supported, err := SupportedMediaFilter(effMsg)
	if err != nil {
		return err
	}
	if !supported {
		l.Debug("message not supported")
		return dispatcher.EndGroups
	}
	// ...
	l.Debug("adding media to DB")
	dbDoc, err := FileDocFromMessage(effMsg)
	if err != nil {
		return err
	}
	coll, cl_, err := db.GetFileCollection()
	if err != nil {
		return err
	}
	defer cl_.Disconnect(context.TODO())
	_, err = db.AddDoc(ctx, coll, dbDoc)
	if err != nil {
		return err
	}
	l.Debug("file added to DB")
	return nil
	// TODO: dedup
}
