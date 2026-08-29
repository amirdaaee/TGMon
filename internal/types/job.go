package types

import (
	"github.com/chenmingyong0423/go-mongox/v2"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// ...
type JobTypeEnum string

const (
	THUMBNAILJobType     JobTypeEnum = "THUMBNAIL"
	SPRITEJobType        JobTypeEnum = "SPRITE"
	EmbeddingJobType     JobTypeEnum = "EMBEDDING"
	TranscriptionJobType JobTypeEnum = "TRANSCRIPTION"
)
const (
	JobReqDoc__MediaIDField = "MediaID"
)

type JobReqDoc struct {
	mongox.Model `bson:",inline"`
	MediaID      bson.ObjectID `bson:"MediaID"`
	Type         JobTypeEnum   `bson:"JobType"`
}

func (m JobReqDoc) String() string {
	return m.ID.String()
}

type JobResDoc struct {
	mongox.Model  `bson:",inline"`
	JobReqID      bson.ObjectID `bson:"JobReqID"`
	Thumbnail     []byte        `bson:"-"`
	Sprite        []byte        `bson:"-"`
	Vtt           []byte        `bson:"-"`
	Transcription []byte        `bson:"-"`
}

func (m JobResDoc) String() string {
	return m.ID.String()
}
