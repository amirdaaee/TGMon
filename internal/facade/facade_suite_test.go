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
	newMongoClient := func() (*mongo.Client, error) {
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
						func(ctx context.Context, d *primitive.D, mfd *db.MediaFileDoc, c *mongo.Client) (*db.MediaFileDoc, errs.IMongoErr) {
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
						func(ctx context.Context, jd *db.JobDoc, c *mongo.Client) (*db.JobDoc, errs.IMongoErr) {
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
				d.SetID(primitive.NilObjectID)
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
				filter      *primitive.D       // filter to call Read
				outputDoc   []*db.MediaFileDoc // result docs of ds.list
				listErr     bool               // error calling ds.list
			}
			assertMediaDs_List := func(tc testCase) {
				v := new([]*db.MediaFileDoc)
				err := new(error)
				if tc.listErr {
					*err = fmt.Errorf("mock mediaDSMock.List err")
				} else {
					*v = tc.outputDoc
				}
				mediaDSMock.EXPECT().List(mock.Anything, tc.filter, mock.Anything).Return(tc.outputDoc, *err)
			}
			newBsonEmptyFilter := func() *primitive.D {
				return &primitive.D{}
			}
			newBsonIDFilter := func() *primitive.D {
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
						listErr:     true,
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
				filter             primitive.D      // filter to call Read
				outputDoc          *db.MediaFileDoc // result doc of ds.find
				withThumbMedia     bool             // doc has Thumbnail
				withVttMedia       bool             // doc has Vtt
				withSpriteMedia    bool             // doc has Sprite
				findErr            error            // error calling jobDs.find
				jobDeleteManyError bool             // error calling jobDs.deleteMany
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
					jobDSMock.EXPECT().DeleteMany(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, d *primitive.D, c *mongo.Client) errs.IMongoErr {
						Expect(*d).To(BeEquivalentTo(primitive.D{{Key: "MediaID", Value: tc.outputDoc.ID}}))
						err := new(error)
						if tc.jobDeleteManyError {
							*err = fmt.Errorf("mock mediaDSMock.DeleteMany err")
						}
						return *err
					})
				}
			}
			assertMinioMock_RmAdd := func(tc testCase) {
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
			newBsonIDFilter := func() *primitive.D {
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
						assertMinioMock_RmAdd(tc)
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
						description:        "Successfully Delete media while error creating job",
						outputDoc:          newFakeMediaDoc(),
						filter:             *newBsonIDFilter(),
						withThumbMedia:     true,
						withVttMedia:       true,
						withSpriteMedia:    true,
						expectDelete:       true,
						expectJob:          true,
						expectMinioRm:      true,
						jobDeleteManyError: true,
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
						assertMinioMock_RmAdd(tc)
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
				jobDsListErr      error      // error calling ds.list
				jobDsCreateErr    error      // error calling ds.create
				expectErr         error
			}
			assertJobDs_List := func(tc testCase) {
				if tc.expectJobDsList {
					jobDSMock.EXPECT().List(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(
						func(ctx context.Context, d *primitive.D, c *mongo.Client) ([]*db.JobDoc, errs.IMongoErr) {
							filterObj := db.JobDoc{MediaID: tc.inputDoc.MediaID, Type: tc.inputDoc.Type}
							expectedFilter, _err := db.MarshalOmitEmpty(&filterObj)
							Expect(_err).To(BeNil())
							Expect(len(*d)).Should(Equal(len(*expectedFilter)))
							Expect(*d).To(ContainElements(*expectedFilter))
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
						func(ctx context.Context, jd *db.JobDoc, c *mongo.Client) (*db.JobDoc, errs.IMongoErr) {
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
				d.SetID(primitive.NilObjectID)
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
						jobDsListErr:    errs.NewMongoOpErr(fmt.Errorf("mock jobDSMock.List err")),
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
				filter       *primitive.D // filter to call Read
				outputDoc    []*db.JobDoc // result docs of ds.list
				jobDsListErr error        // error calling ds.list
				expectErr    error
			}
			assertJobDs_List := func(tc testCase) {
				jobDSMock.EXPECT().List(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(
					func(ctx context.Context, d *primitive.D, c *mongo.Client) ([]*db.JobDoc, errs.IMongoErr) {
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
			newBsonEmptyFilter := func() *primitive.D {
				return &primitive.D{}
			}
			newBsonIDFilter := func() *primitive.D {
				return db.GetIDFilter(primitive.NewObjectID())
			}
			newFakeJobDoc := func(jt db.JobType) *db.JobDoc {
				d := new(db.JobDoc)
				gofakeit.Struct(&d)
				d.SetID(primitive.NilObjectID)
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
						jobDsListErr: errs.NewMongoOpErr(fmt.Errorf("mock jobDSMock.List err")),
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

	})
})
