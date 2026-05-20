package ws

import (
	"context"
	"encoding/json"
	"time"

	"go.uber.org/zap"

	dtoWs "github.com/go-park-mail-ru/2026_1_ASAP/internal/chat/dto/ws"
	onlinerepo "github.com/go-park-mail-ru/2026_1_ASAP/internal/chat/repository/online"
	"github.com/go-park-mail-ru/2026_1_ASAP/pkg/loggerctx"
)

func (s *ChatServer) wasFirstConnection(userID int64) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	userHub, ok := s.subscribersByUserID[userID]
	return ok && len(userHub) == 1
}

func (s *ChatServer) isLastConnection(userID int64) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	userHub, ok := s.subscribersByUserID[userID]
	return !ok || len(userHub) == 0
}

func (s *ChatServer) userAllConnectionsAway(userID int64) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	userHub, ok := s.subscribersByUserID[userID]
	if !ok || len(userHub) == 0 {
		return true
	}
	for sub := range userHub {
		if !sub.away.Load() {
			return false
		}
	}
	return true
}

func (s *ChatServer) broadcastPresenceToOthers(ctx context.Context, excludeUserID int64, frame []byte) {
	s.mu.RLock()
	subs := make([]*subscriber, 0, len(s.subscribers))
	for sub := range s.subscribers {
		if sub.userID != excludeUserID {
			subs = append(subs, sub)
		}
	}
	s.mu.RUnlock()

	for _, sub := range subs {
		payload := append([]byte(nil), frame...)
		s.enqueueToSubscriber(sub, payload)
	}
}

func (s *ChatServer) publishPresenceOnline(ctx context.Context, userID int64) {
	log := loggerctx.From(ctx)
	if s.onlineRepo == nil {
		return
	}
	wasOnline, err := s.onlineRepo.IsOnline(ctx, userID)
	if err != nil {
		log.Warn("presence is online check", zap.Int64("user_id", userID), zap.Error(err))
	}
	if err := s.onlineRepo.SetOnline(ctx, userID); err != nil {
		log.Warn("presence set online", zap.Int64("user_id", userID), zap.Error(err))
		return
	}
	if wasOnline {
		return
	}
	frame, err := dtoWs.EncodePresenceOnline(userID)
	if err != nil {
		log.Error("ws encode presence online", zap.Error(err))
		return
	}
	s.broadcastPresenceToOthers(ctx, userID, frame)
}

func (s *ChatServer) publishPresenceOffline(ctx context.Context, userID int64, forcePeerNotify bool) {
	log := loggerctx.From(ctx)
	if s.onlineRepo != nil {
		wasOnline, err := s.onlineRepo.IsOnline(ctx, userID)
		if err != nil {
			log.Warn("presence is online check", zap.Int64("user_id", userID), zap.Error(err))
		}
		if !wasOnline && !forcePeerNotify {
			return
		}
		if wasOnline {
			if err := s.onlineRepo.SetOffline(ctx, userID); err != nil {
				log.Warn("presence set offline", zap.Int64("user_id", userID), zap.Error(err))
				return
			}
		}
	}

	lastSeenAt := time.Now().UTC()
	if s.presenceService != nil {
		if err := s.presenceService.UpdateLastSeen(ctx, userID); err != nil {
			log.Warn("ws update last seen", zap.Int64("user_id", userID), zap.Error(err))
		}
	}

	if frame, err := dtoWs.EncodePresenceOffline(userID); err != nil {
		log.Error("ws encode presence offline", zap.Error(err))
	} else {
		s.broadcastPresenceToOthers(ctx, userID, frame)
	}
	if frame, err := dtoWs.EncodePresenceLastSeen(userID, lastSeenAt); err != nil {
		log.Error("ws encode presence last seen", zap.Error(err))
	} else {
		s.broadcastPresenceToOthers(ctx, userID, frame)
	}
}

func (s *ChatServer) notifyUserOnline(ctx context.Context, userID int64) {
	if !s.wasFirstConnection(userID) {
		return
	}
	if s.userAllConnectionsAway(userID) {
		return
	}
	s.publishPresenceOnline(ctx, userID)
}

