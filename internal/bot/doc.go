package bot

import (
	"fmt"

	"github.com/gotd/td/tg"
)

// ...
type DocumentMetadata struct {
	FileName string
	Duration float64

	DocID    int64
	FileSize int64
	MimeType string
	Location *tg.InputDocumentFileLocation
}

// ...
type TelegramDocument struct {
	*tg.Document
	MessageID int
}

func (d *TelegramDocument) FromMessage(msg tg.MessageClass) error {
	m, ok := msg.(*tg.Message)
	if !ok {
		return fmt.Errorf("provided message type is %T", msg)
	}
	media := m.Media
	switch media := media.(type) {
	case *tg.MessageMediaDocument:
		document, ok := media.Document.AsNotEmpty()
		if !ok {
			return fmt.Errorf("unexpected media.document type %T", media.Document)
		}
		d.Document = document
		d.MessageID = m.ID
		return nil
	default:
		return fmt.Errorf("unexpected media type %T", media)
	}
}
func (d *TelegramDocument) GetMetadata() *DocumentMetadata {
	meta := DocumentMetadata{}
	for _, attribute := range d.Attributes {
		switch v := attribute.(type) {
		case *tg.DocumentAttributeFilename:
			meta.FileName = v.FileName
		case *tg.DocumentAttributeVideo:
			meta.Duration = v.Duration
		}
	}

	meta.DocID = d.ID
	meta.FileSize = d.Size
	meta.MimeType = d.MimeType
	meta.Location = d.AsInputDocumentFileLocation()
	return &meta

}
