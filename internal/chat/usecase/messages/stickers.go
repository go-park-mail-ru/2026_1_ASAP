package messages

import (
	"context"
	"fmt"

	domain "github.com/go-park-mail-ru/2026_1_ASAP/internal/chat/domain/chat"
	dto "github.com/go-park-mail-ru/2026_1_ASAP/internal/chat/dto/message"
	stickerdto "github.com/go-park-mail-ru/2026_1_ASAP/internal/chat/dto/sticker"
)

const stickerMessagePreview = "[Стикер]"

func (m MessageService) SendSticker(ctx context.Context, userID, chatID int64, req *dto.RequestSendSticker) (*dto.ResponseSendMessage, error) {
	if req == nil {
		return nil, fmt.Errorf("send sticker nil request")
	}
	if req.StickerID <= 0 {
		return nil, domain.ErrInvalidSticker
	}
	if m.stickerRepo == nil {
		return nil, fmt.Errorf("sticker repository is nil")
	}

	isUserMember, err := m.chatRepo.IsMember(ctx, chatID, userID)
	if err != nil {
		return nil, fmt.Errorf("chatrepo check is member: %w", err)
	}
	if !isUserMember {
		return nil, domain.ErrMessageNotMember
	}

	chat, err := m.chatRepo.GetChatByID(ctx, chatID)
	if err != nil {
		return nil, fmt.Errorf("failed to get chat by id: %w", err)
	}
	if chat.Type == domain.ChatTypeChannel && userID != chat.OwnerId {
		return nil, domain.ErrOnlyOwnerCanSendMessaage
	}

	sticker, err := m.stickerRepo.GetStickerByID(ctx, req.StickerID)
	if err != nil {
		return nil, err
	}

	message := &domain.Message{
		ChatId:    chatID,
		SenderId:  userID,
		Content:   stickerMessagePreview,
		StickerId: &sticker.Id,
	}
	createdMessage, err := m.messageRepo.CreateMessage(ctx, message)
	if err != nil {
		return nil, fmt.Errorf("messageRepo create sticker message: %w", err)
	}
	resp := messageToSendResponse(createdMessage, false, false)
	resp.Sticker = mapStickerToDTO(*sticker)
	return resp, nil
}

func (m MessageService) stickersByMessage(ctx context.Context, messages []*domain.Message) (map[int64]domain.Sticker, error) {
	if m.stickerRepo == nil {
		return map[int64]domain.Sticker{}, nil
	}
	seen := make(map[int64]struct{})
	ids := make([]int64, 0)
	for _, msg := range messages {
		if msg == nil || msg.StickerId == nil {
			continue
		}
		if _, ok := seen[*msg.StickerId]; ok {
			continue
		}
		seen[*msg.StickerId] = struct{}{}
		ids = append(ids, *msg.StickerId)
	}
	if len(ids) == 0 {
		return map[int64]domain.Sticker{}, nil
	}
	stickers, err := m.stickerRepo.GetStickersByIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("stickerRepo get stickers by ids: %w", err)
	}
	return stickers, nil
}

func stickerDTOFromMessage(msg *domain.Message, stickers map[int64]domain.Sticker) *stickerdto.StickerDTO {
	if msg == nil || msg.StickerId == nil {
		return nil
	}
	sticker, ok := stickers[*msg.StickerId]
	if !ok {
		return nil
	}
	return mapStickerToDTO(sticker)
}

func mapStickerToDTO(sticker domain.Sticker) *stickerdto.StickerDTO {
	return &stickerdto.StickerDTO{
		ID:      sticker.Id,
		PackID:  sticker.PackID,
		Slug:    sticker.Slug,
		Emoji:   sticker.Emoji,
		FileURL: sticker.FileURL,
		Width:   sticker.Width,
		Height:  sticker.Height,
	}
}
