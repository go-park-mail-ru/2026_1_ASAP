package ws

import (
	"context"

	chatdto "github.com/go-park-mail-ru/2026_1_ASAP/internal/chat/dto/chat"
	dtoWs "github.com/go-park-mail-ru/2026_1_ASAP/internal/chat/dto/ws"
	"go.uber.org/zap"
)

type RealtimeNotifier struct {
	hub *ChatServer
	log *zap.Logger
}

func NewRealtimeNotifier(log *zap.Logger) *RealtimeNotifier {
	if log == nil {
		log = zap.NewNop()
	}
	return &RealtimeNotifier{log: log}
}

func (n *RealtimeNotifier) BindHub(h *ChatServer) {
	if n != nil {
		n.hub = h
	}
}

func (n *RealtimeNotifier) NotifyChatNew(ctx context.Context, viewerUserID int64, chat *chatdto.ChatInformationDTO) {
	if n == nil || n.hub == nil || chat == nil {
		return
	}
	b, err := dtoWs.EncodeChatNew(chat)
	if err != nil {
		n.log.Warn("ws encode chat.New", zap.Error(err))
		return
	}
	n.hub.PublishToUser(ctx, viewerUserID, b)
}

func (n *RealtimeNotifier) NotifyChatDeleted(ctx context.Context, formerMemberUserIDs []int64, chatID int64) {
	if n == nil || n.hub == nil || len(formerMemberUserIDs) == 0 {
		return
	}
	b, err := dtoWs.EncodeChatDeleted(chatID)
	if err != nil {
		n.log.Warn("ws encode chat.Deleted", zap.Error(err))
		return
	}
	for _, uid := range formerMemberUserIDs {
		p := append([]byte(nil), b...)
		n.hub.PublishToUser(ctx, uid, p)
	}
}

func (n *RealtimeNotifier) NotifyChatAvatarUpdated(ctx context.Context, memberUserIDs []int64, chatID int64, avatarURL string) {
	if n == nil || n.hub == nil || len(memberUserIDs) == 0 {
		return
	}
	b, err := dtoWs.EncodeChatUpdatedAvatar(chatID, avatarURL)
	if err != nil {
		n.log.Warn("ws encode chat.Updated.Avatar", zap.Error(err))
		return
	}
	for _, uid := range memberUserIDs {
		p := append([]byte(nil), b...)
		n.hub.PublishToUser(ctx, uid, p)
	}
}

func (n *RealtimeNotifier) NotifyChatTitleUpdated(ctx context.Context, memberUserIDs []int64, chatID int64, title string) {
	if n == nil || n.hub == nil || len(memberUserIDs) == 0 {
		return
	}
	b, err := dtoWs.EncodeChatUpdatedTitle(chatID, title)
	if err != nil {
		n.log.Warn("ws encode chat.Updated.Title", zap.Error(err))
		return
	}
	for _, uid := range memberUserIDs {
		p := append([]byte(nil), b...)
		n.hub.PublishToUser(ctx, uid, p)
	}
}

func (n *RealtimeNotifier) NotifyChatDescriptionUpdated(ctx context.Context, memberUserIDs []int64, chatID int64, description *string) {
	if n == nil || n.hub == nil || len(memberUserIDs) == 0 {
		return
	}
	b, err := dtoWs.EncodeChatUpdatedDescription(chatID, description)
	if err != nil {
		n.log.Warn("ws encode chat.Updated.Description", zap.Error(err))
		return
	}
	for _, uid := range memberUserIDs {
		p := append([]byte(nil), b...)
		n.hub.PublishToUser(ctx, uid, p)
	}
}

func (n *RealtimeNotifier) NotifyChatMembersUpdated(ctx context.Context, memberUserIDs []int64, chatID int64, changeType string, updatedMemberIDs []int64, name string) {
	if n == nil || n.hub == nil || len(memberUserIDs) == 0 || len(updatedMemberIDs) == 0 {
		return
	}
	b, err := dtoWs.EncodeChatUpdatedMembers(chatID, changeType, updatedMemberIDs, name)
	if err != nil {
		n.log.Warn("ws encode chat.Updated.Members", zap.Error(err))
		return
	}
	for _, uid := range memberUserIDs {
		p := append([]byte(nil), b...)
		n.hub.PublishToUser(ctx, uid, p)
	}
}
