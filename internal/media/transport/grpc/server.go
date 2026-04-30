package grpc

import (
	"bytes"
	"context"
	"errors"
	"net/http"

	mediav1 "github.com/go-park-mail-ru/2026_1_ASAP/gen/go/media/v1"
	mediadto "github.com/go-park-mail-ru/2026_1_ASAP/internal/media/dto"
	"github.com/go-park-mail-ru/2026_1_ASAP/pkg/grpcerr"
	"github.com/go-park-mail-ru/2026_1_ASAP/pkg/loggerctx"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
)

type MediaRepositoryInterface interface {
	UploadAvatar(ctx context.Context, userId int64, input *mediadto.FileInput) (string, error)
	UploadChatAvatar(ctx context.Context, chatID int64, input *mediadto.FileInput) (string, error)
	UploadComplaint(ctx context.Context, complaintID int64, input *mediadto.FileInput) (string, error)
	DeleteAvatar(ctx context.Context, userID int64) error
}

type MediaServer struct {
	mediav1.UnimplementedMediaServer
	MediaRepository MediaRepositoryInterface
	logger          *zap.Logger
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
		return grpcerr.New(codes.InvalidArgument, int32(mediav1.MediaErrorCode_MEDIA_ERROR_FILE_TOO_LARGE), "file too large")
	case errors.Is(err, mediadto.ErrEmptyFile):
		return grpcerr.New(codes.InvalidArgument, int32(mediav1.MediaErrorCode_MEDIA_ERROR_FILE_EMPTY), "file is empty")
	default:
		return grpcerr.New(codes.Internal, int32(mediav1.MediaErrorCode_MEDIA_ERROR_INTERNAL), "internal server error")
	}
}

func (m MediaServer) UpdateUserAvatar(ctx context.Context, req *mediav1.RequestUpdateUserAvatar) (*mediav1.ResponseUpdateUserAvatar, error) {
	if req == nil || req.GetUserId() <= 0 {
		return nil, grpcerr.New(codes.InvalidArgument, int32(mediav1.MediaErrorCode_MEDIA_ERROR_INVALID_INPUT), "user_id is required")
	}
	in, err := protoFileToValidatedInput(req.GetAvatar())
	if err != nil {
		m.Log(ctx).Info("invalid avatar file", zap.Int64("user_id", req.GetUserId()), zap.Error(err))
		return nil, statusFromAvatarFileError(err)
	}
	url, err := m.MediaRepository.UploadAvatar(ctx, req.GetUserId(), in)
	if err != nil {
		m.Log(ctx).Error("failed to upload user avatar", zap.Int64("user_id", req.GetUserId()), zap.Error(err))
		return nil, statusFromAvatarFileError(err)
	}
	return &mediav1.ResponseUpdateUserAvatar{AvatarUrl: url}, nil
}

func (m MediaServer) UploadChatAvatar(ctx context.Context, req *mediav1.RequestUpdateChatAvatar) (*mediav1.ResponseUpdateChatAvatar, error) {
	if req == nil || req.GetChatId() <= 0 {
		return nil, grpcerr.New(codes.InvalidArgument, int32(mediav1.MediaErrorCode_MEDIA_ERROR_INVALID_INPUT), "chat_id is required")
	}
	in, err := protoFileToValidatedInput(req.GetAvatar())
	if err != nil {
		m.Log(ctx).Info("invalid chat avatar file", zap.Int64("chat_id", req.GetChatId()), zap.Error(err))
		return nil, statusFromAvatarFileError(err)
	}
	url, err := m.MediaRepository.UploadChatAvatar(ctx, req.GetChatId(), in)
	if err != nil {
		m.Log(ctx).Error("failed to upload chat avatar", zap.Int64("chat_id", req.GetChatId()), zap.Error(err))
		return nil, statusFromAvatarFileError(err)
	}
	return &mediav1.ResponseUpdateChatAvatar{AvatarUrl: url}, nil
}

func (m MediaServer) UploadComplaintAttachment(ctx context.Context, req *mediav1.RequestUpdateComplaintAttachment) (*mediav1.ResponseUpdateComplaintAttachment, error) {
	if req == nil || req.GetComplaintId() <= 0 {
		return nil, grpcerr.New(codes.InvalidArgument, int32(mediav1.MediaErrorCode_MEDIA_ERROR_INVALID_INPUT), "complaint_id is required")
	}
	in, err := protoFileToValidatedInput(req.GetAttachment())
	if err != nil {
		m.Log(ctx).Info("invalid complaint attachment file", zap.Int64("complaint_id", req.GetComplaintId()), zap.Error(err))
		return nil, statusFromAvatarFileError(err)
	}
	url, err := m.MediaRepository.UploadComplaint(ctx, req.GetComplaintId(), in)
	if err != nil {
		m.Log(ctx).Error("failed to upload complaint attachment", zap.Int64("complaint_id", req.GetComplaintId()), zap.Error(err))
		return nil, statusFromAvatarFileError(err)
	}
	return &mediav1.ResponseUpdateComplaintAttachment{AttachmentUrl: url}, nil
}

func (m MediaServer) DeleteUserAvatar(ctx context.Context, req *mediav1.RequestDeleteUserAvatar) (*mediav1.ResponseDeleteUserAvatar, error) {
	if req == nil || req.GetUserId() <= 0 {
		return nil, grpcerr.New(codes.InvalidArgument, int32(mediav1.MediaErrorCode_MEDIA_ERROR_INVALID_INPUT), "user_id is required")
	}
	if err := m.MediaRepository.DeleteAvatar(ctx, req.GetUserId()); err != nil {
		m.Log(ctx).Error("failed to delete user avatar", zap.Int64("user_id", req.GetUserId()), zap.Error(err))
		return nil, grpcerr.New(codes.Internal, int32(mediav1.MediaErrorCode_MEDIA_ERROR_INTERNAL), "delete failed")
	}
	return &mediav1.ResponseDeleteUserAvatar{}, nil
}

func NewMediaServer(mediaRepository MediaRepositoryInterface, logger *zap.Logger) *MediaServer {
	return &MediaServer{
		MediaRepository: mediaRepository,
		logger:          logger,
	}
}

func (m MediaServer) Log(ctx context.Context) *zap.Logger {
	return loggerctx.EnrichLoggerFromContext(ctx, m.logger)
}
