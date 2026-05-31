package grpc

import (
	"google.golang.org/protobuf/types/known/timestamppb"

	chatv1 "github.com/go-park-mail-ru/2026_1_ASAP/gen/go/chat/v1"
	dto "github.com/go-park-mail-ru/2026_1_ASAP/internal/chat/dto/chat"
	stickerdto "github.com/go-park-mail-ru/2026_1_ASAP/internal/chat/dto/sticker"
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
		Id:                dto.ID,
		Title:             dto.Title,
		Type:              mapChatTypeToProto(&dto.ChatType),
		Avatar:            dto.Avatar,
		LastMessage:       mapMessageDTOToProto(&dto.LastMessage),
		OwnerId:           dto.OwnerID,
		Description:       dto.Description,
		UnreadCount:       dto.UnreadCount,
		LastReadMessageId: dto.LastReadMessageID,
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
	return dto.ChatType("")
}

func mapStickerPacksDTOToProto(resp *stickerdto.ResponseGetStickerPacks) *chatv1.ResponseGetStickerPacks {
	if resp == nil {
		return &chatv1.ResponseGetStickerPacks{}
	}
	out := make([]*chatv1.StickerPack, 0, len(resp.Packs))
	for _, pack := range resp.Packs {
		item := &chatv1.StickerPack{
			Id:       pack.ID,
			Name:     pack.Name,
			Title:    pack.Title,
			Slug:     pack.Slug,
			Stickers: make([]*chatv1.Sticker, 0, len(pack.Stickers)),
		}
		if pack.ThumbnailURL != nil {
			item.ThumbnailUrl = pack.ThumbnailURL
		}
		for _, sticker := range pack.Stickers {
			item.Stickers = append(item.Stickers, mapStickerDTOToProto(sticker))
		}
		out = append(out, item)
	}
	return &chatv1.ResponseGetStickerPacks{Packs: out}
}

func mapStickerDTOToProto(sticker stickerdto.StickerDTO) *chatv1.Sticker {
	out := &chatv1.Sticker{
		Id:      sticker.ID,
		PackId:  sticker.PackID,
		FileUrl: sticker.FileURL,
		Slug:    sticker.Slug,
		Emoji:   sticker.Emoji,
	}
	if sticker.Width != nil {
		width := int32(*sticker.Width)
		out.Width = &width
	}
	if sticker.Height != nil {
		height := int32(*sticker.Height)
		out.Height = &height
	}
	return out
}
