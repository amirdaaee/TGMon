package facade_test

import (
	"context"
	"fmt"
	"time"

	"github.com/amirdaaee/TGMon/internal/facade"
	ftypes "github.com/amirdaaee/TGMon/internal/facade/types"
	"github.com/amirdaaee/TGMon/internal/repository"
	mFacade "github.com/amirdaaee/TGMon/mocks/facade/types"
	mRepo "github.com/amirdaaee/TGMon/mocks/repository"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.uber.org/mock/gomock"
)

var _ = Describe("BaseFacade", func() {
	type testDoc struct{}
	var (
		mockStore   *mRepo.MockStore[testDoc]
		mockCrud    *mFacade.MockICrud[testDoc]
		testContext context.Context
		ctrl        *gomock.Controller
		fac         ftypes.IFacade[testDoc]
		tDoc        *testDoc
		testID      bson.ObjectID
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		DeferCleanup(ctrl.Finish)
		testContext = context.Background()
		tDoc = &testDoc{}
		testID = bson.NewObjectID()
		mockStore = mRepo.NewMockStore[testDoc](ctrl)
		mockCrud = mFacade.NewMockICrud[testDoc](ctrl)
		fac = facade.NewFacade(mockStore, mockCrud)
	})

	DescribeTable("CreateOne",
		func(nilDoc bool, preErr bool, insertErr bool, postErr bool, expectErr bool) {
			ctx := testContext
			doc := tDoc
			if nilDoc {
				doc = nil
			}
			if !nilDoc {
				mockCrud.EXPECT().PreCreate(ctx, gomock.Any()).DoAndReturn(func(context.Context, *testDoc) error {
					if preErr {
						return fmt.Errorf("mock pre create error")
					}
					return nil
				})
			}
			if !nilDoc && !preErr {
				mockStore.EXPECT().Insert(ctx, gomock.Any()).DoAndReturn(func(context.Context, *testDoc) error {
					if insertErr {
						return fmt.Errorf("mock insert error")
					}
					return nil
				})
			}
			postCalled := make(chan struct{})
			if !nilDoc && !preErr && !insertErr {
				mockCrud.EXPECT().PostCreate(gomock.Any(), gomock.Any()).DoAndReturn(func(context.Context, *testDoc) error {
					defer close(postCalled)
					if postErr {
						return fmt.Errorf("mock post create error")
					}
					return nil
				})
			} else {
				close(postCalled)
			}
			res, err := fac.CreateOne(ctx, doc)
			if expectErr {
				Expect(err).To(HaveOccurred())
				Expect(res).To(BeNil())
			} else {
				Expect(err).NotTo(HaveOccurred())
				Expect(res).NotTo(BeNil())
			}
			Eventually(postCalled).WithTimeout(1 * time.Second).Should(BeClosed())
		},
		Entry("creates a document", false, false, false, false, false),
		Entry("fails on pre-create error", false, true, false, false, true),
		Entry("fails on insert error", false, false, true, false, true),
		Entry("succeeds when post-create errors", false, false, false, true, false),
		Entry("fails on nil document", true, false, false, false, true),
	)

	DescribeTable("DeleteByID",
		func(findErr error, preErr bool, deleteErr bool, postErr bool, expectErr bool) {
			ctx := testContext
			if findErr != nil {
				mockStore.EXPECT().FindByID(ctx, testID).Return(nil, findErr)
			} else {
				mockStore.EXPECT().FindByID(ctx, testID).Return(tDoc, nil)
				mockCrud.EXPECT().PreDelete(ctx, gomock.Any()).DoAndReturn(func(context.Context, *testDoc) error {
					if preErr {
						return fmt.Errorf("mock pre delete error")
					}
					return nil
				})
				if !preErr {
					mockStore.EXPECT().DeleteByID(ctx, testID).DoAndReturn(func(context.Context, bson.ObjectID) error {
						if deleteErr {
							return fmt.Errorf("mock delete error")
						}
						return nil
					})
				}
			}
			postCalled := make(chan struct{})
			if findErr == nil && !preErr && !deleteErr {
				mockCrud.EXPECT().PostDelete(gomock.Any(), gomock.Any()).DoAndReturn(func(context.Context, *testDoc) error {
					defer close(postCalled)
					if postErr {
						return fmt.Errorf("mock post delete error")
					}
					return nil
				})
			} else {
				close(postCalled)
			}
			res, err := fac.DeleteByID(ctx, testID)
			if expectErr {
				Expect(err).To(HaveOccurred())
			} else {
				Expect(err).NotTo(HaveOccurred())
				Expect(res).NotTo(BeNil())
			}
			Eventually(postCalled).WithTimeout(1 * time.Second).Should(BeClosed())
		},
		Entry("deletes a document", nil, false, false, false, false),
		Entry("maps not found", repository.ErrNotFound, false, false, false, true),
		Entry("fails on find error", fmt.Errorf("find failed"), false, false, false, true),
		Entry("fails on pre-delete error", nil, true, false, false, true),
		Entry("fails on delete error", nil, false, true, false, true),
		Entry("succeeds when post-delete errors", nil, false, false, true, false),
	)

	DescribeTable("FindByID",
		func(findErr error, expectErr bool, expectNotFound bool) {
			if findErr != nil {
				mockStore.EXPECT().FindByID(testContext, testID).Return(nil, findErr)
			} else {
				mockStore.EXPECT().FindByID(testContext, testID).Return(tDoc, nil)
			}
			res, err := fac.FindByID(testContext, testID)
			if expectErr {
				Expect(err).To(HaveOccurred())
				if expectNotFound {
					Expect(err).To(MatchError(ftypes.ErrNoDocumentsFound))
				}
				Expect(res).To(BeNil())
			} else {
				Expect(err).NotTo(HaveOccurred())
				Expect(res).To(Equal(tDoc))
			}
		},
		Entry("returns a document", nil, false, false),
		Entry("maps not found", repository.ErrNotFound, true, true),
		Entry("propagates other errors", fmt.Errorf("db down"), true, false),
	)
})
