package grpc

import (
	"bytes"
	"context"
	"errors"
	"net/http"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"

	mediav1 "github.com/go-park-mail-ru/2026_1_ASAP/gen/go/media/v1"
	mediadto "github.com/go-park-mail-ru/2026_1_ASAP/internal/media/dto"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/media/repository"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/media/speechkit"
	"github.com/go-park-mail-ru/2026_1_ASAP/pkg/grpcerr"
	"github.com/go-park-mail-ru/2026_1_ASAP/pkg/loggerctx"
)

type MediaRepositoryInterface interface {
	UploadAvatar(ctx context.Context, userId int64, input *mediadto.FileInput) (string, error)
	UploadChatAvatar(ctx context.Context, chatID int64, input *mediadto.FileInput) (string, error)
	UploadComplaint(ctx context.Context, complaintID int64, input *mediadto.FileInput) (string, error)
	UploadMessageAttachment(ctx context.Context, userID int64, kind mediadto.MessageAttachmentKind, input *mediadto.FileInput) (*repository.MessageAttachmentObject, error)
	GetMessageAttachment(ctx context.Context, objectKey string) ([]byte, string, error)
	GetMessageVoiceMetadata(ctx context.Context, objectKey string) (*repository.VoiceMetadata, error)
	ClassifyMessagePhoto(ctx context.Context, objectKey string) (bool, error)
	DeleteAvatar(ctx context.Context, userID int64) error
}

type VoiceTranscriber interface {
	Transcribe(ctx context.Context, data []byte, contentType string) (string, error)
}

