package types

import (
	"github.com/chenmingyong0423/go-mongox/v2"
)

const (
	WorkerMediaDoc__WorkerIDField    = "WorkerID"
	WorkerMediaDoc__MessageIDField   = "MessageID"
	WorkerMediaDoc__TelegramDocField = "TelegramDoc"
)

type WorkerMediaDoc struct {
	mongox.Model `bson:",inline"`
	WorkerID     int64  `bson:"WorkerID"`
	MessageID    int    `bson:"MessageID"`
	TelegramDoc  []byte `bson:"TelegramDoc"`
}
