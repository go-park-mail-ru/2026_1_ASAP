package messages

import (
	domain "github.com/go-park-mail-ru/2026_1_ASAP/internal/chat/domain/chat"
	"github.com/go-park-mail-ru/2026_1_ASAP/pkg/null"
)

func toDomainModel(model *MessageModel) *domain.Message {
	return &domain.Message{
		Id:        model.Id,
		ChatId:    model.ChatId,
		SenderId:  model.SenderId,
		Content:   model.Content,
		StickerId: null.NullInt64ToPtrInt64(model.StickerId),
		Edited:    model.Edited,
		CreatedAt: model.CreatedAt,
		UpdatedAt: model.UpdatedAt,
		DeletedAt: null.NullTimeToPtrTime(model.DeletedAt),
	}
}

func toModel(domainModel *domain.Message) *MessageModel {
	return &MessageModel{
		Id:        domainModel.Id,
		ChatId:    domainModel.ChatId,
		SenderId:  domainModel.SenderId,
		Content:   domainModel.Content,
		StickerId: null.PtrInt64ToNullInt64(domainModel.StickerId),
		Edited:    domainModel.Edited,
		CreatedAt: domainModel.CreatedAt,
		UpdatedAt: domainModel.UpdatedAt,
		DeletedAt: null.TimePtrToNullTime(domainModel.DeletedAt),
	}
}
