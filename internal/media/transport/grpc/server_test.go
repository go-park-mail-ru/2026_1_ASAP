//go:generate mockgen -destination=mock/media_repository_mock.go -package=mock github.com/go-park-mail-ru/2026_1_ASAP/internal/media/transport/grpc MediaRepositoryInterface
package grpc

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	mediav1 "github.com/go-park-mail-ru/2026_1_ASAP/gen/go/media/v1"
	mediadto "github.com/go-park-mail-ru/2026_1_ASAP/internal/media/dto"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/media/transport/grpc/mock"
)

func TestPositiveMediaServer_UpdateUserAvatar(t *testing.T) {
	type fields struct {
		mediaRepo *mock.MockMediaRepositoryInterface
	}

	type args struct {
		ctx context.Context
		req *mediav1.RequestUpdateUserAvatar
	}

	tests := []struct {
		prepare func(*fields)
		want    *mediav1.ResponseUpdateUserAvatar
		name    string
		args    args
	}{
		{
			name: "Successful update user avatar",
			prepare: func(f *fields) {
				f.mediaRepo.EXPECT().UploadAvatar(gomock.Any(), int64(100), gomock.Any()).DoAndReturn(
					func(ctx context.Context, userID int64, input *mediadto.FileInput) (string, error) {
						require.Equal(t, int64(100), userID)
						require.NotNil(t, input)
						return "https://storage.example.com/avatar.jpg", nil
					})
			},
			args: args{
				ctx: context.Background(),
				req: &mediav1.RequestUpdateUserAvatar{
					UserId: 100,
					Avatar: &mediav1.File{
						Content: []byte("fake image data"),
						Type:    "image/jpeg",
					},
				},
			},
			want: &mediav1.ResponseUpdateUserAvatar{AvatarUrl: "https://storage.example.com/avatar.jpg"},
		},
		{
			name: "Successful update user avatar with content type detection",
			prepare: func(f *fields) {
				f.mediaRepo.EXPECT().UploadAvatar(gomock.Any(), int64(100), gomock.Any()).DoAndReturn(
					func(ctx context.Context, userID int64, input *mediadto.FileInput) (string, error) {
						require.Equal(t, "image/png", input.ContentType)
						return "https://storage.example.com/avatar.png", nil
					})
			},
			args: args{
				ctx: context.Background(),
				req: &mediav1.RequestUpdateUserAvatar{
					UserId: 100,
					Avatar: &mediav1.File{
						Content: []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}, // PNG magic bytes
						Type:    "",
					},
				},
			},
			want: &mediav1.ResponseUpdateUserAvatar{AvatarUrl: "https://storage.example.com/avatar.png"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				mediaRepo: mock.NewMockMediaRepositoryInterface(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &MediaServer{
				MediaRepository: f.mediaRepo,
				logger:          zap.NewNop(),
			}

			resp, err := s.UpdateUserAvatar(tt.args.ctx, tt.args.req)
			require.NoError(t, err)
			require.Equal(t, tt.want.GetAvatarUrl(), resp.GetAvatarUrl())
		})
	}
}

