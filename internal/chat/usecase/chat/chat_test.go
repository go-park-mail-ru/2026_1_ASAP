// internal/services/chat/chat_test.go
package chat

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	domain "github.com/go-park-mail-ru/2026_1_ASAP/internal/domain/chat"
	domainProfile "github.com/go-park-mail-ru/2026_1_ASAP/internal/domain/profile"
	domainUser "github.com/go-park-mail-ru/2026_1_ASAP/internal/domain/user"
	dto "github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/chat"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/media"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/services/chat/mock"
)

func strPtr(s string) *string {
	return &s
}

func TestPositiveChatService_GetChatByID(t *testing.T) {
	type fields struct {
		chatRepo  *mock.MockChatRepositoryInterface
		userRepo  *mock.MockUserRepositoryInterface
		mediaRepo *mock.MockMediaRepositoryInterface
	}

	type args struct {
		ctx    context.Context
		chatID int64
		userID int64
	}

	now := time.Now()

	tests := []struct {
		prepare func(*fields)
		want    *dto.ChatInformationDTO
		name    string
		args    args
	}{
		{
			name: "Get group chat by id",
			prepare: func(f *fields) {
				f.chatRepo.EXPECT().IsMember(context.Background(), int64(1), int64(100)).Return(true, nil)
				f.chatRepo.EXPECT().GetChatByID(context.Background(), int64(1)).Return(&domain.Chat{
					Id:        1,
					Type:      domain.ChatTypeGroup,
					Title:     "Group Chat",
					OwnerId:   100,
					AvatarUrl: strPtr("avatar.jpg"),
					CreatedAt: now,
					UpdatedAt: now,
				}, nil)
				f.chatRepo.EXPECT().GetLastMessageOfChat(context.Background(), int64(1)).Return(&domain.Message{
					Id:        10,
					ChatId:    1,
					SenderId:  101,
					Content:   "Hello!",
					CreatedAt: now,
				}, nil)
			},
			args: args{ctx: context.Background(), chatID: 1, userID: 100},
			want: &dto.ChatInformationDTO{
				ID:       1,
				ChatType: dto.ChatTypeGroup,
				Title:    "Group Chat",
				LastMessage: dto.MessageDTO{
					SenderId:  101,
					Text:      "Hello!",
					CreatedAt: now,
				},
				Avatar: strPtr("avatar.jpg"),
			},
		},
		{
			name: "Get dialog chat by id with friend name",
			prepare: func(f *fields) {
				f.chatRepo.EXPECT().IsMember(context.Background(), int64(2), int64(100)).Return(true, nil)
				f.chatRepo.EXPECT().GetChatByID(context.Background(), int64(2)).Return(&domain.Chat{
					Id:        2,
					Type:      domain.ChatTypeDialog,
					Title:     "",
					OwnerId:   100,
					AvatarUrl: nil,
					CreatedAt: now,
					UpdatedAt: now,
				}, nil)
				f.chatRepo.EXPECT().GetLastMessageOfChat(context.Background(), int64(2)).Return(&domain.Message{
					Id:        20,
					ChatId:    2,
					SenderId:  101,
					Content:   "Hi!",
					CreatedAt: now,
				}, nil)
				f.chatRepo.EXPECT().GetChatMembers(context.Background(), int64(2)).Return([]int64{100, 101}, nil)
				f.userRepo.EXPECT().GetUserByID(context.Background(), int64(101)).Return(&domainUser.User{
					Id:        101,
					Login:     "friend_user",
					FirstName: "Friend",
					LastName:  strPtr("User"),
				}, nil)
				f.chatRepo.EXPECT().GetChatMembers(context.Background(), int64(2)).Return([]int64{100, 101}, nil)
				f.userRepo.EXPECT().GetUserByID(context.Background(), int64(101)).Return(&domainUser.User{
					Id:        101,
					Login:     "friend_user",
					FirstName: "Friend",
					LastName:  strPtr("User"),
					AvatarUrl: strPtr("friend_avatar.jpg"),
				}, nil)
			},
			args: args{ctx: context.Background(), chatID: 2, userID: 100},
			want: &dto.ChatInformationDTO{
				ID:       2,
				ChatType: dto.ChatTypeDialog,
				Title:    "Friend User",
				LastMessage: dto.MessageDTO{
					SenderId:  101,
					Text:      "Hi!",
					CreatedAt: now,
				},
				Avatar: strPtr("friend_avatar.jpg"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				chatRepo:  mock.NewMockChatRepositoryInterface(ctrl),
				userRepo:  mock.NewMockUserRepositoryInterface(ctrl),
				mediaRepo: mock.NewMockMediaRepositoryInterface(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &ChatService{
				chatRepo:  f.chatRepo,
				userRepo:  f.userRepo,
				mediaRepo: f.mediaRepo,
			}

			result, err := s.GetChatByID(tt.args.ctx, tt.args.chatID, tt.args.userID)
			require.NoError(t, err)
			require.Equal(t, tt.want, result)
		})
	}
}

func TestNegativeChatService_GetChatByID(t *testing.T) {
	type fields struct {
		chatRepo  *mock.MockChatRepositoryInterface
		userRepo  *mock.MockUserRepositoryInterface
		mediaRepo *mock.MockMediaRepositoryInterface
	}

	type args struct {
		ctx    context.Context
		chatID int64
		userID int64
	}

	tests := []struct {
		wantErr    error
		prepare    func(*fields)
		name       string
		args       args
		wantAnyErr bool
	}{
		{
			name: "User is not member of chat",
			prepare: func(f *fields) {
				f.chatRepo.EXPECT().IsMember(context.Background(), int64(1), int64(100)).Return(false, nil)
			},
			args:    args{ctx: context.Background(), chatID: 1, userID: 100},
			wantErr: domain.ErrNotMember,
		},
		{
			name: "Chat not found",
			prepare: func(f *fields) {
				f.chatRepo.EXPECT().IsMember(context.Background(), int64(1), int64(100)).Return(true, nil)
				f.chatRepo.EXPECT().GetChatByID(context.Background(), int64(1)).Return(nil, domain.ErrChatNotFound)
			},
			args:    args{ctx: context.Background(), chatID: 1, userID: 100},
			wantErr: domain.ErrChatNotFound,
		},
		{
			name: "IsMember error",
			prepare: func(f *fields) {
				f.chatRepo.EXPECT().IsMember(context.Background(), int64(1), int64(100)).Return(false, errors.New("db error"))
			},
			args:       args{ctx: context.Background(), chatID: 1, userID: 100},
			wantAnyErr: true,
		},
		{
			name: "GetLastMessage error (non-ErrNoMessage)",
			prepare: func(f *fields) {
				f.chatRepo.EXPECT().IsMember(context.Background(), int64(1), int64(100)).Return(true, nil)
				f.chatRepo.EXPECT().GetChatByID(context.Background(), int64(1)).Return(&domain.Chat{
					Id:   1,
					Type: domain.ChatTypeGroup,
				}, nil)
				f.chatRepo.EXPECT().GetLastMessageOfChat(context.Background(), int64(1)).Return(nil, errors.New("db error"))
			},
			args:       args{ctx: context.Background(), chatID: 1, userID: 100},
			wantAnyErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				chatRepo:  mock.NewMockChatRepositoryInterface(ctrl),
				userRepo:  mock.NewMockUserRepositoryInterface(ctrl),
				mediaRepo: mock.NewMockMediaRepositoryInterface(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &ChatService{
				chatRepo:  f.chatRepo,
				userRepo:  f.userRepo,
				mediaRepo: f.mediaRepo,
			}

			result, err := s.GetChatByID(tt.args.ctx, tt.args.chatID, tt.args.userID)
			require.Nil(t, result)
			if tt.wantAnyErr {
				require.Error(t, err)
			} else {
				require.EqualError(t, err, tt.wantErr.Error())
			}
		})
	}
}

func TestPositiveChatService_CreateChat(t *testing.T) {
	type fields struct {
		chatRepo  *mock.MockChatRepositoryInterface
		userRepo  *mock.MockUserRepositoryInterface
		mediaRepo *mock.MockMediaRepositoryInterface
	}

	type args struct {
		ctx     context.Context
		chatDTO dto.ChatCreate
		ownerID int64
	}

	now := time.Now()

	tests := []struct {
		prepare func(*fields)
		want    *dto.ChatInformationDTO
		name    string
		args    args
	}{
		{
			name: "Create group chat",
			prepare: func(f *fields) {
				f.chatRepo.EXPECT().CreateChat(gomock.Any(), gomock.Any()).Return(&domain.Chat{
					Id:        10,
					Type:      domain.ChatTypeGroup,
					Title:     "New Group",
					OwnerId:   100,
					CreatedAt: now,
					UpdatedAt: now,
				}, nil)

				f.userRepo.EXPECT().GetUserByID(context.Background(), int64(100)).Return(&domainUser.User{Id: 100}, nil)
				f.userRepo.EXPECT().GetUserByID(context.Background(), int64(101)).Return(&domainUser.User{Id: 101}, nil)

				f.chatRepo.EXPECT().AddMember(context.Background(), int64(10), int64(100), "owner").Return(nil)
				f.chatRepo.EXPECT().AddMember(context.Background(), int64(10), int64(101), "member").Return(nil)
			},
			args: args{
				ctx: context.Background(),
				chatDTO: dto.ChatCreate{
					Type:      dto.ChatTypeGroup,
					Title:     "New Group",
					MembersID: []int64{101},
				},
				ownerID: 100,
			},
			want: &dto.ChatInformationDTO{
				ID:          10,
				ChatType:    dto.ChatTypeGroup,
				Title:       "New Group",
				LastMessage: dto.MessageDTO{},
				Avatar:      nil,
			},
		},
		{
			name: "Create dialog chat",
			prepare: func(f *fields) {
				f.chatRepo.EXPECT().GetDialogBetweenUsers(context.Background(), int64(100), int64(101)).Return(nil, nil)
				f.chatRepo.EXPECT().CreateChat(gomock.Any(), gomock.Any()).Return(&domain.Chat{
					Id:        20,
					Type:      domain.ChatTypeDialog,
					Title:     "",
					OwnerId:   100,
					CreatedAt: now,
					UpdatedAt: now,
				}, nil)

				f.userRepo.EXPECT().GetUserByID(context.Background(), int64(100)).Return(&domainUser.User{Id: 100}, nil)
				f.userRepo.EXPECT().GetUserByID(context.Background(), int64(101)).Return(&domainUser.User{Id: 101}, nil)

				f.chatRepo.EXPECT().AddMember(context.Background(), int64(20), int64(100), "owner").Return(nil)
				f.chatRepo.EXPECT().AddMember(context.Background(), int64(20), int64(101), "member").Return(nil)

				f.chatRepo.EXPECT().GetChatMembers(context.Background(), int64(20)).Return([]int64{100, 101}, nil).Times(2)
				f.userRepo.EXPECT().GetUserByID(context.Background(), int64(101)).Return(&domainUser.User{
					Id:        101,
					Login:     "friend",
					AvatarUrl: strPtr("avatar.jpg"),
				}, nil).Times(2)
			},
			args: args{
				ctx: context.Background(),
				chatDTO: dto.ChatCreate{
					Type:      dto.ChatTypeDialog,
					Title:     "",
					MembersID: []int64{100, 101},
				},
				ownerID: 100,
			},
			want: &dto.ChatInformationDTO{
				ID:          20,
				ChatType:    dto.ChatTypeDialog,
				Title:       "",
				LastMessage: dto.MessageDTO{},
				Avatar:      strPtr("avatar.jpg"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				chatRepo:  mock.NewMockChatRepositoryInterface(ctrl),
				userRepo:  mock.NewMockUserRepositoryInterface(ctrl),
				mediaRepo: mock.NewMockMediaRepositoryInterface(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &ChatService{
				chatRepo:  f.chatRepo,
				userRepo:  f.userRepo,
				mediaRepo: f.mediaRepo,
			}

			result, err := s.CreateChat(tt.args.ctx, tt.args.chatDTO, tt.args.ownerID)
			require.NoError(t, err)
			require.Equal(t, tt.want, result)
		})
	}
}

func TestNegativeChatService_CreateChat(t *testing.T) {
	type fields struct {
		chatRepo  *mock.MockChatRepositoryInterface
		userRepo  *mock.MockUserRepositoryInterface
		mediaRepo *mock.MockMediaRepositoryInterface
	}

	type args struct {
		ctx     context.Context
		chatDTO dto.ChatCreate
		ownerID int64
	}

	tests := []struct {
		wantErr    error
		prepare    func(*fields)
		name       string
		args       args
		wantAnyErr bool
	}{
		{
			name: "Dialog must have exactly 2 users",
			args: args{
				ctx: context.Background(),
				chatDTO: dto.ChatCreate{
					Type:      dto.ChatTypeDialog,
					MembersID: []int64{100},
				},
				ownerID: 100,
			},
			wantErr: domain.ErrDialogMustHave2Users,
		},
		{
			name: "Cannot create dialog with yourself",
			args: args{
				ctx: context.Background(),
				chatDTO: dto.ChatCreate{
					Type:      dto.ChatTypeDialog,
					MembersID: []int64{100, 100},
				},
				ownerID: 100,
			},
			wantErr: domain.ErrCantCreateDialogWithYourself,
		},
		{
			name: "Dialog already exists",
			prepare: func(f *fields) {
				f.chatRepo.EXPECT().GetDialogBetweenUsers(context.Background(), int64(100), int64(101)).Return(&domain.Chat{Id: 1}, nil)
			},
			args: args{
				ctx: context.Background(),
				chatDTO: dto.ChatCreate{
					Type:      dto.ChatTypeDialog,
					MembersID: []int64{100, 101},
				},
				ownerID: 100,
			},
			wantErr: domain.ErrDialogAlreadyExists,
		},
		{
			name: "User not found when adding member",
			prepare: func(f *fields) {
				f.chatRepo.EXPECT().CreateChat(gomock.Any(), gomock.Any()).Return(&domain.Chat{Id: 10}, nil)
				f.userRepo.EXPECT().GetUserByID(context.Background(), int64(101)).Return(nil, domainUser.ErrNotFound)
			},
			args: args{
				ctx: context.Background(),
				chatDTO: dto.ChatCreate{
					Type:      dto.ChatTypeGroup,
					MembersID: []int64{101},
				},
				ownerID: 100,
			},
			wantErr: domainUser.ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				chatRepo:  mock.NewMockChatRepositoryInterface(ctrl),
				userRepo:  mock.NewMockUserRepositoryInterface(ctrl),
				mediaRepo: mock.NewMockMediaRepositoryInterface(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &ChatService{
				chatRepo:  f.chatRepo,
				userRepo:  f.userRepo,
				mediaRepo: f.mediaRepo,
			}

			result, err := s.CreateChat(tt.args.ctx, tt.args.chatDTO, tt.args.ownerID)
			require.Nil(t, result)
			if tt.wantAnyErr {
				require.Error(t, err)
			} else {
				require.EqualError(t, err, tt.wantErr.Error())
			}
		})
	}
}

func TestPositiveChatService_DeleteChat(t *testing.T) {
	type fields struct {
		chatRepo  *mock.MockChatRepositoryInterface
		userRepo  *mock.MockUserRepositoryInterface
		mediaRepo *mock.MockMediaRepositoryInterface
	}

	type args struct {
		ctx    context.Context
		userID int64
		chatID int64
	}

	tests := []struct {
		name    string
		prepare func(*fields)
		args    args
	}{
		{
			name: "Delete group chat as owner",
			prepare: func(f *fields) {
				f.chatRepo.EXPECT().IsMember(context.Background(), int64(1), int64(100)).Return(true, nil)
				f.chatRepo.EXPECT().GetChatByID(context.Background(), int64(1)).Return(&domain.Chat{
					Id:      1,
					Type:    domain.ChatTypeGroup,
					OwnerId: 100,
				}, nil)
				f.chatRepo.EXPECT().GetMemberRole(context.Background(), int64(100), int64(1)).Return("owner", nil)
				f.chatRepo.EXPECT().DeleteChat(context.Background(), int64(1)).Return(nil)
			},
			args: args{ctx: context.Background(), userID: 100, chatID: 1},
		},
		{
			name: "Delete dialog chat",
			prepare: func(f *fields) {
				f.chatRepo.EXPECT().IsMember(context.Background(), int64(2), int64(100)).Return(true, nil)
				f.chatRepo.EXPECT().GetChatByID(context.Background(), int64(2)).Return(&domain.Chat{
					Id:      2,
					Type:    domain.ChatTypeDialog,
					OwnerId: 100,
				}, nil)
				f.chatRepo.EXPECT().DeleteChat(context.Background(), int64(2)).Return(nil)
			},
			args: args{ctx: context.Background(), userID: 100, chatID: 2},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				chatRepo:  mock.NewMockChatRepositoryInterface(ctrl),
				userRepo:  mock.NewMockUserRepositoryInterface(ctrl),
				mediaRepo: mock.NewMockMediaRepositoryInterface(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &ChatService{
				chatRepo:  f.chatRepo,
				userRepo:  f.userRepo,
				mediaRepo: f.mediaRepo,
			}

			err := s.DeleteChat(tt.args.ctx, tt.args.userID, tt.args.chatID)
			require.NoError(t, err)
		})
	}
}

func TestNegativeChatService_DeleteChat(t *testing.T) {
	type fields struct {
		chatRepo  *mock.MockChatRepositoryInterface
		userRepo  *mock.MockUserRepositoryInterface
		mediaRepo *mock.MockMediaRepositoryInterface
	}

	type args struct {
		ctx    context.Context
		userID int64
		chatID int64
	}

	tests := []struct {
		wantErr    error
		prepare    func(*fields)
		name       string
		args       args
		wantAnyErr bool
	}{
		{
			name: "User not member",
			prepare: func(f *fields) {
				f.chatRepo.EXPECT().IsMember(context.Background(), int64(1), int64(100)).Return(false, nil)
			},
			args:    args{ctx: context.Background(), userID: 100, chatID: 1},
			wantErr: domain.ErrNotMember,
		},
		{
			name: "Group chat - user not owner",
			prepare: func(f *fields) {
				f.chatRepo.EXPECT().IsMember(context.Background(), int64(1), int64(101)).Return(true, nil)
				f.chatRepo.EXPECT().GetChatByID(context.Background(), int64(1)).Return(&domain.Chat{
					Id:      1,
					Type:    domain.ChatTypeGroup,
					OwnerId: 100,
				}, nil)
				f.chatRepo.EXPECT().GetMemberRole(context.Background(), int64(101), int64(1)).Return("member", nil)
			},
			args:    args{ctx: context.Background(), userID: 101, chatID: 1},
			wantErr: domain.ErrOnlyOwnerCanDeleteChat,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				chatRepo:  mock.NewMockChatRepositoryInterface(ctrl),
				userRepo:  mock.NewMockUserRepositoryInterface(ctrl),
				mediaRepo: mock.NewMockMediaRepositoryInterface(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &ChatService{
				chatRepo:  f.chatRepo,
				userRepo:  f.userRepo,
				mediaRepo: f.mediaRepo,
			}

			err := s.DeleteChat(tt.args.ctx, tt.args.userID, tt.args.chatID)
			if tt.wantAnyErr {
				require.Error(t, err)
			} else {
				require.EqualError(t, err, tt.wantErr.Error())
			}
		})
	}
}

func TestPositiveChatService_UpdateChatAvatar(t *testing.T) {
	type fields struct {
		chatRepo  *mock.MockChatRepositoryInterface
		userRepo  *mock.MockUserRepositoryInterface
		mediaRepo *mock.MockMediaRepositoryInterface
	}

	type args struct {
		ctx     context.Context
		request *dto.RequestUpdateAvatar
		userID  int64
		chatID  int64
	}

	fileInput := &media.FileInput{
		Body:        bytes.NewBufferString("avatar-bytes"),
		ContentType: "image/png",
		Size:        1024,
	}

	now := time.Now()

	tests := []struct {
		prepare func(*fields)
		want    *dto.ChatInformationDTO
		name    string
		args    args
	}{
		{
			name: "Update group chat avatar",
			prepare: func(f *fields) {
				f.chatRepo.EXPECT().IsMember(context.Background(), int64(1), int64(100)).Return(true, nil)
				f.chatRepo.EXPECT().GetChatByID(context.Background(), int64(1)).Return(&domain.Chat{
					Id:   1,
					Type: domain.ChatTypeGroup,
				}, nil)
				f.mediaRepo.EXPECT().UploadChatAvatar(context.Background(), int64(1), fileInput).Return("new_avatar_url", nil)
				f.chatRepo.EXPECT().UploadAvatarUrl(context.Background(), int64(1), "new_avatar_url").Return(&domain.Chat{
					Id:        1,
					Type:      domain.ChatTypeGroup,
					Title:     "Group",
					AvatarUrl: strPtr("new_avatar_url"),
				}, nil)
				f.chatRepo.EXPECT().GetLastMessageOfChat(context.Background(), int64(1)).Return(&domain.Message{
					SenderId:  101,
					Content:   "Hello",
					CreatedAt: now,
				}, nil)
			},
			args: args{
				ctx:     context.Background(),
				userID:  100,
				chatID:  1,
				request: &dto.RequestUpdateAvatar{File: fileInput},
			},
			want: &dto.ChatInformationDTO{
				ID:       1,
				ChatType: dto.ChatTypeGroup,
				Title:    "Group",
				LastMessage: dto.MessageDTO{
					SenderId:  101,
					Text:      "Hello",
					CreatedAt: now,
				},
				Avatar: strPtr("new_avatar_url"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				chatRepo:  mock.NewMockChatRepositoryInterface(ctrl),
				userRepo:  mock.NewMockUserRepositoryInterface(ctrl),
				mediaRepo: mock.NewMockMediaRepositoryInterface(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &ChatService{
				chatRepo:  f.chatRepo,
				userRepo:  f.userRepo,
				mediaRepo: f.mediaRepo,
			}

			result, err := s.UpdateChatAvatar(tt.args.ctx, tt.args.userID, tt.args.chatID, tt.args.request)
			require.NoError(t, err)
			require.Equal(t, tt.want, result)
		})
	}
}

func TestNegativeChatService_UpdateChatAvatar(t *testing.T) {
	type fields struct {
		chatRepo  *mock.MockChatRepositoryInterface
		userRepo  *mock.MockUserRepositoryInterface
		mediaRepo *mock.MockMediaRepositoryInterface
	}

	type args struct {
		ctx     context.Context
		request *dto.RequestUpdateAvatar
		userID  int64
		chatID  int64
	}

	tests := []struct {
		wantErr    error
		prepare    func(*fields)
		name       string
		args       args
		wantAnyErr bool
	}{
		{
			name: "Empty file - nil request",
			args: args{
				ctx:     context.Background(),
				userID:  100,
				chatID:  1,
				request: nil,
			},
			wantErr: errors.New("update profile avatar nil request"),
		},
		{
			name: "Empty file - nil file input",
			args: args{
				ctx:     context.Background(),
				userID:  100,
				chatID:  1,
				request: &dto.RequestUpdateAvatar{File: nil},
			},
			wantErr: domainProfile.ErrEmptyAvatar,
		},
		{
			name: "Empty file - nil body",
			args: args{
				ctx:    context.Background(),
				userID: 100,
				chatID: 1,
				request: &dto.RequestUpdateAvatar{File: &media.FileInput{
					Body:        nil,
					ContentType: "image/png",
					Size:        1024,
				}},
			},
			wantErr: domainProfile.ErrEmptyAvatar,
		},
		{
			name: "Empty file - zero size",
			args: args{
				ctx:    context.Background(),
				userID: 100,
				chatID: 1,
				request: &dto.RequestUpdateAvatar{File: &media.FileInput{
					Body:        bytes.NewBufferString("test"),
					ContentType: "image/png",
					Size:        0,
				}},
			},
			wantErr: domainProfile.ErrEmptyAvatar,
		},
		{
			name: "Invalid file type",
			args: args{
				ctx:    context.Background(),
				userID: 100,
				chatID: 1,
				request: &dto.RequestUpdateAvatar{File: &media.FileInput{
					Body:        bytes.NewBufferString("test"),
					ContentType: "text/plain",
					Size:        1024,
				}},
			},
			wantErr: domainProfile.ErrInvalidAvatarType,
		},
		{
			name: "File too large",
			args: args{
				ctx:    context.Background(),
				userID: 100,
				chatID: 1,
				request: &dto.RequestUpdateAvatar{File: &media.FileInput{
					Body:        bytes.NewBufferString("test"),
					ContentType: "image/png",
					Size:        6 * 1024 * 1024,
				}},
			},
			wantErr: domainProfile.ErrAvatarTooLarge,
		},
		{
			name: "Cannot update dialog avatar",
			prepare: func(f *fields) {
				f.chatRepo.EXPECT().IsMember(context.Background(), int64(1), int64(100)).Return(true, nil)
				f.chatRepo.EXPECT().GetChatByID(context.Background(), int64(1)).Return(&domain.Chat{
					Id:   1,
					Type: domain.ChatTypeDialog,
				}, nil)
			},
			args: args{
				ctx:    context.Background(),
				userID: 100,
				chatID: 1,
				request: &dto.RequestUpdateAvatar{File: &media.FileInput{
					Body:        bytes.NewBufferString("test"),
					ContentType: "image/png",
					Size:        1024,
				}},
			},
			wantErr: domain.ErrDialogCannotHaveCustomAvatar,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				chatRepo:  mock.NewMockChatRepositoryInterface(ctrl),
				userRepo:  mock.NewMockUserRepositoryInterface(ctrl),
				mediaRepo: mock.NewMockMediaRepositoryInterface(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &ChatService{
				chatRepo:  f.chatRepo,
				userRepo:  f.userRepo,
				mediaRepo: f.mediaRepo,
			}

			result, err := s.UpdateChatAvatar(tt.args.ctx, tt.args.userID, tt.args.chatID, tt.args.request)
			require.Nil(t, result)
			if tt.wantAnyErr {
				require.Error(t, err)
			} else {
				require.EqualError(t, err, tt.wantErr.Error())
			}
		})
	}
}

func TestPositiveChatService_QuitChat(t *testing.T) {
	type fields struct {
		chatRepo  *mock.MockChatRepositoryInterface
		userRepo  *mock.MockUserRepositoryInterface
		mediaRepo *mock.MockMediaRepositoryInterface
	}

	type args struct {
		ctx    context.Context
		userID int64
		chatID int64
	}

	tests := []struct {
		name    string
		prepare func(*fields)
		args    args
	}{
		{
			name: "Quit group chat successfully",
			prepare: func(f *fields) {
				f.chatRepo.EXPECT().IsMember(context.Background(), int64(1), int64(101)).Return(true, nil)
				f.chatRepo.EXPECT().GetChatByID(context.Background(), int64(1)).Return(&domain.Chat{
					Id:      1,
					Type:    domain.ChatTypeGroup,
					OwnerId: 100,
				}, nil)
				f.chatRepo.EXPECT().DeleteMember(context.Background(), int64(1), int64(101)).Return(nil)
			},
			args: args{ctx: context.Background(), userID: 101, chatID: 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				chatRepo:  mock.NewMockChatRepositoryInterface(ctrl),
				userRepo:  mock.NewMockUserRepositoryInterface(ctrl),
				mediaRepo: mock.NewMockMediaRepositoryInterface(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &ChatService{
				chatRepo:  f.chatRepo,
				userRepo:  f.userRepo,
				mediaRepo: f.mediaRepo,
			}

			err := s.QuitChat(tt.args.ctx, tt.args.userID, tt.args.chatID)
			require.NoError(t, err)
		})
	}
}

func TestNegativeChatService_QuitChat(t *testing.T) {
	type fields struct {
		chatRepo  *mock.MockChatRepositoryInterface
		userRepo  *mock.MockUserRepositoryInterface
		mediaRepo *mock.MockMediaRepositoryInterface
	}

	type args struct {
		ctx    context.Context
		userID int64
		chatID int64
	}

	tests := []struct {
		wantErr    error
		prepare    func(*fields)
		name       string
		args       args
		wantAnyErr bool
	}{
		{
			name: "Cannot quit dialog",
			prepare: func(f *fields) {
				f.chatRepo.EXPECT().IsMember(context.Background(), int64(1), int64(100)).Return(true, nil)
				f.chatRepo.EXPECT().GetChatByID(context.Background(), int64(1)).Return(&domain.Chat{
					Id:   1,
					Type: domain.ChatTypeDialog,
				}, nil)
			},
			args:    args{ctx: context.Background(), userID: 100, chatID: 1},
			wantErr: domain.ErrCantQuitDialog,
		},
		{
			name: "Owner cannot quit group",
			prepare: func(f *fields) {
				f.chatRepo.EXPECT().IsMember(context.Background(), int64(1), int64(100)).Return(true, nil)
				f.chatRepo.EXPECT().GetChatByID(context.Background(), int64(1)).Return(&domain.Chat{
					Id:      1,
					Type:    domain.ChatTypeGroup,
					OwnerId: 100,
				}, nil)
			},
			args:    args{ctx: context.Background(), userID: 100, chatID: 1},
			wantErr: domain.ErrOwnerCantQuitGroup,
		},
		{
			name: "User not member",
			prepare: func(f *fields) {
				f.chatRepo.EXPECT().IsMember(context.Background(), int64(1), int64(101)).Return(false, nil)
			},
			args:    args{ctx: context.Background(), userID: 101, chatID: 1},
			wantErr: domain.ErrNotMember,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				chatRepo:  mock.NewMockChatRepositoryInterface(ctrl),
				userRepo:  mock.NewMockUserRepositoryInterface(ctrl),
				mediaRepo: mock.NewMockMediaRepositoryInterface(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &ChatService{
				chatRepo:  f.chatRepo,
				userRepo:  f.userRepo,
				mediaRepo: f.mediaRepo,
			}

			err := s.QuitChat(tt.args.ctx, tt.args.userID, tt.args.chatID)
			if tt.wantAnyErr {
				require.Error(t, err)
			} else {
				require.EqualError(t, err, tt.wantErr.Error())
			}
		})
	}
}

func TestPositiveChatService_GetAllChats(t *testing.T) {
	type fields struct {
		chatRepo  *mock.MockChatRepositoryInterface
		userRepo  *mock.MockUserRepositoryInterface
		mediaRepo *mock.MockMediaRepositoryInterface
	}

	type args struct {
		ctx    context.Context
		userID int64
	}

	now := time.Now()

	tests := []struct {
		name    string
		prepare func(*fields)
		args    args
		want    []dto.ChatInformationDTO
	}{
		{
			name: "Get all chats with group and dialog",
			prepare: func(f *fields) {
				// Мок для получения всех чатов пользователя
				f.chatRepo.EXPECT().GetAllChatsByUserID(context.Background(), int64(100)).Return([]*domain.Chat{
					{
						Id:        1,
						Type:      domain.ChatTypeGroup,
						Title:     "Group Chat",
						OwnerId:   100,
						AvatarUrl: strPtr("group_avatar.jpg"),
						CreatedAt: now,
						UpdatedAt: now,
					},
					{
						Id:        2,
						Type:      domain.ChatTypeDialog,
						Title:     "",
						OwnerId:   100,
						AvatarUrl: nil,
						CreatedAt: now,
						UpdatedAt: now,
					},
				}, nil)

				// Мок для получения последних сообщений
				f.chatRepo.EXPECT().GetLastMessagesOfChats(context.Background(), int64(100)).Return([]*domain.Message{
					{
						Id:        10,
						ChatId:    1,
						SenderId:  101,
						Content:   "Hello group!",
						CreatedAt: now,
					},
					{
						Id:        20,
						ChatId:    2,
						SenderId:  102,
						Content:   "Hi in dialog!",
						CreatedAt: now,
					},
				}, nil)

				// Моки для GetDialogName и GetDialogAvatar (для диалога)
				f.chatRepo.EXPECT().GetChatMembers(context.Background(), int64(2)).Return([]int64{100, 102}, nil).Times(2)
				f.userRepo.EXPECT().GetUserByID(context.Background(), int64(102)).Return(&domainUser.User{
					Id:        102,
					Login:     "friend_user",
					FirstName: "Friend",
					LastName:  strPtr("User"),
					AvatarUrl: strPtr("friend_avatar.jpg"),
				}, nil).Times(2)
			},
			args: args{ctx: context.Background(), userID: 100},
			want: []dto.ChatInformationDTO{
				{
					ID:       1,
					ChatType: dto.ChatTypeGroup,
					Title:    "Group Chat",
					LastMessage: dto.MessageDTO{
						SenderId:  101,
						Text:      "Hello group!",
						CreatedAt: now,
					},
					Avatar: strPtr("group_avatar.jpg"),
				},
				{
					ID:       2,
					ChatType: dto.ChatTypeDialog,
					Title:    "Friend User",
					LastMessage: dto.MessageDTO{
						SenderId:  102,
						Text:      "Hi in dialog!",
						CreatedAt: now,
					},
					Avatar: strPtr("friend_avatar.jpg"),
				},
			},
		},
		{
			name: "Get chats when no last messages",
			prepare: func(f *fields) {
				f.chatRepo.EXPECT().GetAllChatsByUserID(context.Background(), int64(100)).Return([]*domain.Chat{
					{
						Id:        1,
						Type:      domain.ChatTypeGroup,
						Title:     "Empty Group",
						OwnerId:   100,
						CreatedAt: now,
						UpdatedAt: now,
					},
				}, nil)

				f.chatRepo.EXPECT().GetLastMessagesOfChats(context.Background(), int64(100)).Return(nil, errors.New("no messages"))
			},
			args: args{ctx: context.Background(), userID: 100},
			want: []dto.ChatInformationDTO{
				{
					ID:          1,
					ChatType:    dto.ChatTypeGroup,
					Title:       "Empty Group",
					LastMessage: dto.MessageDTO{},
					Avatar:      nil,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				chatRepo:  mock.NewMockChatRepositoryInterface(ctrl),
				userRepo:  mock.NewMockUserRepositoryInterface(ctrl),
				mediaRepo: mock.NewMockMediaRepositoryInterface(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &ChatService{
				chatRepo:  f.chatRepo,
				userRepo:  f.userRepo,
				mediaRepo: f.mediaRepo,
			}

			result, err := s.GetAllChats(tt.args.ctx, tt.args.userID)
			require.NoError(t, err)
			require.Equal(t, len(tt.want), len(result))
			for i := range tt.want {
				require.Equal(t, tt.want[i].ID, result[i].ID)
				require.Equal(t, tt.want[i].ChatType, result[i].ChatType)
				require.Equal(t, tt.want[i].Title, result[i].Title)
				require.Equal(t, tt.want[i].Avatar, result[i].Avatar)
			}
		})
	}
}

func TestNegativeChatService_GetAllChats(t *testing.T) {
	type fields struct {
		chatRepo  *mock.MockChatRepositoryInterface
		userRepo  *mock.MockUserRepositoryInterface
		mediaRepo *mock.MockMediaRepositoryInterface
	}

	type args struct {
		ctx    context.Context
		userID int64
	}

	tests := []struct {
		name       string
		prepare    func(*fields)
		args       args
		wantAnyErr bool
	}{
		{
			name: "GetAllChatsByUserID error",
			prepare: func(f *fields) {
				f.chatRepo.EXPECT().GetAllChatsByUserID(context.Background(), int64(100)).Return(nil, errors.New("db error"))
			},
			args:       args{ctx: context.Background(), userID: 100},
			wantAnyErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				chatRepo:  mock.NewMockChatRepositoryInterface(ctrl),
				userRepo:  mock.NewMockUserRepositoryInterface(ctrl),
				mediaRepo: mock.NewMockMediaRepositoryInterface(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &ChatService{
				chatRepo:  f.chatRepo,
				userRepo:  f.userRepo,
				mediaRepo: f.mediaRepo,
			}

			_, err := s.GetAllChats(tt.args.ctx, tt.args.userID)
			if tt.wantAnyErr {
				require.Error(t, err)
			}
		})
	}
}

func TestPositiveChatService_CreateChatEscapesTitle(t *testing.T) {
	type fields struct {
		chatRepo  *mock.MockChatRepositoryInterface
		userRepo  *mock.MockUserRepositoryInterface
		mediaRepo *mock.MockMediaRepositoryInterface
	}

	type args struct {
		ctx     context.Context
		payload dto.ChatCreate
		ownerID int64
	}

	tests := []struct {
		name      string
		prepare   func(*fields)
		args      args
		wantTitle string
	}{
		{
			name: "bold markup",
			prepare: func(f *fields) {
				rawTitle := `<b>Group</b>`
				f.chatRepo.EXPECT().
					CreateChat(gomock.Any(), gomock.AssignableToTypeOf(&domain.Chat{})).
					DoAndReturn(func(_ context.Context, c *domain.Chat) (*domain.Chat, error) {
						require.Equal(t, rawTitle, c.Title)
						return &domain.Chat{Id: 77, Type: domain.ChatTypeGroup, Title: rawTitle}, nil
					})
				f.userRepo.EXPECT().GetUserByID(context.Background(), int64(100)).Return(&domainUser.User{Id: 100}, nil)
				f.userRepo.EXPECT().GetUserByID(context.Background(), int64(101)).Return(&domainUser.User{Id: 101}, nil)
				f.chatRepo.EXPECT().AddMember(context.Background(), int64(77), int64(100), "owner").Return(nil)
				f.chatRepo.EXPECT().AddMember(context.Background(), int64(77), int64(101), "member").Return(nil)
			},
			args: args{
				ctx: context.Background(),
				payload: dto.ChatCreate{
					Type:      dto.ChatTypeGroup,
					Title:     `<b>Group</b>`,
					MembersID: []int64{101},
				},
				ownerID: 100,
			},
			wantTitle: `&lt;b&gt;Group&lt;/b&gt;`,
		},
		{
			name: "ampersand in title",
			prepare: func(f *fields) {
				rawTitle := `A & B team`
				f.chatRepo.EXPECT().
					CreateChat(gomock.Any(), gomock.AssignableToTypeOf(&domain.Chat{})).
					DoAndReturn(func(_ context.Context, c *domain.Chat) (*domain.Chat, error) {
						require.Equal(t, rawTitle, c.Title)
						return &domain.Chat{Id: 78, Type: domain.ChatTypeGroup, Title: rawTitle}, nil
					})
				f.userRepo.EXPECT().GetUserByID(context.Background(), int64(10)).Return(&domainUser.User{Id: 10}, nil)
				f.userRepo.EXPECT().GetUserByID(context.Background(), int64(11)).Return(&domainUser.User{Id: 11}, nil)
				f.chatRepo.EXPECT().AddMember(context.Background(), int64(78), int64(10), "owner").Return(nil)
				f.chatRepo.EXPECT().AddMember(context.Background(), int64(78), int64(11), "member").Return(nil)
			},
			args: args{
				ctx: context.Background(),
				payload: dto.ChatCreate{
					Type:      dto.ChatTypeGroup,
					Title:     `A & B team`,
					MembersID: []int64{11},
				},
				ownerID: 10,
			},
			wantTitle: `A &amp; B team`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				chatRepo:  mock.NewMockChatRepositoryInterface(ctrl),
				userRepo:  mock.NewMockUserRepositoryInterface(ctrl),
				mediaRepo: mock.NewMockMediaRepositoryInterface(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &ChatService{
				chatRepo:  f.chatRepo,
				userRepo:  f.userRepo,
				mediaRepo: f.mediaRepo,
			}
			resp, err := s.CreateChat(tt.args.ctx, tt.args.payload, tt.args.ownerID)
			require.NoError(t, err)
			require.Equal(t, tt.wantTitle, resp.Title)
		})
	}
}

func TestPositiveChatService_UpdateChatTitle(t *testing.T) {
	type fields struct {
		chatRepo  *mock.MockChatRepositoryInterface
		userRepo  *mock.MockUserRepositoryInterface
		mediaRepo *mock.MockMediaRepositoryInterface
	}

	type args struct {
		ctx     context.Context
		request *dto.RequestUpdateTitle
		userID  int64
		chatID  int64
	}

	now := time.Now()

	tests := []struct {
		prepare func(*fields)
		want    *dto.ChatInformationDTO
		name    string
		args    args
	}{
		{
			name: "Update group chat title successfully",
			prepare: func(f *fields) {
				f.chatRepo.EXPECT().IsMember(context.Background(), int64(1), int64(100)).Return(true, nil)
				f.chatRepo.EXPECT().GetChatByID(context.Background(), int64(1)).Return(&domain.Chat{
					Id:   1,
					Type: domain.ChatTypeGroup,
				}, nil)
				f.chatRepo.EXPECT().UpdateTitle(context.Background(), int64(1), "New Title").Return(&domain.Chat{
					Id:        1,
					Type:      domain.ChatTypeGroup,
					Title:     "New Title",
					AvatarUrl: strPtr("avatar.jpg"),
				}, nil)
				f.chatRepo.EXPECT().GetLastMessageOfChat(context.Background(), int64(1)).Return(&domain.Message{
					SenderId:  101,
					Content:   "Hello",
					CreatedAt: now,
				}, nil)
			},
			args: args{
				ctx:     context.Background(),
				userID:  100,
				chatID:  1,
				request: &dto.RequestUpdateTitle{Title: "New Title"},
			},
			want: &dto.ChatInformationDTO{
				ID:       1,
				ChatType: dto.ChatTypeGroup,
				Title:    "New Title",
				LastMessage: dto.MessageDTO{
					SenderId:  101,
					Text:      "Hello",
					CreatedAt: now,
				},
				Avatar: strPtr("avatar.jpg"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				chatRepo:  mock.NewMockChatRepositoryInterface(ctrl),
				userRepo:  mock.NewMockUserRepositoryInterface(ctrl),
				mediaRepo: mock.NewMockMediaRepositoryInterface(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &ChatService{
				chatRepo:  f.chatRepo,
				userRepo:  f.userRepo,
				mediaRepo: f.mediaRepo,
			}

			result, err := s.UpdateChatTitle(tt.args.ctx, tt.args.userID, tt.args.chatID, tt.args.request)
			require.NoError(t, err)
			require.Equal(t, tt.want, result)
		})
	}
}

func TestPositiveChatService_UpdateChatTitleEscapesHTML(t *testing.T) {
	type fields struct {
		chatRepo  *mock.MockChatRepositoryInterface
		userRepo  *mock.MockUserRepositoryInterface
		mediaRepo *mock.MockMediaRepositoryInterface
	}

	type args struct {
		ctx     context.Context
		userID  int64
		chatID  int64
		request *dto.RequestUpdateTitle
	}

	tests := []struct {
		name      string
		prepare   func(*fields)
		args      args
		wantTitle string
	}{
		{
			name: "script in title",
			prepare: func(f *fields) {
				rawTitle := `<script>alert(1)</script>`
				f.chatRepo.EXPECT().IsMember(context.Background(), int64(11), int64(100)).Return(true, nil)
				f.chatRepo.EXPECT().GetChatByID(context.Background(), int64(11)).Return(&domain.Chat{
					Id:   11,
					Type: domain.ChatTypeGroup,
				}, nil)
				f.chatRepo.EXPECT().UpdateTitle(context.Background(), int64(11), rawTitle).Return(&domain.Chat{
					Id:    11,
					Type:  domain.ChatTypeGroup,
					Title: rawTitle,
				}, nil)
				f.chatRepo.EXPECT().GetLastMessageOfChat(context.Background(), int64(11)).Return(&domain.Message{}, domain.ErrNoMessage)
			},
			args: args{
				ctx:     context.Background(),
				userID:  100,
				chatID:  11,
				request: &dto.RequestUpdateTitle{Title: `<script>alert(1)</script>`},
			},
			wantTitle: `&lt;script&gt;alert(1)&lt;/script&gt;`,
		},
		{
			name: "ampersand",
			prepare: func(f *fields) {
				rawTitle := `X & Y`
				f.chatRepo.EXPECT().IsMember(context.Background(), int64(12), int64(200)).Return(true, nil)
				f.chatRepo.EXPECT().GetChatByID(context.Background(), int64(12)).Return(&domain.Chat{
					Id:   12,
					Type: domain.ChatTypeGroup,
				}, nil)
				f.chatRepo.EXPECT().UpdateTitle(context.Background(), int64(12), rawTitle).Return(&domain.Chat{
					Id:    12,
					Type:  domain.ChatTypeGroup,
					Title: rawTitle,
				}, nil)
				f.chatRepo.EXPECT().GetLastMessageOfChat(context.Background(), int64(12)).Return(&domain.Message{}, domain.ErrNoMessage)
			},
			args: args{
				ctx:     context.Background(),
				userID:  200,
				chatID:  12,
				request: &dto.RequestUpdateTitle{Title: `X & Y`},
			},
			wantTitle: `X &amp; Y`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				chatRepo:  mock.NewMockChatRepositoryInterface(ctrl),
				userRepo:  mock.NewMockUserRepositoryInterface(ctrl),
				mediaRepo: mock.NewMockMediaRepositoryInterface(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &ChatService{
				chatRepo:  f.chatRepo,
				userRepo:  f.userRepo,
				mediaRepo: f.mediaRepo,
			}
			resp, err := s.UpdateChatTitle(tt.args.ctx, tt.args.userID, tt.args.chatID, tt.args.request)
			require.NoError(t, err)
			require.Equal(t, tt.wantTitle, resp.Title)
		})
	}
}

func TestNegativeChatService_UpdateChatTitle(t *testing.T) {
	type fields struct {
		chatRepo  *mock.MockChatRepositoryInterface
		userRepo  *mock.MockUserRepositoryInterface
		mediaRepo *mock.MockMediaRepositoryInterface
	}

	type args struct {
		ctx     context.Context
		request *dto.RequestUpdateTitle
		userID  int64
		chatID  int64
	}

	tests := []struct {
		wantErr    error
		prepare    func(*fields)
		name       string
		args       args
		wantAnyErr bool
	}{
		{
			name: "User not member",
			prepare: func(f *fields) {
				f.chatRepo.EXPECT().IsMember(context.Background(), int64(1), int64(100)).Return(false, nil)
			},
			args: args{
				ctx:     context.Background(),
				userID:  100,
				chatID:  1,
				request: &dto.RequestUpdateTitle{Title: "New Title"},
			},
			wantErr: domain.ErrNotMember,
		},
		{
			name: "Dialog cannot have custom title",
			prepare: func(f *fields) {
				f.chatRepo.EXPECT().IsMember(context.Background(), int64(1), int64(100)).Return(true, nil)
				f.chatRepo.EXPECT().GetChatByID(context.Background(), int64(1)).Return(&domain.Chat{
					Id:   1,
					Type: domain.ChatTypeDialog,
				}, nil)
			},
			args: args{
				ctx:     context.Background(),
				userID:  100,
				chatID:  1,
				request: &dto.RequestUpdateTitle{Title: "New Title"},
			},
			wantErr: domain.ErrDialogCannotHaveCustomTitle,
		},
		{
			name: "Chat not found",
			prepare: func(f *fields) {
				f.chatRepo.EXPECT().IsMember(context.Background(), int64(1), int64(100)).Return(true, nil)
				f.chatRepo.EXPECT().GetChatByID(context.Background(), int64(1)).Return(nil, domain.ErrChatNotFound)
			},
			args: args{
				ctx:     context.Background(),
				userID:  100,
				chatID:  1,
				request: &dto.RequestUpdateTitle{Title: "New Title"},
			},
			wantErr: domain.ErrChatNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				chatRepo:  mock.NewMockChatRepositoryInterface(ctrl),
				userRepo:  mock.NewMockUserRepositoryInterface(ctrl),
				mediaRepo: mock.NewMockMediaRepositoryInterface(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &ChatService{
				chatRepo:  f.chatRepo,
				userRepo:  f.userRepo,
				mediaRepo: f.mediaRepo,
			}

			result, err := s.UpdateChatTitle(tt.args.ctx, tt.args.userID, tt.args.chatID, tt.args.request)
			require.Nil(t, result)
			if tt.wantAnyErr {
				require.Error(t, err)
			} else {
				require.EqualError(t, err, tt.wantErr.Error())
			}
		})
	}
}

func TestPositiveChatService_AddMembersToChat(t *testing.T) {
	type fields struct {
		chatRepo  *mock.MockChatRepositoryInterface
		userRepo  *mock.MockUserRepositoryInterface
		mediaRepo *mock.MockMediaRepositoryInterface
	}

	type args struct {
		ctx     context.Context
		request *dto.RequestAddMember
		userID  int64
		chatID  int64
	}

	tests := []struct {
		prepare func(*fields)
		name    string
		args    args
	}{
		{
			name: "Add member to group chat as owner",
			prepare: func(f *fields) {
				f.chatRepo.EXPECT().IsMember(context.Background(), int64(1), int64(100)).Return(true, nil)
				f.chatRepo.EXPECT().GetChatByID(context.Background(), int64(1)).Return(&domain.Chat{
					Id:      1,
					Type:    domain.ChatTypeGroup,
					OwnerId: 100,
				}, nil)
				f.chatRepo.EXPECT().IsMember(context.Background(), int64(1), int64(101)).Return(false, nil)
				f.chatRepo.EXPECT().AddMember(context.Background(), int64(1), int64(101), "member").Return(nil)
			},
			args: args{
				ctx:     context.Background(),
				userID:  100,
				chatID:  1,
				request: &dto.RequestAddMember{MembersId: []int64{101}},
			},
		},
		{
			name: "Add multiple members to group chat",
			prepare: func(f *fields) {
				f.chatRepo.EXPECT().IsMember(context.Background(), int64(1), int64(100)).Return(true, nil)
				f.chatRepo.EXPECT().GetChatByID(context.Background(), int64(1)).Return(&domain.Chat{
					Id:      1,
					Type:    domain.ChatTypeGroup,
					OwnerId: 100,
				}, nil)
				f.chatRepo.EXPECT().IsMember(context.Background(), int64(1), int64(101)).Return(false, nil)
				f.chatRepo.EXPECT().IsMember(context.Background(), int64(1), int64(102)).Return(false, nil)
				f.chatRepo.EXPECT().AddMember(context.Background(), int64(1), int64(101), "member").Return(nil)
				f.chatRepo.EXPECT().AddMember(context.Background(), int64(1), int64(102), "member").Return(nil)
			},
			args: args{
				ctx:     context.Background(),
				userID:  100,
				chatID:  1,
				request: &dto.RequestAddMember{MembersId: []int64{101, 102}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				chatRepo:  mock.NewMockChatRepositoryInterface(ctrl),
				userRepo:  mock.NewMockUserRepositoryInterface(ctrl),
				mediaRepo: mock.NewMockMediaRepositoryInterface(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &ChatService{
				chatRepo:  f.chatRepo,
				userRepo:  f.userRepo,
				mediaRepo: f.mediaRepo,
			}

			err := s.AddMembersToChat(tt.args.ctx, tt.args.userID, tt.args.chatID, tt.args.request)
			require.NoError(t, err)
		})
	}
}

func TestNegativeChatService_AddMembersToChat(t *testing.T) {
	type fields struct {
		chatRepo  *mock.MockChatRepositoryInterface
		userRepo  *mock.MockUserRepositoryInterface
		mediaRepo *mock.MockMediaRepositoryInterface
	}

	type args struct {
		ctx     context.Context
		request *dto.RequestAddMember
		userID  int64
		chatID  int64
	}

	tests := []struct {
		wantErr    error
		prepare    func(*fields)
		name       string
		args       args
		wantAnyErr bool
	}{
		{
			name: "User not member",
			prepare: func(f *fields) {
				f.chatRepo.EXPECT().IsMember(context.Background(), int64(1), int64(100)).Return(false, nil)
			},
			args: args{
				ctx:     context.Background(),
				userID:  100,
				chatID:  1,
				request: &dto.RequestAddMember{MembersId: []int64{101}},
			},
			wantErr: domain.ErrNotMember,
		},
		{
			name: "Cannot add member to dialog",
			prepare: func(f *fields) {
				f.chatRepo.EXPECT().IsMember(context.Background(), int64(1), int64(100)).Return(true, nil)
				f.chatRepo.EXPECT().GetChatByID(context.Background(), int64(1)).Return(&domain.Chat{
					Id:   1,
					Type: domain.ChatTypeDialog,
				}, nil)
			},
			args: args{
				ctx:     context.Background(),
				userID:  100,
				chatID:  1,
				request: &dto.RequestAddMember{MembersId: []int64{101}},
			},
			wantErr: domain.ErrCantAddMemberToDialog,
		},
		{
			name: "Only owner can add members",
			prepare: func(f *fields) {
				f.chatRepo.EXPECT().IsMember(context.Background(), int64(1), int64(101)).Return(true, nil)
				f.chatRepo.EXPECT().GetChatByID(context.Background(), int64(1)).Return(&domain.Chat{
					Id:      1,
					Type:    domain.ChatTypeGroup,
					OwnerId: 100,
				}, nil)
			},
			args: args{
				ctx:     context.Background(),
				userID:  101,
				chatID:  1,
				request: &dto.RequestAddMember{MembersId: []int64{102}},
			},
			wantErr: domain.ErrOnlyOwnerCanAddPeople,
		},
		{
			name: "Member already in chat",
			prepare: func(f *fields) {
				f.chatRepo.EXPECT().IsMember(context.Background(), int64(1), int64(100)).Return(true, nil)
				f.chatRepo.EXPECT().GetChatByID(context.Background(), int64(1)).Return(&domain.Chat{
					Id:      1,
					Type:    domain.ChatTypeGroup,
					OwnerId: 100,
				}, nil)
				f.chatRepo.EXPECT().IsMember(context.Background(), int64(1), int64(101)).Return(true, nil)
			},
			args: args{
				ctx:     context.Background(),
				userID:  100,
				chatID:  1,
				request: &dto.RequestAddMember{MembersId: []int64{101}},
			},
			wantErr: domain.ErrMemberAlreadyInChat,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				chatRepo:  mock.NewMockChatRepositoryInterface(ctrl),
				userRepo:  mock.NewMockUserRepositoryInterface(ctrl),
				mediaRepo: mock.NewMockMediaRepositoryInterface(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &ChatService{
				chatRepo:  f.chatRepo,
				userRepo:  f.userRepo,
				mediaRepo: f.mediaRepo,
			}

			err := s.AddMembersToChat(tt.args.ctx, tt.args.userID, tt.args.chatID, tt.args.request)
			if tt.wantAnyErr {
				require.Error(t, err)
			} else {
				require.EqualError(t, err, tt.wantErr.Error())
			}
		})
	}
}

func TestPositiveChatService_DeleteMemberFromChat(t *testing.T) {
	type fields struct {
		chatRepo  *mock.MockChatRepositoryInterface
		userRepo  *mock.MockUserRepositoryInterface
		mediaRepo *mock.MockMediaRepositoryInterface
	}

	type args struct {
		ctx     context.Context
		request *dto.RequestDeleteMember
		userID  int64
		chatID  int64
	}

	tests := []struct {
		prepare func(*fields)
		name    string
		args    args
	}{
		{
			name: "Delete member from group chat as owner",
			prepare: func(f *fields) {
				f.chatRepo.EXPECT().IsMember(context.Background(), int64(1), int64(100)).Return(true, nil)
				f.chatRepo.EXPECT().GetChatByID(context.Background(), int64(1)).Return(&domain.Chat{
					Id:      1,
					Type:    domain.ChatTypeGroup,
					OwnerId: 100,
				}, nil)
				f.chatRepo.EXPECT().IsMember(context.Background(), int64(1), int64(101)).Return(true, nil)
				f.chatRepo.EXPECT().DeleteMember(context.Background(), int64(1), int64(101)).Return(nil)
			},
			args: args{
				ctx:     context.Background(),
				userID:  100,
				chatID:  1,
				request: &dto.RequestDeleteMember{MemberId: 101},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				chatRepo:  mock.NewMockChatRepositoryInterface(ctrl),
				userRepo:  mock.NewMockUserRepositoryInterface(ctrl),
				mediaRepo: mock.NewMockMediaRepositoryInterface(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &ChatService{
				chatRepo:  f.chatRepo,
				userRepo:  f.userRepo,
				mediaRepo: f.mediaRepo,
			}

			err := s.DeleteMemberFromChat(tt.args.ctx, tt.args.userID, tt.args.chatID, tt.args.request)
			require.NoError(t, err)
		})
	}
}

func TestNegativeChatService_DeleteMemberFromChat(t *testing.T) {
	type fields struct {
		chatRepo  *mock.MockChatRepositoryInterface
		userRepo  *mock.MockUserRepositoryInterface
		mediaRepo *mock.MockMediaRepositoryInterface
	}

	type args struct {
		ctx     context.Context
		request *dto.RequestDeleteMember
		userID  int64
		chatID  int64
	}

	tests := []struct {
		wantErr    error
		prepare    func(*fields)
		name       string
		args       args
		wantAnyErr bool
	}{
		{
			name: "User not member",
			prepare: func(f *fields) {
				f.chatRepo.EXPECT().IsMember(context.Background(), int64(1), int64(100)).Return(false, nil)
			},
			args: args{
				ctx:     context.Background(),
				userID:  100,
				chatID:  1,
				request: &dto.RequestDeleteMember{MemberId: 101},
			},
			wantErr: domain.ErrNotMember,
		},
		{
			name: "Cannot delete member from dialog",
			prepare: func(f *fields) {
				f.chatRepo.EXPECT().IsMember(context.Background(), int64(1), int64(100)).Return(true, nil)
				f.chatRepo.EXPECT().GetChatByID(context.Background(), int64(1)).Return(&domain.Chat{
					Id:   1,
					Type: domain.ChatTypeDialog,
				}, nil)
			},
			args: args{
				ctx:     context.Background(),
				userID:  100,
				chatID:  1,
				request: &dto.RequestDeleteMember{MemberId: 101},
			},
			wantErr: domain.ErrCantDeleteMemberFromDialog,
		},
		{
			name: "Only owner can delete members",
			prepare: func(f *fields) {
				f.chatRepo.EXPECT().IsMember(context.Background(), int64(1), int64(101)).Return(true, nil)
				f.chatRepo.EXPECT().GetChatByID(context.Background(), int64(1)).Return(&domain.Chat{
					Id:      1,
					Type:    domain.ChatTypeGroup,
					OwnerId: 100,
				}, nil)
			},
			args: args{
				ctx:     context.Background(),
				userID:  101,
				chatID:  1,
				request: &dto.RequestDeleteMember{MemberId: 102},
			},
			wantErr: domain.ErrOnlyOwnerCanDeletePeople,
		},
		{
			name: "Cannot delete owner",
			prepare: func(f *fields) {
				f.chatRepo.EXPECT().IsMember(context.Background(), int64(1), int64(100)).Return(true, nil)
				f.chatRepo.EXPECT().GetChatByID(context.Background(), int64(1)).Return(&domain.Chat{
					Id:      1,
					Type:    domain.ChatTypeGroup,
					OwnerId: 100,
				}, nil)
			},
			args: args{
				ctx:     context.Background(),
				userID:  100,
				chatID:  1,
				request: &dto.RequestDeleteMember{MemberId: 100},
			},
			wantErr: domain.ErrCantDeleteOwnerOfChat,
		},
		{
			name: "Member not found in chat",
			prepare: func(f *fields) {
				f.chatRepo.EXPECT().IsMember(context.Background(), int64(1), int64(100)).Return(true, nil)
				f.chatRepo.EXPECT().GetChatByID(context.Background(), int64(1)).Return(&domain.Chat{
					Id:      1,
					Type:    domain.ChatTypeGroup,
					OwnerId: 100,
				}, nil)
				f.chatRepo.EXPECT().IsMember(context.Background(), int64(1), int64(999)).Return(false, nil)
			},
			args: args{
				ctx:     context.Background(),
				userID:  100,
				chatID:  1,
				request: &dto.RequestDeleteMember{MemberId: 999},
			},
			wantErr: domain.ErrUserNotMember,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				chatRepo:  mock.NewMockChatRepositoryInterface(ctrl),
				userRepo:  mock.NewMockUserRepositoryInterface(ctrl),
				mediaRepo: mock.NewMockMediaRepositoryInterface(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &ChatService{
				chatRepo:  f.chatRepo,
				userRepo:  f.userRepo,
				mediaRepo: f.mediaRepo,
			}

			err := s.DeleteMemberFromChat(tt.args.ctx, tt.args.userID, tt.args.chatID, tt.args.request)
			if tt.wantAnyErr {
				require.Error(t, err)
			} else {
				require.EqualError(t, err, tt.wantErr.Error())
			}
		})
	}
}

func TestPositiveChatService_GetAllChatMembers(t *testing.T) {
	type fields struct {
		chatRepo  *mock.MockChatRepositoryInterface
		userRepo  *mock.MockUserRepositoryInterface
		mediaRepo *mock.MockMediaRepositoryInterface
	}

	type args struct {
		ctx    context.Context
		userID int64
		chatID int64
	}

	tests := []struct {
		prepare func(*fields)
		want    *dto.ResponseGetChatMembers
		name    string
		args    args
	}{
		{
			name: "Get chat members successfully",
			prepare: func(f *fields) {
				f.chatRepo.EXPECT().IsMember(context.Background(), int64(1), int64(100)).Return(true, nil)
				f.chatRepo.EXPECT().GetChatMembers(context.Background(), int64(1)).Return([]int64{100, 101, 102}, nil)
			},
			args: args{ctx: context.Background(), userID: 100, chatID: 1},
			want: &dto.ResponseGetChatMembers{
				MembersId: []int64{100, 101, 102},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				chatRepo:  mock.NewMockChatRepositoryInterface(ctrl),
				userRepo:  mock.NewMockUserRepositoryInterface(ctrl),
				mediaRepo: mock.NewMockMediaRepositoryInterface(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &ChatService{
				chatRepo:  f.chatRepo,
				userRepo:  f.userRepo,
				mediaRepo: f.mediaRepo,
			}

			result, err := s.GetAllChatMembers(tt.args.ctx, tt.args.userID, tt.args.chatID)
			require.NoError(t, err)
			require.Equal(t, tt.want, result)
		})
	}
}

func TestNegativeChatService_GetAllChatMembers(t *testing.T) {
	type fields struct {
		chatRepo  *mock.MockChatRepositoryInterface
		userRepo  *mock.MockUserRepositoryInterface
		mediaRepo *mock.MockMediaRepositoryInterface
	}

	type args struct {
		ctx    context.Context
		userID int64
		chatID int64
	}

	tests := []struct {
		wantErr    error
		prepare    func(*fields)
		name       string
		args       args
		wantAnyErr bool
	}{
		{
			name: "User not member",
			prepare: func(f *fields) {
				f.chatRepo.EXPECT().IsMember(context.Background(), int64(1), int64(100)).Return(false, nil)
			},
			args:    args{ctx: context.Background(), userID: 100, chatID: 1},
			wantErr: domain.ErrNotMember,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				chatRepo:  mock.NewMockChatRepositoryInterface(ctrl),
				userRepo:  mock.NewMockUserRepositoryInterface(ctrl),
				mediaRepo: mock.NewMockMediaRepositoryInterface(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &ChatService{
				chatRepo:  f.chatRepo,
				userRepo:  f.userRepo,
				mediaRepo: f.mediaRepo,
			}

			_, err := s.GetAllChatMembers(tt.args.ctx, tt.args.userID, tt.args.chatID)
			if tt.wantAnyErr {
				require.Error(t, err)
			} else {
				require.EqualError(t, err, tt.wantErr.Error())
			}
		})
	}
}
