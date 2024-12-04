package db

import (
	"reflect"

	"github.com/amirdaaee/TGMon/internal/errs"
	"github.com/gotd/td/tg"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type IMongoDoc interface {
	GetID() primitive.ObjectID
	SetID(primitive.ObjectID)
	GetIDStr() string
	SetIDStr(string) error
	MarshalOmitEmpty() (*primitive.M, errs.IMongoErr)
}
type baseMongoDoc struct {
	ID primitive.ObjectID `bson:"_id,omitempty"`
}

func (d *baseMongoDoc) GetID() primitive.ObjectID {
	return getID(d)
}
func (d *baseMongoDoc) GetIDStr() string {
	return getIDStr(d)
}
func (d *baseMongoDoc) SetID(id primitive.ObjectID) {
	setID(d, id)
}
func (d *baseMongoDoc) SetIDStr(id string) error {
	return setIDStr(d, id)
}
func (d *baseMongoDoc) MarshalOmitEmpty() (*primitive.M, errs.IMongoErr) {
	marsh, err := bson.Marshal(d)
	if err != nil {
		return nil, errs.NewMongoMarshalErr(err)
	}
	unmarsh := new(bson.M)
	if err := bson.Unmarshal(marsh, unmarsh); err != nil {
		return nil, errs.NewMongoUnMarshalErr(err)
	}
	res := bson.M{}
	for k, v := range *unmarsh {
		if !reflect.ValueOf(v).IsZero() {
			res[k] = v
		}
	}
	return &res, nil
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
	MediaID      primitive.ObjectID `bson:"MediaID" json:"mediaID"`
	Type         JobType            `bson:"JobType" json:"type"`
}

func getID(v *baseMongoDoc) primitive.ObjectID {
	return v.ID
}
func getIDStr(v *baseMongoDoc) string {
	return v.ID.Hex()
}

func setID(v *baseMongoDoc, id primitive.ObjectID) {
	v.ID = id
}
func setIDStr(v *baseMongoDoc, id string) error {
	idObj, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errs.NewMongoUnMarshalErr(err)
	}
	v.ID = idObj
	return nil
}
