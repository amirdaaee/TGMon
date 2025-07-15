package types

import (
	"go.mongodb.org/mongo-driver/v2/bson"
)

func ToBson(p any) (bson.D, error) {
	bsonData, err := bson.Marshal(p)
	if err != nil {
		return nil, err
	}
	var bsonMap bson.D
	err = bson.Unmarshal(bsonData, &bsonMap)
	if err != nil {
		return nil, err
	}
	return bsonMap, nil
}
