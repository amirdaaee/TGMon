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
type MongoMultipleObjectfound struct{ *baseMongoErr }

func NewMongoClientErr(e error) IMongoErr {
	return MongoClientErr{&baseMongoErr{message: "error getting mongo client", txt: e.Error()}}
}
func NewMongoOpErr(e error) IMongoErr {
	return MongoOpErr{&baseMongoErr{message: "error performain mongo action", txt: e.Error()}}
}
func NewMongoMarshalErr(e error) IMongoErr {
	return MongoMarshalErr{&baseMongoErr{message: "error marshaling mongo", txt: e.Error()}}
}
func NewMongoUnMarshalErr(e error) IMongoErr {
	return MongoUnMarshalErr{&baseMongoErr{message: "error unmarshaling mongo", txt: e.Error()}}
}
func NewMongoObjectNotfound(q bson.D) IMongoErr {
	return MongoObjectNotfound{&baseMongoErr{message: "requested object not found", txt: fmt.Sprintf("filter: %+v", q)}}
}
func NewMongoMultipleObjectfound(q bson.D) IMongoErr {
	return MongoMultipleObjectfound{&baseMongoErr{message: "multiple object found for query", txt: fmt.Sprintf("filter: %+v", q)}}
}
