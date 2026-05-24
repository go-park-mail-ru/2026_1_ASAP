package messages

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	domain "github.com/go-park-mail-ru/2026_1_ASAP/internal/chat/domain/chat"
	dto "github.com/go-park-mail-ru/2026_1_ASAP/internal/chat/dto/message"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/chat/usecase/messages/mock"
)

func TestPositiveMessageService_SendMessageEscapesHTML(t *testing.T) {
	type fields struct {
		msgRepo  *mock.MockMessageRepositoryInterface
		chatRepo *mock.MockChatRepositoryInterface
	}

	type args struct {
		ctx    context.Context
		userID int64
		chatID int64
		req    *dto.RequestSendMessage
	}

	tests := []struct {
		name     string
		prepare  func(*fields)
		args     args
		wantText string
	}{
		{
			name: "img onerror",
			prepare: func(f *fields) {
				f.chatRepo.EXPECT().IsMember(context.Background(), int64(10), int64(55)).Return(true, nil)
				f.chatRepo.EXPECT().GetChatByID(context.Background(), int64(10)).Return(&domain.Chat{
					Id:   10,
					Type: domain.ChatTypeGroup,
				}, nil)
				f.msgRepo.EXPECT().
					CreateMessage(context.Background(), &domain.Message{
						ChatId:   10,
						SenderId: 55,
						Content:  `<img src=x onerror=alert(1)>`,
					}).
					Return(&domain.Message{
						Id:        1,
						ChatId:    10,
						SenderId:  55,
						Content:   `<img src=x onerror=alert(1)>`,
						CreatedAt: time.Unix(1700000000, 0).UTC(),
					}, nil)
			},
			args: args{
				ctx:    context.Background(),
				userID: 55,
				chatID: 10,
				req: &dto.RequestSendMessage{
					ChatID: 10,
					Text:   `<img src=x onerror=alert(1)>`,
				},
			},
			wantText: `&lt;img src=x onerror=alert(1)&gt;`,
		},
		{
			name: "ampersand",
			prepare: func(f *fields) {
				f.chatRepo.EXPECT().IsMember(context.Background(), int64(3), int64(2)).Return(true, nil)
				f.chatRepo.EXPECT().GetChatByID(context.Background(), int64(3)).Return(&domain.Chat{
					Id:   3,
					Type: domain.ChatTypeGroup,
				}, nil)
				f.msgRepo.EXPECT().
					CreateMessage(context.Background(), &domain.Message{
						ChatId:   3,
						SenderId: 2,
						Content:  `a & b`,
					}).
					Return(&domain.Message{
						Id:        99,
						ChatId:    3,
						SenderId:  2,
						Content:   `a & b`,
						CreatedAt: time.Unix(1700000001, 0).UTC(),
					}, nil)
			},
			args: args{
				ctx:    context.Background(),
				userID: 2,
				chatID: 3,
				req: &dto.RequestSendMessage{
					ChatID: 3,
					Text:   `a & b`,
				},
			},
			wantText: `a &amp; b`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				msgRepo:  mock.NewMockMessageRepositoryInterface(ctrl),
				chatRepo: mock.NewMockChatRepositoryInterface(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := NewMessageService(f.msgRepo, f.chatRepo, nil, nil, "http://localhost:8088")
			resp, err := s.SendMessage(tt.args.ctx, tt.args.userID, tt.args.chatID, tt.args.req)
			require.NoError(t, err)
			require.Equal(t, tt.wantText, resp.Text)
		})
	}
}

