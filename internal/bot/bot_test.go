package bot_test

import (
	"context"
	"fmt"

	"github.com/amirdaaee/TGMon/internal/bot"
	mockBot "github.com/amirdaaee/TGMon/mocks/bot"
	"github.com/gotd/td/tg"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	"github.com/stretchr/testify/mock"
)

var _ = Describe("Bot", func() {
	var testContext context.Context
	var testChannelId int64 = 100
	var (
		tgClMock *mockBot.MockIClient
	)
	//
	resetTgClMock := func() {
		tgClMock = mockBot.NewMockIClient(GinkgoT())
		tgClMock.EXPECT().GetChannelID().Return(testChannelId).Maybe()
	}
	asserTgClMockCall := func() {
		tgClMock.AssertExpectations(GinkgoT())
	}

	assertTgCl_ChannelsGetChannels := func(ret tg.ChatClass, err error) {
		tgClMock.EXPECT().ChannelsGetChannels(mock.Anything, mock.Anything).RunAndReturn(
			func(ctx context.Context, icc []tg.InputChannelClass) (tg.MessagesChatsClass, error) {
				Expect(icc).To(HaveLen(1))
				Expect(icc[0]).To(HaveField("ChannelID", testChannelId))
				if err != nil {
					return nil, err
				}
				if ret == nil {
					ret = &tg.Channel{ID: testChannelId}
				}
				return &tg.MessagesChatsSlice{Count: 1, Chats: []tg.ChatClass{ret}}, err
			},
		)
	}
	assertTgCl_ChannelsGetMessages := func(inputIds []int, ret tg.MessagesMessagesClass, err error) {
		tgClMock.EXPECT().ChannelsGetMessages(mock.Anything, mock.Anything).RunAndReturn(
			func(ctx context.Context, cgmr *tg.ChannelsGetMessagesRequest) (tg.MessagesMessagesClass, error) {
				expectedHaveField := []types.GomegaMatcher{}
				for _, i := range inputIds {
					expectedHaveField = append(expectedHaveField, HaveField("ID", i))
				}
				Expect(cgmr.ID).To(ConsistOf(expectedHaveField))
				if err != nil {
					return nil, err
				}
				if ret == nil {
					_ret := tg.MessagesChannelMessages{Count: len(inputIds), Messages: []tg.MessageClass{}}
					for _, i := range inputIds {
						_ret.Messages = append(_ret.Messages, &tg.Message{ID: i})
					}
					ret = &_ret
				}
				return ret, nil
			},
		)
	}
	assertTgCl_DeleteMessages := func(inputIds []int, err error) {
		tgClMock.EXPECT().DeleteMessages(mock.Anything).RunAndReturn(func(i []int) error {
			Expect(i).To(ConsistOf(inputIds))
			return err
		})
	}
	Describe("Worker", Label("Worker"), func() {
		resetMock := func() {
			resetTgClMock()
		}
		asserMockCall := func() {
			asserTgClMockCall()
		}
		mockClientFactory := func(string, *bot.SessionConfig) bot.IClient {
			return tgClMock
		}
		workerFactory := func() *bot.Worker {
			sessCfg := bot.SessionConfig{
				SocksProxy: "",
				SessionDir: "",
				ChannelId:  testChannelId,
			}
			w, e := bot.NewWorker(mockClientFactory("", &sessCfg))
			Expect(e).ToNot(HaveOccurred())
			return w
		}
		Describe("GetChannel", Label("GetChannel"), func() {
			type testCase struct {
				description                string
				tType                      TestCaseType
				tgClChannelsGetChannelsRes tg.ChatClass // returrned value from ChannelsGetChannels
				tgClChannelsGetChannelsErr error        // error calling ChannelsGetChannels
				expectErr                  bool         // whether or not expect failure
			}
			// ...
			BeforeEach(func() {
				resetMock()
			})

			AfterEach(func() {
				asserMockCall()
			})
			// ...
			tests := []testCase{
				{
					description: "successfully get channel",
					tType:       HAPPY_PATH,
				},
				{
					description:                "chat is not channel",
					tType:                      FAILURE,
					tgClChannelsGetChannelsRes: &tg.ChatEmpty{ID: testChannelId},

					expectErr: true,
				},
				{
					description:                "chat id missmatch",
					tType:                      FAILURE,
					tgClChannelsGetChannelsRes: &tg.Channel{ID: -10},
					expectErr:                  true,
				},
				{
					description:                "error calling ChannelsGetChannels",
					tType:                      FAILURE,
					tgClChannelsGetChannelsErr: fmt.Errorf("mock ChannelsGetChannels error"),
					expectErr:                  true,
				},
			}
			// ...
			for _, tc := range tests {
				tc := tc
				It(tc.description, Label(string(tc.tType)), func() {
					// Arrange
					wrkr := workerFactory()
					assertTgCl_ChannelsGetChannels(tc.tgClChannelsGetChannelsRes, tc.tgClChannelsGetChannelsErr)
					// Act
					res, err := wrkr.GetChannel(testContext)
					// Assert
					if tc.expectErr {
						Expect(err).To(HaveOccurred())
						GinkgoWriter.Println(err.Error())
					} else {
						Expect(err).NotTo(HaveOccurred())
						Expect(res).NotTo(BeNil())
					}
				})
			}
		})
		Describe("GetChannelMessages", Label("GetChannelMessages"), func() {
			type testCase struct {
				description                 string
				tType                       TestCaseType
				msgIDs                      []int
				tgClChannelsGetMessagesCall bool                     // whether or not expect call ChannelsGetMessages
				tgClChannelsGetMessagesRet  tg.MessagesMessagesClass // returrned value from ChannelsGetMessages
				tgClChannelsGetMessagesErr  error                    // error calling ChannelsGetMessages
				tgClChannelsGetChannelsErr  error                    // error calling ChannelsGetChannels
				expectErr                   bool                     // whether or not expect failure
			}
			// ...
			BeforeEach(func() {
				resetMock()
			})

			AfterEach(func() {
				asserMockCall()
			})
			// ...
			tests := []testCase{
				{
					description:                 "successfully get empty message",
					tType:                       HAPPY_PATH,
					msgIDs:                      []int{},
					tgClChannelsGetMessagesCall: true,
				},
				{
					description:                 "successfully get single message",
					tType:                       HAPPY_PATH,
					msgIDs:                      []int{1},
					tgClChannelsGetMessagesCall: true,
				},
				{
					description:                 "successfully get multiple message",
					tType:                       HAPPY_PATH,
					msgIDs:                      []int{1, 2, 3, 4, 5},
					tgClChannelsGetMessagesCall: true,
				},
				{
					description:                "error calling ChannelsGetChannels",
					tType:                      FAILURE,
					tgClChannelsGetChannelsErr: fmt.Errorf("mock ChannelsGetChannels error"),
					expectErr:                  true,
				},
				{
					description:                 "error calling ChannelsGetMessages",
					tType:                       FAILURE,
					msgIDs:                      []int{1, 2, 3, 4, 5},
					tgClChannelsGetMessagesCall: true,
					tgClChannelsGetMessagesErr:  fmt.Errorf("mock ChannelsGetMessages error"),
					expectErr:                   true,
				},
				{
					description:                 "unexpected result from ChannelsGetMessages",
					tType:                       FAILURE,
					msgIDs:                      []int{1, 2, 3, 4, 5},
					tgClChannelsGetMessagesCall: true,
					tgClChannelsGetMessagesRet:  &tg.MessagesMessages{},
					expectErr:                   true,
				},
			}
			// ...
			for _, tc := range tests {
				tc := tc
				It(tc.description, Label(string(tc.tType)), func() {
					// Arrange
					wrkr := workerFactory()
					assertTgCl_ChannelsGetChannels(nil, tc.tgClChannelsGetChannelsErr)
					if tc.tgClChannelsGetMessagesCall {
						assertTgCl_ChannelsGetMessages(tc.msgIDs, tc.tgClChannelsGetMessagesRet, tc.tgClChannelsGetMessagesErr)
					}
					// Act
					res, err := wrkr.GetChannelMessages(testContext, tc.msgIDs)
					// Assert
					if tc.expectErr {
						Expect(err).To(HaveOccurred())
						GinkgoWriter.Println(err.Error())
					} else {
						Expect(err).NotTo(HaveOccurred())
						Expect(res).NotTo(BeNil())
					}
				})
			}
		})
		Describe("DeleteMessages", Label("DeleteMessages"), func() {
			type testCase struct {
				description           string
				tType                 TestCaseType
				msgIDs                []int
				tgClDeleteMessagesErr error // error calling DeleteMessages
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
			tests := []testCase{
				{
					description: "successfully delete empty message",
					tType:       HAPPY_PATH,
					msgIDs:      []int{},
				},
				{
					description: "successfully delete single message",
					tType:       HAPPY_PATH,
					msgIDs:      []int{1},
				},
				{
					description: "successfully delete multiple message",
					tType:       HAPPY_PATH,
					msgIDs:      []int{1, 2, 3, 4, 5},
				},
				{
					description:           "error calling DeleteMessages",
					tType:                 FAILURE,
					msgIDs:                []int{1, 2, 3, 4, 5},
					tgClDeleteMessagesErr: fmt.Errorf("mock DeleteMessages error"),
					expectErr:             true,
				},
			}
			// ...
			for _, tc := range tests {
				tc := tc
				It(tc.description, Label(string(tc.tType)), func() {
					// Arrange
					wrkr := workerFactory()
					assertTgCl_DeleteMessages(tc.msgIDs, tc.tgClDeleteMessagesErr)
					// Act
					err := wrkr.DeleteMessages(tc.msgIDs)
					// Assert
					if tc.expectErr {
						Expect(err).To(HaveOccurred())
						GinkgoWriter.Println(err.Error())
					} else {
						Expect(err).NotTo(HaveOccurred())
					}
				})
			}
		})
	})
})
