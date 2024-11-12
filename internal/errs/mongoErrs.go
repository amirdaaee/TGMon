package errs

import (
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
)

type IMongoErr interface {
	error
}
type baseMongoErr struct {
	message string
	txt     string
}

func (e *baseMongoErr) Error() string {
	if e.txt != "" {
		return fmt.Sprintf("%s: %s", e.message, e.txt)
	} else {
		return e.message
	}
}

type MongoClientErr struct{ *baseMongoErr }
type MongoOpErr struct{ *baseMongoErr }
type MongoMarshalErr struct{ *baseMongoErr }
type MongoUnMarshalErr struct{ *baseMongoErr }
type MongoObjectNotfound struct{ *baseMongoErr }

func NewMongoClientErr(e error) IMongoErr {
	return MongoClientErr{&baseMongoErr{message: "error getting mongo client", txt: e.Error()}}
}
func NewMongoOpErr(e error) IMongoErr {
	return MongoClientErr{&baseMongoErr{message: "error performain mongo action", txt: e.Error()}}
}
func NewMongoMarshalErr(e error) IMongoErr {
	return MongoClientErr{&baseMongoErr{message: "error marshaling mongo", txt: e.Error()}}
}
func NewMongoUnMarshalErr(e error) IMongoErr {
	return MongoClientErr{&baseMongoErr{message: "error unmarshaling mongo", txt: e.Error()}}
}
func NewMongoObjectNotfound(q bson.D) IMongoErr {
	return MongoClientErr{&baseMongoErr{message: "requested object not found", txt: fmt.Sprintf("filter: %+v", q)}}
}
