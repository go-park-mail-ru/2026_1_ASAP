package grpc

import (
	"bytes"
	"context"
	"errors"
	"net/http"

	mediav1 "github.com/go-park-mail-ru/2026_1_ASAP/gen/go/media/v1"
	mediadto "github.com/go-park-mail-ru/2026_1_ASAP/internal/media/dto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type MediaRepositoryInterface interface {
	UploadAvatar(ctx context.Context, userId int64, input *mediadto.FileInput) (string, error)
	UploadChatAvatar(ctx context.Context, chatID int64, input *mediadto.FileInput) (string, error)
	DeleteAvatar(ctx context.Context, userID int64) error
}

type MediaServer struct {
	mediav1.UnimplementedMediaServer
	MediaRepository MediaRepositoryInterface
}

func protoFileToValidatedInput(f *mediav1.File) (*mediadto.FileInput, error) {
	if f == nil || len(f.GetContent()) == 0 {
		return nil, mediadto.ErrEmptyFile
	}
	content := f.GetContent()
	ct := f.GetType()
	if ct == "" || ct == "application/octet-stream" {
		n := 512
		if len(content) < n {
			n = len(content)
		}
		if n > 0 {
			ct = http.DetectContentType(content[:n])
		}
	}
	in := &mediadto.FileInput{
		Body:        bytes.NewReader(content),
		ContentType: ct,
		Size:        int64(len(content)),
	}
	if err := in.Validate(); err != nil {
		return nil, err
	}
	return in, nil
}

func statusFromAvatarFileError(err error) error {
	switch {
	case errors.Is(err, mediadto.ErrFileTooLarge):
		return status.Error(codes.InvalidArgument, mediadto.ErrFileTooLarge.Error())
	case errors.Is(err, mediadto.ErrInvalidFileType):
		return status.Error(codes.InvalidArgument, mediadto.ErrInvalidFileType.Error())
	case errors.Is(err, mediadto.ErrEmptyFile):
		return status.Error(codes.InvalidArgument, mediadto.ErrEmptyFile.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}

func (m MediaServer) UpdateUserAvatar(ctx context.Context, req *mediav1.RequestUpdateUserAvatar) (*mediav1.ResponseUpdateUserAvatar, error) {
	if req == nil || req.GetUserId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	in, err := protoFileToValidatedInput(req.GetAvatar())
	if err != nil {
		return nil, statusFromAvatarFileError(err)
	}
	url, err := m.MediaRepository.UploadAvatar(ctx, req.GetUserId(), in)
	if err != nil {
		if errors.Is(err, mediadto.ErrEmptyFile) {
			return nil, status.Error(codes.InvalidArgument, mediadto.ErrEmptyFile.Error())
		}
		return nil, status.Error(codes.Internal, "upload failed")
	}
	return &mediav1.ResponseUpdateUserAvatar{AvatarUrl: url}, nil
}

func (m MediaServer) UploadChatAvatar(ctx context.Context, req *mediav1.RequestUpdateChatAvatar) (*mediav1.ResponseUpdateChatAvatar, error) {
	if req == nil || req.GetChatId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "chat_id is required")
	}
	in, err := protoFileToValidatedInput(req.GetAvatar())
	if err != nil {
		return nil, statusFromAvatarFileError(err)
	}
	url, err := m.MediaRepository.UploadChatAvatar(ctx, req.GetChatId(), in)
	if err != nil {
		if errors.Is(err, mediadto.ErrEmptyFile) {
			return nil, status.Error(codes.InvalidArgument, mediadto.ErrEmptyFile.Error())
		}
		return nil, status.Error(codes.Internal, "upload failed")
	}
	return &mediav1.ResponseUpdateChatAvatar{AvatarUrl: url}, nil
}

func (m MediaServer) DeleteUserAvatar(ctx context.Context, req *mediav1.RequestDeleteUserAvatar) (*mediav1.ResponseDeleteUserAvatar, error) {
	if req == nil || req.GetUserId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	if err := m.MediaRepository.DeleteAvatar(ctx, req.GetUserId()); err != nil {
		return nil, status.Error(codes.Internal, "delete failed")
	}
	return &mediav1.ResponseDeleteUserAvatar{}, nil
}

func NewMediaServer(mediaRepository MediaRepositoryInterface) *MediaServer {
	return &MediaServer{
		MediaRepository: mediaRepository,
	}
}