func TestNegativeMediaServer_UpdateUserAvatar(t *testing.T) {
	type fields struct {
		mediaRepo *mock.MockMediaRepositoryInterface
	}

	type args struct {
		ctx context.Context
		req *mediav1.RequestUpdateUserAvatar
	}

	tests := []struct {
		wantCode codes.Code
		prepare  func(*fields)
		name     string
		args     args
	}{
		{
			name: "Nil request",
			args: args{
				ctx: context.Background(),
				req: nil,
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "Invalid user_id (zero)",
			args: args{
				ctx: context.Background(),
				req: &mediav1.RequestUpdateUserAvatar{
					UserId: 0,
					Avatar: &mediav1.File{Content: []byte("data"), Type: "image/jpeg"},
				},
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "Nil avatar",
			args: args{
				ctx: context.Background(),
				req: &mediav1.RequestUpdateUserAvatar{
					UserId: 100,
					Avatar: nil,
				},
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "Empty avatar content",
			args: args{
				ctx: context.Background(),
				req: &mediav1.RequestUpdateUserAvatar{
					UserId: 100,
					Avatar: &mediav1.File{Content: []byte{}, Type: "image/jpeg"},
				},
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "File too large",
			args: args{
				ctx: context.Background(),
				req: &mediav1.RequestUpdateUserAvatar{
					UserId: 100,
					Avatar: &mediav1.File{Content: make([]byte, 6*1024*1024), Type: "image/jpeg"},
				},
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "Repository upload error",
			prepare: func(f *fields) {
				f.mediaRepo.EXPECT().UploadAvatar(gomock.Any(), int64(100), gomock.Any()).Return("", errors.New("storage error"))
			},
			args: args{
				ctx: context.Background(),
				req: &mediav1.RequestUpdateUserAvatar{
					UserId: 100,
					Avatar: &mediav1.File{Content: []byte("data"), Type: "image/jpeg"},
				},
			},
			wantCode: codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				mediaRepo: mock.NewMockMediaRepositoryInterface(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &MediaServer{
				MediaRepository: f.mediaRepo,
				logger:          zap.NewNop(),
			}

			_, err := s.UpdateUserAvatar(tt.args.ctx, tt.args.req)
			require.Error(t, err)
			st, ok := status.FromError(err)
			require.True(t, ok)
			require.Equal(t, tt.wantCode, st.Code())
		})
	}
}

func TestPositiveMediaServer_UploadChatAvatar(t *testing.T) {
	type fields struct {
		mediaRepo *mock.MockMediaRepositoryInterface
	}

	type args struct {
		ctx context.Context
		req *mediav1.RequestUpdateChatAvatar
	}

	tests := []struct {
		prepare func(*fields)
		want    *mediav1.ResponseUpdateChatAvatar
		name    string
		args    args
	}{
		{
			name: "Successful upload chat avatar",
			prepare: func(f *fields) {
				f.mediaRepo.EXPECT().UploadChatAvatar(gomock.Any(), int64(1), gomock.Any()).DoAndReturn(
					func(ctx context.Context, chatID int64, input *mediadto.FileInput) (string, error) {
						require.Equal(t, int64(1), chatID)
						return "https://storage.example.com/chat_avatar.jpg", nil
					})
			},
			args: args{
				ctx: context.Background(),
				req: &mediav1.RequestUpdateChatAvatar{
					ChatId: 1,
					Avatar: &mediav1.File{
						Content: []byte("fake image data"),
						Type:    "image/jpeg",
					},
				},
			},
			want: &mediav1.ResponseUpdateChatAvatar{AvatarUrl: "https://storage.example.com/chat_avatar.jpg"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				mediaRepo: mock.NewMockMediaRepositoryInterface(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &MediaServer{
				MediaRepository: f.mediaRepo,
				logger:          zap.NewNop(),
			}

			resp, err := s.UploadChatAvatar(tt.args.ctx, tt.args.req)
			require.NoError(t, err)
			require.Equal(t, tt.want.GetAvatarUrl(), resp.GetAvatarUrl())
		})
	}
}

func TestNegativeMediaServer_UploadChatAvatar(t *testing.T) {
	type fields struct {
		mediaRepo *mock.MockMediaRepositoryInterface
	}

	type args struct {
		ctx context.Context
		req *mediav1.RequestUpdateChatAvatar
	}

	tests := []struct {
		wantCode codes.Code
		prepare  func(*fields)
		name     string
		args     args
	}{
		{
			name: "Nil request",
			args: args{
				ctx: context.Background(),
				req: nil,
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "Invalid chat_id (zero)",
			args: args{
				ctx: context.Background(),
				req: &mediav1.RequestUpdateChatAvatar{
					ChatId: 0,
					Avatar: &mediav1.File{Content: []byte("data"), Type: "image/jpeg"},
				},
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "Nil avatar",
			args: args{
				ctx: context.Background(),
				req: &mediav1.RequestUpdateChatAvatar{
					ChatId: 1,
					Avatar: nil,
				},
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "Repository upload error",
			prepare: func(f *fields) {
				f.mediaRepo.EXPECT().UploadChatAvatar(gomock.Any(), int64(1), gomock.Any()).Return("", errors.New("storage error"))
			},
			args: args{
				ctx: context.Background(),
				req: &mediav1.RequestUpdateChatAvatar{
					ChatId: 1,
					Avatar: &mediav1.File{Content: []byte("data"), Type: "image/jpeg"},
				},
			},
			wantCode: codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				mediaRepo: mock.NewMockMediaRepositoryInterface(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &MediaServer{
				MediaRepository: f.mediaRepo,
				logger:          zap.NewNop(),
			}

			_, err := s.UploadChatAvatar(tt.args.ctx, tt.args.req)
			require.Error(t, err)
			st, ok := status.FromError(err)
			require.True(t, ok)
			require.Equal(t, tt.wantCode, st.Code())
		})
	}
}

func TestPositiveMediaServer_DeleteUserAvatar(t *testing.T) {
	type fields struct {
		mediaRepo *mock.MockMediaRepositoryInterface
	}

	type args struct {
		ctx context.Context
		req *mediav1.RequestDeleteUserAvatar
	}

	tests := []struct {
		prepare func(*fields)
		want    *mediav1.ResponseDeleteUserAvatar
		name    string
		args    args
	}{
		{
			name: "Successful delete user avatar",
			prepare: func(f *fields) {
				f.mediaRepo.EXPECT().DeleteAvatar(gomock.Any(), int64(100)).Return(nil)
			},
			args: args{
				ctx: context.Background(),
				req: &mediav1.RequestDeleteUserAvatar{UserId: 100},
			},
			want: &mediav1.ResponseDeleteUserAvatar{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				mediaRepo: mock.NewMockMediaRepositoryInterface(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &MediaServer{
				MediaRepository: f.mediaRepo,
				logger:          zap.NewNop(),
			}

			resp, err := s.DeleteUserAvatar(tt.args.ctx, tt.args.req)
			require.NoError(t, err)
			require.NotNil(t, resp)
		})
	}
}

func TestNegativeMediaServer_DeleteUserAvatar(t *testing.T) {
	type fields struct {
		mediaRepo *mock.MockMediaRepositoryInterface
	}

	type args struct {
		ctx context.Context
		req *mediav1.RequestDeleteUserAvatar
	}

	tests := []struct {
		wantCode codes.Code
		prepare  func(*fields)
		name     string
		args     args
	}{
		{
			name: "Nil request",
			args: args{
				ctx: context.Background(),
				req: nil,
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "Invalid user_id (zero)",
			args: args{
				ctx: context.Background(),
				req: &mediav1.RequestDeleteUserAvatar{UserId: 0},
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "Repository delete error",
			prepare: func(f *fields) {
				f.mediaRepo.EXPECT().DeleteAvatar(gomock.Any(), int64(100)).Return(errors.New("delete failed"))
			},
			args: args{
				ctx: context.Background(),
				req: &mediav1.RequestDeleteUserAvatar{UserId: 100},
			},
			wantCode: codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				mediaRepo: mock.NewMockMediaRepositoryInterface(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &MediaServer{
				MediaRepository: f.mediaRepo,
				logger:          zap.NewNop(),
			}

			_, err := s.DeleteUserAvatar(tt.args.ctx, tt.args.req)
			require.Error(t, err)
			st, ok := status.FromError(err)
			require.True(t, ok)
			require.Equal(t, tt.wantCode, st.Code())
		})
	}
}

func TestProtoFileToValidatedInput(t *testing.T) {
	tests := []struct {
		name    string
		file    *mediav1.File
		wantErr error
	}{
		{
			name:    "Nil file",
			file:    nil,
			wantErr: mediadto.ErrEmptyFile,
		},
		{
			name:    "Empty content",
			file:    &mediav1.File{Content: []byte{}, Type: "image/jpeg"},
			wantErr: mediadto.ErrEmptyFile,
		},
		{
			name:    "Valid file with content type",
			file:    &mediav1.File{Content: []byte("data"), Type: "image/jpeg"},
			wantErr: nil,
		},
		{
			name:    "Valid file without content type - detects from content",
			file:    &mediav1.File{Content: []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}, Type: ""},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := protoFileToValidatedInput(tt.file)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestStatusFromAvatarFileError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantCode codes.Code
	}{
		{
			name:     "File too large",
			err:      mediadto.ErrFileTooLarge,
			wantCode: codes.InvalidArgument,
		},
		{
			name:     "Empty file",
			err:      mediadto.ErrEmptyFile,
			wantCode: codes.InvalidArgument,
		},
		{
			name:     "Other error",
			err:      errors.New("some error"),
			wantCode: codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := statusFromAvatarFileError(tt.err)
			require.Error(t, err)
			st, ok := status.FromError(err)
			require.True(t, ok)
			require.Equal(t, tt.wantCode, st.Code())
		})
	}
}

func TestMediaServer_Log(t *testing.T) {
	tests := []struct {
		name   string
		logger *zap.Logger
	}{
		{
			name:   "With logger",
			logger: zap.NewNop(),
		},
		{
			name:   "Nil logger",
			logger: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &MediaServer{
				logger: tt.logger,
			}
			got := s.Log(context.Background())
			require.NotNil(t, got)
		})
	}
}

func TestNewMediaServer(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mock.NewMockMediaRepositoryInterface(ctrl)
	logger := zap.NewNop()

	server := NewMediaServer(mockRepo, nil, logger)
	require.NotNil(t, server)
	require.Equal(t, mockRepo, server.MediaRepository)
	require.Equal(t, logger, server.logger)
}
