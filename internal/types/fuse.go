package types

import "github.com/chenmingyong0423/go-mongox/v2"

// ...
const (
	FuseStateDoc__RenameField = "Rename"
)

type FuseStateDoc struct {
	mongox.Model `bson:",inline"`
	SrcID        string `bson:"SrcID"`
	UID          string `bson:"UID"`
	Name         string `bson:"Name"`
	NameSuffix   int    `bson:"NameSuffix"`
	Ext          string `bson:"Ext"`
	Rename       string `bson:"Rename"`
}
