package facade_test

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"

	"github.com/amirdaaee/TGMon/internal/db"
	"github.com/amirdaaee/TGMon/internal/errs"
	"github.com/amirdaaee/TGMon/internal/facade"
	mockDB "github.com/amirdaaee/TGMon/mocks/db"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

var _ = Describe("MediaFacade", func() {
	var (
		mongoMock   *mockDB.MockIMongo
		minioMock   *mockDB.MockIMinioClient
		jobDSMock   *mockDB.MockIDataStore[*db.JobDoc]
		mediaDSMock *mockDB.MockIDataStore[*db.MediaFileDoc]
		mediaFacade *facade.MediaFacade
		mockClient  *mongo.Client
		testContext context.Context
	)
	resetMock := func() {
		mongoMock = mockDB.NewMockIMongo(GinkgoT())
		minioMock = mockDB.NewMockIMinioClient(GinkgoT())
		jobDSMock = mockDB.NewMockIDataStore[*db.JobDoc](GinkgoT())
		mediaDSMock = mockDB.NewMockIDataStore[*db.MediaFileDoc](GinkgoT())
		mediaFacade = facade.NewMediaFacade(mongoMock, minioMock, jobDSMock, mediaDSMock)
	}
	asserMockCall := func() {
		mongoMock.AssertExpectations(GinkgoT())
		minioMock.AssertExpectations(GinkgoT())
		jobDSMock.AssertExpectations(GinkgoT())
		mediaDSMock.AssertExpectations(GinkgoT())
	}
	BeforeEach(func() {
		mockClient = &mongo.Client{}
		testContext = context.Background()
	})

	Describe("Create", func() {
		type testCase struct {
			description    string
			inputDoc       *db.MediaFileDoc
			inputThumb     []byte
			createErr      bool
			jobCreateError bool
			minioAddError  bool
			expectJob      bool
			expectMinioAdd bool
		}

		var (
			testCases []testCase
		)

		BeforeEach(func() {
			resetMock()
		})

		AfterEach(func() {
			asserMockCall()
		})

		testCases = []testCase{
			{
				description:    "Successfully create media document with thumbnail",
				inputDoc:       &db.MediaFileDoc{},
				inputThumb:     []byte{0x01, 0x02},
				expectJob:      true,
				expectMinioAdd: true,
			},
			{
				description: "Successfully create media document without thumbnail",
				inputDoc:    &db.MediaFileDoc{},
				expectJob:   true,
			},
			{
				description:    "Successfully create media document while error creating job",
				inputDoc:       &db.MediaFileDoc{},
				inputThumb:     []byte{0x01, 0x02},
				jobCreateError: true,
				expectJob:      true,
				expectMinioAdd: true,
			},
			{
				description:    "Successfully create media document while error adding thumbnail",
				inputDoc:       &db.MediaFileDoc{},
				inputThumb:     []byte{0x01, 0x02},
				minioAddError:  true,
				expectJob:      true,
				expectMinioAdd: true,
			},
			{
				description: "Error create doc",
				inputDoc:    &db.MediaFileDoc{},
				inputThumb:  []byte{0x01, 0x02},
				createErr:   true,
			},
		}

		for _, tc := range testCases {
			It(tc.description, func() {
				// Arrange
				// Mock baseCreate (which calls ds.Create)
				outDoc := tc.inputDoc
				outDoc.SetID(primitive.NewObjectID())
				if tc.createErr {
					mediaDSMock.EXPECT().Create(mock.Anything, tc.inputDoc, mock.Anything).Return(nil, fmt.Errorf("test add error"))
				} else {
					mediaDSMock.EXPECT().Create(mock.Anything, tc.inputDoc, mock.Anything).Return(outDoc, nil)
					mongoMock.EXPECT().GetClient().RunAndReturn(func() (*mongo.Client, error) {
						return &mongo.Client{}, nil
					})
				}
				if tc.expectJob {
					if tc.jobCreateError {
						jobDSMock.EXPECT().Create(mock.Anything, mock.Anything, mock.Anything).Return(nil, fmt.Errorf("test job err"))
					} else {
						jobDSMock.EXPECT().Create(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(
							func(ctx context.Context, jDoc *db.JobDoc, cl *mongo.Client) (*db.JobDoc, errs.IMongoErr) {
								Expect(jDoc.Type).To(Equal(db.SPRITEJobType))
								return jDoc, nil
							},
						)
					}
				}
				if tc.expectMinioAdd {
					if tc.minioAddError {
						minioMock.EXPECT().FileAdd(mock.Anything, tc.inputThumb, mock.Anything).Return(fmt.Errorf("test minio err"))
					} else {
						minioMock.EXPECT().FileAdd(mock.Anything, tc.inputThumb, mock.Anything).Return(nil)
						_f := &bson.D{{Key: "_id", Value: primitive.NewObjectID()}}
						mediaDSMock.EXPECT().GetIDFilter(mock.Anything).Return(_f)
						mediaDSMock.EXPECT().Replace(mock.Anything, _f, mock.Anything, mock.Anything).Return(&db.MediaFileDoc{}, nil)
					}
				}
				// Act
				res, err := mediaFacade.Create(testContext, facade.NewFullMediaData(tc.inputDoc, tc.inputThumb), mockClient)
				if tc.createErr {
					Expect(res).To(BeNil())
					Expect(err).NotTo(BeNil())
				} else {
					Expect(res).NotTo(BeNil())
					Expect(err).To(BeNil())
				}

				time.Sleep(10 * time.Millisecond)
			})
		}
	})
	Describe("Read", func() {
		type testCase struct {
			description string
			filter      *primitive.D
			err         bool
			res         []*db.MediaFileDoc
		}

		var (
			testCases []testCase
		)

		BeforeEach(func() {
			resetMock()
		})

		AfterEach(func() {
			asserMockCall()
		})
		smapleFilter := &primitive.D{}
		testCases = []testCase{
			{
				description: "Successfully read many media documents",
				res:         []*db.MediaFileDoc{{}, {}},
				filter:      smapleFilter,
			},
			{
				description: "Successfully read empty media document",
				res:         []*db.MediaFileDoc{},
				filter:      smapleFilter,
			},
			{
				description: "Error read from datastore",
				err:         true,
				filter:      smapleFilter,
			},
		}

		for _, tc := range testCases {
			It(tc.description, func() {
				// Arrange
				if tc.err {
					mediaDSMock.EXPECT().List(mock.Anything, tc.filter, mockClient).Return(nil, fmt.Errorf("test mediaDSMock err"))
				} else {
					mediaDSMock.EXPECT().List(mock.Anything, tc.filter, mockClient).Return(tc.res, nil)
				}
				res, err := mediaFacade.Read(testContext, tc.filter, mockClient)
				if tc.err {
					Expect(res).To(BeNil())
					Expect(err).ToNot(BeNil())
				} else {
					Expect(res).To(Equal(tc.res))
					Expect(err).To(BeNil())
				}

			})
		}
	})
	Describe("Delete", func() {
		type testCase struct {
			description              string
			filter                   *primitive.D
			hasMedia                 bool
			err                      bool
			notFoundErr              bool
			expectPurgeJob           bool
			expectMinioRm            bool
			expectPurgeJobMarshalErr bool
			expectPurgeJobDeleteErr  bool
			expectMinioRmErr         bool
		}

		var (
			testCases []testCase
		)

		BeforeEach(func() {
			resetMock()
		})

		AfterEach(func() {
			asserMockCall()
		})
		smapleFilter := &primitive.D{}
		testCases = []testCase{
			{
				description:    "Successfully delete media document",
				filter:         smapleFilter,
				expectPurgeJob: true,
			},
			{
				description:    "Successfully delete media document with media",
				filter:         smapleFilter,
				hasMedia:       true,
				expectMinioRm:  true,
				expectPurgeJob: true,
			},
			{
				description:              "Successfully delete media document with media while job marshal fail",
				filter:                   smapleFilter,
				hasMedia:                 true,
				expectMinioRm:            true,
				expectPurgeJob:           true,
				expectPurgeJobMarshalErr: true,
			},
			{
				description:             "Successfully delete media document with media while job delete fail",
				filter:                  smapleFilter,
				hasMedia:                true,
				expectMinioRm:           true,
				expectPurgeJob:          true,
				expectPurgeJobDeleteErr: true,
			},
			{
				description:      "Successfully delete media document with media while minio fail",
				filter:           smapleFilter,
				hasMedia:         true,
				expectMinioRm:    true,
				expectPurgeJob:   true,
				expectMinioRmErr: true,
			},
			{
				description: "Error delete media document",
				filter:      smapleFilter,
				hasMedia:    true,
				err:         true,
			},
			{
				description: "Error delete media not found",
				filter:      smapleFilter,
				hasMedia:    true,
				err:         true,
				notFoundErr: true,
			},
		}

		for _, tc := range testCases {
			It(tc.description, func() {
				// Arrange
				mongoMock.EXPECT().GetClient().RunAndReturn(func() (*mongo.Client, error) { return &mongo.Client{}, nil }).Maybe()
				d := &db.MediaFileDoc{}
				if tc.hasMedia {
					d.Vtt = "vtt.jpeg"
					d.Thumbnail = "Thumb.jpeg"
					d.Sprite = "sprite.jpeg"
				}
				if tc.err {
					e := new(error)
					if tc.notFoundErr {
						*e = errs.NewMongoObjectNotfound(nil)
					} else {
						*e = fmt.Errorf("test mediaDSMock find err")
					}
					mediaDSMock.EXPECT().Find(mock.Anything, tc.filter, mockClient).Return(nil, *e)
				} else {
					mediaDSMock.EXPECT().Find(mock.Anything, tc.filter, mockClient).Return(d, nil)
					mediaDSMock.EXPECT().Delete(mock.Anything, tc.filter, mockClient).Return(nil)
				}

				if tc.expectPurgeJob {
					mE := new(errs.IMongoErr)
					mD := new(errs.IMongoErr)
					if tc.expectPurgeJobMarshalErr {
						*mE = errs.NewMongoMarshalErr(fmt.Errorf("test marshal err"))
					}
					if tc.expectPurgeJobDeleteErr {
						*mD = errs.NewMongoOpErr(fmt.Errorf("test delete err"))
					}
					jobDSMock.EXPECT().MarshalOmitEmpty(mock.Anything).Return(&bson.D{}, *mE)
					if !tc.expectPurgeJobMarshalErr {
						jobDSMock.EXPECT().DeleteMany(mock.Anything, mock.Anything, mock.Anything).Return(*mD)
					}
				}
				if tc.expectMinioRm {
					e := new(error)
					if tc.expectMinioRmErr {
						*e = fmt.Errorf("test minio err")
					}
					if d.Vtt != "" {
						minioMock.EXPECT().FileRm(d.Vtt, mock.Anything).Return(*e)
					}
					if d.Thumbnail != "" {
						minioMock.EXPECT().FileRm(d.Thumbnail, mock.Anything).Return(*e)
					}
					if d.Sprite != "" {
						minioMock.EXPECT().FileRm(d.Sprite, mock.Anything).Return(*e)
					}
				}
				// Act
				err := mediaFacade.Delete(testContext, tc.filter, mockClient)

				if tc.err {
					Expect(err).NotTo(BeNil())
					if tc.notFoundErr {
						Expect(errs.IsErr(err, errs.NewMongoObjectNotfound(nil))).To(BeTrue())
					}
				} else {
					Expect(err).To(BeNil())
				}
				time.Sleep(10 * time.Millisecond)
			})
		}
	})
})
