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
	})
})
