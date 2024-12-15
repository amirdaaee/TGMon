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
	"github.com/stretchr/testify/mock"
)

var _ = Describe("Minio", func() {
	var testContext context.Context
	// ...
	BeforeEach(func() {
		testContext = context.Background()
	})
	// ================================
	Describe("MinioClient", Label("MinioClient"), func() {
		var (
			mockMinio *mockDB.MockIMinioCl
		)
		resetMock := func() {
			mockMinio = mockDB.NewMockIMinioCl(GinkgoT())
		}
		asserMockCall := func() {
			mockMinio.AssertExpectations(GinkgoT())
		}
		newMinioClient := func(bucket string) db.IMinioClient {
			cl, err := db.NewMinioClient(&db.MinioConfig{MinioBucket: bucket}, func(s string, o *minio.Options) (db.IMinioCl, error) { return mockMinio, nil }, true)
			Expect(err).ToNot(HaveOccurred())
			return cl
		}
		// ...
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
			BeforeEach(func() {
				resetMock()
			})

			AfterEach(func() {
				asserMockCall()
			})
			// ...
			assertMinio_BucketExists := func(tc testCase) {
				mockMinio.EXPECT().BucketExists(mock.Anything, tc.bucketName).Return(tc.expectBucketExistsRes, tc.expectBucketExistsErr)
			}
			assertMinio_MakeBucket := func(tc testCase) {
				if tc.expectMakeBucketCall {
					mockMinio.EXPECT().MakeBucket(mock.Anything, tc.bucketName, mock.Anything).Return(tc.expectMakeBucketErr)
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
					cl := newMinioClient(tc.bucketName)
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
			BeforeEach(func() {
				resetMock()
			})

			AfterEach(func() {
				asserMockCall()
			})
			// ...
			assertMinio_PutObjec := func(tc testCase) {
				mockMinio.EXPECT().PutObject(mock.Anything, tc.bucketName, tc.filename, mock.Anything, mock.Anything, mock.Anything).RunAndReturn(
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
					cl := newMinioClient(tc.bucketName)
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
			BeforeEach(func() {
				resetMock()
			})

			AfterEach(func() {
				asserMockCall()
			})
			// ...
			assertMinio_PutObjec := func(tc testCase) {
				mockMinio.EXPECT().PutObject(mock.Anything, tc.bucketName, tc.filename, mock.Anything, mock.Anything, mock.Anything).RunAndReturn(
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
					cl := newMinioClient(tc.bucketName)
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
			BeforeEach(func() {
				resetMock()
			})

			AfterEach(func() {
				asserMockCall()
			})
			// ...
			assertMinio_RemoveObject := func(tc testCase) {
				mockMinio.EXPECT().RemoveObject(mock.Anything, tc.bucketName, tc.filename, mock.Anything).Return(tc.expectRemoveObjectErr)
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
					cl := newMinioClient(tc.bucketName)
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
