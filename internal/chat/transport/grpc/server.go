package grpc

import (
	"bytes"
	"context"
	"errors"

	chatv1 "github.com/go-park-mail-ru/2026_1_ASAP/gen/go/chat/v1"
	domain "github.com/go-park-mail-ru/2026_1_ASAP/internal/chat/domain/chat"
	pdomain "github.com/go-park-mail-ru/2026_1_ASAP/internal/chat/domain/profile"
	dto "github.com/go-park-mail-ru/2026_1_ASAP/internal/chat/dto/chat"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/chat/dto/media"
	msgdto "github.com/go-park-mail-ru/2026_1_ASAP/internal/chat/dto/message"
	"github.com/go-park-mail-ru/2026_1_ASAP/pkg/grpcerr"
	"github.com/go-park-mail-ru/2026_1_ASAP/pkg/loggerctx"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type ChatUsecaseInterface interface {
	GetAllChats(ctx context.Context, userID int64) ([]dto.ChatInformationDTO, error)
	CreateChat(ctx context.Context, chatDTO dto.ChatCreate, ownerID int64) (*dto.ChatInformationDTO, error)
	GetChatByID(ctx context.Context, chatID, userID int64) (*dto.ChatInformationDTO, error)
	DeleteChat(ctx context.Context, userID, chatID int64) error
	UpdateChatAvatar(ctx context.Context, userID, chatID int64, req *dto.RequestUpdateAvatar) (*dto.ChatInformationDTO, error)
	UpdateChatTitle(ctx context.Context, userID, chatID int64, req *dto.RequestUpdateTitle) (*dto.ChatInformationDTO, error)
	AddMembersToChat(ctx context.Context, userID, chatID int64, req *dto.RequestAddMember) error
	DeleteMemberFromChat(ctx context.Context, userID, chatID int64, req *dto.RequestDeleteMember) error
	GetAllChatMembers(ctx context.Context, userID, chatID int64) (*dto.ResponseGetChatMembers, error)
	QuitChat(ctx context.Context, userID, chatID int64) error
	JoinChannel(ctx context.Context, userID, chatID int64) error
	UpdateChatDescription(ctx context.Context, userID, chatID int64, request *dto.RequestUpdateDescription) (*dto.ChatInformationDTO, error)
}

type MessageUsecaseInterface interface {
	GetMessagesByChatId(ctx context.Context, userID, chatID int64, req *msgdto.RequestGetMessages) (*msgdto.ResponseGetMessages, error)
	SendMessage(ctx context.Context, userID, chatID int64, req *msgdto.RequestSendMessage) (*msgdto.ResponseSendMessage, error)
	SendMessageWithAttachments(ctx context.Context, userID, chatID int64, req *msgdto.RequestSendMessageAttachments) (*msgdto.ResponseSendMessage, error)
	UploadMessageAttachment(ctx context.Context, userID int64, kind chatv1.MessageAttachmentKind, input *media.FileInput, fileName string) (*msgdto.UploadAttachmentResponse, error)
	AuthorizeMessageAttachment(ctx context.Context, userID int64, objectKey string) error
	EditMessage(ctx context.Context, userID, chatID int64, req *msgdto.RequestEditMessage) (*msgdto.ResponseEditMessage, error)
}

func mapDomainErr(err error) error {
	switch {
	case errors.Is(err, domain.ErrChatNotFound),
		errors.Is(err, pdomain.ErrNotFound),
		errors.Is(err, domain.ErrMemberNotFound):
		return grpcerr.New(
			codes.NotFound,
			int32(chatv1.ChatErrorCode_CHAT_ERROR_NOT_FOUND),
			err.Error(),
		)

	case errors.Is(err, domain.ErrDialogAlreadyExists),
		errors.Is(err, domain.ErrChatAlreadyExists):
		return grpcerr.New(
			codes.FailedPrecondition,
			int32(chatv1.ChatErrorCode_CHAT_ERROR_DIALOG_EXISTS),
			err.Error(),
		)

	case errors.Is(err, domain.ErrUserNotFound):
		return grpcerr.New(
			codes.NotFound,
			int32(chatv1.ChatErrorCode_CHAT_ERROR_USER_NOT_FOUND),
			err.Error(),
		)

	case errors.Is(err, domain.ErrOnlyOwnerCanDeleteChat),
		errors.Is(err, domain.ErrOnlyOwnerCanAddPeople),
		errors.Is(err, domain.ErrOnlyOwnerCanChangeAvatar),
		errors.Is(err, domain.ErrOnlyOwnerCanChangeTitle),
		errors.Is(err, domain.ErrOnlyOwnerCanDeletePeople):
		return grpcerr.New(
			codes.PermissionDenied,
			int32(chatv1.ChatErrorCode_CHAT_ERROR_ONLY_OWNER_CAN_DELETE_CHAT),
			err.Error(),
		)

	case errors.Is(err, domain.ErrNotMember),
		errors.Is(err, domain.ErrUserNotMember),
		errors.Is(err, domain.ErrMessageNotMember):
		return grpcerr.New(
			codes.PermissionDenied,
			int32(chatv1.ChatErrorCode_CHAT_ERROR_USER_NOT_MEMBER),
			err.Error(),
		)

	case errors.Is(err, domain.ErrMessageEmpty),
		errors.Is(err, domain.ErrMessageTooLong),
		errors.Is(err, domain.ErrCantCreateDialogWithYourself),
		errors.Is(err, domain.ErrDialogMustHave2Users):
		return grpcerr.New(
			codes.InvalidArgument,
			int32(chatv1.ChatErrorCode_CHAT_ERROR_INVALID_INPUT),
			err.Error(),
		)

	case errors.Is(err, domain.ErrDialogCannotHaveCustomTitle):
		return grpcerr.New(
			codes.FailedPrecondition,
			int32(chatv1.ChatErrorCode_CHAT_ERROR_DIALOG_CANT_HAVE_CUSTOM_TITLE),
			err.Error(),
		)

	case errors.Is(err, domain.ErrCanJoinOnlyChannel):
		return grpcerr.New(
			codes.FailedPrecondition,
			int32(chatv1.ChatErrorCode_CHAT_ERROR_ONLY_CHANNEL_CAN_BE_JOINED),
			err.Error(),
		)

	case errors.Is(err, domain.ErrOnlyOwnerCanChangeDescription):
		return grpcerr.New(
			codes.PermissionDenied,
			int32(chatv1.ChatErrorCode_CHAT_ERROR_ONLY_OWNER_CAN_CHANGE_DESCRIPTION),
			err.Error(),
		)

	case errors.Is(err, domain.ErrDialogCannotHaveCustomDescription):
		return grpcerr.New(
			codes.FailedPrecondition,
			int32(chatv1.ChatErrorCode_CHAT_ERROR_DIALOG_CANT_HAVE_CUSTOM_DESCRIPTION),
			err.Error(),
		)

	case errors.Is(err, domain.ErrInvalidAttachment):
		return grpcerr.New(
			codes.InvalidArgument,
			int32(chatv1.ChatErrorCode_CHAT_ERROR_INVALID_ATTACHMENT),
			err.Error(),
		)

	case errors.Is(err, domain.ErrAttachmentNotOwned):
		return grpcerr.New(
			codes.PermissionDenied,
			int32(chatv1.ChatErrorCode_CHAT_ERROR_ATTACHMENT_NOT_OWNED),
			err.Error(),
		)

	case errors.Is(err, domain.ErrContactNotFound):
		return grpcerr.New(
			codes.NotFound,
			int32(chatv1.ChatErrorCode_CHAT_ERROR_CONTACT_NOT_FOUND),
			err.Error(),
		)

	case errors.Is(err, domain.ErrTooManyAttachments):
		return grpcerr.New(
			codes.InvalidArgument,
			int32(chatv1.ChatErrorCode_CHAT_ERROR_TOO_MANY_ATTACHMENTS),
			err.Error(),
		)

	case errors.Is(err, domain.ErrAttachmentForbidden):
		return grpcerr.New(
			codes.PermissionDenied,
			int32(chatv1.ChatErrorCode_CHAT_ERROR_NOT_MEMBER),
			err.Error(),
		)

	default:
		return grpcerr.New(
			codes.Internal,
			int32(chatv1.ChatErrorCode_CHAT_ERROR_INTERNAL),
			err.Error(),
		)
	}
}

type ChatServer struct {
	chatv1.UnimplementedChatServer
	chatUsecase    ChatUsecaseInterface
	messageUsecase MessageUsecaseInterface
	logger         *zap.Logger
}

func NewChatServer(chatSvc ChatUsecaseInterface, messageSvc MessageUsecaseInterface, logger *zap.Logger) *ChatServer {
	return &ChatServer{
		chatUsecase:    chatSvc,
		messageUsecase: messageSvc,
		logger:         logger,
	}
}

func (s *ChatServer) log(ctx context.Context) *zap.Logger {
	return loggerctx.EnrichLoggerFromContext(ctx, s.logger)
}

func (s *ChatServer) GetChats(ctx context.Context, chats *chatv1.RequestGetUserChats) (*chatv1.ResponseGetUserChats, error) {
	chatsRes, err := s.chatUsecase.GetAllChats(ctx, chats.GetUserId())
	if err != nil {
		return nil, mapDomainErr(err)
	}

	return &chatv1.ResponseGetUserChats{
		Chats: mapChatsInformationDTOToProtos(chatsRes),
	}, nil
}

func (s *ChatServer) Create(ctx context.Context, create *chatv1.RequestChatCreate) (*chatv1.ChatInformation, error) {
	createdChat, err := s.chatUsecase.CreateChat(ctx, dto.ChatCreate{
		Title:     create.GetTitle(),
		Type:      mapProtoToChatType(create.GetType()),
		MembersID: create.GetMembersId(),
	}, create.GetUserId())
	if err != nil {
		return nil, mapDomainErr(err)
	}

	return mapChatInformationDTOToProto(createdChat), nil
}

func (s *ChatServer) GetChatByID(ctx context.Context, id *chatv1.RequestGetChatByID) (*chatv1.ChatInformation, error) {
	chat, err := s.chatUsecase.GetChatByID(ctx, id.GetChatId(), id.GetUserId())
	if err != nil {
		return nil, mapDomainErr(err)
	}

	return mapChatInformationDTOToProto(chat), nil
}

func (s *ChatServer) DeleteChat(ctx context.Context, chat *chatv1.RequestDeleteChat) (*emptypb.Empty, error) {
	err := s.chatUsecase.DeleteChat(ctx, chat.GetUserId(), chat.GetChatId())
	if err != nil {
		return nil, mapDomainErr(err)
	}
	return &emptypb.Empty{}, nil
}

func (s *ChatServer) UpdateChatAvatar(ctx context.Context, avatar *chatv1.RequestUpdateAvatar) (*chatv1.ChatInformation, error) {
	responseUpdate, err := s.chatUsecase.UpdateChatAvatar(ctx, avatar.GetUserId(), avatar.ChatId, &dto.RequestUpdateAvatar{
		File: &media.FileInput{
			Body:        bytes.NewReader(avatar.GetContent()),
			ContentType: avatar.GetType(),
			Size:        int64(len(avatar.GetContent())),
		},
	})
	if err != nil {
		return nil, mapDomainErr(err)
	}

	return mapChatInformationDTOToProto(responseUpdate), nil
}

func (s *ChatServer) UpdateChatTitle(ctx context.Context, title *chatv1.RequestUpdateTitle) (*chatv1.ChatInformation, error) {
	responseUpdate, err := s.chatUsecase.UpdateChatTitle(ctx, title.GetUserId(), title.GetChatId(), &dto.RequestUpdateTitle{
		Title: title.GetTitle(),
	})
	if err != nil {
		return nil, mapDomainErr(err)
	}
	return mapChatInformationDTOToProto(responseUpdate), nil
}

func (s *ChatServer) UpdateChatDescription(ctx context.Context, description *chatv1.RequestUpdateDescription) (*chatv1.ChatInformation, error) {
	responseUpdate, err := s.chatUsecase.UpdateChatDescription(ctx, description.GetUserId(), description.GetChatId(), &dto.RequestUpdateDescription{
		Description: description.GetDescription(),
	})
	if err != nil {
		return nil, mapDomainErr(err)
	}
	return mapChatInformationDTOToProto(responseUpdate), nil
}

func (s *ChatServer) AddMembersToChat(ctx context.Context, chat *chatv1.RequestAddMembersToChat) (*emptypb.Empty, error) {
	err := s.chatUsecase.AddMembersToChat(ctx, chat.GetUserId(), chat.GetChatId(), &dto.RequestAddMember{
		MembersId: chat.GetMembersId(),
	})
	if err != nil {
		return nil, mapDomainErr(err)
	}
	return &emptypb.Empty{}, nil
}

func (s *ChatServer) JoinChannel(ctx context.Context, chat *chatv1.RequestJoinChannel) (*emptypb.Empty, error) {
	err := s.chatUsecase.JoinChannel(ctx, chat.GetUserId(), chat.GetChatId())
	if err != nil {
		return nil, mapDomainErr(err)
	}
	return &emptypb.Empty{}, nil
}

func (s *ChatServer) DeleteMemberFromChat(ctx context.Context, chat *chatv1.RequestDeleteMemberFromChat) (*emptypb.Empty, error) {
	err := s.chatUsecase.DeleteMemberFromChat(ctx, chat.GetUserId(), chat.GetChatId(), &dto.RequestDeleteMember{
		MemberId: chat.GetMemberId(),
	})
	if err != nil {
		return nil, mapDomainErr(err)
	}
	return &emptypb.Empty{}, nil
}

func (s *ChatServer) GetChatMembers(ctx context.Context, members *chatv1.RequestChatMembers) (*chatv1.ResponseGetChatMembers, error) {
	membersRes, err := s.chatUsecase.GetAllChatMembers(ctx, members.GetUserId(), members.GetChatId())
	if err != nil {
		return nil, mapDomainErr(err)
	}

	return &chatv1.ResponseGetChatMembers{
		MembersId: membersRes.MembersId,
	}, nil
}

func (s *ChatServer) UploadMessageAttachment(ctx context.Context, req *chatv1.RequestUploadMessageAttachment) (*chatv1.ResponseUploadMessageAttachment, error) {
	if req == nil || req.GetUserId() <= 0 {
		return nil, grpcerr.New(codes.InvalidArgument, int32(chatv1.ChatErrorCode_CHAT_ERROR_INVALID_INPUT), "user_id is required")
	}
	resp, err := s.messageUsecase.UploadMessageAttachment(ctx, req.GetUserId(), req.GetKind(), &media.FileInput{
		Body:        bytes.NewReader(req.GetContent()),
		ContentType: req.GetType(),
		Size:        int64(len(req.GetContent())),
	}, req.GetFileName())
	if err != nil {
		if _, ok := status.FromError(err); ok {
			return nil, err
		}
		return nil, mapDomainErr(err)
	}
	out := &chatv1.ResponseUploadMessageAttachment{
		AttachmentUrl: resp.AttachmentURL,
		MimeType:      resp.MimeType,
		FileSize:      resp.FileSize,
		ObjectKey:     resp.ObjectKey,
	}
	if resp.FileName != nil {
		out.FileName = resp.FileName
	}
	return out, nil
}

func (s *ChatServer) AuthorizeMessageAttachment(ctx context.Context, req *chatv1.RequestAuthorizeMessageAttachment) (*chatv1.ResponseAuthorizeMessageAttachment, error) {
	if req == nil || req.GetUserId() <= 0 || req.GetObjectKey() == "" {
		return nil, grpcerr.New(codes.InvalidArgument, int32(chatv1.ChatErrorCode_CHAT_ERROR_INVALID_INPUT), "user_id and object_key are required")
	}
	if err := s.messageUsecase.AuthorizeMessageAttachment(ctx, req.GetUserId(), req.GetObjectKey()); err != nil {
		return nil, mapDomainErr(err)
	}
	return &chatv1.ResponseAuthorizeMessageAttachment{}, nil
}

func (s *ChatServer) QuitChat(ctx context.Context, chat *chatv1.RequestQuitChat) (*emptypb.Empty, error) {
	err := s.chatUsecase.QuitChat(ctx, chat.GetUserId(), chat.GetChatId())
	if err != nil {
		return nil, mapDomainErr(err)
	}

	return &emptypb.Empty{}, nil
}
