package db_test

import (
	"github.com/amirdaaee/TGMon/internal/db"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var _ = Describe("MongoDocs", func() {
	Describe("MarshalOmitEmpty", Label("MarshalOmitEmpty"), func() {
		type testStruct struct {
			OID    primitive.ObjectID
			Str    string
			Number int
		}
		// ...
		type testCase struct {
			description string
			tType       TestCaseType
			input       testStruct
			output      primitive.M
			expectErr   error
		}
		sample_OID := primitive.NewObjectID()
		sample_str := "hello-mortal"
		tests := []testCase{
			{
				description: "marshal non zero OID",
				tType:       HAPPY_PATH,
				input:       testStruct{OID: sample_OID},
				output:      primitive.M{"oid": sample_OID},
			},
			{
				description: "not marshal nil OID",
				tType:       HAPPY_PATH,
				input:       testStruct{OID: primitive.NilObjectID},
				output:      primitive.M{},
			},
			{
				description: "marshal non zero str",
				tType:       HAPPY_PATH,
				input:       testStruct{Str: sample_str},
				output:      primitive.M{"str": sample_str},
			},
			{
				description: "not marshal zero str",
				tType:       HAPPY_PATH,
				input:       testStruct{Str: ""},
				output:      primitive.M{},
			},
			{
				description: "marshal non zero number",
				tType:       HAPPY_PATH,
				input:       testStruct{Number: 10},
				output:      primitive.M{"number": int32(10)},
			},
			{
				description: "marshal non zero number",
				tType:       HAPPY_PATH,
				input:       testStruct{Number: -10},
				output:      primitive.M{"number": int32(-10)},
			},
			{
				description: "not marshal zero number",
				tType:       HAPPY_PATH,
				input:       testStruct{Number: 0},
				output:      primitive.M{},
			},
		}
		// ...
		for _, tc := range tests {
			tc := tc
			It(tc.description, Label(string(tc.tType)), func() {
				// Arrange
				// Act
				res, err := db.MarshalOmitEmpty(tc.input)
				// Assert
				if tc.expectErr == nil {
					Expect(err).NotTo(HaveOccurred())
					Expect(*res).To(Equal(tc.output))
				}

			})
		}
	},
	)
})
