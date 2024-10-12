package db

import (
	"github.com/gotd/td/tg"
)

type MediaFileDoc struct {
	ID           string                        `bson:"_id,omitempty"`
	Location     *tg.InputDocumentFileLocation `bson:"Location"`
	FileSize     int64                         `bson:"FileSize"`
	FileName     string                        `bson:"FileName"`
	MimeType     string                        `bson:"MimeType"`
	FileID       int64                         `bson:"FileID"`
	MessageID    int                           `bson:"MessageID"`
	Thumbnail    string                        `bson:"Thumbnail"`
	DateAdded    int64                         `bson:"DateAdded"`
	Duration     float64                       `bson:"Duration"`
	VttFileName  string                        `bson:"VttFileName"`
	VttImageName string                        `bson:"VttImageName"`
}
type JobType string

const (
	THUMBNAILJobType JobType = "THUMBNAIL"
	SPRITEJobType    JobType = "SPRITE"
)

type JobDoc struct {
	ID      string  `bson:"_id,omitempty"`
	MediaID string  `bson:"MediaID" json:"mediaID"`
	Type    JobType `bson:"JobType" json:"type"`
}
