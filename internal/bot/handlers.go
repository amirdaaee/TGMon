package bot

import (
	"context"
	"fmt"
	"time"

	"github.com/amirdaaee/TGMon/config"
	"github.com/amirdaaee/TGMon/internal/db"
	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/ext"
	"github.com/celestix/gotgproto/types"
	"github.com/gotd/td/tg"
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
	dbDoc, err := fileDocFromMessage(effMsg)
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

func fileDocFromMessage(msg *types.Message) (*db.MediaFileDoc, error) {
	media := msg.Media
	switch media := media.(type) {
	case *tg.MessageMediaDocument:
		document, ok := media.Document.AsNotEmpty()
		if !ok {
			return nil, fmt.Errorf("unexpected type %T", media)
		}
		document.AsInput()
		tmbSize := document.Thumbs[0].(*tg.PhotoSize)
		tmb, err := GetThumbnail(context.Background(), document.AsInputDocumentFileLocation(), tmbSize.Type, tmbSize.Size)
		if err != nil {
			fmt.Println(err.Error())
			tmb = nil
		}
		var fileName string
		for _, attribute := range document.Attributes {
			if name, ok := attribute.(*tg.DocumentAttributeFilename); ok {
				fileName = name.FileName
			}
		}
		return &db.MediaFileDoc{
			Location:  document.AsInputDocumentFileLocation(),
			FileSize:  document.Size,
			FileName:  fileName,
			MimeType:  document.MimeType,
			FileID:    document.ID,
			MessageID: msg.ID,
			Thumbnail: tmb,
			DateAdded: time.Now().Unix(),
		}, nil
	}
	return nil, fmt.Errorf("unexpected type %T", media)
}
