package chat

import (
	domain "github.com/go-park-mail-ru/2026_1_ASAP/internal/chat/domain/chat"
	"github.com/go-park-mail-ru/2026_1_ASAP/pkg/null"
)

func toDomainChat(chatModel *ChatModel) *domain.Chat {
	return &domain.Chat{
		Id:          chatModel.Id,
		Type:        domain.ChatType(chatModel.Type),
		Title:       chatModel.Title,
		Description: null.NullStringToPtrString(chatModel.Description),
		OwnerId:     chatModel.OwnerId,
		AvatarUrl:   null.NullStringToPtrString(chatModel.AvatarUrl),
		CreatedAt:   chatModel.CreatedAt,
		UpdatedAt:   chatModel.UpdatedAt,
	}
}

func toModelChat(chatDomain *domain.Chat) *ChatModel {
	return &ChatModel{
		Id:          chatDomain.Id,
		Type:        ChatType(chatDomain.Type),
		Title:       chatDomain.Title,
		Description: null.StringPtrToNullString(chatDomain.Description),
		OwnerId:     chatDomain.OwnerId,
		AvatarUrl:   null.StringPtrToNullString(chatDomain.AvatarUrl),
		CreatedAt:   chatDomain.CreatedAt,
		UpdatedAt:   chatDomain.UpdatedAt,
	}
}

func toDomainMessage(msgModel *MessageModel) *domain.Message {
	return &domain.Message{
		Id:        msgModel.Id,
		ChatId:    msgModel.ChatId,
		SenderId:  msgModel.SenderId,
		Content:   msgModel.Content,
		StickerId: null.NullInt64ToPtrInt64(msgModel.StickerId),
		Edited:    msgModel.Edited,
		CreatedAt: msgModel.CreatedAt,
		UpdatedAt: msgModel.UpdatedAt,
		DeletedAt: null.NullTimeToPtrTime(msgModel.DeletedAt),
	}
}