func TestNegativeMessageService_SendMessage(t *testing.T) {
	type fields struct {
		msgRepo  *mock.MockMessageRepositoryInterface
		chatRepo *mock.MockChatRepositoryInterface
	}

	type args struct {
		ctx    context.Context
		userID int64
		chatID int64
		req    *dto.RequestSendMessage
	}

	longText := make([]byte, 2001)
	for i := range longText {
		longText[i] = 'a'
	}

	tests := []struct {
		wantErr    error
		prepare    func(*fields)
		name       string
		args       args
		wantAnyErr bool
		wantSubstr string
	}{
		{
			name:       "nil request",
			args:       args{ctx: context.Background(), userID: 1, chatID: 1, req: nil},
			wantAnyErr: true,
			wantSubstr: "send message nil request",
		},
		{
			name:    "empty message",
			args:    args{ctx: context.Background(), userID: 1, chatID: 1, req: &dto.RequestSendMessage{Text: ""}},
			wantErr: domain.ErrMessageEmpty,
		},
		{
			name:    "message too long",
			args:    args{ctx: context.Background(), userID: 1, chatID: 1, req: &dto.RequestSendMessage{Text: string(longText)}},
			wantErr: domain.ErrMessageTooLong,
		},
		{
			name: "user not member",
			prepare: func(f *fields) {
				f.chatRepo.EXPECT().IsMember(context.Background(), int64(1), int64(10)).Return(false, nil)
			},
			args:    args{ctx: context.Background(), userID: 10, chatID: 1, req: &dto.RequestSendMessage{Text: "hi"}},
			wantErr: domain.ErrMessageNotMember,
		},
		{
			name: "channel only owner can send",
			prepare: func(f *fields) {
				f.chatRepo.EXPECT().IsMember(context.Background(), int64(1), int64(10)).Return(true, nil)
				f.chatRepo.EXPECT().GetChatByID(context.Background(), int64(1)).Return(&domain.Chat{
					Id:      1,
					Type:    domain.ChatTypeChannel,
					OwnerId: 99,
				}, nil)
			},
			args:    args{ctx: context.Background(), userID: 10, chatID: 1, req: &dto.RequestSendMessage{Text: "hi"}},
			wantErr: domain.ErrOnlyOwnerCanSendMessaage,
		},
		{
			name: "repo create error",
			prepare: func(f *fields) {
				f.chatRepo.EXPECT().IsMember(context.Background(), int64(1), int64(10)).Return(true, nil)
				f.chatRepo.EXPECT().GetChatByID(context.Background(), int64(1)).Return(&domain.Chat{
					Id:   1,
					Type: domain.ChatTypeGroup,
				}, nil)
				f.msgRepo.EXPECT().CreateMessage(context.Background(), &domain.Message{
					ChatId:   1,
					SenderId: 10,
					Content:  "hi",
				}).Return(nil, errors.New("db down"))
			},
			args:       args{ctx: context.Background(), userID: 10, chatID: 1, req: &dto.RequestSendMessage{Text: "hi"}},
			wantAnyErr: true,
			wantSubstr: "messageRepo create message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				msgRepo:  mock.NewMockMessageRepositoryInterface(ctrl),
				chatRepo: mock.NewMockChatRepositoryInterface(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := NewMessageService(f.msgRepo, f.chatRepo, nil, nil, "http://localhost:8088")
			resp, err := s.SendMessage(tt.args.ctx, tt.args.userID, tt.args.chatID, tt.args.req)
			require.Nil(t, resp)
			if tt.wantAnyErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.wantSubstr)
			} else {
				require.EqualError(t, err, tt.wantErr.Error())
			}
		})
	}
}