func (s *ChatServer) notifyUserOffline(ctx context.Context, userID int64) {
	if !s.isLastConnection(userID) {
		return
	}
	s.publishPresenceOffline(ctx, userID, false)
}

func (s *ChatServer) touchOnlineIfPresent(ctx context.Context, userID int64, sub *subscriber) {
	if sub.away.Load() || s.onlineRepo == nil {
		return
	}
	if err := s.onlineRepo.TouchOnline(ctx, userID); err != nil {
		loggerctx.From(ctx).Warn("presence touch online", zap.Int64("user_id", userID), zap.Error(err))
	}
}

func (s *ChatServer) handlePresencePing(ctx context.Context, userID int64, sub *subscriber) {
	s.touchOnlineIfPresent(ctx, userID, sub)
}

func (s *ChatServer) handlePresenceBackground(ctx context.Context, userID int64, sub *subscriber) {
	sub.away.Store(true)
	if s.userAllConnectionsAway(userID) {
		s.publishPresenceOffline(ctx, userID, true)
	}
}

func (s *ChatServer) handlePresenceForeground(ctx context.Context, userID int64, sub *subscriber) {
	sub.away.Store(false)
	s.publishPresenceOnline(ctx, userID)
	s.touchOnlineIfPresent(ctx, userID, sub)
}

// runPresenceRedisRefresh extends Redis presence TTL while the WebSocket stays open and
// the client is not in presence.Background, so profile/search IsOnline stays aligned with
// chat presence without relying only on client pings or unrelated WS traffic.
func (s *ChatServer) runPresenceRedisRefresh(ctx context.Context, userID int64, sub *subscriber) {
	interval := onlinerepo.OnlineTTL / 2
	if interval < 10*time.Second {
		interval = 10 * time.Second
	}
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			s.touchOnlineIfPresent(ctx, userID, sub)
		}
	}
}

func (s *ChatServer) publishTypingToChatMembers(ctx context.Context, chatID, userID int64, typing bool, excludeSender bool) {
	log := loggerctx.From(ctx)
	frame, err := dtoWs.EncodePresenceTyping(chatID, userID, typing)
	if err != nil {
		log.Error("ws encode presence typing", zap.Error(err))
		return
	}

	memberIDs, err := s.chatService.GetChatMemberIDs(ctx, chatID)
	if err != nil {
		log.Warn("ws get chat members for typing", zap.Int64("chat_id", chatID), zap.Error(err))
		return
	}

	s.mu.RLock()
	subs := make([]*subscriber, 0)
	for _, uid := range memberIDs {
		if excludeSender && uid == userID {
			continue
		}
		if userHub, ok := s.subscribersByUserID[uid]; ok && len(userHub) > 0 {
			for sub := range userHub {
				subs = append(subs, sub)
			}
		}
	}
	s.mu.RUnlock()

	for _, sub := range subs {
		payload := append([]byte(nil), frame...)
		s.enqueueToSubscriber(sub, payload)
	}
}

func (s *ChatServer) handlePresenceTyping(ctx context.Context, userID int64, sub *subscriber, env dtoWs.WsRequest, typing bool) {
	log := loggerctx.From(ctx)
	var req dtoWs.RequestPresenceTyping
	if len(env.Payload) == 0 {
		s.sendErr(sub, dtoWs.WsErrorPayload{
			Code:    dtoWs.ErrCodeInvalidPayload,
			Message: dtoWs.ErrCodeInvalidPayloadMsg,
		})
		return
	}
	if err := json.Unmarshal(env.Payload, &req); err != nil {
		s.sendErr(sub, dtoWs.WsErrorPayload{
			Code:    dtoWs.ErrCodeInvalidPayload,
			Message: dtoWs.ErrCodeInvalidPayloadMsg,
		})
		return
	}
	if req.ChatID <= 0 {
		s.sendErr(sub, dtoWs.WsErrorPayload{
			Code:    dtoWs.ErrCodeInvalidPayload,
			Message: dtoWs.ErrCodeInvalidPayloadMsg,
		})
		return
	}

	s.publishTypingToChatMembers(ctx, req.ChatID, userID, typing, true)
	log.Debug("ws presence typing", zap.Int64("chat_id", req.ChatID), zap.Bool("typing", typing))
}
