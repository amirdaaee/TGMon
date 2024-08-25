package bot

import (
	"context"
	"fmt"
	"time"

	"github.com/amirdaaee/TGMon/internal/db"
	"github.com/celestix/gotgproto/types"
	"github.com/gotd/td/tg"
)

func FileDocFromMessage(msg *types.Message) (*db.MediaFileDoc, error) {
	media := msg.Media
	switch media := media.(type) {
	case *tg.MessageMediaDocument:
		document, ok := media.Document.AsNotEmpty()
		if !ok {
			return nil, fmt.Errorf("unexpected type %T", media)
		}
		tmb, err := StoreThumbnail(context.Background(), document, nil)
		if err != nil {
			fmt.Println(err.Error())
			tmb = ""
		}
		dbDoc := db.MediaFileDoc{
			Location:  document.AsInputDocumentFileLocation(),
			FileID:    document.ID,
			MessageID: msg.ID,
			Thumbnail: tmb,
			DateAdded: time.Now().Unix(),
		}
		FillDocMetadata(document, &dbDoc)
		return &dbDoc, nil
	}
	return nil, fmt.Errorf("unexpected type %T", media)
}
func FillDocMetadata(doc *tg.Document, dbDoc *db.MediaFileDoc) {
	var fileName string
	var dur float64
	for _, attribute := range doc.Attributes {
		switch v := attribute.(type) {
		case *tg.DocumentAttributeFilename:
			fileName = v.FileName
		case *tg.DocumentAttributeVideo:
			dur = v.Duration
		}
	}
	dbDoc.FileSize = doc.Size
	dbDoc.FileName = fileName
	dbDoc.MimeType = doc.MimeType
	dbDoc.Duration = dur
}
