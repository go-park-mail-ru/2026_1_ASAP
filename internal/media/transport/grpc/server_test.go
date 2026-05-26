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
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/media/repository"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/media/speechkit"
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
						Content: []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A},
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

func TestProtoFileToMessageInput(t *testing.T) {
	tests := []struct {
		name     string
		file     *mediav1.File
		wantErr  error
		wantType string
		wantSize int64
	}{
		{name: "nil file", wantErr: mediadto.ErrEmptyFile},
		{name: "empty content", file: &mediav1.File{Type: "image/png"}, wantErr: mediadto.ErrEmptyFile},
		{name: "uses type", file: &mediav1.File{Content: []byte("hello"), Type: "text/plain"}, wantType: "text/plain", wantSize: 5},
		{name: "detects type", file: &mediav1.File{Content: []byte{0x89, 0x50, 0x4E, 0x47}, Type: ""}, wantType: "text/plain; charset=utf-8", wantSize: 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := protoFileToMessageInput(tt.file)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				require.Nil(t, got)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.wantType, got.ContentType)
			require.Equal(t, tt.wantSize, got.Size)
		})
	}
}

func TestProtoKindToDTO(t *testing.T) {
	tests := []struct {
		name    string
		kind    mediav1.MessageAttachmentKind
		want    mediadto.MessageAttachmentKind
		wantErr error
	}{
		{name: "photo", kind: mediav1.MessageAttachmentKind_MESSAGE_ATTACHMENT_KIND_PHOTO, want: mediadto.MessageAttachmentKindPhoto},
		{name: "video", kind: mediav1.MessageAttachmentKind_MESSAGE_ATTACHMENT_KIND_VIDEO, want: mediadto.MessageAttachmentKindVideo},
		{name: "file", kind: mediav1.MessageAttachmentKind_MESSAGE_ATTACHMENT_KIND_FILE, want: mediadto.MessageAttachmentKindFile},
		{name: "voice", kind: mediav1.MessageAttachmentKind_MESSAGE_ATTACHMENT_KIND_VOICE, want: mediadto.MessageAttachmentKindVoice},
		{name: "invalid", kind: mediav1.MessageAttachmentKind_MESSAGE_ATTACHMENT_KIND_UNSPECIFIED, wantErr: mediadto.ErrInvalidFileType},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := protoKindToDTO(tt.kind)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestStatusFromFileError_AppCodes(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		wantCode    codes.Code
		wantAppCode int32
	}{
		{name: "too large", err: mediadto.ErrFileTooLarge, wantCode: codes.InvalidArgument, wantAppCode: int32(mediav1.MediaErrorCode_MEDIA_ERROR_FILE_TOO_LARGE)},
		{name: "invalid type", err: mediadto.ErrInvalidFileType, wantCode: codes.InvalidArgument, wantAppCode: int32(mediav1.MediaErrorCode_MEDIA_ERROR_FILE_INVALID_TYPE)},
		{name: "empty", err: mediadto.ErrEmptyFile, wantCode: codes.InvalidArgument, wantAppCode: int32(mediav1.MediaErrorCode_MEDIA_ERROR_FILE_EMPTY)},
		{name: "voice too long", err: mediadto.ErrVoiceTooLong, wantCode: codes.InvalidArgument, wantAppCode: int32(mediav1.MediaErrorCode_MEDIA_ERROR_VOICE_TOO_LONG)},
		{name: "speechkit failed", err: speechkit.ErrTranscriptionFailed, wantCode: codes.Internal, wantAppCode: int32(mediav1.MediaErrorCode_MEDIA_ERROR_TRANSCRIPTION_FAILED)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := statusFromFileError(tt.err)
			require.Error(t, err)
			st, ok := status.FromError(err)
			require.True(t, ok)
			require.Equal(t, tt.wantCode, st.Code())
		})
	}
}

