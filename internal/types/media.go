package types

import (
	"fmt"
	"strings"
	"time"

	"github.com/chenmingyong0423/go-mongox/v2"
	"github.com/gotd/td/tg"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// ...
const (
	MediaFileDoc__SrtField       = "Srt"
	MediaFileDoc__VttField       = "Vtt"
	MediaFileDoc__SpriteField    = "Sprite"
	MediaFileDoc__ThumbnailField = "Thumbnail"
	MediaFileDoc__FileIDField    = "Meta.FileID"
	MediaFileDoc__UnameField     = "UName"
	MediaFileDoc__HasHashField   = "HasHash"
)

const (
	MediaExtendedMeta__MediaFileIDField  = "MediaFileID"
	MediaExtendedMeta__LastPlayedAtField = "LastPlayedAt"
	MediaExtendedMeta__CheckpointField   = "Checkpoint"
	MediaExtendedMeta__PlayCountField    = "PlayCount"
	MediaExtendedMeta__ScoreField        = "Score"
	MediaExtendedMeta__LikesField        = "Likes"
	MediaExtendedMeta__IsFavoriteField   = "IsFavorite"
)

type MediaFileMeta struct {
	FileSize int64   `bson:"FileSize"`
	FileName string  `bson:"FileName"`
	MimeType string  `bson:"MimeType"`
	FileID   int64   `bson:"FileID"`
	Duration float64 `bson:"Duration"`
}

type MediaFileDoc struct {
	mongox.Model `bson:",inline"`
	Meta         MediaFileMeta `bson:"Meta"`
	MessageID    int           `bson:"MessageID"`
	Thumbnail    string        `bson:"Thumbnail"`
	Vtt          string        `bson:"Vtt"`
	Sprite       string        `bson:"Sprite"`
	Srt          string        `bson:"Srt"`
	HasHash      bool          `bson:"HasHash"`
	UName        string        `bson:"UName"`
}

func (m MediaFileDoc) String() string {
	return m.ID.String()
}
func (m MediaFileDoc) Name() string {
	if m.UName != "" {
		return m.UName
	}
	baseName := m.Meta.FileName
	if baseName == "" {
		baseName = m.ID.Hex()
	}
	baseName = RemoveExtension(baseName)
	idStr := m.ID.Hex()
	idStr = idStr[len(idStr)-8:]
	return fmt.Sprintf("%s-%s", baseName, idStr)
}
func (m MediaFileDoc) NameExt() string {
	baseName := m.Name()
	return fmt.Sprintf("%s%s", baseName, GetExtensionFromMimeType(m.Meta.MimeType))
}

func (m *MediaFileMeta) FillFromDocument(doc *tg.Document) error {
	for _, attr := range doc.Attributes {
		switch v := attr.(type) {
		case *tg.DocumentAttributeFilename:
			m.FileName = v.FileName
		case *tg.DocumentAttributeVideo:
			m.Duration = v.Duration
		}
	}
	m.FileSize = doc.Size
	m.MimeType = doc.MimeType
	m.FileID = doc.ID
	return nil
}

type MediaExtendedMeta struct {
	mongox.Model `bson:",inline"`
	MediaFileID  bson.ObjectID `bson:"MediaFileID"`
	LastPlayedAt time.Time     `bson:"LastPlayedAt"`
	Checkpoint   uint64        `bson:"Checkpoint"`
	PlayCount    int           `bson:"PlayCount"`
	Score        int           `bson:"Score"`
	Likes        int           `bson:"Likes"`
	IsFavorite   bool          `bson:"IsFavorite"`
}

// MediaExtendedMetaPatch holds optional fields for a partial update. Nil means leave unchanged.
type MediaExtendedMetaPatch struct {
	LastPlayedAt *time.Time
	PlayCount    *int
	Checkpoint   *uint64 //!
	Score        *int
	Likes        *int
	IsFavorite   *bool
}

func RemoveExtension(fileName string) string {
	exts := []string{".mp4", ".webm", ".mkv", ".mov", ".mp3", ".ogg", ".weba", ".jpg", ".png", ".gif", ".bin"}
	for _, ext := range exts {
		if strings.HasSuffix(fileName, ext) {
			return fileName[:len(fileName)-len(ext)]
		}
	}
	return fileName
}

// GetExtensionFromMimeType returns a file extension based on mime type
func GetExtensionFromMimeType(mimeType string) string {
	switch mimeType {
	case "video/mp4":
		return ".mp4"
	case "video/webm":
		return ".webm"
	case "video/x-matroska":
		return ".mkv"
	case "video/quicktime":
		return ".mov"
	case "audio/mpeg":
		return ".mp3"
	case "audio/ogg":
		return ".ogg"
	case "audio/webm":
		return ".weba"
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/gif":
		return ".gif"
	default:
		return ".bin"
	}
}
