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
	newMongoClient := func() (*mongo.Client, error) {
		return &mongo.Client{}, nil
	}
	BeforeEach(func() {
		mockClient = &mongo.Client{}
		testContext = context.Background()
	})

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
			newFakeDoc := func() *db.MediaFileDoc {
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
						inputDoc:       newFakeDoc(),
						inputThumb:     []byte{0x01, 0x02},
						expectJob:      true,
						expectMinioAdd: true,
					},
					{
						description:    "Successfully create media document without thumbnail",
						inputDoc:       newFakeDoc(),
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
						Expect(res).NotTo(BeNil())
						Expect(err).To(BeNil())
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
						inputDoc:       newFakeDoc(),
						inputThumb:     []byte{0x01, 0x02},
						jobCreateError: true,
						expectJob:      true,
						expectMinioAdd: true,
					},
					{
						description:    "Successfully create media document while error adding thumbnail",
						inputDoc:       newFakeDoc(),
						inputThumb:     []byte{0x01, 0x02},
						minioAddError:  true,
						expectJob:      true,
						expectMinioAdd: true,
					},
					{
						description:    "Successfully create media document while error replacing while adding thumbnail",
						inputDoc:       newFakeDoc(),
						inputThumb:     []byte{0x01, 0x02},
						replaceErr:     true,
						expectJob:      true,
						expectMinioAdd: true,
					},
					{
						description: "Error create doc (datastore)",
						inputDoc:    newFakeDoc(),
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
						} else {
							Expect(res).NotTo(BeNil())
							Expect(err).To(BeNil())
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
			newFakeDoc := func() *db.MediaFileDoc {
				d := new(db.MediaFileDoc)
				gofakeit.Struct(&d)
				return d
			}
			newManyFakeDoc := func(n uint) []*db.MediaFileDoc {
				res := []*db.MediaFileDoc{}
				for range n {
					res = append(res, newFakeDoc())
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
						Expect(res).To(Equal(tc.outputDoc))
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
			newFakeDoc := func() *db.MediaFileDoc {
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
						outputDoc:       newFakeDoc(),
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
						outputDoc:      newFakeDoc(),
						filter:         *newBsonIDFilter(),
						withThumbMedia: true,
						withVttMedia:   true,
						expectDelete:   true,
						expectJob:      true,
						expectMinioRm:  true,
					},
					{
						description:     "Successfully Delete media with all minio src (thumb+sprite)",
						outputDoc:       newFakeDoc(),
						filter:          *newBsonIDFilter(),
						withThumbMedia:  true,
						withSpriteMedia: true,
						expectDelete:    true,
						expectJob:       true,
						expectMinioRm:   true,
					},
					{
						description:     "Successfully Delete media with all minio src (vtt+sprite)",
						outputDoc:       newFakeDoc(),
						filter:          *newBsonIDFilter(),
						withVttMedia:    true,
						withSpriteMedia: true,
						expectDelete:    true,
						expectJob:       true,
						expectMinioRm:   true,
					},
					{
						description:    "Successfully Delete media with all minio src (thumb)",
						outputDoc:      newFakeDoc(),
						filter:         *newBsonIDFilter(),
						withThumbMedia: true,
						expectDelete:   true,
						expectJob:      true,
						expectMinioRm:  true,
					},
					{
						description:   "Successfully Delete media with all minio src (vtt)",
						outputDoc:     newFakeDoc(),
						filter:        *newBsonIDFilter(),
						withVttMedia:  true,
						expectDelete:  true,
						expectJob:     true,
						expectMinioRm: true,
					},
					{
						description:     "Successfully Delete media with all minio src (sprite)",
						outputDoc:       newFakeDoc(),
						filter:          *newBsonIDFilter(),
						withSpriteMedia: true,
						expectDelete:    true,
						expectJob:       true,
						expectMinioRm:   true,
					},
					{
						description:   "Successfully Delete media without minio src",
						outputDoc:     newFakeDoc(),
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
						outputDoc:          newFakeDoc(),
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
						outputDoc:       newFakeDoc(),
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
						outputDoc:       newFakeDoc(),
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
})
