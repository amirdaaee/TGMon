package facade_test

import (
	"context"
	"fmt"
	"time"

	"github.com/amirdaaee/TGMon/internal/facade"
	mDb "github.com/amirdaaee/TGMon/mocks/db"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	mMongo "github.com/amirdaaee/TGMon/mocks/db/mongo"
	mFacade "github.com/amirdaaee/TGMon/mocks/facade"
	mMongoX "github.com/chenmingyong0423/go-mongox/v2/mock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("BaseFacade", func() {
	type testDoc struct{}
	var (
		mockMongoContainer *mMongo.MockIMongoContainer
		mockCreator        *mMongoX.MockICreator[testDoc]
		mockDeleter        *mMongoX.MockIDeleter[testDoc]
		mockFinder         *mMongoX.MockIFinder[testDoc]
		mockCrud           *mFacade.MockICrud[testDoc]
		mockContainer      *mDb.MockIDbContainer
		mockCollection     *mMongo.MockICollection[testDoc]
		testContext        context.Context
		ctrl               *gomock.Controller
		fac                facade.IFacade[testDoc]
		tDoc               *testDoc
		testQ              bson.D
	)
	// ...
	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		testContext = context.Background()
		tDoc = &testDoc{}
		testQ = bson.D{}
		// ...
		mockCreator = mMongoX.NewMockICreator[testDoc](ctrl)
		mockDeleter = mMongoX.NewMockIDeleter[testDoc](ctrl)
		mockFinder = mMongoX.NewMockIFinder[testDoc](ctrl)
		mockCollection = mMongo.NewMockICollection[testDoc](ctrl)
		mockCollection.EXPECT().Creator().Return(mockCreator).AnyTimes()
		mockCollection.EXPECT().Deleter().Return(mockDeleter).AnyTimes()
		mockCollection.EXPECT().Finder().Return(mockFinder).AnyTimes()
		// ...
		mockCrud = mFacade.NewMockICrud[testDoc](ctrl)
		mockCrud.EXPECT().GetCollection().Return(mockCollection).AnyTimes()
		// ...
		mockMongoContainer = mMongo.NewMockIMongoContainer(ctrl)
		mockContainer = mDb.NewMockIDbContainer(ctrl)
		mockContainer.EXPECT().GetMongoContainer().Return(mockMongoContainer).AnyTimes()
	})
	Describe("CreateOne", func() {
		type testCase struct {
			nilDoc        bool
			expectPreErr  bool
			expectPostErr bool
			expectDbErr   bool
			expectErr     bool
			preCall       bool
			dbCall        bool
			postCall      bool
		}
		DescribeTable("", func(tc testCase) {
			ctx := testContext
			doc := tDoc
			if tc.nilDoc {
				doc = nil
			}
			if tc.dbCall {
				mockCreator.EXPECT().InsertOne(ctx, gomock.AssignableToTypeOf(doc)).DoAndReturn(func(ctx context.Context, doc any, _ ...any) (*mongo.InsertOneResult, error) {
					if tc.expectDbErr {
						return nil, fmt.Errorf("mock insert error")
					}
					return &mongo.InsertOneResult{}, nil
				})
			}
			if tc.preCall {
				mockCrud.EXPECT().PreCreate(ctx, gomock.Any()).DoAndReturn(func(ctx context.Context, doc *testDoc) error {
					if tc.expectPreErr {
						return fmt.Errorf("mock pre create error")
					}
					return nil
				})
			}
			postCalled := make(chan struct{})
			if tc.postCall {
				mockCrud.EXPECT().PostCreate(ctx, gomock.Any()).DoAndReturn(func(ctx context.Context, doc *testDoc) error {
					defer close(postCalled)
					if tc.expectPostErr {
						return fmt.Errorf("mock post create error")
					}
					return nil
				})
			} else {
				close(postCalled)
			}
			fac = facade.NewFacade(mockCrud)
			var res any
			var err error
			res, err = fac.CreateOne(ctx, doc)
			if tc.expectErr {
				Expect(err).To(HaveOccurred())
				Expect(res).To(BeNil())
			} else {
				Expect(err).ToNot(HaveOccurred())
				Expect(res).ToNot(BeNil())
			}
			Eventually(postCalled).WithTimeout(1 * time.Second).Should(BeClosed())
		},
			Entry("should create a document", testCase{
				dbCall:   true,
				postCall: true,
				preCall:  true,
			}),
			Entry("should not create a document with pre create error", testCase{
				expectPreErr: true,
				expectErr:    true,
				preCall:      true,
			}),
			Entry("should not create a document with insert error", testCase{
				dbCall:      true,
				expectDbErr: true,
				expectErr:   true,
				preCall:     true,
			}),
			Entry("should create a document with post create error", testCase{
				dbCall:        true,
				postCall:      true,
				expectPostErr: true,
				preCall:       true,
			}),
			Entry("should not create with nil document", testCase{
				nilDoc:    true,
				expectErr: true,
			}),
		)
	})
	Describe("DeleteOne", func() {
		type testCase struct {
			nilQuery       bool
			findOneErr     bool
			expectPreErr   bool
			expectPostErr  bool
			expectDbErr    bool
			expectCountErr bool
			expectErr      bool
			dbCall         bool
			countCall      bool
			findCall       bool
			preCall        bool
			postCall       bool
			count          int64
		}
		DescribeTable("", func(tc testCase) {
			ctx := testContext
			query := testQ
			if tc.nilQuery {
				query = nil
			}
			if tc.countCall {
				mockFinder.EXPECT().Filter(gomock.AssignableToTypeOf(query)).Return(mockFinder)
				mockFinder.EXPECT().Count(ctx).DoAndReturn(func(ctx context.Context, _ ...any) (int64, error) {
					if tc.expectCountErr {
						return 0, fmt.Errorf("mock count error")
					}
					return tc.count, nil
				})
			}
			if tc.findCall {
				if tc.findOneErr {
					mockFinder.EXPECT().FindOne(ctx).Return(nil, fmt.Errorf("mock findOne error"))
				} else {
					mockFinder.EXPECT().FindOne(ctx).Return(tDoc, nil)
				}
			}
			if tc.dbCall {
				mockDeleter.EXPECT().Filter(gomock.AssignableToTypeOf(query)).Return(mockDeleter)
				mockDeleter.EXPECT().DeleteOne(ctx).DoAndReturn(func(ctx context.Context, _ ...any) (*mongo.DeleteResult, error) {
					if tc.expectDbErr {
						return nil, fmt.Errorf("mock delete error")
					}
					return &mongo.DeleteResult{}, nil
				})
			}
			if tc.preCall {
				mockCrud.EXPECT().PreDelete(ctx, gomock.Any()).DoAndReturn(func(ctx context.Context, doc *testDoc) error {
					if tc.expectPreErr {
						return fmt.Errorf("mock pre delete error")
					}
					return nil
				})
			}
			postCalled := make(chan struct{})
			if tc.postCall {
				mockCrud.EXPECT().PostDelete(ctx, gomock.Any()).DoAndReturn(func(ctx context.Context, doc *testDoc) error {
					defer close(postCalled)
					if tc.expectPostErr {
						return fmt.Errorf("mock post delete error")
					}
					return nil
				})
			} else {
				close(postCalled)
			}
			fac = facade.NewFacade(mockCrud)
			var res any
			var err error
			res, err = fac.DeleteOne(ctx, query)
			if tc.expectErr {
				Expect(err).To(HaveOccurred())
			} else {
				Expect(err).ToNot(HaveOccurred())
				Expect(res).ToNot(BeNil())
			}
			Eventually(postCalled).WithTimeout(1 * time.Second).Should(BeClosed())
		},
			Entry("should delete a signle document", testCase{
				countCall: true,
				findCall:  true,
				preCall:   true,
				dbCall:    true,
				postCall:  true,
				count:     1,
			}),
			Entry("should not delete multiple documents", testCase{
				countCall: true,
				findCall:  false,
				preCall:   false,
				dbCall:    false,
				postCall:  false,
				count:     2,
				expectErr: true,
			}),
			Entry("should not delete zero documents", testCase{
				countCall: true,
				findCall:  false,
				preCall:   false,
				dbCall:    false,
				postCall:  false,
				count:     0,
				expectErr: true,
			}),
			Entry("should not delete with count error", testCase{
				countCall:      true,
				expectCountErr: true,
				findCall:       false,
				preCall:        false,
				dbCall:         false,
				postCall:       false,
				count:          1,
				expectErr:      true,
			}),
			Entry("should not delete with pre error", testCase{
				countCall:    true,
				findCall:     true,
				preCall:      true,
				dbCall:       false,
				postCall:     false,
				count:        1,
				expectPreErr: true,
				expectErr:    true,
			}),
			Entry("should not delete with delete error", testCase{
				countCall:   true,
				findCall:    true,
				preCall:     true,
				dbCall:      true,
				postCall:    false,
				count:       1,
				expectDbErr: true,
				expectErr:   true,
			}),
			Entry("should delete with post error", testCase{
				countCall:     true,
				findCall:      true,
				preCall:       true,
				dbCall:        true,
				postCall:      true,
				count:         1,
				expectPostErr: true,
				expectErr:     false,
			}),
			Entry("should not delete with nil query", testCase{
				countCall: true,
				findCall:  false,
				preCall:   false,
				dbCall:    false,
				postCall:  false,
				nilQuery:  true,
				expectErr: true,
			}),
			Entry("should not delete with FindOne returning error", testCase{
				countCall:  true,
				findCall:   true,
				findOneErr: true,
				preCall:    false,
				dbCall:     false,
				postCall:   false,
				count:      1,
				expectErr:  true,
			}),
		)
	})
})