func TestMediaServer_UploadMessageAttachment(t *testing.T) {
	fileName := "photo.png"

	tests := []struct {
		name       string
		req        *mediav1.RequestUploadMessageAttachment
		prepare    func(*mock.MockMediaRepositoryInterface)
		wantCode   codes.Code
		wantObject string
	}{
		{
			name:     "nil request",
			wantCode: codes.InvalidArgument,
		},
		{
			name: "invalid kind",
			req: &mediav1.RequestUploadMessageAttachment{
				UserId: 1,
				Kind:   mediav1.MessageAttachmentKind_MESSAGE_ATTACHMENT_KIND_UNSPECIFIED,
				File:   &mediav1.File{Content: []byte("x"), Type: "image/png"},
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "empty file",
			req: &mediav1.RequestUploadMessageAttachment{
				UserId: 1,
				Kind:   mediav1.MessageAttachmentKind_MESSAGE_ATTACHMENT_KIND_PHOTO,
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "repo error",
			req: &mediav1.RequestUploadMessageAttachment{
				UserId: 1,
				Kind:   mediav1.MessageAttachmentKind_MESSAGE_ATTACHMENT_KIND_PHOTO,
				File:   &mediav1.File{Content: []byte("x"), Type: "image/png"},
			},
			prepare: func(repo *mock.MockMediaRepositoryInterface) {
				repo.EXPECT().
					UploadMessageAttachment(gomock.Any(), int64(1), mediadto.MessageAttachmentKindPhoto, gomock.Any()).
					Return(nil, mediadto.ErrFileTooLarge)
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "success",
			req: &mediav1.RequestUploadMessageAttachment{
				UserId:   1,
				Kind:     mediav1.MessageAttachmentKind_MESSAGE_ATTACHMENT_KIND_PHOTO,
				File:     &mediav1.File{Content: []byte("x"), Type: "image/png"},
				FileName: &fileName,
			},
			prepare: func(repo *mock.MockMediaRepositoryInterface) {
				repo.EXPECT().
					UploadMessageAttachment(gomock.Any(), int64(1), mediadto.MessageAttachmentKindPhoto, gomock.Any()).
					Return(&repository.MessageAttachmentObject{
						ObjectKey:   "message/1/file.png",
						ContentType: "image/png",
						Size:        1,
						DurationMs:  12,
						Waveform:    []uint8{1, 2},
						IsCapybara:  true,
					}, nil)
			},
			wantObject: "message/1/file.png",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			repo := mock.NewMockMediaRepositoryInterface(ctrl)
			if tt.prepare != nil {
				tt.prepare(repo)
			}
			s := &MediaServer{MediaRepository: repo, logger: zap.NewNop()}

			got, err := s.UploadMessageAttachment(context.Background(), tt.req)
			if tt.wantCode != 0 {
				require.Error(t, err)
				st, ok := status.FromError(err)
				require.True(t, ok)
				require.Equal(t, tt.wantCode, st.Code())
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.wantObject, got.GetObjectKey())
			require.Equal(t, "photo.png", got.GetFileName())
			require.Equal(t, []uint32{1, 2}, got.GetWaveform())
			require.True(t, got.GetIsCapybara())
		})
	}
}

func TestMediaServer_MessageAttachmentReadAndMetadata(t *testing.T) {
	tests := []struct {
		name     string
		call     func(MediaServer) (any, error)
		prepare  func(*mock.MockMediaRepositoryInterface)
		wantCode codes.Code
		assert   func(t *testing.T, got any)
	}{
		{
			name: "classify success",
			call: func(s MediaServer) (any, error) {
				return s.ClassifyMessagePhoto(context.Background(), &mediav1.RequestClassifyMessagePhoto{ObjectKey: "message/1/a.png"})
			},
			prepare: func(repo *mock.MockMediaRepositoryInterface) {
				repo.EXPECT().ClassifyMessagePhoto(gomock.Any(), "message/1/a.png").Return(true, nil)
			},
			assert: func(t *testing.T, got any) {
				require.True(t, got.(*mediav1.ResponseClassifyMessagePhoto).GetIsCapybara())
			},
		},
		{
			name: "metadata success",
			call: func(s MediaServer) (any, error) {
				return s.GetMessageVoiceMetadata(context.Background(), &mediav1.RequestGetMessageVoiceMetadata{ObjectKey: "message/1/a.webm"})
			},
			prepare: func(repo *mock.MockMediaRepositoryInterface) {
				repo.EXPECT().GetMessageVoiceMetadata(gomock.Any(), "message/1/a.webm").Return(&repository.VoiceMetadata{
					DurationMs: 100,
					Waveform:   []uint8{1, 3},
					MimeType:   "audio/webm",
					FileSize:   7,
				}, nil)
			},
			assert: func(t *testing.T, got any) {
				resp := got.(*mediav1.ResponseGetMessageVoiceMetadata)
				require.Equal(t, int32(100), resp.GetDurationMs())
				require.Equal(t, []uint32{1, 3}, resp.GetWaveform())
				require.Equal(t, int64(7), resp.GetFileSize())
			},
		},
		{
			name: "get attachment success",
			call: func(s MediaServer) (any, error) {
				return s.GetMessageAttachment(context.Background(), &mediav1.RequestGetMessageAttachment{ObjectKey: "message/1/a.png"})
			},
			prepare: func(repo *mock.MockMediaRepositoryInterface) {
				repo.EXPECT().GetMessageAttachment(gomock.Any(), "message/1/a.png").Return([]byte("data"), "image/png", nil)
			},
			assert: func(t *testing.T, got any) {
				resp := got.(*mediav1.ResponseGetMessageAttachment)
				require.Equal(t, []byte("data"), resp.GetContent())
				require.Equal(t, int64(4), resp.GetSize())
			},
		},
		{
			name: "empty classify key",
			call: func(s MediaServer) (any, error) {
				return s.ClassifyMessagePhoto(context.Background(), &mediav1.RequestClassifyMessagePhoto{})
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "metadata repo error",
			call: func(s MediaServer) (any, error) {
				return s.GetMessageVoiceMetadata(context.Background(), &mediav1.RequestGetMessageVoiceMetadata{ObjectKey: "message/1/a.webm"})
			},
			prepare: func(repo *mock.MockMediaRepositoryInterface) {
				repo.EXPECT().GetMessageVoiceMetadata(gomock.Any(), "message/1/a.webm").Return(nil, mediadto.ErrInvalidFileType)
			},
			wantCode: codes.InvalidArgument,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			repo := mock.NewMockMediaRepositoryInterface(ctrl)
			if tt.prepare != nil {
				tt.prepare(repo)
			}
			got, err := tt.call(MediaServer{MediaRepository: repo, logger: zap.NewNop()})
			if tt.wantCode != 0 {
				require.Error(t, err)
				st, ok := status.FromError(err)
				require.True(t, ok)
				require.Equal(t, tt.wantCode, st.Code())
				return
			}
			require.NoError(t, err)
			tt.assert(t, got)
		})
	}
}

type fakeTranscriber struct {
	text string
	err  error
}

func (t fakeTranscriber) Transcribe(ctx context.Context, data []byte, contentType string) (string, error) {
	return t.text, t.err
}

func TestMediaServer_TranscribeVoice(t *testing.T) {
	tests := []struct {
		name        string
		req         *mediav1.RequestTranscribeVoice
		transcriber VoiceTranscriber
		prepare     func(*mock.MockMediaRepositoryInterface)
		wantCode    codes.Code
		wantText    string
	}{
		{name: "nil request", wantCode: codes.InvalidArgument},
		{name: "no transcriber", req: &mediav1.RequestTranscribeVoice{ObjectKey: "message/1/a.webm"}, wantCode: codes.Internal},
		{
			name:        "load error",
			req:         &mediav1.RequestTranscribeVoice{ObjectKey: "message/1/a.webm"},
			transcriber: fakeTranscriber{text: "ignored"},
			prepare: func(repo *mock.MockMediaRepositoryInterface) {
				repo.EXPECT().GetMessageAttachment(gomock.Any(), "message/1/a.webm").Return(nil, "", mediadto.ErrFileTooLarge)
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name:        "transcribe error",
			req:         &mediav1.RequestTranscribeVoice{ObjectKey: "message/1/a.webm"},
			transcriber: fakeTranscriber{err: speechkit.ErrTranscriptionFailed},
			prepare: func(repo *mock.MockMediaRepositoryInterface) {
				repo.EXPECT().GetMessageAttachment(gomock.Any(), "message/1/a.webm").Return([]byte("voice"), "audio/webm", nil)
			},
			wantCode: codes.Internal,
		},
		{
			name:        "success",
			req:         &mediav1.RequestTranscribeVoice{ObjectKey: "message/1/a.webm"},
			transcriber: fakeTranscriber{text: "hello"},
			prepare: func(repo *mock.MockMediaRepositoryInterface) {
				repo.EXPECT().GetMessageAttachment(gomock.Any(), "message/1/a.webm").Return([]byte("voice"), "audio/webm", nil)
			},
			wantText: "hello",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			repo := mock.NewMockMediaRepositoryInterface(ctrl)
			if tt.prepare != nil {
				tt.prepare(repo)
			}
			got, err := (MediaServer{MediaRepository: repo, transcriber: tt.transcriber, logger: zap.NewNop()}).
				TranscribeVoice(context.Background(), tt.req)
			if tt.wantCode != 0 {
				require.Error(t, err)
				st, ok := status.FromError(err)
				require.True(t, ok)
				require.Equal(t, tt.wantCode, st.Code())
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.wantText, got.GetTranscript())
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
