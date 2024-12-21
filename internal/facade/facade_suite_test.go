package facade_test

import (
	"testing"

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
	"github.com/brianvoe/gofakeit/v7"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func TestFacade(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Facade Suite")
}
func newResDoc[T db.IMongoDoc](d T) T {
	dCopy := d
	dCopy.SetID(primitive.NewObjectID())
	return dCopy
}

var _ = Describe("Facade", func() {
	var (
		mongoMock   *mockDB.MockIMongo
		minioMock   *mockDB.MockIMinioClient
		jobDSMock   *mockDB.MockIDataStore[*db.JobDoc]
		mediaDSMock *mockDB.MockIDataStore[*db.MediaFileDoc]
		mediaFacade *facade.MediaFacade
		jobFacade   *facade.JobFacade
		mockClient  *mongo.Client
		testContext context.Context
	)
	resetMock := func() {
		mongoMock = mockDB.NewMockIMongo(GinkgoT())
		minioMock = mockDB.NewMockIMinioClient(GinkgoT())
		jobDSMock = mockDB.NewMockIDataStore[*db.JobDoc](GinkgoT())
		mediaDSMock = mockDB.NewMockIDataStore[*db.MediaFileDoc](GinkgoT())
		mediaFacade = facade.NewMediaFacade(mongoMock, minioMock, jobDSMock, mediaDSMock)
		jobFacade = facade.NewJobFacade(mongoMock, minioMock, jobDSMock, mediaDSMock)
	}
	asserMockCall := func() {
		mongoMock.AssertExpectations(GinkgoT())
		minioMock.AssertExpectations(GinkgoT())
		jobDSMock.AssertExpectations(GinkgoT())
		mediaDSMock.AssertExpectations(GinkgoT())
	}
	newMongoClient := func() (db.IMongoClient, error) {
		return &mongo.Client{}, nil
	}
	BeforeEach(func() {
		mockClient = &mongo.Client{}
		testContext = context.Background()
	})

	// ================================
	Describe("MediaFacade", Label("MediaFacade"), func() {
		Describe("Create", Label("Create"), func() {
			type testCase struct {
				description    string
				inputDoc       *db.MediaFileDoc // doc to be created
				outputDoc      *db.MediaFileDoc // result doc of ds.create
				inputThumb     []byte           // thumbnail to add
				createErr      bool             // error calling ds.create
				replaceErr     bool             // error calling ds.replace
				jobCreateError bool             // error calling jobDs.create
				minioAddError  bool             // error calling minio.fileAdd
				expectJob      bool             // expect job creating
				expectMinioAdd bool             // expect thumbnail storage
			}
			assertMongo_GetClient := func(tc testCase) {
				if tc.expectJob || tc.expectMinioAdd {
					// used for job and thumb update
					mongoMock.EXPECT().GetClient().RunAndReturn(newMongoClient)
				}
			}
			assertMediaDs_Create := func(tc testCase) {
				v := new(db.MediaFileDoc)
				err := new(error)
				if tc.createErr {
					*err = fmt.Errorf("mock mediaDSMock.Create err")
				} else {
					v = tc.outputDoc
				}
				mediaDSMock.EXPECT().Create(mock.Anything, tc.inputDoc, mock.Anything).Return(v, *err)
			}
			assertMediaDs_Replace := func(tc testCase) {
				if tc.expectMinioAdd && !tc.minioAddError {
					mediaDSMock.EXPECT().Replace(mock.Anything, mock.Anything, mock.Anything, mock.Anything).RunAndReturn(
						func(ctx context.Context, d *primitive.M, mfd *db.MediaFileDoc, c db.IMongoClient) (*db.MediaFileDoc, errs.IMongoErr) {
							Expect(*d).To(BeEquivalentTo(*db.GetIDFilter(tc.outputDoc.GetID())))
							Expect(mfd.GetID()).To(BeEquivalentTo(tc.outputDoc.GetID()))
							Expect(mfd.Thumbnail).NotTo(BeEmpty())
							v := new(db.MediaFileDoc)
							err := new(error)
							if tc.replaceErr {
								*err = fmt.Errorf("mock mediaDSMock.Replace err")
							} else {
								mfdCopy := *mfd
								v = &mfdCopy
							}
							return v, *err
						})
				}
			}
			assertJobDs_Create := func(tc testCase) {
				if tc.expectJob {
					jobDSMock.EXPECT().Create(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(
						func(ctx context.Context, jd *db.JobDoc, c db.IMongoClient) (*db.JobDoc, errs.IMongoErr) {
							Expect(jd.MediaID).To(BeEquivalentTo(tc.outputDoc.GetID()))
							Expect(jd.Type).To(BeEquivalentTo(db.SPRITEJobType))
							v := new(db.JobDoc)
							err := new(error)
							if tc.jobCreateError {
								*err = fmt.Errorf("mock jobDSMock.Create err")
							} else {
								v = newResDoc(jd)
							}
							return v, *err
						})
				}
			}
			assertMinioMock_FileAdd := func(tc testCase) {
				if tc.expectMinioAdd && tc.inputThumb != nil {
					err := new(error)
					if tc.minioAddError {
						*err = fmt.Errorf("mock minioMock.FileAdd err")
					}
					minioMock.EXPECT().FileAdd(mock.Anything, mock.Anything, tc.inputThumb).Return(*err)
				}
			}
			newFakeMediaDoc := func() *db.MediaFileDoc {
				d := new(db.MediaFileDoc)
				gofakeit.Struct(&d)
				d.SetID(primitive.NewObjectID())
				d.DateAdded = 0
				d.Sprite = ""
				d.Vtt = ""
				d.Thumbnail = ""
				return d
			}
			Describe("Happy path", Label("Happy"), func() {
				BeforeEach(func() {
					resetMock()
				})

				AfterEach(func() {
					asserMockCall()
				})
				testCases := []testCase{
					{
						description:    "Successfully create media document with thumbnail",
						inputDoc:       newFakeMediaDoc(),
						inputThumb:     []byte{0x01, 0x02},
						expectJob:      true,
						expectMinioAdd: true,
					},
					{
						description:    "Successfully create media document without thumbnail",
						inputDoc:       newFakeMediaDoc(),
						inputThumb:     nil,
						expectJob:      true,
						expectMinioAdd: false,
					},
				}
				for _, tc := range testCases {
					tc := tc
					It(tc.description, func() {
						// Arrange
						tc.outputDoc = newResDoc(tc.inputDoc)
						// ...
						assertMongo_GetClient(tc)
						// ...
						assertMediaDs_Create(tc)
						assertMediaDs_Replace(tc)
						// ...
						assertJobDs_Create(tc)
						// ...
						assertMinioMock_FileAdd(tc)
						// Act
						res, err := mediaFacade.Create(testContext, facade.NewFullMediaData(tc.inputDoc, tc.inputThumb), mockClient)
						time.Sleep(10 * time.Millisecond) // wait for coroutines
						// Assert
						Expect(err).To(BeNil())
						Expect(res).NotTo(BeNil())
						Expect(*res).To(Equal(*tc.outputDoc))
					})
				}
			})
			Describe("Failure mode", Label("Failure"), func() {
				BeforeEach(func() {
					resetMock()
				})

				AfterEach(func() {
					asserMockCall()
				})
				testCases := []testCase{
					{
						description:    "Successfully create media document while error creating job",
						inputDoc:       newFakeMediaDoc(),
						inputThumb:     []byte{0x01, 0x02},
						jobCreateError: true,
						expectJob:      true,
						expectMinioAdd: true,
					},
					{
						description:    "Successfully create media document while error adding thumbnail",
						inputDoc:       newFakeMediaDoc(),
						inputThumb:     []byte{0x01, 0x02},
						minioAddError:  true,
						expectJob:      true,
						expectMinioAdd: true,
					},
					{
						description:    "Successfully create media document while error replacing while adding thumbnail",
						inputDoc:       newFakeMediaDoc(),
						inputThumb:     []byte{0x01, 0x02},
						replaceErr:     true,
						expectJob:      true,
						expectMinioAdd: true,
					},
					{
						description: "Error create doc (datastore)",
						inputDoc:    newFakeMediaDoc(),
						inputThumb:  []byte{0x01, 0x02},
						createErr:   true,
					},
				}
				for _, tc := range testCases {
					It(tc.description, func() {
						// Arrange
						tc.outputDoc = newResDoc(tc.inputDoc)
						// ...
						assertMongo_GetClient(tc)
						// ...
						assertMediaDs_Create(tc)
						assertMediaDs_Replace(tc)
						// ...
						assertJobDs_Create(tc)
						// ...
						assertMinioMock_FileAdd(tc)
						// Act
						res, err := mediaFacade.Create(testContext, facade.NewFullMediaData(tc.inputDoc, tc.inputThumb), mockClient)
						time.Sleep(10 * time.Millisecond) // wait for coroutines
						// Assert
						if tc.createErr {
							Expect(res).To(BeNil())
							Expect(err).NotTo(BeNil())
							// todo: assert error type
						} else {
							Expect(err).To(BeNil())
							Expect(res).NotTo(BeNil())
							Expect(*res).To(Equal(*tc.outputDoc))
						}
					})
				}
			})
		})
		Describe("Read", Label("Read"), func() {
			type testCase struct {
				description string
				filter      *primitive.M       // filter to call Read
				outputDoc   []*db.MediaFileDoc // result docs of ds.findMany
				findManyErr bool               // error calling ds.findMany
			}
			assertMediaDs_List := func(tc testCase) {
				v := new([]*db.MediaFileDoc)
				err := new(error)
				if tc.findManyErr {
					*err = fmt.Errorf("mock mediaDSMock.FindMany err")
				} else {
					*v = tc.outputDoc
				}
				mediaDSMock.EXPECT().FindMany(mock.Anything, tc.filter, mock.Anything).Return(tc.outputDoc, *err)
			}
			newBsonEmptyFilter := func() *primitive.M {
				return &primitive.M{}
			}
			newBsonIDFilter := func() *primitive.M {
				return db.GetIDFilter(primitive.NewObjectID())
			}
			newFakeMediaDoc := func() *db.MediaFileDoc {
				d := new(db.MediaFileDoc)
				gofakeit.Struct(&d)
				return d
			}
			newManyFakeDoc := func(n uint) []*db.MediaFileDoc {
				res := []*db.MediaFileDoc{}
				for range n {
					res = append(res, newFakeMediaDoc())
				}
				return res
			}
			Describe("Happy path", Label("Happy"), func() {
				BeforeEach(func() {
					resetMock()
				})
				AfterEach(func() {
					asserMockCall()
				})
				testCases := []testCase{
					{
						description: "Successfully read many media documents",
						outputDoc:   newManyFakeDoc(10),
						filter:      newBsonIDFilter(),
					},
					{
						description: "Successfully read zero media document",
						outputDoc:   newManyFakeDoc(0),
					},
					{
						description: "Successfully read media with nil filter",
						outputDoc:   newManyFakeDoc(10),
						filter:      nil,
					},
					{
						description: "Successfully read media with empty filter",
						outputDoc:   newManyFakeDoc(10),
						filter:      newBsonEmptyFilter(),
					},
				}
				for _, tc := range testCases {
					It(tc.description, func() {
						// Arrange
						assertMediaDs_List(tc)
						// Act
						res, err := mediaFacade.Read(testContext, tc.filter, mockClient)
						// Assert
						Expect(err).To(BeNil())
						Expect(res).To(Equal(tc.outputDoc))
						Expect(len(res)).To(Equal(len(tc.outputDoc)))
						Expect(res).To(ContainElements(tc.outputDoc))
					})
				}
			})
			Describe("Failure path", Label("Failure"), func() {
				BeforeEach(func() {
					resetMock()
				})
				AfterEach(func() {
					asserMockCall()
				})
				testCases := []testCase{
					{
						description: "Error list media (datastore)",
						outputDoc:   nil,
						filter:      newBsonIDFilter(),
						findManyErr: true,
					},
				}
				for _, tc := range testCases {
					It(tc.description, func() {
						// Arrange
						assertMediaDs_List(tc)
						// Act
						res, err := mediaFacade.Read(testContext, tc.filter, mockClient)
						// Assert
						Expect(res).To(BeNil())
						Expect(err).ToNot(BeNil())
						// todo: assert error type
					})
				}
			})
		})
		Describe("Delete", Label("Delete"), func() {
			type testCase struct {
				description        string
				filter             primitive.M      // filter to call Read
				outputDoc          *db.MediaFileDoc // result doc of ds.find
				withThumbMedia     bool             // doc has Thumbnail
				withVttMedia       bool             // doc has Vtt
				withSpriteMedia    bool             // doc has Sprite
				findErr            error            // error calling jobDs.find
				jobDeleteManyError error            // error calling jobDs.deleteMany
				minioRmError       bool             // error calling minio.rmFile
				expectDelete       bool             // expect ds.delete
				expectJob          bool             // expect job purge
				expectMinioRm      bool             // expect minio media purge

			}
			assertMongo_GetClient := func(tc testCase) {
				if tc.expectJob || tc.expectMinioRm {
					// used for job and thumb update
					mongoMock.EXPECT().GetClient().RunAndReturn(newMongoClient)
				}
			}
			assertMediaDs_Find := func(tc testCase) {
				v := new(db.MediaFileDoc)
				err := new(error)
				if tc.findErr != nil {
					*err = tc.findErr
				} else {
					v = tc.outputDoc
				}
				mediaDSMock.EXPECT().Find(mock.Anything, &tc.filter, mock.Anything).Return(v, *err)
			}
			assertMediaDs_Delete := func(tc testCase) {
				if tc.expectDelete {
					mediaDSMock.EXPECT().Delete(mock.Anything, &tc.filter, mock.Anything).Return(nil)
				}
			}
			assertJobDs_DeleteMany := func(tc testCase) {
				if tc.expectJob {
					jobDSMock.EXPECT().DeleteMany(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(
						func(ctx context.Context, d *primitive.M, c db.IMongoClient) errs.IMongoErr {
							Expect(*d).To(Equal(primitive.M{"MediaID": tc.outputDoc.ID}))
							return tc.jobDeleteManyError
						})
				}
			}
			assertMinioMock_FileRm := func(tc testCase) {
				if tc.expectMinioRm {
					err := new(error)
					if tc.minioRmError {
						*err = fmt.Errorf("mock minioMock.FileRm err")
					}
					for _, s := range []string{tc.outputDoc.Thumbnail, tc.outputDoc.Sprite, tc.outputDoc.Vtt} {
						if s != "" {
							minioMock.EXPECT().FileRm(mock.Anything, s).Return(*err)
						}
					}
				}
			}
			newFakeMediaDoc := func() *db.MediaFileDoc {
				d := new(db.MediaFileDoc)
				gofakeit.Struct(&d)
				return d
			}
			setDocMedia := func(tc testCase) {
				if tc.outputDoc == nil {
					return
				}
				if !tc.withThumbMedia {
					tc.outputDoc.Thumbnail = ""
				}
				if !tc.withSpriteMedia {
					tc.outputDoc.Sprite = ""
				}
				if !tc.withVttMedia {
					tc.outputDoc.Vtt = ""
				}
			}
			newBsonIDFilter := func() *primitive.M {
				return db.GetIDFilter(primitive.NewObjectID())
			}
			Describe("Happy path", Label("Happy"), func() {
				BeforeEach(func() {
					resetMock()
				})
				AfterEach(func() {
					asserMockCall()
				})
				testCases := []testCase{
					{
						description:     "Successfully Delete media with all minio src (thumb+vtt+sprite)",
						outputDoc:       newFakeMediaDoc(),
						filter:          *newBsonIDFilter(),
						withThumbMedia:  true,
						withVttMedia:    true,
						withSpriteMedia: true,
						expectDelete:    true,
						expectJob:       true,
						expectMinioRm:   true,
					},
					{
						description:    "Successfully Delete media with all minio src (thumb+vtt)",
						outputDoc:      newFakeMediaDoc(),
						filter:         *newBsonIDFilter(),
						withThumbMedia: true,
						withVttMedia:   true,
						expectDelete:   true,
						expectJob:      true,
						expectMinioRm:  true,
					},
					{
						description:     "Successfully Delete media with all minio src (thumb+sprite)",
						outputDoc:       newFakeMediaDoc(),
						filter:          *newBsonIDFilter(),
						withThumbMedia:  true,
						withSpriteMedia: true,
						expectDelete:    true,
						expectJob:       true,
						expectMinioRm:   true,
					},
					{
						description:     "Successfully Delete media with all minio src (vtt+sprite)",
						outputDoc:       newFakeMediaDoc(),
						filter:          *newBsonIDFilter(),
						withVttMedia:    true,
						withSpriteMedia: true,
						expectDelete:    true,
						expectJob:       true,
						expectMinioRm:   true,
					},
					{
						description:    "Successfully Delete media with all minio src (thumb)",
						outputDoc:      newFakeMediaDoc(),
						filter:         *newBsonIDFilter(),
						withThumbMedia: true,
						expectDelete:   true,
						expectJob:      true,
						expectMinioRm:  true,
					},
					{
						description:   "Successfully Delete media with all minio src (vtt)",
						outputDoc:     newFakeMediaDoc(),
						filter:        *newBsonIDFilter(),
						withVttMedia:  true,
						expectDelete:  true,
						expectJob:     true,
						expectMinioRm: true,
					},
					{
						description:     "Successfully Delete media with all minio src (sprite)",
						outputDoc:       newFakeMediaDoc(),
						filter:          *newBsonIDFilter(),
						withSpriteMedia: true,
						expectDelete:    true,
						expectJob:       true,
						expectMinioRm:   true,
					},
					{
						description:   "Successfully Delete media without minio src",
						outputDoc:     newFakeMediaDoc(),
						filter:        *newBsonIDFilter(),
						expectDelete:  true,
						expectJob:     true,
						expectMinioRm: true,
					},
				}
				for _, tc := range testCases {
					tc := tc
					It(tc.description, func() {
						// Arrange
						setDocMedia(tc)
						// ...
						assertMongo_GetClient(tc)
						// ...
						assertMediaDs_Find(tc)
						assertMediaDs_Delete(tc)
						// ...
						assertJobDs_DeleteMany(tc)
						// ...
						assertMinioMock_FileRm(tc)
						// Act
						err := mediaFacade.Delete(testContext, &tc.filter, mockClient)
						time.Sleep(10 * time.Millisecond) // wait for coroutines
						// Assert
						Expect(err).To(BeNil())
					})
				}
			})
			Describe("Failure path", Label("Failure"), func() {
				BeforeEach(func() {
					resetMock()
				})
				AfterEach(func() {
					asserMockCall()
				})
				testCases := []testCase{
					{
						description:        "Successfully Delete media while error delete job",
						outputDoc:          newFakeMediaDoc(),
						filter:             *newBsonIDFilter(),
						withThumbMedia:     true,
						withVttMedia:       true,
						withSpriteMedia:    true,
						expectDelete:       true,
						expectJob:          true,
						expectMinioRm:      true,
						jobDeleteManyError: fmt.Errorf("mock jobDs.DeleteMany error"),
					},
					{
						description:        "Successfully Delete media while no job found",
						outputDoc:          newFakeMediaDoc(),
						filter:             *newBsonIDFilter(),
						withThumbMedia:     true,
						withVttMedia:       true,
						withSpriteMedia:    true,
						expectDelete:       true,
						expectJob:          true,
						expectMinioRm:      true,
						jobDeleteManyError: errs.NewMongoObjectNotfound(primitive.M{}),
					},
					{
						description:     "Successfully Delete media while error rm minio files",
						outputDoc:       newFakeMediaDoc(),
						filter:          *newBsonIDFilter(),
						withThumbMedia:  true,
						withVttMedia:    true,
						withSpriteMedia: true,
						expectDelete:    true,
						expectJob:       true,
						expectMinioRm:   true,
						minioRmError:    true,
					},
					{
						description:     "Error find doc (datastore)",
						outputDoc:       newFakeMediaDoc(),
						filter:          *newBsonIDFilter(),
						withThumbMedia:  true,
						withVttMedia:    true,
						withSpriteMedia: true,
						findErr:         errs.NewMongoObjectNotfound(*newBsonIDFilter()),
					},
				}
				for _, tc := range testCases {
					tc := tc
					It(tc.description, func() {
						// Arrange
						setDocMedia(tc)
						// ...
						assertMongo_GetClient(tc)
						// ...
						assertMediaDs_Find(tc)
						assertMediaDs_Delete(tc)
						// ...
						assertJobDs_DeleteMany(tc)
						// ...
						assertMinioMock_FileRm(tc)
						// Act
						err := mediaFacade.Delete(testContext, &tc.filter, mockClient)
						time.Sleep(10 * time.Millisecond) // wait for coroutines
						// Assert
						if tc.findErr == nil {
							Expect(err).To(BeNil())
						} else {
							Expect(err).To(BeEquivalentTo(tc.findErr))
						}
					})
				}
			})
		})
	})
	// ================================
	Describe("JobFacade", Label("JobFacade"), func() {
		Describe("Create", Label("Create"), func() {
			type testCase struct {
				description       string
				inputDoc          *db.JobDoc // doc to be created
				outputDoc         *db.JobDoc // result doc of ds.create
				isDuplicated      bool       // job is duplicated
				expectJobDsList   bool       // expect job ds list
				expectJobDsCreate bool       // expect job ds create
				jobDsListErr      error      // error calling ds.findMany
				jobDsCreateErr    error      // error calling ds.create
				expectErr         error
			}
			assertJobDs_List := func(tc testCase) {
				if tc.expectJobDsList {
					jobDSMock.EXPECT().FindMany(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(
						func(ctx context.Context, d *primitive.M, c db.IMongoClient) ([]*db.JobDoc, errs.IMongoErr) {
							filterObj := db.JobDoc{MediaID: tc.inputDoc.MediaID, Type: tc.inputDoc.Type}
							expectedFilter, _err := filterObj.MarshalOmitEmpty()
							Expect(_err).To(BeNil())
							Expect(len(*d)).Should(Equal(len(*expectedFilter)))
							Expect(*d).To(Equal(*expectedFilter))
							// ...
							v := []*db.JobDoc{}
							err := new(error)
							if tc.jobDsListErr == nil {
								if tc.isDuplicated {
									filterObj.SetID(tc.outputDoc.GetID())
									v = append(v, &filterObj)
								}
							} else {
								v = nil
								*err = tc.jobDsListErr
							}
							return v, *err
						},
					)
				}
			}
			assertJobDs_Create := func(tc testCase) {
				if tc.expectJobDsCreate {
					jobDSMock.EXPECT().Create(mock.Anything, tc.inputDoc, mock.Anything).RunAndReturn(
						func(ctx context.Context, jd *db.JobDoc, c db.IMongoClient) (*db.JobDoc, errs.IMongoErr) {
							v := new(db.JobDoc)
							err := new(error)
							if tc.jobDsCreateErr == nil {
								v = tc.outputDoc
							} else {
								*err = tc.jobDsCreateErr
							}
							return v, *err
						},
					)
				}
			}
			newFakeJobDoc := func(jt db.JobType) *db.JobDoc {
				d := new(db.JobDoc)
				gofakeit.Struct(&d)
				d.SetID(primitive.NewObjectID())
				d.Type = jt
				return d
			}
			Describe("Happy path", Label("Happy"), func() {
				BeforeEach(func() {
					resetMock()
				})

				AfterEach(func() {
					asserMockCall()
				})
				testCases := []testCase{
					{
						description:       "Successfully create thumbnail job document",
						inputDoc:          newFakeJobDoc(db.THUMBNAILJobType),
						expectJobDsList:   true,
						expectJobDsCreate: true,
					},
					{
						description:       "Successfully create sprite job document",
						inputDoc:          newFakeJobDoc(db.SPRITEJobType),
						expectJobDsList:   true,
						expectJobDsCreate: true,
					},
					{
						description:     "Not create duplicated thumbnail job document",
						inputDoc:        newFakeJobDoc(db.THUMBNAILJobType),
						isDuplicated:    true,
						expectJobDsList: true,
					},
					{
						description:     "Not create duplicated sprite job document",
						inputDoc:        newFakeJobDoc(db.SPRITEJobType),
						isDuplicated:    true,
						expectJobDsList: true,
					},
				}
				for _, tc := range testCases {
					tc := tc
					It(tc.description, func() {
						// Arrange
						tc.outputDoc = newResDoc(tc.inputDoc)
						// ...
						assertJobDs_List(tc)
						assertJobDs_Create(tc)
						// Act
						res, err := jobFacade.Create(testContext, tc.inputDoc, mockClient)
						// Assert
						Expect(err).To(BeNil())
						Expect(*res).To(BeEquivalentTo(*tc.outputDoc))
						Expect(res).NotTo(BeNil())
					})
				}
			})
			Describe("Failure path", Label("Failure"), func() {
				BeforeEach(func() {
					resetMock()
				})

				AfterEach(func() {
					asserMockCall()
				})
				testCases := []testCase{
					{
						description:     "Error list docs (datastore)",
						inputDoc:        newFakeJobDoc(db.THUMBNAILJobType),
						expectJobDsList: true,
						jobDsListErr:    errs.NewMongoOpErr(fmt.Errorf("mock jobDSMock.FindMany err")),
						expectErr:       fmt.Errorf("sample err"),
					},
					{
						description:       "Error create docs (datastore)",
						inputDoc:          newFakeJobDoc(db.THUMBNAILJobType),
						expectJobDsList:   true,
						expectJobDsCreate: true,
						jobDsCreateErr:    errs.NewMongoOpErr(fmt.Errorf("mock jobDSMock.Create err")),
						expectErr:         fmt.Errorf("sample err"),
					},
				}
				for _, tc := range testCases {
					tc := tc
					It(tc.description, func() {
						// Arrange
						tc.outputDoc = newResDoc(tc.inputDoc)
						// ...
						assertJobDs_List(tc)
						assertJobDs_Create(tc)
						// Act
						res, err := jobFacade.Create(testContext, tc.inputDoc, mockClient)
						// Assert
						if tc.expectErr == nil {
							Expect(err).To(BeNil())
							Expect(*res).To(BeEquivalentTo(*tc.outputDoc))
							Expect(res).NotTo(BeNil())
						} else {
							Expect(err).ToNot(BeNil())
							Expect(res).To(BeNil())
							// todo: assert err type
						}
					})
				}
			})
		})
		Describe("Read", Label("Read"), func() {
			type testCase struct {
				description  string
				filter       *primitive.M // filter to call Read
				outputDoc    []*db.JobDoc // result docs of ds.findMany
				jobDsListErr error        // error calling ds.findMany
				expectErr    error
			}
			assertJobDs_List := func(tc testCase) {
				jobDSMock.EXPECT().FindMany(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(
					func(ctx context.Context, d *primitive.M, c db.IMongoClient) ([]*db.JobDoc, errs.IMongoErr) {
						Expect(d).Should(Equal(tc.filter))
						// ...
						v := new([]*db.JobDoc)
						err := new(error)
						if tc.jobDsListErr == nil {
							*v = tc.outputDoc
						} else {
							*err = tc.jobDsListErr
						}
						return *v, *err
					},
				)
			}
			newBsonEmptyFilter := func() *primitive.M {
				return &primitive.M{}
			}
			newBsonIDFilter := func() *primitive.M {
				return db.GetIDFilter(primitive.NewObjectID())
			}
			newFakeJobDoc := func(jt db.JobType) *db.JobDoc {
				d := new(db.JobDoc)
				gofakeit.Struct(&d)
				d.SetID(primitive.NewObjectID())
				d.Type = jt
				return d
			}
			newManyFakeJobDoc := func(n uint) []*db.JobDoc {
				res := []*db.JobDoc{}
				jdArr := []db.JobType{db.SPRITEJobType, db.THUMBNAILJobType}
				for range n {
					res = append(res, newFakeJobDoc(jdArr[n%2]))
				}
				return res
			}
			Describe("Happy path", Label("Happy"), func() {
				BeforeEach(func() {
					resetMock()
				})

				AfterEach(func() {
					asserMockCall()
				})
				testCases := []testCase{
					{
						description: "Successfully read many media documents",
						outputDoc:   newManyFakeJobDoc(10),
						filter:      newBsonIDFilter(),
					},
					{
						description: "Successfully read zero media document",
						outputDoc:   newManyFakeJobDoc(0),
					},
					{
						description: "Successfully read media with nil filter",
						outputDoc:   newManyFakeJobDoc(10),
						filter:      nil,
					},
					{
						description: "Successfully read media with empty filter",
						outputDoc:   newManyFakeJobDoc(10),
						filter:      newBsonEmptyFilter(),
					},
				}
				for _, tc := range testCases {
					tc := tc
					It(tc.description, func() {
						// Arrange
						// ...
						assertJobDs_List(tc)
						// Act
						res, err := jobFacade.Read(testContext, tc.filter, mockClient)
						// Assert
						Expect(err).To(BeNil())
						Expect(res).NotTo(BeNil())
						Expect(len(res)).To(Equal(len(tc.outputDoc)))
						Expect(res).To(ContainElements(tc.outputDoc))
					})
				}
			})
			Describe("Failure path", Label("Failure"), func() {
				BeforeEach(func() {
					resetMock()
				})

				AfterEach(func() {
					asserMockCall()
				})
				testCases := []testCase{
					{
						description:  "Error list media (datastore)",
						outputDoc:    nil,
						filter:       newBsonIDFilter(),
						jobDsListErr: errs.NewMongoOpErr(fmt.Errorf("mock jobDSMock.FindMany err")),
						expectErr:    fmt.Errorf("sample err"),
					},
				}
				for _, tc := range testCases {
					tc := tc
					It(tc.description, func() {
						// Arrange
						// ...
						assertJobDs_List(tc)
						// Act
						res, err := jobFacade.Read(testContext, tc.filter, mockClient)
						// Assert
						if tc.expectErr == nil {
							Expect(err).To(BeNil())
							Expect(res).NotTo(BeNil())
							Expect(len(res)).To(Equal(len(tc.outputDoc)))
							Expect(res).To(ContainElements(tc.outputDoc))

						} else {
							Expect(err).ToNot(BeNil())
							Expect(res).To(BeNil())
							// todo: assert error type
						}
					})
				}
			})
		})
		Describe("Done", Label("Done"), func() {
			type testCase struct {
				description          string
				jobDoc               *db.JobDoc            //job doc in db
				mediaDoc             *db.MediaFileDoc      //media doc in db
				data                 facade.MediaMinioFile // data to pass to done function
				expectJobDsFind      bool                  // expect job ds find
				expectJobDsDelete    bool                  // expect job ds delete
				expectMediaDsFind    bool                  // expect media ds find
				expectMediaDsReplace bool                  // expect media ds replace
				expectminioAddFile   bool                  // expect minio fileadd
				expectminioRmFile    bool                  // expect minio filerm
				jobDsFindErr         error                 // error calling ds.find
				jobDsDeleteErr       error                 // error calling ds.delete
				mediaDsFindErr       error                 // error calling mediaDs.find
				mediaDsReplaceErr    error                 // error calling mediaDs.replace
				minioAddFileErr      error                 // error calling minio.fileAdd
				minioRmFileErr       error                 // error calling minio.fileRm
				expectErr            bool
			}
			assertMongo_GetClient := func(tc testCase) {
				if tc.expectMediaDsFind || tc.expectMediaDsReplace || tc.expectJobDsDelete {
					mongoMock.EXPECT().GetClient().RunAndReturn(newMongoClient)
				}
			}
			assertJobDs_Find := func(tc testCase) {
				if tc.expectJobDsFind {
					jobDSMock.EXPECT().Find(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(
						func(ctx context.Context, d *primitive.M, c db.IMongoClient) (*db.JobDoc, errs.IMongoErr) {
							Expect(*d).To(Equal(*db.GetIDFilter(tc.jobDoc.GetID())))
							// ...
							v := new(db.JobDoc)
							err := new(error)
							if tc.jobDsFindErr == nil {
								v = tc.jobDoc
							} else {
								*err = tc.jobDsFindErr
							}
							// ...
							return v, *err
						},
					)
				}
			}
			assertJobDs_Delete := func(tc testCase) {
				if tc.expectJobDsDelete {
					jobDSMock.EXPECT().Delete(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(
						func(ctx context.Context, d *primitive.M, c db.IMongoClient) errs.IMongoErr {
							Expect(*d).To(Equal(*db.GetIDFilter(tc.jobDoc.GetID())))
							// ...
							err := new(error)
							if tc.jobDsDeleteErr != nil {
								*err = tc.jobDsDeleteErr
							}
							// ...
							return *err
						},
					)
				}
			}
			assertMediaDs_Find := func(tc testCase) {
				if tc.expectMediaDsFind {
					mediaDSMock.EXPECT().Find(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(
						func(ctx context.Context, d *primitive.M, c db.IMongoClient) (*db.MediaFileDoc, errs.IMongoErr) {
							Expect(*d).To(Equal(*db.GetIDFilter(tc.jobDoc.MediaID)))
							// ...
							v := new(db.MediaFileDoc)
							err := new(error)
							if tc.mediaDsFindErr == nil {
								v = tc.mediaDoc
							} else {
								*err = tc.mediaDsFindErr
							}
							// ...
							return v, *err
						},
					)
				}
			}
			assertMediaDs_Replace := func(tc testCase) {
				if tc.expectMediaDsReplace {
					mediaDSMock.EXPECT().Replace(mock.Anything, mock.Anything, mock.Anything, mock.Anything).RunAndReturn(
						func(ctx context.Context, d *primitive.M, mfd *db.MediaFileDoc, c db.IMongoClient) (*db.MediaFileDoc, errs.IMongoErr) {
							Expect(*d).To(Equal(*db.GetIDFilter(tc.jobDoc.MediaID)))
							switch tc.jobDoc.Type {
							case db.SPRITEJobType:
								Expect(mfd.Sprite).NotTo(BeEmpty())
								Expect(mfd.Vtt).NotTo(BeEmpty())
								Expect(mfd.Sprite).NotTo(Equal(tc.mediaDoc.Sprite))
								Expect(mfd.Vtt).NotTo(Equal(tc.mediaDoc.Vtt))
							case db.THUMBNAILJobType:
								Expect(mfd.Thumbnail).NotTo(BeEmpty())
								Expect(mfd.Thumbnail).NotTo(Equal(tc.mediaDoc.Thumbnail))
							}
							// ...
							v := new(db.MediaFileDoc)
							err := new(error)
							// ...
							if tc.mediaDsReplaceErr == nil {
								v = mfd
							} else {
								*err = tc.mediaDsReplaceErr
							}
							// ...
							return v, *err
						},
					)
				}
			}
			assertMinio_FileAdd := func(tc testCase) func() {
				if tc.expectminioAddFile {
					err := new(error)
					if tc.minioAddFileErr != nil {
						*err = tc.minioAddFileErr
					}
					switch tc.jobDoc.Type {
					case db.SPRITEJobType:
						minioMock.EXPECT().FileAdd(mock.Anything, mock.Anything, tc.data.SpriteData).Return(*err)
						minioMock.EXPECT().FileAdd(mock.Anything, mock.Anything, tc.data.VttData).Return(*err)
					case db.THUMBNAILJobType:
						minioMock.EXPECT().FileAdd(mock.Anything, mock.Anything, tc.data.ThumbData).Return(*err)
					}
					return func() {
						fileNames := []string{}
						for _, call := range minioMock.Calls {
							if call.Method == "FileAdd" {
								fileNames = append(fileNames, call.Arguments.String(1))
							}
						}
						// ...
						expectedCall := mock.Call{}
						for _, call := range mediaDSMock.Calls {
							if call.Method == "Replace" {
								expectedCall = call
								break
							}
						}
						_doc := expectedCall.Arguments.Get(2).(*db.MediaFileDoc)
						expectedFileNames := []string{}
						switch tc.jobDoc.Type {
						case db.SPRITEJobType:
							expectedFileNames = append(expectedFileNames, _doc.Vtt)
							expectedFileNames = append(expectedFileNames, _doc.Sprite)
						case db.THUMBNAILJobType:
							expectedFileNames = append(expectedFileNames, _doc.Thumbnail)
						}
						// ...
						Expect(len(fileNames)).To(Equal(len(expectedFileNames)))
						Expect(fileNames).To(ContainElements(expectedFileNames))
					}
				}
				return func() {}
			}
			assertMinio_FileRm := func(tc testCase) {
				if tc.expectminioRmFile {
					err := new(error)
					if tc.minioRmFileErr != nil {
						*err = tc.minioRmFileErr
					}
					switch tc.jobDoc.Type {
					case db.SPRITEJobType:
						minioMock.EXPECT().FileRm(mock.Anything, tc.mediaDoc.Sprite).Return(*err)
						minioMock.EXPECT().FileRm(mock.Anything, tc.mediaDoc.Vtt).Return(*err)
					case db.THUMBNAILJobType:
						minioMock.EXPECT().FileRm(mock.Anything, tc.mediaDoc.Thumbnail).Return(*err)
					}
				}
			}
			newFakeJobDoc := func(jt db.JobType) *db.JobDoc {
				d := new(db.JobDoc)
				gofakeit.Struct(&d)
				d.Type = jt
				return d
			}
			newFakeMediaDoc := func(hasThumbnail bool, hasVtt bool, hasSprite bool) *db.MediaFileDoc {
				d := new(db.MediaFileDoc)
				gofakeit.Struct(&d)
				if !hasThumbnail {
					d.Thumbnail = ""
				}
				if !hasVtt {
					d.Vtt = ""
				}
				if !hasSprite {
					d.Sprite = ""
				}
				return d
			}
			Describe("Happy path", Label("Happy"), func() {
				BeforeEach(func() {
					resetMock()
				})

				AfterEach(func() {
					asserMockCall()
				})
				testCases := []testCase{
					{
						description: "Successfully done thumbnail (no thumbnail, no vtt, no sprite)",
						jobDoc:      newFakeJobDoc(db.THUMBNAILJobType),
						mediaDoc:    newFakeMediaDoc(false, false, false),
						data: facade.MediaMinioFile{
							ThumbData: []byte("thumb-data"),
						},
						expectJobDsFind:      true,
						expectJobDsDelete:    true,
						expectMediaDsFind:    true,
						expectMediaDsReplace: true,
						expectminioAddFile:   true,
						expectminioRmFile:    false,
					},
					{
						description: "Successfully done thumbnail",
						jobDoc:      newFakeJobDoc(db.THUMBNAILJobType),
						mediaDoc:    newFakeMediaDoc(true, true, true),
						data: facade.MediaMinioFile{
							ThumbData: []byte("thumb-data"),
						},
						expectJobDsFind:      true,
						expectJobDsDelete:    true,
						expectMediaDsFind:    true,
						expectMediaDsReplace: true,
						expectminioAddFile:   true,
						expectminioRmFile:    true,
					},
					{
						description: "Successfully done thumbnail with redunant data",
						jobDoc:      newFakeJobDoc(db.THUMBNAILJobType),
						mediaDoc:    newFakeMediaDoc(true, true, true),
						data: facade.MediaMinioFile{
							ThumbData:  []byte("thumb-data"),
							SpriteData: []byte("sprite-data"),
							VttData:    []byte("vtt-data"),
						},
						expectJobDsFind:      true,
						expectJobDsDelete:    true,
						expectMediaDsFind:    true,
						expectMediaDsReplace: true,
						expectminioAddFile:   true,
						expectminioRmFile:    true,
					},
					{
						description: "Successfully done sprite (no thumbnail, no vtt, no sprite)",
						jobDoc:      newFakeJobDoc(db.SPRITEJobType),
						mediaDoc:    newFakeMediaDoc(false, false, false),
						data: facade.MediaMinioFile{
							SpriteData: []byte("sprite-data"),
							VttData:    []byte("vtt-data"),
						},
						expectJobDsFind:      true,
						expectJobDsDelete:    true,
						expectMediaDsFind:    true,
						expectMediaDsReplace: true,
						expectminioAddFile:   true,
						expectminioRmFile:    false,
					},
					{
						description: "Successfully done sprite",
						jobDoc:      newFakeJobDoc(db.SPRITEJobType),
						mediaDoc:    newFakeMediaDoc(true, true, true),
						data: facade.MediaMinioFile{
							SpriteData: []byte("sprite-data"),
							VttData:    []byte("vtt-data"),
						},
						expectJobDsFind:      true,
						expectJobDsDelete:    true,
						expectMediaDsFind:    true,
						expectMediaDsReplace: true,
						expectminioAddFile:   true,
						expectminioRmFile:    true,
					},
					{
						description: "Successfully done sprite with redunant data",
						jobDoc:      newFakeJobDoc(db.SPRITEJobType),
						mediaDoc:    newFakeMediaDoc(true, true, true),
						data: facade.MediaMinioFile{
							ThumbData:  []byte("thumb-data"),
							SpriteData: []byte("sprite-data"),
							VttData:    []byte("vtt-data"),
						},
						expectJobDsFind:      true,
						expectJobDsDelete:    true,
						expectMediaDsFind:    true,
						expectMediaDsReplace: true,
						expectminioAddFile:   true,
						expectminioRmFile:    true,
					},
				}
				// ...

				for _, tc := range testCases {
					tc := tc
					It(tc.description, func() {
						// Arrange
						tc := tc
						// ...
						tc.mediaDoc.SetID(tc.jobDoc.MediaID)
						// ...
						assertMongo_GetClient(tc)
						// ...
						assertJobDs_Find(tc)
						assertJobDs_Delete(tc)
						// ...
						assertMediaDs_Replace(tc)
						assertMediaDs_Find(tc)
						// ...
						assertFileName := assertMinio_FileAdd(tc)
						assertMinio_FileRm(tc)
						// Act
						err := jobFacade.Done(testContext, tc.jobDoc.ID, mockClient, &tc.data)
						time.Sleep(10 * time.Millisecond) // wait for coroutines
						// Assert
						Expect(err).To(BeNil())
						assertFileName()
					})
				}
			})
			Describe("Failure path", Label("Failure"), func() {
				BeforeEach(func() {
					resetMock()
				})

				AfterEach(func() {
					asserMockCall()
				})
				testCases := []testCase{
					{
						description: "Fail on thumbnail job without thumb data",
						jobDoc:      newFakeJobDoc(db.THUMBNAILJobType),
						mediaDoc:    newFakeMediaDoc(false, false, false),
						data: facade.MediaMinioFile{
							SpriteData: []byte("sprite-data"),
							VttData:    []byte("vtt-data"),
						},
						expectJobDsFind:      true,
						expectJobDsDelete:    false,
						expectMediaDsFind:    false,
						expectMediaDsReplace: false,
						expectminioAddFile:   false,
						expectminioRmFile:    false,
						expectErr:            true,
					},
					{
						description: "Fail on sprite job without sprite data",
						jobDoc:      newFakeJobDoc(db.SPRITEJobType),
						mediaDoc:    newFakeMediaDoc(false, false, false),
						data: facade.MediaMinioFile{
							SpriteData: []byte("sprite-data"),
							ThumbData:  []byte("thumb-data"),
						},
						expectJobDsFind:      true,
						expectJobDsDelete:    false,
						expectMediaDsFind:    false,
						expectMediaDsReplace: false,
						expectminioAddFile:   false,
						expectminioRmFile:    false,
						expectErr:            true,
					},
					{
						description: "Fail on sprite job without vtt data",
						jobDoc:      newFakeJobDoc(db.SPRITEJobType),
						mediaDoc:    newFakeMediaDoc(false, false, false),
						data: facade.MediaMinioFile{
							VttData:   []byte("vtt-data"),
							ThumbData: []byte("thumb-data"),
						},
						expectJobDsFind:      true,
						expectJobDsDelete:    false,
						expectMediaDsFind:    false,
						expectMediaDsReplace: false,
						expectminioAddFile:   false,
						expectminioRmFile:    false,
						expectErr:            true,
					},
					{
						description: "Fail at jobDS.read",
						jobDoc:      newFakeJobDoc(db.THUMBNAILJobType),
						mediaDoc:    newFakeMediaDoc(false, false, false),
						data: facade.MediaMinioFile{
							ThumbData: []byte("thumb-data"),
						},
						expectJobDsFind:      true,
						expectJobDsDelete:    false,
						expectMediaDsFind:    false,
						expectMediaDsReplace: false,
						expectminioAddFile:   false,
						expectminioRmFile:    false,
						jobDsFindErr:         errs.NewMongoOpErr(fmt.Errorf("mock jobDSMock.Find err")),
						expectErr:            true,
					},
					{
						description: "successfully done while minio rm fail in goroutine",
						jobDoc:      newFakeJobDoc(db.THUMBNAILJobType),
						mediaDoc:    newFakeMediaDoc(true, true, true),
						data: facade.MediaMinioFile{
							ThumbData: []byte("thumb-data"),
						},
						expectJobDsFind:      true,
						expectJobDsDelete:    true,
						expectMediaDsFind:    true,
						expectMediaDsReplace: true,
						expectminioAddFile:   true,
						expectminioRmFile:    true,
						minioRmFileErr:       errs.NewMongoOpErr(fmt.Errorf("mock minio.RmFile err")),
						expectErr:            false,
					},
					{
						description: "successfully done while jobds.delete fail in goroutine",
						jobDoc:      newFakeJobDoc(db.THUMBNAILJobType),
						mediaDoc:    newFakeMediaDoc(true, true, true),
						data: facade.MediaMinioFile{
							ThumbData: []byte("thumb-data"),
						},
						expectJobDsFind:      true,
						expectJobDsDelete:    true,
						expectMediaDsFind:    true,
						expectMediaDsReplace: true,
						expectminioAddFile:   true,
						expectminioRmFile:    true,
						jobDsDeleteErr:       errs.NewMongoOpErr(fmt.Errorf("mock jobDS.Delete err")),
						expectErr:            false,
					},
					{
						description: "mongo JobDs.Find failure",
						jobDoc:      newFakeJobDoc(db.THUMBNAILJobType),
						mediaDoc:    newFakeMediaDoc(false, false, false),
						data: facade.MediaMinioFile{
							ThumbData: []byte("thumb-data"),
						},
						expectJobDsFind:      true,
						jobDsFindErr:         fmt.Errorf("mock jobDs.Find error"),
						expectJobDsDelete:    false,
						expectMediaDsFind:    false,
						expectMediaDsReplace: false,
						expectminioAddFile:   false,
						expectminioRmFile:    false,
						expectErr:            true,
					},
					{
						description: "success while mongo JobDs.Delete failure",
						jobDoc:      newFakeJobDoc(db.THUMBNAILJobType),
						mediaDoc:    newFakeMediaDoc(false, false, false),
						data: facade.MediaMinioFile{
							ThumbData: []byte("thumb-data"),
						},
						expectJobDsFind:      true,
						expectJobDsDelete:    true,
						jobDsDeleteErr:       fmt.Errorf("mock JobDs.Delete error"),
						expectMediaDsFind:    true,
						expectMediaDsReplace: true,
						expectminioAddFile:   true,
						expectminioRmFile:    false,
						expectErr:            false,
					},
					{
						description: "mongo MediaDs.Find failure",
						jobDoc:      newFakeJobDoc(db.THUMBNAILJobType),
						mediaDoc:    newFakeMediaDoc(false, false, false),
						data: facade.MediaMinioFile{
							ThumbData: []byte("thumb-data"),
						},
						expectJobDsFind:      true,
						expectJobDsDelete:    true,
						expectMediaDsFind:    true,
						mediaDsFindErr:       fmt.Errorf("mock mediaDs.find error"),
						expectMediaDsReplace: false,
						expectminioAddFile:   false,
						expectminioRmFile:    false,
						expectErr:            false,
					},
					{
						description: "success while mongo MediaDs.Replace failure",
						jobDoc:      newFakeJobDoc(db.THUMBNAILJobType),
						mediaDoc:    newFakeMediaDoc(false, false, false),
						data: facade.MediaMinioFile{
							ThumbData: []byte("thumb-data"),
						},
						expectJobDsFind:      true,
						expectJobDsDelete:    true,
						expectMediaDsFind:    true,
						expectMediaDsReplace: true,
						mediaDsReplaceErr:    fmt.Errorf("mock MediaDs.Replace error"),
						expectminioAddFile:   true,
						expectminioRmFile:    false,
						expectErr:            false,
					},
				}
				// ...

				for _, tc := range testCases {
					tc := tc
					It(tc.description, func() {
						// Arrange
						tc := tc
						// ...
						tc.mediaDoc.SetID(tc.jobDoc.MediaID)
						// ...
						assertMongo_GetClient(tc)
						// ...
						assertJobDs_Find(tc)
						assertJobDs_Delete(tc)
						// ...
						assertMediaDs_Replace(tc)
						assertMediaDs_Find(tc)
						// ...
						assertFileName := assertMinio_FileAdd(tc)
						assertMinio_FileRm(tc)
						// Act
						err := jobFacade.Done(testContext, tc.jobDoc.ID, mockClient, &tc.data)
						time.Sleep(10 * time.Millisecond) // wait for coroutines
						// Assert
						if tc.expectErr {
							Expect(err).ToNot(BeNil())
						} else {
							Expect(err).To(BeNil())
						}
						assertFileName()
					})
				}
			})
		})

	})
})
