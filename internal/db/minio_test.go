package db_test

import (
	"context"
	"fmt"
	"io"

	"github.com/amirdaaee/TGMon/internal/db"
	mockDB "github.com/amirdaaee/TGMon/mocks/db"
	"github.com/minio/minio-go/v7"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gomock "go.uber.org/mock/gomock"
)

var _ = Describe("Minio", func() {
	var (
		mockMinio   *mockDB.MockIMinioCl
		testContext context.Context
		ctrl        *gomock.Controller
	)
	// ...
	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		testContext = context.Background()
		mockMinio = mockDB.NewMockIMinioCl(ctrl)

		db.DefaultMinioRegistry.InitMinioClient(testContext,
			&db.MinioConfig{MinioBucket: "mock_bucket"},
			true,
			func(s string, o *minio.Options) (db.IMinioCl, error) {
				return mockMinio, nil
			}, nil)
	})
	// ================================
	Describe("MinioClient", Label("MinioClient"), func() {

		Describe("CreateBucket", Label("CreateBucket"), func() {
			type testCase struct {
				description           string
				tType                 TestCaseType
				bucketName            string
				expectBucketExistsRes bool  // returned result from minio.BucketExist
				expectBucketExistsErr error // returned error from minio.BucketExist
				expectMakeBucketCall  bool  // whether or not to expect calling minio.MakeBucket
				expectMakeBucketErr   error // return error from minio.MakeBucket
				expectErr             bool  // whether or not expect failure
			}
			// ...
			assertMinio_BucketExists := func(tc testCase) {
				mockMinio.EXPECT().BucketExists(gomock.Any(), tc.bucketName).Return(tc.expectBucketExistsRes, tc.expectBucketExistsErr)
			}
			assertMinio_MakeBucket := func(tc testCase) {
				if tc.expectMakeBucketCall {
					mockMinio.EXPECT().MakeBucket(gomock.Any(), tc.bucketName, gomock.Any()).Return(tc.expectMakeBucketErr)
				}
			}
			// ...
			tests := []testCase{
				{
					description:           "successfully create bucket",
					tType:                 HAPPY_PATH,
					bucketName:            "mock_bucket",
					expectBucketExistsRes: false,
					expectMakeBucketCall:  true,
				},
				{
					description:           "don't create existing bucket",
					tType:                 HAPPY_PATH,
					bucketName:            "mock_bucket",
					expectBucketExistsRes: true,
					expectMakeBucketCall:  false,
				},
				{
					description:           "error calling BucketExists",
					tType:                 FAILURE,
					bucketName:            "mock_bucket",
					expectBucketExistsErr: fmt.Errorf("mock BucketExists err"),
					expectErr:             true,
				},
				{
					description:          "error calling MakeBucket",
					tType:                FAILURE,
					bucketName:           "mock_bucket",
					expectMakeBucketCall: true,
					expectMakeBucketErr:  fmt.Errorf("mock MakeBucket err"),
					expectErr:            true,
				},
			}
			// ...
			for _, tc := range tests {
				tc := tc
				It(tc.description, Label(string(tc.tType)), func() {
					// Arrange
					cl := db.DefaultMinioRegistry.GetMinioClient()
					assertMinio_BucketExists(tc)
					assertMinio_MakeBucket(tc)
					// Act
					err := cl.CreateBucket(testContext)
					// Assert
					if tc.expectErr {
						Expect(err).To(HaveOccurred())
					} else {
						Expect(err).NotTo(HaveOccurred())
					}
				})
			}
		})
		Describe("FileAdd", Label("FileAdd"), func() {
			type testCase struct {
				description        string
				tType              TestCaseType
				bucketName         string
				filename           string
				data               []byte
				expectPutObjectErr error // returned error from minio.PutObject
				expectErr          bool  // whether or not expect failure
			}
			// ...
			assertMinio_PutObjec := func(tc testCase) {
				mockMinio.EXPECT().PutObject(gomock.Any(), tc.bucketName, tc.filename, gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
					func(ctx context.Context, s1, s2 string, r io.Reader, i int64, poo minio.PutObjectOptions) (minio.UploadInfo, error) {
						d := make([]byte, len(tc.data))
						n, err := r.Read(d)
						Expect(err).ToNot(HaveOccurred())
						Expect(d).To(Equal(tc.data))
						Expect(i).To(BeEquivalentTo(n))
						Expect(i).To(BeEquivalentTo(len(tc.data)))
						return minio.UploadInfo{}, tc.expectPutObjectErr
					},
				)
			}
			// ...
			tests := []testCase{
				{
					description: "successfully add data",
					tType:       HAPPY_PATH,
					bucketName:  "mock_bucket",
					data:        []byte("test-data"),
					filename:    "test.file",
				},
				{
					description:        "error calling PutObjec",
					tType:              FAILURE,
					bucketName:         "mock_bucket",
					data:               []byte("test-data"),
					filename:           "test.file",
					expectPutObjectErr: fmt.Errorf("mock PutObjec error"),
					expectErr:          true,
				},
			}
			// ...
			for _, tc := range tests {
				tc := tc
				It(tc.description, Label(string(tc.tType)), func() {
					// Arrange
					cl := db.DefaultMinioRegistry.GetMinioClient()
					assertMinio_PutObjec(tc)
					// Act
					err := cl.FileAdd(testContext, tc.filename, tc.data)
					// Assert
					if tc.expectErr {
						Expect(err).To(HaveOccurred())
					} else {
						Expect(err).NotTo(HaveOccurred())
					}
				})
			}
		})
		Describe("FileAddStr", Label("FileAddStr"), func() {
			type testCase struct {
				description        string
				tType              TestCaseType
				bucketName         string
				filename           string
				data               string
				expectPutObjectErr error // returned error from minio.PutObject
				expectErr          bool  // whether or not expect failure
			}
			// ...
			assertMinio_PutObjec := func(tc testCase) {
				mockMinio.EXPECT().PutObject(gomock.Any(), tc.bucketName, tc.filename, gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
					func(ctx context.Context, s1, s2 string, r io.Reader, i int64, poo minio.PutObjectOptions) (minio.UploadInfo, error) {
						d := make([]byte, len(tc.data))
						n, err := r.Read(d)
						Expect(err).ToNot(HaveOccurred())
						Expect(d).To(Equal([]byte(tc.data)))
						Expect(i).To(BeEquivalentTo(n))
						Expect(i).To(BeEquivalentTo(len(tc.data)))
						return minio.UploadInfo{}, tc.expectPutObjectErr
					},
				)
			}
			// ...
			tests := []testCase{
				{
					description: "successfully add data",
					tType:       HAPPY_PATH,
					bucketName:  "mock_bucket",
					data:        "test-data",
					filename:    "test.file",
				},
				{
					description:        "error calling PutObjec",
					tType:              FAILURE,
					bucketName:         "mock_bucket",
					data:               "test-data",
					filename:           "test.file",
					expectPutObjectErr: fmt.Errorf("mock PutObjec error"),
					expectErr:          true,
				},
			}
			// ...
			for _, tc := range tests {
				tc := tc
				It(tc.description, Label(string(tc.tType)), func() {
					// Arrange
					cl := db.DefaultMinioRegistry.GetMinioClient()
					assertMinio_PutObjec(tc)
					// Act
					err := cl.FileAddStr(testContext, tc.filename, tc.data)
					// Assert
					if tc.expectErr {
						Expect(err).To(HaveOccurred())
					} else {
						Expect(err).NotTo(HaveOccurred())
					}
				})
			}
		})
		Describe("RemoveObject", Label("RemoveObject"), func() {
			type testCase struct {
				description           string
				tType                 TestCaseType
				bucketName            string
				filename              string
				expectRemoveObjectErr error // returned error from minio.RemoveObject
				expectErr             bool  // whether or not expect failure
			}
			// ...
			assertMinio_RemoveObject := func(tc testCase) {
				mockMinio.EXPECT().RemoveObject(gomock.Any(), tc.bucketName, tc.filename, gomock.Any()).Return(tc.expectRemoveObjectErr)
			}
			// ...
			tests := []testCase{
				{
					description: "successfully add data",
					tType:       HAPPY_PATH,
					bucketName:  "mock_bucket",
					filename:    "test.file",
				},
				{
					description:           "error calling RemoveObject",
					tType:                 FAILURE,
					bucketName:            "mock_bucket",
					filename:              "test.file",
					expectRemoveObjectErr: fmt.Errorf("mock RemoveObject error"),
					expectErr:             true,
				},
			}
			// ...
			for _, tc := range tests {
				tc := tc
				It(tc.description, Label(string(tc.tType)), func() {
					// Arrange
					cl := db.DefaultMinioRegistry.GetMinioClient()
					assertMinio_RemoveObject(tc)
					// Act
					err := cl.FileRm(testContext, tc.filename)
					// Assert
					if tc.expectErr {
						Expect(err).To(HaveOccurred())
					} else {
						Expect(err).NotTo(HaveOccurred())
					}
				})
			}
		})
	})
})
