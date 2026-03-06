package handlers

import (
	chatService "github.com/go-park-mail-ru/2026_1_ASAP/internal/services/chat"
)

type ChatsHandler struct {
	chatService chatService.ChatServiceInterface
}

func NewChatHandler(chatService chatService.ChatServiceInterface) *ChatsHandler {
	return &ChatsHandler{
		chatService: chatService,
	}
}