func TestPositiveMessageService_GetMessagesByChatId(t *testing.T) {
	type fields struct {
		msgRepo  *mock.MockMessageRepositoryInterface
		chatRepo *mock.MockChatRepositoryInterface
	}

	type args struct {
		ctx    context.Context
		userID int64
		chatID int64
		req    *dto.RequestGetMessages
	}

	now := time.Unix(1700001000, 0).UTC()
	beforeID := int64(50)

	tests := []struct {
		prepare func(*fields)
		want    *dto.ResponseGetMessages
		name    string
		args    args
	}{
		{
			name: "returns messages with has more",
			prepare: func(f *fields) {
				f.chatRepo.EXPECT().GetChatByID(context.Background(), int64(1)).Return(&domain.Chat{
					Id:   1,
					Type: domain.ChatTypeGroup,
				}, nil)
				f.chatRepo.EXPECT().IsMember(context.Background(), int64(1), int64(10)).Return(true, nil)
				f.msgRepo.EXPECT().GetMessagesByChatId(context.Background(), int64(1), &beforeID, 3).Return([]*domain.Message{
					{Id: 5, ChatId: 1, SenderId: 10, Content: "one", CreatedAt: now},
					{Id: 4, ChatId: 1, SenderId: 11, Content: "two", CreatedAt: now.Add(-time.Minute), Edited: true},
					{Id: 3, ChatId: 1, SenderId: 12, Content: "three", CreatedAt: now.Add(-2 * time.Minute)},
				}, nil)
				f.chatRepo.EXPECT().GetMemberLastReads(context.Background(), int64(1)).Return(map[int64]*int64{
					10: ptrInt64(50),
					11: ptrInt64(5),
					12: ptrInt64(5),
				}, nil)
				f.msgRepo.EXPECT().GetAttachmentsByMessageIDs(gomock.Any(), []int64{5, 4}).Return(map[int64][]domain.MessageAttachment{}, nil)
			},
			args: args{
				ctx:    context.Background(),
				userID: 10,
				chatID: 1,
				req:    &dto.RequestGetMessages{BeforeID: &beforeID, Limit: 2},
			},
			want: &dto.ResponseGetMessages{
				Messages: []dto.MessageDTO{
					{ID: 5, ChatID: 1, SenderID: 10, Text: "one", CreatedAt: now, Edited: false, Read: true},
					{ID: 4, ChatID: 1, SenderID: 11, Text: "two", CreatedAt: now.Add(-time.Minute), Edited: true, Read: false},
				},
				NextBeforeID: ptrInt64(4),
				HasMore:      true,
			},
		},
		{
			name: "public channel available without membership",
			prepare: func(f *fields) {
				f.chatRepo.EXPECT().GetChatByID(context.Background(), int64(2)).Return(&domain.Chat{
					Id:   2,
					Type: domain.ChatTypeChannel,
				}, nil)
				f.chatRepo.EXPECT().IsMember(context.Background(), int64(2), int64(77)).Return(false, nil)
				f.msgRepo.EXPECT().GetMessagesByChatId(context.Background(), int64(2), (*int64)(nil), 21).Return([]*domain.Message{
					{Id: 9, ChatId: 2, SenderId: 1, Content: "post", CreatedAt: now},
				}, nil)
				f.chatRepo.EXPECT().GetMemberLastReads(context.Background(), int64(2)).Return(map[int64]*int64{
					1: ptrInt64(100),
				}, nil)
				f.msgRepo.EXPECT().GetAttachmentsByMessageIDs(gomock.Any(), []int64{9}).Return(map[int64][]domain.MessageAttachment{}, nil)
			},
			args: args{
				ctx:    context.Background(),
				userID: 77,
				chatID: 2,
				req:    &dto.RequestGetMessages{Limit: 0},
			},
			want: &dto.ResponseGetMessages{
				Messages: []dto.MessageDTO{
					{ID: 9, ChatID: 2, SenderID: 1, Text: "post", CreatedAt: now, Edited: false, Read: false},
				},
				NextBeforeID: ptrInt64(9),
				HasMore:      false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				msgRepo:  mock.NewMockMessageRepositoryInterface(ctrl),
				chatRepo: mock.NewMockChatRepositoryInterface(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := NewMessageService(f.msgRepo, f.chatRepo, nil, nil, "http://localhost:8088")
			got, err := s.GetMessagesByChatId(tt.args.ctx, tt.args.userID, tt.args.chatID, tt.args.req)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestNegativeMessageService_GetMessagesByChatId(t *testing.T) {
	type fields struct {
		msgRepo  *mock.MockMessageRepositoryInterface
		chatRepo *mock.MockChatRepositoryInterface
	}

	type args struct {
		ctx    context.Context
		userID int64
		chatID int64
		req    *dto.RequestGetMessages
	}

	tests := []struct {
		wantErr    error
		prepare    func(*fields)
		name       string
		args       args
		wantAnyErr bool
		wantSubstr string
	}{
		{
			name:       "nil request",
			args:       args{ctx: context.Background(), userID: 1, chatID: 1, req: nil},
			wantAnyErr: true,
			wantSubstr: "get messages nil request",
		},
		{
			name: "not member in non channel",
			prepare: func(f *fields) {
				f.chatRepo.EXPECT().GetChatByID(context.Background(), int64(1)).Return(&domain.Chat{
					Id:   1,
					Type: domain.ChatTypeGroup,
				}, nil)
				f.chatRepo.EXPECT().IsMember(context.Background(), int64(1), int64(10)).Return(false, nil)
			},
			args:    args{ctx: context.Background(), userID: 10, chatID: 1, req: &dto.RequestGetMessages{Limit: 20}},
			wantErr: domain.ErrMessageNotMember,
		},
		{
			name: "repo get messages error",
			prepare: func(f *fields) {
				f.chatRepo.EXPECT().GetChatByID(context.Background(), int64(1)).Return(&domain.Chat{
					Id:   1,
					Type: domain.ChatTypeGroup,
				}, nil)
				f.chatRepo.EXPECT().IsMember(context.Background(), int64(1), int64(10)).Return(true, nil)
				f.msgRepo.EXPECT().GetMessagesByChatId(context.Background(), int64(1), (*int64)(nil), 21).
					Return(nil, errors.New("db down"))
			},
			args:       args{ctx: context.Background(), userID: 10, chatID: 1, req: &dto.RequestGetMessages{Limit: 20}},
			wantAnyErr: true,
			wantSubstr: "messageRepo get messages",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				msgRepo:  mock.NewMockMessageRepositoryInterface(ctrl),
				chatRepo: mock.NewMockChatRepositoryInterface(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := NewMessageService(f.msgRepo, f.chatRepo, nil, nil, "http://localhost:8088")
			got, err := s.GetMessagesByChatId(tt.args.ctx, tt.args.userID, tt.args.chatID, tt.args.req)
			require.Nil(t, got)
			if tt.wantAnyErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.wantSubstr)
			} else {
				require.EqualError(t, err, tt.wantErr.Error())
			}
		})
	}
}

func TestPositiveMessageService_EditMessage(t *testing.T) {
	type fields struct {
		msgRepo  *mock.MockMessageRepositoryInterface
		chatRepo *mock.MockChatRepositoryInterface
	}

	type args struct {
		ctx    context.Context
		userID int64
		chatID int64
		req    *dto.RequestEditMessage
	}

	now := time.Unix(1700002000, 0).UTC()

	tests := []struct {
		prepare func(*fields)
		want    *dto.ResponseEditMessage
		name    string
		args    args
	}{
		{
			name: "edit last message",
			prepare: func(f *fields) {
				f.chatRepo.EXPECT().IsMember(context.Background(), int64(1), int64(10)).Return(true, nil)
				f.msgRepo.EXPECT().UpdateMessage(context.Background(), &domain.Message{
					Id:       7,
					ChatId:   1,
					SenderId: 10,
					Content:  "edited <b>text</b>",
				}).Return(&domain.Message{
					Id:        7,
					ChatId:    1,
					SenderId:  10,
					Content:   "edited <b>text</b>",
					CreatedAt: now,
					Edited:    true,
				}, true, nil)
				f.chatRepo.EXPECT().GetMemberLastReads(context.Background(), int64(1)).Return(map[int64]*int64{
					10: ptrInt64(7),
					11: ptrInt64(7),
					12: ptrInt64(7),
				}, nil)
			},
			args: args{
				ctx:    context.Background(),
				userID: 10,
				chatID: 1,
				req:    &dto.RequestEditMessage{MessageID: 7, ChatID: 1, Text: "edited <b>text</b>"},
			},
			want: &dto.ResponseEditMessage{
				ID:                7,
				ChatID:            1,
				SenderID:          10,
				Text:              "edited &lt;b&gt;text&lt;/b&gt;",
				CreatedAt:         now,
				Edited:            true,
				Read:              true,
				LastMessageEdited: true,
				LastMessage: &dto.LastMessageDTO{
					SenderId:  10,
					Text:      "edited &lt;b&gt;text&lt;/b&gt;",
					CreatedAt: now,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				msgRepo:  mock.NewMockMessageRepositoryInterface(ctrl),
				chatRepo: mock.NewMockChatRepositoryInterface(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := NewMessageService(f.msgRepo, f.chatRepo, nil, nil, "http://localhost:8088")
			got, err := s.EditMessage(tt.args.ctx, tt.args.userID, tt.args.chatID, tt.args.req)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestNegativeMessageService_EditMessage(t *testing.T) {
	type fields struct {
		msgRepo  *mock.MockMessageRepositoryInterface
		chatRepo *mock.MockChatRepositoryInterface
	}

	type args struct {
		ctx    context.Context
		userID int64
		chatID int64
		req    *dto.RequestEditMessage
	}

	longText := make([]byte, 2001)
	for i := range longText {
		longText[i] = 'a'
	}

	tests := []struct {
		wantErr    error
		prepare    func(*fields)
		name       string
		args       args
		wantAnyErr bool
		wantSubstr string
	}{
		{
			name:       "nil request",
			args:       args{ctx: context.Background(), userID: 1, chatID: 1, req: nil},
			wantAnyErr: true,
			wantSubstr: "edit message nil request",
		},
		{
			name:    "empty text",
			args:    args{ctx: context.Background(), userID: 1, chatID: 1, req: &dto.RequestEditMessage{Text: ""}},
			wantErr: domain.ErrMessageEmpty,
		},
		{
			name:    "too long",
			args:    args{ctx: context.Background(), userID: 1, chatID: 1, req: &dto.RequestEditMessage{Text: string(longText)}},
			wantErr: domain.ErrMessageTooLong,
		},
		{
			name: "not member",
			prepare: func(f *fields) {
				f.chatRepo.EXPECT().IsMember(context.Background(), int64(1), int64(10)).Return(false, nil)
			},
			args:    args{ctx: context.Background(), userID: 10, chatID: 1, req: &dto.RequestEditMessage{MessageID: 1, Text: "x"}},
			wantErr: domain.ErrMessageNotMember,
		},
		{
			name: "repo update error",
			prepare: func(f *fields) {
				f.chatRepo.EXPECT().IsMember(context.Background(), int64(1), int64(10)).Return(true, nil)
				f.msgRepo.EXPECT().UpdateMessage(context.Background(), &domain.Message{
					Id:       1,
					ChatId:   1,
					SenderId: 10,
					Content:  "x",
				}).Return(nil, false, errors.New("db down"))
			},
			args:       args{ctx: context.Background(), userID: 10, chatID: 1, req: &dto.RequestEditMessage{MessageID: 1, Text: "x"}},
			wantAnyErr: true,
			wantSubstr: "messageRepo updated message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				msgRepo:  mock.NewMockMessageRepositoryInterface(ctrl),
				chatRepo: mock.NewMockChatRepositoryInterface(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := NewMessageService(f.msgRepo, f.chatRepo, nil, nil, "http://localhost:8088")
			got, err := s.EditMessage(tt.args.ctx, tt.args.userID, tt.args.chatID, tt.args.req)
			require.Nil(t, got)
			if tt.wantAnyErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.wantSubstr)
			} else {
				require.EqualError(t, err, tt.wantErr.Error())
			}
		})
	}
}

func TestPositiveMessageService_DeleteMessage(t *testing.T) {
	type fields struct {
		msgRepo  *mock.MockMessageRepositoryInterface
		chatRepo *mock.MockChatRepositoryInterface
	}

	type args struct {
		ctx    context.Context
		userID int64
		chatID int64
		req    *dto.RequestDeleteMessage
	}

	now := time.Unix(1700003000, 0).UTC()

	tests := []struct {
		prepare func(*fields)
		want    *dto.ResponseClearMessage
		name    string
		args    args
	}{
		{
			name: "delete last message and fetch previous",
			prepare: func(f *fields) {
				f.chatRepo.EXPECT().IsMember(context.Background(), int64(1), int64(10)).Return(true, nil)
				f.msgRepo.EXPECT().DeleteMessage(context.Background(), &domain.Message{
					Id:       7,
					ChatId:   1,
					SenderId: 10,
				}).Return(&domain.Message{
					Id:       7,
					ChatId:   1,
					SenderId: 10,
				}, true, nil)
				f.chatRepo.EXPECT().GetLastMessageOfChat(context.Background(), int64(1)).Return(&domain.Message{
					Id:        6,
					ChatId:    1,
					SenderId:  11,
					Content:   "prev <b>msg</b>",
					CreatedAt: now,
				}, nil)
			},
			args: args{
				ctx:    context.Background(),
				userID: 10,
				chatID: 1,
				req:    &dto.RequestDeleteMessage{MessageID: 7, ChatID: 1},
			},
			want: &dto.ResponseClearMessage{
				ID:                7,
				ChatID:            1,
				SenderID:          10,
				LastMessageEdited: true,
				LastMessage: &dto.LastMessageDTO{
					SenderId:  11,
					Text:      "prev &lt;b&gt;msg&lt;/b&gt;",
					CreatedAt: now,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				msgRepo:  mock.NewMockMessageRepositoryInterface(ctrl),
				chatRepo: mock.NewMockChatRepositoryInterface(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := NewMessageService(f.msgRepo, f.chatRepo, nil, nil, "http://localhost:8088")
			got, err := s.DeleteMessage(tt.args.ctx, tt.args.userID, tt.args.chatID, tt.args.req)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestNegativeMessageService_DeleteMessage(t *testing.T) {
	type fields struct {
		msgRepo  *mock.MockMessageRepositoryInterface
		chatRepo *mock.MockChatRepositoryInterface
	}

	type args struct {
		ctx    context.Context
		userID int64
		chatID int64
		req    *dto.RequestDeleteMessage
	}

	tests := []struct {
		wantErr    error
		prepare    func(*fields)
		name       string
		args       args
		wantAnyErr bool
		wantSubstr string
	}{
		{
			name:       "nil request",
			args:       args{ctx: context.Background(), userID: 1, chatID: 1, req: nil},
			wantAnyErr: true,
			wantSubstr: "delete message nil request",
		},
		{
			name: "not member",
			prepare: func(f *fields) {
				f.chatRepo.EXPECT().IsMember(context.Background(), int64(1), int64(10)).Return(false, nil)
			},
			args:    args{ctx: context.Background(), userID: 10, chatID: 1, req: &dto.RequestDeleteMessage{MessageID: 1}},
			wantErr: domain.ErrMessageNotMember,
		},
		{
			name: "no message",
			prepare: func(f *fields) {
				f.chatRepo.EXPECT().IsMember(context.Background(), int64(1), int64(10)).Return(true, nil)
				f.msgRepo.EXPECT().DeleteMessage(context.Background(), &domain.Message{
					Id:       1,
					ChatId:   1,
					SenderId: 10,
				}).Return(nil, false, domain.ErrNoMessage)
			},
			args:    args{ctx: context.Background(), userID: 10, chatID: 1, req: &dto.RequestDeleteMessage{MessageID: 1}},
			wantErr: domain.ErrNoMessage,
		},
		{
			name: "repo delete error",
			prepare: func(f *fields) {
				f.chatRepo.EXPECT().IsMember(context.Background(), int64(1), int64(10)).Return(true, nil)
				f.msgRepo.EXPECT().DeleteMessage(context.Background(), &domain.Message{
					Id:       1,
					ChatId:   1,
					SenderId: 10,
				}).Return(nil, false, errors.New("db down"))
			},
			args:       args{ctx: context.Background(), userID: 10, chatID: 1, req: &dto.RequestDeleteMessage{MessageID: 1}},
			wantAnyErr: true,
			wantSubstr: "messageRepo delete message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				msgRepo:  mock.NewMockMessageRepositoryInterface(ctrl),
				chatRepo: mock.NewMockChatRepositoryInterface(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := NewMessageService(f.msgRepo, f.chatRepo, nil, nil, "http://localhost:8088")
			got, err := s.DeleteMessage(tt.args.ctx, tt.args.userID, tt.args.chatID, tt.args.req)
			require.Nil(t, got)
			if tt.wantAnyErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.wantSubstr)
			} else {
				require.EqualError(t, err, tt.wantErr.Error())
			}
		})
	}
}

func ptrInt64(v int64) *int64 {
	return &v
}
