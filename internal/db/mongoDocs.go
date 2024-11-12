package db

import (
	"fmt"

	"github.com/gotd/td/tg"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type IMongoDoc interface {
	GetID() primitive.ObjectID
	SetID(primitive.ObjectID)
	GetIDStr() string
	SetIDStr(string)
}
type baseMongoDoc struct {
	ID primitive.ObjectID `bson:"_id,omitempty"`
}

func (d *baseMongoDoc) GetID() primitive.ObjectID {
	return d.ID
}
func (d *baseMongoDoc) GetIDStr() string {
	return d.GetID().Hex()
}
func (d *baseMongoDoc) SetID(id primitive.ObjectID) {
	d.ID = id
}
func (d *baseMongoDoc) SetIDStr(id string) error {
	idObj, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return fmt.Errorf("can not convert string %s to opbject id: %s", id, err)
	}
	d.SetID(idObj)
	return nil
}

// ...
type MediaFileDoc struct {
	baseMongoDoc `bson:"inline"`
	Location     *tg.InputDocumentFileLocation `bson:"Location"`
	FileSize     int64                         `bson:"FileSize"`
	FileName     string                        `bson:"FileName"`
	MimeType     string                        `bson:"MimeType"`
	FileID       int64                         `bson:"FileID"`
	MessageID    int                           `bson:"MessageID"`
	Thumbnail    string                        `bson:"Thumbnail"`
	DateAdded    int64                         `bson:"DateAdded"`
	Duration     float64                       `bson:"Duration"`
	Vtt          string                        `bson:"Vtt"`
	Sprite       string                        `bson:"Sprite"`
}

// ...
type JobType string

const (
	THUMBNAILJobType JobType = "THUMBNAIL"
	SPRITEJobType    JobType = "SPRITE"
)

type JobDoc struct {
	baseMongoDoc `bson:"inline"`
	MediaID      string  `bson:"MediaID" json:"mediaID"`
	Type         JobType `bson:"JobType" json:"type"`
}
