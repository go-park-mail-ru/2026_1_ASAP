//go:generate mockgen -destination=mock/chat_usecase_mock.go -package=mock github.com/go-park-mail-ru/2026_1_ASAP/internal/chat/transport/grpc ChatUsecaseInterface
//go:generate mockgen -destination=mock/message_usecase_mock.go -package=mock github.com/go-park-mail-ru/2026_1_ASAP/internal/chat/transport/grpc MessageUsecaseInterface
package grpc

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	chatv1 "github.com/go-park-mail-ru/2026_1_ASAP/gen/go/chat/v1"
	domain "github.com/go-park-mail-ru/2026_1_ASAP/internal/chat/domain/chat"

	dto "github.com/go-park-mail-ru/2026_1_ASAP/internal/chat/dto/chat"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/chat/dto/media"

	"github.com/go-park-mail-ru/2026_1_ASAP/internal/chat/transport/grpc/mock"
)

func strPtr(s string) *string {
	return &s
}

func TestPositiveChatServer_GetChats(t *testing.T) {
	type fields struct {
		chatUsecase    *mock.MockChatUsecaseInterface
		messageUsecase *mock.MockMessageUsecaseInterface
	}

	type args struct {
		ctx context.Context
		req *chatv1.RequestGetUserChats
	}

	now := time.Now()

	tests := []struct {
		prepare func(*fields)
		want    *chatv1.ResponseGetUserChats
		name    string
		args    args
	}{
		{
			name: "Successful get chats",
			prepare: func(f *fields) {
				f.chatUsecase.EXPECT().GetAllChats(gomock.Any(), int64(100)).Return([]dto.ChatInformationDTO{
					{
						ID:       1,
						ChatType: dto.ChatTypeGroup,
						Title:    "Group Chat",
						LastMessage: dto.MessageDTO{
							SenderId:  101,
							Text:      "Hello!",
							CreatedAt: now,
						},
						Avatar:  strPtr("avatar.jpg"),
						OwnerID: 100,
					},
				}, nil)
			},
			args: args{
				ctx: context.Background(),
				req: &chatv1.RequestGetUserChats{UserId: 100},
			},
			want: &chatv1.ResponseGetUserChats{
				Chats: []*chatv1.ChatInformation{
					{
						Id:      1,
						Type:    chatv1.ChatType_GROUP,
						Title:   "Group Chat",
						Avatar:  strPtr("avatar.jpg"),
						OwnerId: 100,
						LastMessage: &chatv1.MessageInformation{
							SenderId:  101,
							Text:      "Hello!",
							CreatedAt: timestamppb.New(now),
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				chatUsecase:    mock.NewMockChatUsecaseInterface(ctrl),
				messageUsecase: mock.NewMockMessageUsecaseInterface(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &ChatServer{
				chatUsecase:    f.chatUsecase,
				messageUsecase: f.messageUsecase,
				logger:         zap.NewNop(),
			}

			resp, err := s.GetChats(tt.args.ctx, tt.args.req)
			require.NoError(t, err)
			require.Equal(t, tt.want.GetChats()[0].GetId(), resp.GetChats()[0].GetId())
			require.Equal(t, tt.want.GetChats()[0].GetTitle(), resp.GetChats()[0].GetTitle())
		})
	}
}

func TestNegativeChatServer_GetChats(t *testing.T) {
	type fields struct {
		chatUsecase    *mock.MockChatUsecaseInterface
		messageUsecase *mock.MockMessageUsecaseInterface
	}

	type args struct {
		ctx context.Context
		req *chatv1.RequestGetUserChats
	}

	tests := []struct {
		wantErr error
		prepare func(*fields)
		name    string
		args    args
	}{
		{
			name: "Chat not found",
			prepare: func(f *fields) {
				f.chatUsecase.EXPECT().GetAllChats(gomock.Any(), int64(100)).Return(nil, domain.ErrChatNotFound)
			},
			args: args{
				ctx: context.Background(),
				req: &chatv1.RequestGetUserChats{UserId: 100},
			},
			wantErr: domain.ErrChatNotFound,
		},
		{
			name: "User not member",
			prepare: func(f *fields) {
				f.chatUsecase.EXPECT().GetAllChats(gomock.Any(), int64(100)).Return(nil, domain.ErrNotMember)
			},
			args: args{
				ctx: context.Background(),
				req: &chatv1.RequestGetUserChats{UserId: 100},
			},
			wantErr: domain.ErrNotMember,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				chatUsecase:    mock.NewMockChatUsecaseInterface(ctrl),
				messageUsecase: mock.NewMockMessageUsecaseInterface(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &ChatServer{
				chatUsecase:    f.chatUsecase,
				messageUsecase: f.messageUsecase,
				logger:         zap.NewNop(),
			}

			_, err := s.GetChats(tt.args.ctx, tt.args.req)
			require.Error(t, err)
		})
	}
}

func TestPositiveChatServer_Create(t *testing.T) {
	type fields struct {
		chatUsecase    *mock.MockChatUsecaseInterface
		messageUsecase *mock.MockMessageUsecaseInterface
	}

	type args struct {
		ctx context.Context
		req *chatv1.RequestChatCreate
	}

	tests := []struct {
		prepare func(*fields)
		want    *chatv1.ChatInformation
		name    string
		args    args
	}{
		{
			name: "Successful create group chat",
			prepare: func(f *fields) {
				f.chatUsecase.EXPECT().CreateChat(gomock.Any(), dto.ChatCreate{
					Title:     "New Group",
					Type:      dto.ChatTypeGroup,
					MembersID: []int64{101},
				}, int64(100)).Return(&dto.ChatInformationDTO{
					ID:       10,
					ChatType: dto.ChatTypeGroup,
					Title:    "New Group",
					OwnerID:  100,
				}, nil)
			},
			args: args{
				ctx: context.Background(),
				req: &chatv1.RequestChatCreate{
					UserId:    100,
					Type:      chatv1.ChatType_GROUP,
					Title:     "New Group",
					MembersId: []int64{101},
				},
			},
			want: &chatv1.ChatInformation{
				Id:      10,
				Type:    chatv1.ChatType_GROUP,
				Title:   "New Group",
				OwnerId: 100,
			},
		},
		{
			name: "Successful create dialog chat",
			prepare: func(f *fields) {
				f.chatUsecase.EXPECT().CreateChat(gomock.Any(), dto.ChatCreate{
					Title:     "",
					Type:      dto.ChatTypeDialog,
					MembersID: []int64{101},
				}, int64(100)).Return(&dto.ChatInformationDTO{
					ID:       20,
					ChatType: dto.ChatTypeDialog,
					Title:    "Friend",
					OwnerID:  100,
				}, nil)
			},
			args: args{
				ctx: context.Background(),
				req: &chatv1.RequestChatCreate{
					UserId:    100,
					Type:      chatv1.ChatType_DIALOG,
					Title:     "",
					MembersId: []int64{101},
				},
			},
			want: &chatv1.ChatInformation{
				Id:      20,
				Type:    chatv1.ChatType_DIALOG,
				Title:   "Friend",
				OwnerId: 100,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				chatUsecase:    mock.NewMockChatUsecaseInterface(ctrl),
				messageUsecase: mock.NewMockMessageUsecaseInterface(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &ChatServer{
				chatUsecase:    f.chatUsecase,
				messageUsecase: f.messageUsecase,
				logger:         zap.NewNop(),
			}

			resp, err := s.Create(tt.args.ctx, tt.args.req)
			require.NoError(t, err)
			require.Equal(t, tt.want.GetId(), resp.GetId())
			require.Equal(t, tt.want.GetTitle(), resp.GetTitle())
		})
	}
}

func TestNegativeChatServer_Create(t *testing.T) {
	type fields struct {
		chatUsecase    *mock.MockChatUsecaseInterface
		messageUsecase *mock.MockMessageUsecaseInterface
	}

	type args struct {
		ctx context.Context
		req *chatv1.RequestChatCreate
	}

	tests := []struct {
		wantErr error
		prepare func(*fields)
		name    string
		args    args
	}{
		{
			name: "Dialog already exists",
			prepare: func(f *fields) {
				f.chatUsecase.EXPECT().CreateChat(gomock.Any(), gomock.Any(), int64(100)).Return(nil, domain.ErrDialogAlreadyExists)
			},
			args: args{
				ctx: context.Background(),
				req: &chatv1.RequestChatCreate{
					UserId: 100,
					Type:   chatv1.ChatType_DIALOG,
				},
			},
			wantErr: domain.ErrDialogAlreadyExists,
		},
		{
			name: "Cannot create dialog with yourself",
			prepare: func(f *fields) {
				f.chatUsecase.EXPECT().CreateChat(gomock.Any(), gomock.Any(), int64(100)).Return(nil, domain.ErrCantCreateDialogWithYourself)
			},
			args: args{
				ctx: context.Background(),
				req: &chatv1.RequestChatCreate{
					UserId: 100,
					Type:   chatv1.ChatType_DIALOG,
				},
			},
			wantErr: domain.ErrCantCreateDialogWithYourself,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				chatUsecase:    mock.NewMockChatUsecaseInterface(ctrl),
				messageUsecase: mock.NewMockMessageUsecaseInterface(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &ChatServer{
				chatUsecase:    f.chatUsecase,
				messageUsecase: f.messageUsecase,
				logger:         zap.NewNop(),
			}

			_, err := s.Create(tt.args.ctx, tt.args.req)
			require.Error(t, err)
		})
	}
}

func TestPositiveChatServer_GetChatByID(t *testing.T) {
	type fields struct {
		chatUsecase    *mock.MockChatUsecaseInterface
		messageUsecase *mock.MockMessageUsecaseInterface
	}

	type args struct {
		ctx context.Context
		req *chatv1.RequestGetChatByID
	}

	now := time.Now()

	tests := []struct {
		prepare func(*fields)
		want    *chatv1.ChatInformation
		name    string
		args    args
	}{
		{
			name: "Successful get chat by id",
			prepare: func(f *fields) {
				f.chatUsecase.EXPECT().GetChatByID(gomock.Any(), int64(1), int64(100)).Return(&dto.ChatInformationDTO{
					ID:       1,
					ChatType: dto.ChatTypeGroup,
					Title:    "Group Chat",
					OwnerID:  100,
					LastMessage: dto.MessageDTO{
						SenderId:  101,
						Text:      "Hello!",
						CreatedAt: now,
					},
				}, nil)
			},
			args: args{
				ctx: context.Background(),
				req: &chatv1.RequestGetChatByID{
					UserId: 100,
					ChatId: 1,
				},
			},
			want: &chatv1.ChatInformation{
				Id:      1,
				Type:    chatv1.ChatType_GROUP,
				Title:   "Group Chat",
				OwnerId: 100,
				LastMessage: &chatv1.MessageInformation{
					SenderId:  101,
					Text:      "Hello!",
					CreatedAt: timestamppb.New(now),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				chatUsecase:    mock.NewMockChatUsecaseInterface(ctrl),
				messageUsecase: mock.NewMockMessageUsecaseInterface(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &ChatServer{
				chatUsecase:    f.chatUsecase,
				messageUsecase: f.messageUsecase,
				logger:         zap.NewNop(),
			}

			resp, err := s.GetChatByID(tt.args.ctx, tt.args.req)
			require.NoError(t, err)
			require.Equal(t, tt.want.GetId(), resp.GetId())
			require.Equal(t, tt.want.GetTitle(), resp.GetTitle())
		})
	}
}

func TestPositiveChatServer_DeleteChat(t *testing.T) {
	type fields struct {
		chatUsecase    *mock.MockChatUsecaseInterface
		messageUsecase *mock.MockMessageUsecaseInterface
	}

	type args struct {
		ctx context.Context
		req *chatv1.RequestDeleteChat
	}

	tests := []struct {
		prepare func(*fields)
		want    *emptypb.Empty
		name    string
		args    args
	}{
		{
			name: "Successful delete chat",
			prepare: func(f *fields) {
				f.chatUsecase.EXPECT().DeleteChat(gomock.Any(), int64(100), int64(1)).Return(nil)
			},
			args: args{
				ctx: context.Background(),
				req: &chatv1.RequestDeleteChat{
					UserId: 100,
					ChatId: 1,
				},
			},
			want: &emptypb.Empty{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				chatUsecase:    mock.NewMockChatUsecaseInterface(ctrl),
				messageUsecase: mock.NewMockMessageUsecaseInterface(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &ChatServer{
				chatUsecase:    f.chatUsecase,
				messageUsecase: f.messageUsecase,
				logger:         zap.NewNop(),
			}

			resp, err := s.DeleteChat(tt.args.ctx, tt.args.req)
			require.NoError(t, err)
			require.Equal(t, tt.want, resp)
		})
	}
}

func TestPositiveChatServer_UpdateChatAvatar(t *testing.T) {
	type fields struct {
		chatUsecase    *mock.MockChatUsecaseInterface
		messageUsecase *mock.MockMessageUsecaseInterface
	}

	type args struct {
		ctx context.Context
		req *chatv1.RequestUpdateAvatar
	}

	tests := []struct {
		prepare func(*fields)
		want    *chatv1.ChatInformation
		name    string
		args    args
	}{
		{
			name: "Successful update chat avatar",
			prepare: func(f *fields) {
				f.chatUsecase.EXPECT().UpdateChatAvatar(gomock.Any(), int64(100), int64(1), &dto.RequestUpdateAvatar{
					File: &media.FileInput{
						Body:        bytes.NewReader([]byte("image data")),
						ContentType: "image/png",
						Size:        10,
					},
				}).Return(&dto.ChatInformationDTO{
					ID:       1,
					ChatType: dto.ChatTypeGroup,
					Title:    "Group Chat",
					Avatar:   strPtr("new_avatar.jpg"),
				}, nil)
			},
			args: args{
				ctx: context.Background(),
				req: &chatv1.RequestUpdateAvatar{
					UserId:  100,
					ChatId:  1,
					Content: []byte("image data"),
					Type:    "image/png",
				},
			},
			want: &chatv1.ChatInformation{
				Id:     1,
				Type:   chatv1.ChatType_GROUP,
				Title:  "Group Chat",
				Avatar: strPtr("new_avatar.jpg"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				chatUsecase:    mock.NewMockChatUsecaseInterface(ctrl),
				messageUsecase: mock.NewMockMessageUsecaseInterface(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &ChatServer{
				chatUsecase:    f.chatUsecase,
				messageUsecase: f.messageUsecase,
				logger:         zap.NewNop(),
			}

			resp, err := s.UpdateChatAvatar(tt.args.ctx, tt.args.req)
			require.NoError(t, err)
			require.Equal(t, tt.want.GetId(), resp.GetId())
			require.Equal(t, tt.want.GetAvatar(), resp.GetAvatar())
		})
	}
}

func TestPositiveChatServer_UpdateChatTitle(t *testing.T) {
	type fields struct {
		chatUsecase    *mock.MockChatUsecaseInterface
		messageUsecase *mock.MockMessageUsecaseInterface
	}

	type args struct {
		ctx context.Context
		req *chatv1.RequestUpdateTitle
	}

	tests := []struct {
		prepare func(*fields)
		want    *chatv1.ChatInformation
		name    string
		args    args
	}{
		{
			name: "Successful update chat title",
			prepare: func(f *fields) {
				f.chatUsecase.EXPECT().UpdateChatTitle(gomock.Any(), int64(100), int64(1), &dto.RequestUpdateTitle{
					Title: "New Title",
				}).Return(&dto.ChatInformationDTO{
					ID:       1,
					ChatType: dto.ChatTypeGroup,
					Title:    "New Title",
				}, nil)
			},
			args: args{
				ctx: context.Background(),
				req: &chatv1.RequestUpdateTitle{
					UserId: 100,
					ChatId: 1,
					Title:  "New Title",
				},
			},
			want: &chatv1.ChatInformation{
				Id:    1,
				Type:  chatv1.ChatType_GROUP,
				Title: "New Title",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				chatUsecase:    mock.NewMockChatUsecaseInterface(ctrl),
				messageUsecase: mock.NewMockMessageUsecaseInterface(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &ChatServer{
				chatUsecase:    f.chatUsecase,
				messageUsecase: f.messageUsecase,
				logger:         zap.NewNop(),
			}

			resp, err := s.UpdateChatTitle(tt.args.ctx, tt.args.req)
			require.NoError(t, err)
			require.Equal(t, tt.want.GetTitle(), resp.GetTitle())
		})
	}
}

func TestPositiveChatServer_AddMembersToChat(t *testing.T) {
	type fields struct {
		chatUsecase    *mock.MockChatUsecaseInterface
		messageUsecase *mock.MockMessageUsecaseInterface
	}

	type args struct {
		ctx context.Context
		req *chatv1.RequestAddMembersToChat
	}

	tests := []struct {
		prepare func(*fields)
		want    *emptypb.Empty
		name    string
		args    args
	}{
		{
			name: "Successful add members",
			prepare: func(f *fields) {
				f.chatUsecase.EXPECT().AddMembersToChat(gomock.Any(), int64(100), int64(1), &dto.RequestAddMember{
					MembersId: []int64{101, 102},
				}).Return(nil)
			},
			args: args{
				ctx: context.Background(),
				req: &chatv1.RequestAddMembersToChat{
					UserId:    100,
					ChatId:    1,
					MembersId: []int64{101, 102},
				},
			},
			want: &emptypb.Empty{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				chatUsecase:    mock.NewMockChatUsecaseInterface(ctrl),
				messageUsecase: mock.NewMockMessageUsecaseInterface(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &ChatServer{
				chatUsecase:    f.chatUsecase,
				messageUsecase: f.messageUsecase,
				logger:         zap.NewNop(),
			}

			resp, err := s.AddMembersToChat(tt.args.ctx, tt.args.req)
			require.NoError(t, err)
			require.Equal(t, tt.want, resp)
		})
	}
}

func TestPositiveChatServer_DeleteMemberFromChat(t *testing.T) {
	type fields struct {
		chatUsecase    *mock.MockChatUsecaseInterface
		messageUsecase *mock.MockMessageUsecaseInterface
	}

	type args struct {
		ctx context.Context
		req *chatv1.RequestDeleteMemberFromChat
	}

	tests := []struct {
		prepare func(*fields)
		want    *emptypb.Empty
		name    string
		args    args
	}{
		{
			name: "Successful delete member",
			prepare: func(f *fields) {
				f.chatUsecase.EXPECT().DeleteMemberFromChat(gomock.Any(), int64(100), int64(1), &dto.RequestDeleteMember{
					MemberId: 101,
				}).Return(nil)
			},
			args: args{
				ctx: context.Background(),
				req: &chatv1.RequestDeleteMemberFromChat{
					UserId:   100,
					ChatId:   1,
					MemberId: 101,
				},
			},
			want: &emptypb.Empty{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				chatUsecase:    mock.NewMockChatUsecaseInterface(ctrl),
				messageUsecase: mock.NewMockMessageUsecaseInterface(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &ChatServer{
				chatUsecase:    f.chatUsecase,
				messageUsecase: f.messageUsecase,
				logger:         zap.NewNop(),
			}

			resp, err := s.DeleteMemberFromChat(tt.args.ctx, tt.args.req)
			require.NoError(t, err)
			require.Equal(t, tt.want, resp)
		})
	}
}

func TestPositiveChatServer_GetChatMembers(t *testing.T) {
	type fields struct {
		chatUsecase    *mock.MockChatUsecaseInterface
		messageUsecase *mock.MockMessageUsecaseInterface
	}

	type args struct {
		ctx context.Context
		req *chatv1.RequestChatMembers
	}

	tests := []struct {
		prepare func(*fields)
		want    *chatv1.ResponseGetChatMembers
		name    string
		args    args
	}{
		{
			name: "Successful get chat members",
			prepare: func(f *fields) {
				f.chatUsecase.EXPECT().GetAllChatMembers(gomock.Any(), int64(100), int64(1)).Return(&dto.ResponseGetChatMembers{
					MembersId: []int64{100, 101, 102},
				}, nil)
			},
			args: args{
				ctx: context.Background(),
				req: &chatv1.RequestChatMembers{
					UserId: 100,
					ChatId: 1,
				},
			},
			want: &chatv1.ResponseGetChatMembers{
				MembersId: []int64{100, 101, 102},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				chatUsecase:    mock.NewMockChatUsecaseInterface(ctrl),
				messageUsecase: mock.NewMockMessageUsecaseInterface(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &ChatServer{
				chatUsecase:    f.chatUsecase,
				messageUsecase: f.messageUsecase,
				logger:         zap.NewNop(),
			}

			resp, err := s.GetChatMembers(tt.args.ctx, tt.args.req)
			require.NoError(t, err)
			require.Equal(t, tt.want.GetMembersId(), resp.GetMembersId())
		})
	}
}

func TestPositiveChatServer_QuitChat(t *testing.T) {
	type fields struct {
		chatUsecase    *mock.MockChatUsecaseInterface
		messageUsecase *mock.MockMessageUsecaseInterface
	}

	type args struct {
		ctx context.Context
		req *chatv1.RequestQuitChat
	}

	tests := []struct {
		prepare func(*fields)
		want    *emptypb.Empty
		name    string
		args    args
	}{
		{
			name: "Successful quit chat",
			prepare: func(f *fields) {
				f.chatUsecase.EXPECT().QuitChat(gomock.Any(), int64(101), int64(1)).Return(nil)
			},
			args: args{
				ctx: context.Background(),
				req: &chatv1.RequestQuitChat{
					UserId: 101,
					ChatId: 1,
				},
			},
			want: &emptypb.Empty{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				chatUsecase:    mock.NewMockChatUsecaseInterface(ctrl),
				messageUsecase: mock.NewMockMessageUsecaseInterface(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &ChatServer{
				chatUsecase:    f.chatUsecase,
				messageUsecase: f.messageUsecase,
				logger:         zap.NewNop(),
			}

			resp, err := s.QuitChat(tt.args.ctx, tt.args.req)
			require.NoError(t, err)
			require.Equal(t, tt.want, resp)
		})
	}
}

func TestPositiveChatServer_JoinChannel(t *testing.T) {
	type fields struct {
		chatUsecase    *mock.MockChatUsecaseInterface
		messageUsecase *mock.MockMessageUsecaseInterface
	}

	type args struct {
		ctx context.Context
		req *chatv1.RequestJoinChannel
	}

	tests := []struct {
		prepare func(*fields)
		want    *emptypb.Empty
		name    string
		args    args
	}{
		{
			name: "Successful join channel",
			prepare: func(f *fields) {
				f.chatUsecase.EXPECT().JoinChannel(gomock.Any(), int64(100), int64(1)).Return(nil)
			},
			args: args{
				ctx: context.Background(),
				req: &chatv1.RequestJoinChannel{
					UserId: 100,
					ChatId: 1,
				},
			},
			want: &emptypb.Empty{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				chatUsecase:    mock.NewMockChatUsecaseInterface(ctrl),
				messageUsecase: mock.NewMockMessageUsecaseInterface(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &ChatServer{
				chatUsecase:    f.chatUsecase,
				messageUsecase: f.messageUsecase,
				logger:         zap.NewNop(),
			}

			resp, err := s.JoinChannel(tt.args.ctx, tt.args.req)
			require.NoError(t, err)
			require.Equal(t, tt.want, resp)
		})
	}
}

func TestPositiveChatServer_UpdateChatDescription(t *testing.T) {
	type fields struct {
		chatUsecase    *mock.MockChatUsecaseInterface
		messageUsecase *mock.MockMessageUsecaseInterface
	}

	type args struct {
		ctx context.Context
		req *chatv1.RequestUpdateDescription
	}

	tests := []struct {
		prepare func(*fields)
		want    *chatv1.ChatInformation
		name    string
		args    args
	}{
		{
			name: "Successful update chat description",
			prepare: func(f *fields) {
				f.chatUsecase.EXPECT().UpdateChatDescription(gomock.Any(), int64(100), int64(1), &dto.RequestUpdateDescription{
					Description: "New description",
				}).Return(&dto.ChatInformationDTO{
					ID:          1,
					ChatType:    dto.ChatTypeGroup,
					Title:       "Group Chat",
					Description: strPtr("New description"),
				}, nil)
			},
			args: args{
				ctx: context.Background(),
				req: &chatv1.RequestUpdateDescription{
					UserId:      100,
					ChatId:      1,
					Description: "New description",
				},
			},
			want: &chatv1.ChatInformation{
				Id:          1,
				Type:        chatv1.ChatType_GROUP,
				Title:       "Group Chat",
				Description: strPtr("New description"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				chatUsecase:    mock.NewMockChatUsecaseInterface(ctrl),
				messageUsecase: mock.NewMockMessageUsecaseInterface(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &ChatServer{
				chatUsecase:    f.chatUsecase,
				messageUsecase: f.messageUsecase,
				logger:         zap.NewNop(),
			}

			resp, err := s.UpdateChatDescription(tt.args.ctx, tt.args.req)
			require.NoError(t, err)
			require.Equal(t, tt.want.GetId(), resp.GetId())
			require.Equal(t, tt.want.GetDescription(), resp.GetDescription())
		})
	}
}
