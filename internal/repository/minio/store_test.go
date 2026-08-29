package minio_test

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/amirdaaee/TGMon/internal/repository/minio"
	mockMinio "github.com/amirdaaee/TGMon/mocks/repository/minio"
	mnio "github.com/minio/minio-go/v7"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("Store", func() {
	const bucket = "mock_bucket"
	var (
		ctrl        *gomock.Controller
		mockAPI     *mockMinio.MockObjectAPI
		st          *minio.Store
		testContext context.Context
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		DeferCleanup(ctrl.Finish)
		testContext = context.Background()
		mockAPI = mockMinio.NewMockObjectAPI(ctrl)
		st = minio.NewStore(mockAPI, bucket)
	})

	DescribeTable("EnsureBucket",
		func(exists bool, existsErr error, makeErr error, callMake bool, expectErr bool) {
			mockAPI.EXPECT().BucketExists(gomock.Any(), bucket).Return(exists, existsErr)
			if callMake {
				mockAPI.EXPECT().MakeBucket(gomock.Any(), bucket, gomock.Any()).Return(makeErr)
			}
			err := st.EnsureBucket(testContext)
			if expectErr {
				Expect(err).To(HaveOccurred())
			} else {
				Expect(err).NotTo(HaveOccurred())
			}
		},
		Entry("creates missing bucket", false, nil, nil, true, false),
		Entry("skips existing bucket", true, nil, nil, false, false),
		Entry("fails when BucketExists errors", false, fmt.Errorf("exists failed"), nil, false, true),
		Entry("fails when MakeBucket errors", false, nil, fmt.Errorf("make failed"), true, true),
	)

	DescribeTable("Put",
		func(putErr error, expectErr bool) {
			data := []byte("test-data")
			mockAPI.EXPECT().PutObject(gomock.Any(), bucket, "test.file", gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, _, _ string, r io.Reader, size int64, _ mnio.PutObjectOptions) (mnio.UploadInfo, error) {
					d := make([]byte, len(data))
					n, err := r.Read(d)
					Expect(err).NotTo(HaveOccurred())
					Expect(d).To(Equal(data))
					Expect(size).To(BeEquivalentTo(n))
					return mnio.UploadInfo{}, putErr
				},
			)
			err := st.Put(testContext, "test.file", data)
			if expectErr {
				Expect(err).To(HaveOccurred())
			} else {
				Expect(err).NotTo(HaveOccurred())
			}
		},
		Entry("uploads data", nil, false),
		Entry("fails when PutObject errors", fmt.Errorf("put failed"), true),
	)

	DescribeTable("Delete",
		func(rmErr error, expectErr bool) {
			mockAPI.EXPECT().RemoveObject(gomock.Any(), bucket, "test.file", gomock.Any()).Return(rmErr)
			err := st.Delete(testContext, "test.file")
			if expectErr {
				Expect(err).To(HaveOccurred())
			} else {
				Expect(err).NotTo(HaveOccurred())
			}
		},
		Entry("removes a file", nil, false),
		Entry("fails when RemoveObject errors", fmt.Errorf("remove failed"), true),
	)

	DescribeTable("Stat",
		func(statErr error, expectErr bool) {
			info := mnio.ObjectInfo{Size: 12, LastModified: time.Unix(100, 0).UTC()}
			mockAPI.EXPECT().StatObject(gomock.Any(), bucket, "test.file", gomock.Any()).Return(info, statErr)
			got, err := st.Stat(testContext, "test.file")
			if expectErr {
				Expect(err).To(HaveOccurred())
				Expect(got).To(BeNil())
			} else {
				Expect(err).NotTo(HaveOccurred())
				Expect(got.Size).To(Equal(int64(12)))
				Expect(got.LastModified).To(Equal(info.LastModified))
			}
		},
		Entry("returns object info", nil, false),
		Entry("fails when StatObject errors", fmt.Errorf("stat failed"), true),
	)

	DescribeTable("Get",
		func(getErr error, expectErr bool) {
			mockAPI.EXPECT().GetObject(gomock.Any(), bucket, "test.file", gomock.Any()).Return(nil, getErr)
			obj, err := st.Get(testContext, "test.file")
			if expectErr {
				Expect(err).To(HaveOccurred())
				Expect(obj).To(BeNil())
			} else {
				Expect(err).NotTo(HaveOccurred())
			}
		},
		Entry("fails when GetObject errors", fmt.Errorf("get failed"), true),
	)
})

var _ = Describe("Connect", func() {
	It("returns an error when config is empty", func() {
		_, err := minio.Connect(context.Background(), minio.Config{}, false)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("minio endpoint is required"))
	})
})