type MediaServer struct {
	mediav1.UnimplementedMediaServer
	MediaRepository MediaRepositoryInterface
	transcriber     VoiceTranscriber
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

func statusFromFileError(err error) error {
	switch {
	case errors.Is(err, mediadto.ErrFileTooLarge):
		return grpcerr.New(codes.InvalidArgument, int32(mediav1.MediaErrorCode_MEDIA_ERROR_FILE_TOO_LARGE), "file too large")
	case errors.Is(err, mediadto.ErrInvalidFileType):
		return grpcerr.New(codes.InvalidArgument, int32(mediav1.MediaErrorCode_MEDIA_ERROR_FILE_INVALID_TYPE), "invalid file type")
	case errors.Is(err, mediadto.ErrEmptyFile):
		return grpcerr.New(codes.InvalidArgument, int32(mediav1.MediaErrorCode_MEDIA_ERROR_FILE_EMPTY), "file is empty")
	case errors.Is(err, mediadto.ErrVoiceTooLong):
		return grpcerr.New(codes.InvalidArgument, int32(mediav1.MediaErrorCode_MEDIA_ERROR_VOICE_TOO_LONG), "voice message too long (max 30 seconds)")
	case errors.Is(err, speechkit.ErrTranscriptionFailed), errors.Is(err, speechkit.ErrNotConfigured):
		return grpcerr.New(codes.Internal, int32(mediav1.MediaErrorCode_MEDIA_ERROR_TRANSCRIPTION_FAILED), "transcription failed")
	default:
		return grpcerr.New(codes.Internal, int32(mediav1.MediaErrorCode_MEDIA_ERROR_INTERNAL), "internal server error")
	}
}

func statusFromAvatarFileError(err error) error {
	return statusFromFileError(err)
}

func protoFileToMessageInput(f *mediav1.File) (*mediadto.FileInput, error) {
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
	return &mediadto.FileInput{
		Body:        bytes.NewReader(content),
		ContentType: ct,
		Size:        int64(len(content)),
	}, nil
}

func protoKindToDTO(kind mediav1.MessageAttachmentKind) (mediadto.MessageAttachmentKind, error) {
	switch kind {
	case mediav1.MessageAttachmentKind_MESSAGE_ATTACHMENT_KIND_PHOTO:
		return mediadto.MessageAttachmentKindPhoto, nil
	case mediav1.MessageAttachmentKind_MESSAGE_ATTACHMENT_KIND_VIDEO:
		return mediadto.MessageAttachmentKindVideo, nil
	case mediav1.MessageAttachmentKind_MESSAGE_ATTACHMENT_KIND_FILE:
		return mediadto.MessageAttachmentKindFile, nil
	case mediav1.MessageAttachmentKind_MESSAGE_ATTACHMENT_KIND_VOICE:
		return mediadto.MessageAttachmentKindVoice, nil
	default:
		return 0, mediadto.ErrInvalidFileType
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

func (m MediaServer) UploadMessageAttachment(ctx context.Context, req *mediav1.RequestUploadMessageAttachment) (*mediav1.ResponseUploadMessageAttachment, error) {
	if req == nil || req.GetUserId() <= 0 {
		return nil, grpcerr.New(codes.InvalidArgument, int32(mediav1.MediaErrorCode_MEDIA_ERROR_INVALID_INPUT), "user_id is required")
	}
	kind, err := protoKindToDTO(req.GetKind())
	if err != nil {
		return nil, statusFromFileError(err)
	}
	in, err := protoFileToMessageInput(req.GetFile())
	if err != nil {
		m.Log(ctx).Info("invalid message attachment file", zap.Int64("user_id", req.GetUserId()), zap.Error(err))
		return nil, statusFromFileError(err)
	}
	if err = in.ValidateMessageAttachment(kind); err != nil {
		m.Log(ctx).Info("message attachment validation failed", zap.Int64("user_id", req.GetUserId()), zap.Error(err))
		return nil, statusFromFileError(err)
	}
	obj, err := m.MediaRepository.UploadMessageAttachment(ctx, req.GetUserId(), kind, in)
	if err != nil {
		m.Log(ctx).Error("failed to upload message attachment", zap.Int64("user_id", req.GetUserId()), zap.Error(err))
		return nil, statusFromFileError(err)
	}
	resp := &mediav1.ResponseUploadMessageAttachment{
		MimeType:  obj.ContentType,
		FileSize:  obj.Size,
		ObjectKey: obj.ObjectKey,
	}
	if obj.DurationMs > 0 {
		resp.DurationMs = int32(obj.DurationMs)
	}
	if len(obj.Waveform) > 0 {
		wf := make([]uint32, len(obj.Waveform))
		for i, v := range obj.Waveform {
			wf[i] = uint32(v)
		}
		resp.Waveform = wf
	}
	if name := req.GetFileName(); name != "" {
		resp.FileName = &name
	}
	resp.IsCapybara = obj.IsCapybara
	return resp, nil
}

func (m MediaServer) ClassifyMessagePhoto(ctx context.Context, req *mediav1.RequestClassifyMessagePhoto) (*mediav1.ResponseClassifyMessagePhoto, error) {
	if req == nil || req.GetObjectKey() == "" {
		return nil, grpcerr.New(codes.InvalidArgument, int32(mediav1.MediaErrorCode_MEDIA_ERROR_INVALID_INPUT), "object_key is required")
	}
	isCapybara, err := m.MediaRepository.ClassifyMessagePhoto(ctx, req.GetObjectKey())
	if err != nil {
		m.Log(ctx).Error("failed to classify message photo", zap.String("object_key", req.GetObjectKey()), zap.Error(err))
		return nil, statusFromFileError(err)
	}
	return &mediav1.ResponseClassifyMessagePhoto{IsCapybara: isCapybara}, nil
}

func (m MediaServer) GetMessageVoiceMetadata(ctx context.Context, req *mediav1.RequestGetMessageVoiceMetadata) (*mediav1.ResponseGetMessageVoiceMetadata, error) {
	if req == nil || req.GetObjectKey() == "" {
		return nil, grpcerr.New(codes.InvalidArgument, int32(mediav1.MediaErrorCode_MEDIA_ERROR_INVALID_INPUT), "object_key is required")
	}
	meta, err := m.MediaRepository.GetMessageVoiceMetadata(ctx, req.GetObjectKey())
	if err != nil {
		m.Log(ctx).Error("failed to get voice metadata", zap.String("object_key", req.GetObjectKey()), zap.Error(err))
		return nil, statusFromFileError(err)
	}
	wf := make([]uint32, len(meta.Waveform))
	for i, v := range meta.Waveform {
		wf[i] = uint32(v)
	}
	return &mediav1.ResponseGetMessageVoiceMetadata{
		DurationMs: int32(meta.DurationMs),
		Waveform:   wf,
		MimeType:   meta.MimeType,
		FileSize:   meta.FileSize,
	}, nil
}

func (m MediaServer) TranscribeVoice(ctx context.Context, req *mediav1.RequestTranscribeVoice) (*mediav1.ResponseTranscribeVoice, error) {
	if req == nil || req.GetObjectKey() == "" {
		return nil, grpcerr.New(codes.InvalidArgument, int32(mediav1.MediaErrorCode_MEDIA_ERROR_INVALID_INPUT), "object_key is required")
	}
	if m.transcriber == nil {
		return nil, grpcerr.New(codes.Internal, int32(mediav1.MediaErrorCode_MEDIA_ERROR_TRANSCRIPTION_FAILED), "transcription not configured")
	}
	data, ct, err := m.MediaRepository.GetMessageAttachment(ctx, req.GetObjectKey())
	if err != nil {
		m.Log(ctx).Error("failed to load voice for transcription", zap.String("object_key", req.GetObjectKey()), zap.Error(err))
		return nil, statusFromFileError(err)
	}
	text, err := m.transcriber.Transcribe(ctx, data, ct)
	if err != nil {
		m.Log(ctx).Error("voice transcription failed", zap.String("object_key", req.GetObjectKey()), zap.Error(err))
		return nil, statusFromFileError(err)
	}
	return &mediav1.ResponseTranscribeVoice{Transcript: text}, nil
}

func (m MediaServer) GetMessageAttachment(ctx context.Context, req *mediav1.RequestGetMessageAttachment) (*mediav1.ResponseGetMessageAttachment, error) {
	if req == nil || req.GetObjectKey() == "" {
		return nil, grpcerr.New(codes.InvalidArgument, int32(mediav1.MediaErrorCode_MEDIA_ERROR_INVALID_INPUT), "object_key is required")
	}
	data, ct, err := m.MediaRepository.GetMessageAttachment(ctx, req.GetObjectKey())
	if err != nil {
		m.Log(ctx).Error("failed to get message attachment", zap.String("object_key", req.GetObjectKey()), zap.Error(err))
		return nil, statusFromFileError(err)
	}
	return &mediav1.ResponseGetMessageAttachment{
		Content:     data,
		ContentType: ct,
		Size:        int64(len(data)),
	}, nil
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

func NewMediaServer(mediaRepository MediaRepositoryInterface, transcriber VoiceTranscriber, logger *zap.Logger) *MediaServer {
	return &MediaServer{
		MediaRepository: mediaRepository,
		transcriber:     transcriber,
		logger:          logger,
	}
}

func (m MediaServer) Log(ctx context.Context) *zap.Logger {
	return loggerctx.EnrichLoggerFromContext(ctx, m.logger)
}
