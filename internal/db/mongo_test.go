package db_test

import (
	"context"
	"fmt"

	"github.com/amirdaaee/TGMon/internal/db"
	"github.com/amirdaaee/TGMon/internal/errs"
	mockDB "github.com/amirdaaee/TGMon/mocks/db"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var _ = Describe("Mongo", func() {
	var (
		mockMongoColl   *mockDB.MockIMongoCollection
		mockMongoClient *mockDB.MockIMongoClient
		mockMongoDoc    *mockDB.MockIMongoDoc
		testContext     context.Context
	)
	resetMock := func() {
		mockMongoColl = mockDB.NewMockIMongoCollection(GinkgoT())
		mockMongoClient = mockDB.NewMockIMongoClient(GinkgoT())
		mockMongoDoc = mockDB.NewMockIMongoDoc(GinkgoT())
	}
	asserMockCall := func() {
		mockMongoColl.AssertExpectations(GinkgoT())
		mockMongoClient.AssertExpectations(GinkgoT())
		mockMongoDoc.AssertExpectations(GinkgoT())
	}
	// ...
	BeforeEach(func() {
		testContext = context.Background()
	})
	// ================================
	Describe("DataStore", Label("DataStore"), func() {
		newDataStore := func() db.IDataStore[*mockDB.MockIMongoDoc] {
			ds := db.NewDatastore[*mockDB.MockIMongoDoc]("testDB", "testColl")
			return ds.WithCollectionFactory(func(ic db.IMongoClient) db.IMongoCollection {
				return mockMongoColl
			})
		}
		Describe("Create", Label("Create"), func() {
			type testCase struct {
				description            string
				tType                  TestCaseType
				createdID              primitive.ObjectID // objectID to assign to newly created doc
				expectDocSetID         bool               // whether or not to expect calling doc.SetID
				expectCollInsertOneErr error              // error to return by coll.InsertOne
				expectErr              error              // error to return by ds.Create
			}
			// ...
			BeforeEach(func() {
				resetMock()
			})

			AfterEach(func() {
				asserMockCall()
			})
			// ...
			assertMongoColl_InsertOne := func(tc testCase) {
				mockMongoColl.EXPECT().InsertOne(mock.Anything, mock.Anything).RunAndReturn(
					func(ctx context.Context, i interface{}, ioo ...*options.InsertOneOptions) (*mongo.InsertOneResult, error) {
						Expect(mockMongoDoc).To(Equal(i))
						if tc.expectCollInsertOneErr == nil {
							return &mongo.InsertOneResult{
								InsertedID: tc.createdID,
							}, nil
						} else {
							return nil, tc.expectCollInsertOneErr
						}
					},
				)
			}
			assertMongoDoc_SetID := func(tc testCase) {
				if tc.expectDocSetID {
					mockMongoDoc.EXPECT().SetID(tc.createdID).RunAndReturn(func(oi primitive.ObjectID) {
						Expect(oi).To(Equal(tc.createdID))
					})
				}
			}
			// ...
			tests := []testCase{
				{
					description:    "successfully create doc",
					tType:          HAPPY_PATH,
					expectDocSetID: true,
				},
				{
					description:            "Failure on mongo.InsertOne error",
					tType:                  FAILURE,
					expectCollInsertOneErr: fmt.Errorf("mock mongo.InsertOne err"),
					expectErr:              errs.MongoOpErr{},
				},
			}
			// ...
			for _, tc := range tests {
				tc := tc
				It(tc.description, Label(string(tc.tType)), func() {
					// Arrange
					ds := newDataStore()
					assertMongoColl_InsertOne(tc)
					assertMongoDoc_SetID(tc)
					// Act
					res, err := ds.Create(testContext, mockMongoDoc, mockMongoClient)
					// Assert
					if tc.expectErr == nil {
						Expect(err).NotTo(HaveOccurred())
						Expect(res).NotTo(BeNil())
						Expect(res).To(Equal(mockMongoDoc))
					} else {
						Expect(err).To(HaveOccurred())
						Expect(err).To(BeAssignableToTypeOf(tc.expectErr))
					}
				})
			}
		})
		// TODO: ds.FindMany test
		Describe("Find", Label("Find"), func() {
			type testCase struct {
				description            string
				tType                  TestCaseType
				filter                 bson.D
				expectCollFindOneCall  bool         // call to coll.FindOne is expected
				expectCollFindOneDoc   db.IMongoDoc // doc to return by coll.FindOne
				expectCollFindOneErr   error        // error to return by coll.FindOne
				expectCollCountDocsDoc int64        // number of docs matching filter
				expectCollCountDocsErr error        // error to return by coll.CountDocuments
				expectErr              error        // error to return by ds.Find
			}
			// ...
			BeforeEach(func() {
				resetMock()
			})
			AfterEach(func() {
				asserMockCall()
			})
			// ...
			assertMongoColl_FindOne := func(tc testCase) {
				if !tc.expectCollFindOneCall {
					return
				}
				mockMongoColl.EXPECT().FindOne(mock.Anything, mock.Anything).RunAndReturn(
					func(ctx context.Context, i interface{}, foo ...*options.FindOneOptions) *mongo.SingleResult {
						Expect(i).To(BeEquivalentTo(&tc.filter))
						return mongo.NewSingleResultFromDocument(tc.expectCollFindOneDoc, tc.expectCollFindOneErr, nil)
					},
				)
			}
			assertMongoColl_CountDocuments := func(tc testCase) {
				mockMongoColl.EXPECT().CountDocuments(mock.Anything, mock.Anything).RunAndReturn(
					func(ctx context.Context, i interface{}, co ...*options.CountOptions) (int64, error) {
						Expect(i).To(BeEquivalentTo(&tc.filter))
						return tc.expectCollCountDocsDoc, tc.expectCollCountDocsErr
					},
				)
			}
			// ...
			tests := []testCase{
				{
					description:            "successfully find doc",
					tType:                  HAPPY_PATH,
					filter:                 bson.D{{Key: "hello", Value: "world"}},
					expectCollFindOneCall:  true,
					expectCollFindOneDoc:   &mockDB.MockIMongoDoc{},
					expectCollCountDocsDoc: 1,
				},
				{
					description:            "error not found",
					tType:                  FAILURE,
					filter:                 bson.D{{Key: "hello", Value: "world"}},
					expectCollCountDocsDoc: 0,
					expectErr:              errs.MongoObjectNotfound{},
				},
				{
					description:            "error multiple found",
					tType:                  FAILURE,
					filter:                 bson.D{{Key: "hello", Value: "world"}},
					expectCollCountDocsDoc: 2,
					expectErr:              errs.MongoMultipleObjectfound{},
				},
				{
					description:            "error count document",
					tType:                  FAILURE,
					filter:                 bson.D{{Key: "hello", Value: "world"}},
					expectCollCountDocsErr: fmt.Errorf("mock coll.CountDocs err"),
					expectCollCountDocsDoc: 1,
					expectErr:              errs.MongoOpErr{},
				},
				{
					description:            "error findOne document",
					tType:                  FAILURE,
					filter:                 bson.D{{Key: "hello", Value: "world"}},
					expectCollFindOneCall:  true,
					expectCollFindOneErr:   fmt.Errorf("mock coll.findOne err"),
					expectCollCountDocsDoc: 1,
					expectErr:              errs.MongoOpErr{},
				},
			}
			// ...
			for _, tc := range tests {
				tc := tc
				It(tc.description, Label(string(tc.tType)), func() {
					// Arrange
					ds := newDataStore()
					assertMongoColl_FindOne(tc)
					assertMongoColl_CountDocuments(tc)
					// Act
					res, err := ds.Find(testContext, &tc.filter, mockMongoClient)
					// Assert
					if tc.expectErr == nil {
						Expect(err).NotTo(HaveOccurred())
						Expect(res).NotTo(BeNil())
						Expect(res).To(Equal(tc.expectCollFindOneDoc))
					} else {
						Expect(err).To(HaveOccurred())
						Expect(err).To(BeAssignableToTypeOf(tc.expectErr))
					}
				})
			}
		})
	})
})
