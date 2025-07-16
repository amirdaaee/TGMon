package types

import (
	"github.com/chenmingyong0423/go-mongox/v2"
	"github.com/gotd/td/tg"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// ...
const (
	MediaFileDoc__VttField       = "Vtt"
	MediaFileDoc__SpriteField    = "Sprite"
	MediaFileDoc__ThumbnailField = "Thumbnail"
	MediaFileDoc__FileIDField    = "Meta.FileID"
)

type MediaFileMeta struct {
	FileSize int64  `bson:"FileSize"`
	FileName string `bson:"FileName"`
	MimeType string `bson:"MimeType"`
	FileID   int64  `bson:"FileID"`
}
type MediaFileDoc struct {
	mongox.Model `bson:",inline"`
	Meta         MediaFileMeta `bson:"Meta"`
	MessageID    int           `bson:"MessageID"`
	Thumbnail    string        `bson:"Thumbnail"`
	Vtt          string        `bson:"Vtt"`
	Sprite       string        `bson:"Sprite"`
}

func (m MediaFileDoc) String() string {
	return m.ID.String()
}

func (m *MediaFileMeta) FillFromDocument(doc *tg.Document) error {
	for _, attr := range doc.Attributes {
		switch v := attr.(type) {
		case *tg.DocumentAttributeFilename:
			m.FileName = v.FileName
		}
	}
	m.FileSize = doc.Size
	m.MimeType = doc.MimeType
	m.FileID = doc.ID
	return nil
}

// ...
type JobTypeEnum string

const (
	THUMBNAILJobType JobTypeEnum = "THUMBNAIL"
	SPRITEJobType    JobTypeEnum = "SPRITE"
)
const (
	JobReqDoc__MediaIDField = "mediaID"
)

type JobReqDoc struct {
	mongox.Model `bson:",inline"`
	MediaID      bson.ObjectID `bson:"MediaID" json:"mediaID"`
	Type         JobTypeEnum   `bson:"JobType" json:"type"`
}

func (m JobReqDoc) String() string {
	return m.ID.String()
}

type JobResDoc struct {
	mongox.Model `bson:",inline"`
	JobReqID     bson.ObjectID `bson:"JobReqID" json:"jobReqID"`
	Thumbnail    []byte        `bson:"-"`
	Sprite       []byte        `bson:"-"`
	Vtt          []byte        `bson:"-"`
}

func (m JobResDoc) String() string {
	return m.ID.String()
}
