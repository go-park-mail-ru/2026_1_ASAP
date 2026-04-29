package grpc

import (
	chatv1 "github.com/go-park-mail-ru/2026_1_ASAP/gen/go/chat/v1"
	dto "github.com/go-park-mail-ru/2026_1_ASAP/internal/chat/dto/chat"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func mapChatsInformationDTOToProtos(chats []dto.ChatInformationDTO) []*chatv1.ChatInformation {
	result := make([]*chatv1.ChatInformation, 0, len(chats))
	for _, chat := range chats {
		result = append(result, mapChatInformationDTOToProto(&chat))
	}

	return result
}

func mapChatInformationDTOToProto(dto *dto.ChatInformationDTO) *chatv1.ChatInformation {
	return &chatv1.ChatInformation{
		Id:          dto.ID,
		Title:       dto.Title,
		Type:        mapChatTypeToProto(&dto.ChatType),
		Avatar:      dto.Avatar,
		LastMessage: mapMessageDTOToProto(&dto.LastMessage),
	}
}

func mapChatTypeToProto(chatType *dto.ChatType) chatv1.ChatType {
	switch *chatType {
	case dto.ChatTypeChannel:
		return chatv1.ChatType_CHANNEL
	case dto.ChatTypeDialog:
		return chatv1.ChatType_DIALOG
	case dto.ChatTypeGroup:
		return chatv1.ChatType_GROUP
	}
	return chatv1.ChatType(0)
}

func mapMessageDTOToProto(message *dto.MessageDTO) *chatv1.MessageInformation {
	return &chatv1.MessageInformation{
		SenderId:  message.SenderId,
		Text:      message.Text,
		CreatedAt: timestamppb.New(message.CreatedAt),
	}
}

func mapProtoToChatType(chatType chatv1.ChatType) dto.ChatType {
	switch chatType {
	case chatv1.ChatType_CHANNEL:
		return dto.ChatTypeChannel
	case chatv1.ChatType_DIALOG:
		return dto.ChatTypeDialog
	case chatv1.ChatType_GROUP:
		return dto.ChatTypeGroup
	}
	return dto.ChatType(0)
}
