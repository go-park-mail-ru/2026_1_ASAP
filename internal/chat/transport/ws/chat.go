package ws

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/coder/websocket"
	"github.com/google/uuid"
	"go.uber.org/zap"

	domain "github.com/go-park-mail-ru/2026_1_ASAP/internal/chat/domain/chat"
	chatdto "github.com/go-park-mail-ru/2026_1_ASAP/internal/chat/dto/chat"
	dto "github.com/go-park-mail-ru/2026_1_ASAP/internal/chat/dto/message"
	dtoWs "github.com/go-park-mail-ru/2026_1_ASAP/internal/chat/dto/ws"
	messagesuc "github.com/go-park-mail-ru/2026_1_ASAP/internal/chat/usecase/messages"
	"github.com/go-park-mail-ru/2026_1_ASAP/pkg/loggerctx"
)

const (
	wsWriteTimeout = 3 * time.Second
	wsPingInterval = 30 * time.Second
	wsPingTimeout  = 10 * time.Second
)

type ctxKey string

const userIDKey ctxKey = "ws_user_id"
const requestIDKey ctxKey = "ws_request_id"

func UserIDFromContext(ctx context.Context) (int64, bool) {
	id, ok := ctx.Value(userIDKey).(int64)
	return id, ok
}

func WithUserID(ctx context.Context, userID int64) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

func WithRequestID(ctx context.Context, reqID string) context.Context {
	return context.WithValue(ctx, requestIDKey, reqID)
}

func requestIDFromContext(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(requestIDKey).(string)
	return id, ok
}

type MessagesServiceInterface interface {
	SendMessage(ctx context.Context, userID int64, chatID int64, req *dto.RequestSendMessage) (*dto.ResponseSendMessage, error)
	SendSticker(ctx context.Context, userID int64, chatID int64, req *dto.RequestSendSticker) (*dto.ResponseSendMessage, error)
	SendMessageWithAttachments(ctx context.Context, userID int64, chatID int64, req *dto.RequestSendMessageAttachments) (*dto.ResponseSendMessage, error)
	GetMessagesByChatId(ctx context.Context, userID int64, chatID int64, req *dto.RequestGetMessages) (*dto.ResponseGetMessages, error)
	EditMessage(ctx context.Context, userID, chatID int64, req *dto.RequestEditMessage) (*dto.ResponseEditMessage, error)
	DeleteMessage(ctx context.Context, userID, chatID int64, req *dto.RequestDeleteMessage) (*dto.ResponseClearMessage, error)
	MarkMessagesRead(ctx context.Context, userID, chatID int64, req *dto.RequestMarkRead) (*dto.ResponseMarkRead, error)
	TranscribeVoiceMessage(ctx context.Context, userID, chatID, messageID int64) (*dto.ResponseVoiceTranscript, error)
}

type SubscriptionChecker interface {
	IsActive(ctx context.Context, userID int64) (bool, error)
}

type ChatServiceInterface interface {
	GetChatMemberIDs(ctx context.Context, chatID int64) ([]int64, error)
	GetChatByID(ctx context.Context, chatID, userID int64) (*chatdto.ChatInformationDTO, error)
}

type PresenceServiceInterface interface {
	UpdateLastSeen(ctx context.Context, userID int64) error
}

type OnlineRepository interface {
	SetOnline(ctx context.Context, userID int64) error
	SetOffline(ctx context.Context, userID int64) error
	TouchOnline(ctx context.Context, userID int64) error
	IsOnline(ctx context.Context, userID int64) (bool, error)
}

type subscriber struct {
	conn      *websocket.Conn
	msgs      chan []byte
	cancel    context.CancelFunc
	closeSlow func()
	userID    int64
	away      atomic.Bool
}

type ChatServer struct {
	messageService          MessagesServiceInterface
	chatService             ChatServiceInterface
	presenceService         PresenceServiceInterface
	subscription            SubscriptionChecker
	onlineRepo              OnlineRepository
	subscribers             map[*subscriber]struct{}
	subscribersByUserID     map[int64]map[*subscriber]struct{}
	logger                  *zap.Logger
	wg                      sync.WaitGroup
	subscriberMessageBuffer int
	mu                      sync.RWMutex
}

func NewChatServer(
	logger *zap.Logger,
	messageService MessagesServiceInterface,
	chatService ChatServiceInterface,
	presenceService PresenceServiceInterface,
	subscription SubscriptionChecker,
	onlineRepo OnlineRepository,
) *ChatServer {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &ChatServer{
		subscribers:             make(map[*subscriber]struct{}),
		subscribersByUserID:     make(map[int64]map[*subscriber]struct{}),
		logger:                  logger,
		messageService:          messageService,
		chatService:             chatService,
		presenceService:         presenceService,
		subscription:            subscription,
		onlineRepo:              onlineRepo,
		subscriberMessageBuffer: 16,
	}
}

func (s *ChatServer) addSubscriber(sub *subscriber) {
	s.mu.Lock()
	s.subscribers[sub] = struct{}{}
	userHub := s.subscribersByUserID[sub.userID]
	if userHub == nil {
		userHub = make(map[*subscriber]struct{})
		s.subscribersByUserID[sub.userID] = userHub
	}
	userHub[sub] = struct{}{}
	s.mu.Unlock()
}

func (s *ChatServer) removeSubscriber(sub *subscriber) {
	s.mu.Lock()
	delete(s.subscribers, sub)
	if userHub := s.subscribersByUserID[sub.userID]; userHub != nil {
		delete(userHub, sub)
		if len(userHub) == 0 {
			delete(s.subscribersByUserID, sub.userID)
		}
	}
	s.mu.Unlock()
}

func (s *ChatServer) enqueueToSubscriber(sub *subscriber, frame []byte) {
	select {
	case sub.msgs <- frame:
	default:
		if sub.closeSlow != nil {
			sub.closeSlow()
		}
	}
}

func (s *ChatServer) sendErr(sub *subscriber, p dtoWs.WsErrorPayload) {
	b, err := dtoWs.EncodeError(p)
	if err != nil {
		return
	}
	s.enqueueToSubscriber(sub, b)
}

func (s *ChatServer) publishMessageNewPerViewer(ctx context.Context, chatID int64, resp *dto.ResponseSendMessage) {
	log := loggerctx.From(ctx)
	if resp == nil {
		return
	}
	memberIDs, err := s.chatService.GetChatMemberIDs(ctx, chatID)
	if err != nil {
		log.Warn("ws get chat members for publish", zap.Int64("chat_id", chatID), zap.Error(err))
		return
	}
	if len(memberIDs) == 0 {
		return
	}

	s.mu.RLock()
	type subTarget struct {
		sub    *subscriber
		userID int64
	}
	targets := make([]subTarget, 0)
	for _, uid := range memberIDs {
		if userHub, ok := s.subscribersByUserID[uid]; ok {
			for sub := range userHub {
				targets = append(targets, subTarget{sub: sub, userID: uid})
			}
		}
	}
	s.mu.RUnlock()

	for _, t := range targets {
		active := false
		if s.subscription != nil {
			var subErr error
			active, subErr = s.subscription.IsActive(ctx, t.userID)
			if subErr != nil {
				log.Warn("ws subscription check for message new", zap.Int64("user_id", t.userID), zap.Error(subErr))
			}
		}
		presented := messagesuc.PresentSendMessageForViewer(resp, active)
		out, encErr := dtoWs.EncodeMessageNew(presented)
		if encErr != nil {
			log.Error("ws encode message new", zap.Error(encErr))
			continue
		}
		s.enqueueToSubscriber(t.sub, out)
	}
}

func (s *ChatServer) publishMessageEditPerViewer(ctx context.Context, chatID int64, resp *dto.ResponseEditMessage) {
	log := loggerctx.From(ctx)
	if resp == nil {
		return
	}
	memberIDs, err := s.chatService.GetChatMemberIDs(ctx, chatID)
	if err != nil {
		log.Warn("ws get chat members for publish", zap.Int64("chat_id", chatID), zap.Error(err))
		return
	}
	if len(memberIDs) == 0 {
		return
	}

	s.mu.RLock()
	type subTarget struct {
		sub    *subscriber
		userID int64
	}
	targets := make([]subTarget, 0)
	for _, uid := range memberIDs {
		if userHub, ok := s.subscribersByUserID[uid]; ok {
			for sub := range userHub {
				targets = append(targets, subTarget{sub: sub, userID: uid})
			}
		}
	}
	s.mu.RUnlock()

	for _, t := range targets {
		active := false
		if s.subscription != nil {
			var subErr error
			active, subErr = s.subscription.IsActive(ctx, t.userID)
			if subErr != nil {
				log.Warn("ws subscription check for message edit", zap.Int64("user_id", t.userID), zap.Error(subErr))
			}
		}
		presented := messagesuc.PresentEditMessageForViewer(resp, active)
		out, encErr := dtoWs.EncodeMessageEdit(presented)
		if encErr != nil {
			log.Error("ws encode message edit", zap.Error(encErr))
			continue
		}
		s.enqueueToSubscriber(t.sub, out)
	}
}

func (s *ChatServer) publishMessageDeletePerViewer(ctx context.Context, chatID int64, resp *dto.ResponseClearMessage) {
	log := loggerctx.From(ctx)
	if resp == nil {
		return
	}
	memberIDs, err := s.chatService.GetChatMemberIDs(ctx, chatID)
	if err != nil {
		log.Warn("ws get chat members for publish", zap.Int64("chat_id", chatID), zap.Error(err))
		return
	}
	if len(memberIDs) == 0 {
		return
	}

	s.mu.RLock()
	type subTarget struct {
		sub    *subscriber
		userID int64
	}
	targets := make([]subTarget, 0)
	for _, uid := range memberIDs {
		if userHub, ok := s.subscribersByUserID[uid]; ok {
			for sub := range userHub {
				targets = append(targets, subTarget{sub: sub, userID: uid})
			}
		}
	}
	s.mu.RUnlock()

	for _, t := range targets {
		active := false
		if s.subscription != nil {
			var subErr error
			active, subErr = s.subscription.IsActive(ctx, t.userID)
			if subErr != nil {
				log.Warn("ws subscription check for message delete", zap.Int64("user_id", t.userID), zap.Error(subErr))
			}
		}
		presented := messagesuc.PresentClearMessageForViewer(resp, active)
		out, encErr := dtoWs.EncodeMessageDelete(presented)
		if encErr != nil {
			log.Error("ws encode message delete", zap.Error(encErr))
			continue
		}
		s.enqueueToSubscriber(t.sub, out)
	}
}

func (s *ChatServer) publishBytesToChatMembers(ctx context.Context, chatID int64, message []byte) {
	log := loggerctx.From(ctx)
	memberIDs, err := s.chatService.GetChatMemberIDs(ctx, chatID)
	if err != nil {
		log.Warn("ws get chat members for publish", zap.Int64("chat_id", chatID), zap.Error(err))
		return
	}
	if len(memberIDs) == 0 {
		return
	}

	s.mu.RLock()
	subs := make([]*subscriber, 0)
	for _, uid := range memberIDs {
		if userHub, ok := s.subscribersByUserID[uid]; ok && len(userHub) > 0 {
			for sub := range userHub {
				subs = append(subs, sub)
			}
		}
	}
	s.mu.RUnlock()

	for _, sub := range subs {
		payload := append([]byte(nil), message...)
		s.enqueueToSubscriber(sub, payload)
	}
}

func (s *ChatServer) publishVoiceTranscriptToSubscribers(ctx context.Context, chatID, requesterUserID int64, payload []byte) {
	log := loggerctx.From(ctx)
	memberIDs, err := s.chatService.GetChatMemberIDs(ctx, chatID)
	if err != nil {
		log.Warn("ws get chat members for voice transcript", zap.Int64("chat_id", chatID), zap.Error(err))
		return
	}
	if len(memberIDs) == 0 {
		return
	}

	s.mu.RLock()
	type subTarget struct {
		sub    *subscriber
		userID int64
	}
	targets := make([]subTarget, 0)
	for _, uid := range memberIDs {
		if userHub, ok := s.subscribersByUserID[uid]; ok {
			for sub := range userHub {
				targets = append(targets, subTarget{sub: sub, userID: uid})
			}
		}
	}
	s.mu.RUnlock()

	for _, t := range targets {
		if t.userID == requesterUserID {
			continue
		}
		if s.subscription == nil {
			continue
		}
		active, err := s.subscription.IsActive(ctx, t.userID)
		if err != nil {
			log.Warn("ws subscription check for voice transcript", zap.Int64("user_id", t.userID), zap.Error(err))
			continue
		}
		if !active {
			continue
		}
		msg := append([]byte(nil), payload...)
		s.enqueueToSubscriber(t.sub, msg)
	}
}

func (s *ChatServer) PublishToUser(_ context.Context, userID int64, message []byte) {
	s.mu.RLock()
	var subs []*subscriber
	if userHub, ok := s.subscribersByUserID[userID]; ok {
		for sub := range userHub {
			subs = append(subs, sub)
		}
	}
	s.mu.RUnlock()

	for _, sub := range subs {
		payload := append([]byte(nil), message...)
		s.enqueueToSubscriber(sub, payload)
	}
}

func (s *ChatServer) SubscribeHandler(w http.ResponseWriter, r *http.Request) {
	reqCtx := r.Context()
	userID, ok := UserIDFromContext(reqCtx)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	reqID := "unknown_request_id"
	if id, ok := requestIDFromContext(reqCtx); ok {
		reqID = id
	}
	connID := uuid.NewString()
	connLog := s.logger.With(
		zap.String("request_id", reqID),
		zap.Int64("user_id", userID),
		zap.String("conn_id", connID),
	)
	reqCtx = loggerctx.With(reqCtx, connLog)

	sub := &subscriber{
		userID: userID,
		msgs:   make(chan []byte, s.subscriberMessageBuffer),
	}

	wsConn, err := websocket.Accept(w, r, &websocket.AcceptOptions{})
	if err != nil {
		connLog.Error("websocket accept", zap.Error(err))
		return
	}
	connLog.Info("websocket connected")

	sub.conn = wsConn

	var closeMu sync.Mutex
	var slowClosed bool
	sub.closeSlow = func() {
		closeMu.Lock()
		if slowClosed {
			closeMu.Unlock()
			return
		}
		slowClosed = true
		closeMu.Unlock()
		_ = wsConn.Close(websocket.StatusPolicyViolation, "connection too slow to keep up with messages")
	}

	s.addSubscriber(sub)
	defer func() {
		s.removeSubscriber(sub)
		s.notifyUserOffline(reqCtx, userID)
	}()

	s.notifyUserOnline(reqCtx, userID)

	defer func() {
		_ = wsConn.CloseNow()
	}()

	ctx, cancel := context.WithCancel(reqCtx)
	defer cancel()
	sub.cancel = cancel

	s.wg.Add(1)
	defer s.wg.Done()

	go s.readClientMessages(ctx, wsConn, userID, sub)
	go s.runWSPing(ctx, wsConn, sub)
	if s.onlineRepo != nil {
		go s.runPresenceRedisRefresh(ctx, userID, sub)
	}
	s.writeClientMessages(ctx, wsConn, sub)
}

func (s *ChatServer) runWSPing(ctx context.Context, wsConn *websocket.Conn, sub *subscriber) {
	log := loggerctx.From(ctx)
	t := time.NewTicker(wsPingInterval)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			pingCtx, pingCancel := context.WithTimeout(ctx, wsPingTimeout)
			err := wsConn.Ping(pingCtx)
			pingCancel()
			if err == nil {
				continue
			}
			if !errors.Is(err, context.Canceled) {
				log.Debug("ws ping failed", zap.Error(err))
			}
			if sub.cancel != nil {
				sub.cancel()
			}
			_ = wsConn.Close(websocket.StatusGoingAway, "ping timeout")
			return
		}
	}
}

func (s *ChatServer) Shutdown(ctx context.Context) error {
	s.mu.RLock()
	subs := make([]*subscriber, 0, len(s.subscribers))
	for sub := range s.subscribers {
		subs = append(subs, sub)
	}
	s.mu.RUnlock()

	for _, sub := range subs {
		if sub.cancel != nil {
			sub.cancel()
		}
		if sub.conn != nil {
			if frame, err := dtoWs.EncodeError(dtoWs.WsErrorPayload{
				Code:    dtoWs.ErrCodeServerShutdown,
				Message: dtoWs.ErrCodeServerShutdownMsg,
			}); err == nil {
				writeCtx, writeCancel := context.WithTimeout(ctx, 1*time.Second)
				_ = sub.conn.Write(writeCtx, websocket.MessageText, frame)
				writeCancel()
			}
			_ = sub.conn.Close(websocket.StatusGoingAway, "server shutdown")
		}
	}

	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *ChatServer) writeClientMessages(ctx context.Context, wsConn *websocket.Conn, sub *subscriber) {
	log := loggerctx.From(ctx)
	for {
		select {
		case msg, ok := <-sub.msgs:
			if !ok {
				return
			}
			ctxWithTimeout, timeoutCancel := context.WithTimeout(ctx, wsWriteTimeout)
			writeErr := wsConn.Write(ctxWithTimeout, websocket.MessageText, msg)
			timeoutCancel()
			if writeErr != nil {
				if errors.Is(writeErr, context.DeadlineExceeded) {
					log.Debug("ws write deadline", zap.Error(writeErr))
					return
				}
				if websocket.CloseStatus(writeErr) == websocket.StatusGoingAway ||
					websocket.CloseStatus(writeErr) == websocket.StatusNormalClosure {
					return
				}
				log.Warn("ws write", zap.Error(writeErr))
				return
			}
		case <-ctx.Done():
			return
		}
	}
}

func (s *ChatServer) readClientMessages(ctx context.Context, wsConn *websocket.Conn, userID int64, sub *subscriber) {
	defer func() {
		if sub.cancel != nil {
			sub.cancel()
		}
	}()
	log := loggerctx.From(ctx)
	for {
		frameType, data, err := wsConn.Read(ctx)
		if err != nil {
			if !errors.Is(err, context.Canceled) {
				log.Debug("ws read closed", zap.Error(err))
			}
			return
		}
		if frameType != websocket.MessageText {
			continue
		}

		var env dtoWs.WsRequest
		if err := json.Unmarshal(data, &env); err != nil || env.Type == "" {
			log.Debug("ws invalid envelope", zap.Error(err))
			s.sendErr(sub, dtoWs.WsErrorPayload{
				Code:    dtoWs.ErrCodeInvalidEnvelope,
				Message: dtoWs.ErrCodeInvalidEnvelopeMsg,
			})
			continue
		}

		switch env.Type {
		case dtoWs.PresencePing:
			s.handlePresencePing(ctx, userID, sub)
		case dtoWs.PresenceBackground:
			s.handlePresenceBackground(ctx, userID, sub)
		case dtoWs.PresenceForeground:
			s.handlePresenceForeground(ctx, userID, sub)
		case dtoWs.MessageSend:
			var req dto.RequestSendMessage
			if len(env.Payload) == 0 {
				s.sendErr(sub, dtoWs.WsErrorPayload{
					Code:    dtoWs.ErrCodeInvalidPayload,
					Message: dtoWs.ErrCodeInvalidPayloadMsg,
				})
				continue
			}
			if err := json.Unmarshal(env.Payload, &req); err != nil {
				s.sendErr(sub, dtoWs.WsErrorPayload{
					Code:    dtoWs.ErrCodeInvalidPayload,
					Message: dtoWs.ErrCodeInvalidPayloadMsg,
				})
				continue
			}
			if req.ChatID <= 0 {
				s.sendErr(sub, dtoWs.WsErrorPayload{
					Code:    dtoWs.ErrCodeInvalidPayload,
					Message: dtoWs.ErrCodeInvalidPayloadMsg,
				})
				continue
			}

			resp, err := s.messageService.SendMessage(ctx, userID, req.ChatID, &req)
			if err != nil {
				log.Warn("ws send message", zap.Int64("chat_id", req.ChatID), zap.Error(err))
				switch {
				case errors.Is(err, domain.ErrMessageEmpty):
					s.sendErr(sub, dtoWs.WsErrorPayload{
						Code:    dtoWs.ErrCodeEmptyText,
						Message: dtoWs.ErrCodeEmptyTextMsg,
					})
				case errors.Is(err, domain.ErrMessageTooLong):
					s.sendErr(sub, dtoWs.WsErrorPayload{
						Code:    dtoWs.ErrCodeMessageTooLong,
						Message: dtoWs.ErrCodeMessageTooLongMsg,
					})
				case errors.Is(err, domain.ErrMessageNotMember):
					s.sendErr(sub, dtoWs.WsErrorPayload{
						Code:    dtoWs.ErrCodeNotMemberOfChat,
						Message: dtoWs.ErrCodeNotMemberOfChatMsg,
					})
				case errors.Is(err, domain.ErrOnlyOwnerCanSendMessaage):
					s.sendErr(sub, dtoWs.WsErrorPayload{
						Code:    dtoWs.ErrCodeOnlyOwnerCanSendMessaage,
						Message: dtoWs.ErrCodeOnlyOwnerCanSendMessaageMsg,
					})
				default:
					s.sendErr(sub, dtoWs.WsErrorPayload{
						Code:    dtoWs.ErrCodeSendFailed,
						Message: dtoWs.ErrCodeSendFailedMsg,
					})
				}
				continue
			}

			s.publishMessageNewPerViewer(ctx, req.ChatID, resp)

		case dtoWs.MessageSendSticker:
			var req dto.RequestSendSticker
			if len(env.Payload) == 0 {
				s.sendErr(sub, dtoWs.WsErrorPayload{
					Code:    dtoWs.ErrCodeInvalidPayload,
					Message: dtoWs.ErrCodeInvalidPayloadMsg,
				})
				continue
			}
			if err := json.Unmarshal(env.Payload, &req); err != nil {
				s.sendErr(sub, dtoWs.WsErrorPayload{
					Code:    dtoWs.ErrCodeInvalidPayload,
					Message: dtoWs.ErrCodeInvalidPayloadMsg,
				})
				continue
			}
			if req.ChatID <= 0 || req.StickerID <= 0 {
				s.sendErr(sub, dtoWs.WsErrorPayload{
					Code:    dtoWs.ErrCodeInvalidPayload,
					Message: dtoWs.ErrCodeInvalidPayloadMsg,
				})
				continue
			}

			resp, err := s.messageService.SendSticker(ctx, userID, req.ChatID, &req)
			if err != nil {
				log.Warn("ws send sticker", zap.Int64("chat_id", req.ChatID), zap.Int64("sticker_id", req.StickerID), zap.Error(err))
				switch {
				case errors.Is(err, domain.ErrInvalidSticker):
					s.sendErr(sub, dtoWs.WsErrorPayload{
						Code:    dtoWs.ErrCodeInvalidSticker,
						Message: dtoWs.ErrCodeInvalidStickerMsg,
					})
				case errors.Is(err, domain.ErrStickerNotFound):
					s.sendErr(sub, dtoWs.WsErrorPayload{
						Code:    dtoWs.ErrCodeStickerNotFound,
						Message: dtoWs.ErrCodeStickerNotFoundMsg,
					})
				case errors.Is(err, domain.ErrMessageNotMember):
					s.sendErr(sub, dtoWs.WsErrorPayload{
						Code:    dtoWs.ErrCodeNotMemberOfChat,
						Message: dtoWs.ErrCodeNotMemberOfChatMsg,
					})
				case errors.Is(err, domain.ErrOnlyOwnerCanSendMessaage):
					s.sendErr(sub, dtoWs.WsErrorPayload{
						Code:    dtoWs.ErrCodeOnlyOwnerCanSendMessaage,
						Message: dtoWs.ErrCodeOnlyOwnerCanSendMessaageMsg,
					})
				default:
					s.sendErr(sub, dtoWs.WsErrorPayload{
						Code:    dtoWs.ErrCodeSendFailed,
						Message: dtoWs.ErrCodeSendFailedMsg,
					})
				}
				continue
			}

			s.publishMessageNewPerViewer(ctx, req.ChatID, resp)

		case dtoWs.MessageSendAttachments:
			var req dto.RequestSendMessageAttachments
			if len(env.Payload) == 0 {
				s.sendErr(sub, dtoWs.WsErrorPayload{
					Code:    dtoWs.ErrCodeInvalidPayload,
					Message: dtoWs.ErrCodeInvalidPayloadMsg,
				})
				continue
			}
			if err := json.Unmarshal(env.Payload, &req); err != nil {
				s.sendErr(sub, dtoWs.WsErrorPayload{
					Code:    dtoWs.ErrCodeInvalidPayload,
					Message: dtoWs.ErrCodeInvalidPayloadMsg,
				})
				continue
			}
			if req.ChatID <= 0 {
				s.sendErr(sub, dtoWs.WsErrorPayload{
					Code:    dtoWs.ErrCodeInvalidPayload,
					Message: dtoWs.ErrCodeInvalidPayloadMsg,
				})
				continue
			}

			resp, err := s.messageService.SendMessageWithAttachments(ctx, userID, req.ChatID, &req)
			if err != nil {
				log.Warn("ws send message attachments", zap.Int64("chat_id", req.ChatID), zap.Error(err))
				switch {
				case errors.Is(err, domain.ErrMessageEmpty):
					s.sendErr(sub, dtoWs.WsErrorPayload{
						Code:    dtoWs.ErrCodeEmptyText,
						Message: dtoWs.ErrCodeEmptyTextMsg,
					})
				case errors.Is(err, domain.ErrMessageTooLong):
					s.sendErr(sub, dtoWs.WsErrorPayload{
						Code:    dtoWs.ErrCodeMessageTooLong,
						Message: dtoWs.ErrCodeMessageTooLongMsg,
					})
				case errors.Is(err, domain.ErrMessageNotMember):
					s.sendErr(sub, dtoWs.WsErrorPayload{
						Code:    dtoWs.ErrCodeNotMemberOfChat,
						Message: dtoWs.ErrCodeNotMemberOfChatMsg,
					})
				case errors.Is(err, domain.ErrOnlyOwnerCanSendMessaage):
					s.sendErr(sub, dtoWs.WsErrorPayload{
						Code:    dtoWs.ErrCodeOnlyOwnerCanSendMessaage,
						Message: dtoWs.ErrCodeOnlyOwnerCanSendMessaageMsg,
					})
				case errors.Is(err, domain.ErrInvalidAttachment):
					s.sendErr(sub, dtoWs.WsErrorPayload{
						Code:    dtoWs.ErrCodeInvalidAttachment,
						Message: dtoWs.ErrCodeInvalidAttachmentMsg,
					})
				case errors.Is(err, domain.ErrAttachmentNotOwned):
					s.sendErr(sub, dtoWs.WsErrorPayload{
						Code:    dtoWs.ErrCodeAttachmentNotOwned,
						Message: dtoWs.ErrCodeAttachmentNotOwnedMsg,
					})
				case errors.Is(err, domain.ErrContactNotFound):
					s.sendErr(sub, dtoWs.WsErrorPayload{
						Code:    dtoWs.ErrCodeContactNotFound,
						Message: dtoWs.ErrCodeContactNotFoundMsg,
					})
				case errors.Is(err, domain.ErrTooManyAttachments):
					s.sendErr(sub, dtoWs.WsErrorPayload{
						Code:    dtoWs.ErrCodeTooManyAttachments,
						Message: dtoWs.ErrCodeTooManyAttachmentsMsg,
					})
				default:
					s.sendErr(sub, dtoWs.WsErrorPayload{
						Code:    dtoWs.ErrCodeSendFailed,
						Message: dtoWs.ErrCodeSendFailedMsg,
					})
				}
				continue
			}

			s.publishMessageNewPerViewer(ctx, req.ChatID, resp)

		case dtoWs.MessageRecv:
			var req dto.RequestGetMessages
			if len(env.Payload) == 0 {
				s.sendErr(sub, dtoWs.WsErrorPayload{
					Code:    dtoWs.ErrCodeInvalidPayload,
					Message: dtoWs.ErrCodeInvalidPayloadMsg,
				})
				continue
			}
			if err := json.Unmarshal(env.Payload, &req); err != nil {
				s.sendErr(sub, dtoWs.WsErrorPayload{
					Code:    dtoWs.ErrCodeInvalidPayload,
					Message: dtoWs.ErrCodeInvalidPayloadMsg,
				})
				continue
			}
			if req.ChatID <= 0 {
				s.sendErr(sub, dtoWs.WsErrorPayload{
					Code:    dtoWs.ErrCodeInvalidPayload,
					Message: dtoWs.ErrCodeInvalidPayloadMsg,
				})
				continue
			}

			resp, err := s.messageService.GetMessagesByChatId(ctx, userID, req.ChatID, &req)
			if err != nil {
				log.Warn("ws get messages", zap.Int64("chat_id", req.ChatID), zap.Error(err))
				if errors.Is(err, domain.ErrMessageNotMember) {
					s.sendErr(sub, dtoWs.WsErrorPayload{
						Code:    dtoWs.ErrCodeNotMemberOfChat,
						Message: dtoWs.ErrCodeNotMemberOfChatMsg,
					})
					continue
				}
				s.sendErr(sub, dtoWs.WsErrorPayload{
					Code:    dtoWs.ErrCodeInternal,
					Message: dtoWs.ErrCodeInternalMsg,
				})
				continue
			}

			out, err := dtoWs.EncodeMessageGet(resp)
			if err != nil {
				log.Error("ws encode message get", zap.Error(err))
				s.sendErr(sub, dtoWs.WsErrorPayload{
					Code:    dtoWs.ErrCodeInternal,
					Message: dtoWs.ErrCodeInternalMsg,
				})
				continue
			}

			s.enqueueToSubscriber(sub, out)

		case dtoWs.MessageTranscribeVoice:
			var req dto.RequestTranscribeVoice
			if len(env.Payload) == 0 {
				s.sendErr(sub, dtoWs.WsErrorPayload{
					Code:    dtoWs.ErrCodeInvalidPayload,
					Message: dtoWs.ErrCodeInvalidPayloadMsg,
				})
				continue
			}
			if err := json.Unmarshal(env.Payload, &req); err != nil {
				s.sendErr(sub, dtoWs.WsErrorPayload{
					Code:    dtoWs.ErrCodeInvalidPayload,
					Message: dtoWs.ErrCodeInvalidPayloadMsg,
				})
				continue
			}
			if req.ChatID <= 0 || req.MessageID <= 0 {
				s.sendErr(sub, dtoWs.WsErrorPayload{
					Code:    dtoWs.ErrCodeInvalidPayload,
					Message: dtoWs.ErrCodeInvalidPayloadMsg,
				})
				continue
			}

			resp, err := s.messageService.TranscribeVoiceMessage(ctx, userID, req.ChatID, req.MessageID)
			if err != nil {
				log.Warn("ws transcribe voice", zap.Int64("chat_id", req.ChatID), zap.Int64("message_id", req.MessageID), zap.Error(err))
				switch {
				case errors.Is(err, domain.ErrSubscriptionRequired):
					s.sendErr(sub, dtoWs.WsErrorPayload{
						Code:    dtoWs.ErrCodeSubscriptionRequired,
						Message: dtoWs.ErrCodeSubscriptionRequiredMsg,
					})
				case errors.Is(err, domain.ErrMessageNotMember):
					s.sendErr(sub, dtoWs.WsErrorPayload{
						Code:    dtoWs.ErrCodeNotMemberOfChat,
						Message: dtoWs.ErrCodeNotMemberOfChatMsg,
					})
				case errors.Is(err, domain.ErrNotVoiceMessage):
					s.sendErr(sub, dtoWs.WsErrorPayload{
						Code:    dtoWs.ErrCodeNotVoiceMessage,
						Message: dtoWs.ErrCodeNotVoiceMessageMsg,
					})
				case errors.Is(err, domain.ErrTranscriptionFailed):
					s.sendErr(sub, dtoWs.WsErrorPayload{
						Code:    dtoWs.ErrCodeTranscriptionFailed,
						Message: dtoWs.ErrCodeTranscriptionFailedMsg,
					})
				case errors.Is(err, domain.ErrNoMessage):
					s.sendErr(sub, dtoWs.WsErrorPayload{
						Code:    dtoWs.ErrCodeInvalidPayload,
						Message: dtoWs.ErrCodeInvalidPayloadMsg,
					})
				default:
					s.sendErr(sub, dtoWs.WsErrorPayload{
						Code:    dtoWs.ErrCodeInternal,
						Message: dtoWs.ErrCodeInternalMsg,
					})
				}
				continue
			}

			out, err := dtoWs.EncodeVoiceTranscript(resp)
			if err != nil {
				log.Error("ws encode voice transcript", zap.Error(err))
				s.sendErr(sub, dtoWs.WsErrorPayload{
					Code:    dtoWs.ErrCodeInternal,
					Message: dtoWs.ErrCodeInternalMsg,
				})
				continue
			}
			s.enqueueToSubscriber(sub, out)
			s.publishVoiceTranscriptToSubscribers(ctx, req.ChatID, userID, out)

		case dtoWs.MessageMarkRead:
			var req dto.RequestMarkRead
			if len(env.Payload) == 0 {
				s.sendErr(sub, dtoWs.WsErrorPayload{
					Code:    dtoWs.ErrCodeInvalidPayload,
					Message: dtoWs.ErrCodeInvalidPayloadMsg,
				})
				continue
			}
			if err := json.Unmarshal(env.Payload, &req); err != nil {
				s.sendErr(sub, dtoWs.WsErrorPayload{
					Code:    dtoWs.ErrCodeInvalidPayload,
					Message: dtoWs.ErrCodeInvalidPayloadMsg,
				})
				continue
			}
			if req.ChatID <= 0 || req.MessageID <= 0 {
				s.sendErr(sub, dtoWs.WsErrorPayload{
					Code:    dtoWs.ErrCodeInvalidPayload,
					Message: dtoWs.ErrCodeInvalidPayloadMsg,
				})
				continue
			}

			resp, err := s.messageService.MarkMessagesRead(ctx, userID, req.ChatID, &req)
			if err != nil {
				log.Warn("ws mark read", zap.Int64("chat_id", req.ChatID), zap.Error(err))
				switch {
				case errors.Is(err, domain.ErrMessageNotMember):
					s.sendErr(sub, dtoWs.WsErrorPayload{
						Code:    dtoWs.ErrCodeNotMemberOfChat,
						Message: dtoWs.ErrCodeNotMemberOfChatMsg,
					})
				case errors.Is(err, domain.ErrReadMessageInvalid):
					s.sendErr(sub, dtoWs.WsErrorPayload{
						Code:    dtoWs.ErrCodeReadMessageInvalid,
						Message: dtoWs.ErrCodeReadMessageInvalidMsg,
					})
				default:
					s.sendErr(sub, dtoWs.WsErrorPayload{
						Code:    dtoWs.ErrCodeInternal,
						Message: dtoWs.ErrCodeInternalMsg,
					})
				}
				continue
			}

			out, err := dtoWs.EncodeMessageRead(resp)
			if err != nil {
				log.Error("ws encode message read", zap.Error(err))
				s.sendErr(sub, dtoWs.WsErrorPayload{
					Code:    dtoWs.ErrCodeInternal,
					Message: dtoWs.ErrCodeInternalMsg,
				})
				continue
			}

			s.publishBytesToChatMembers(ctx, req.ChatID, out)

		case dtoWs.MessageEdit:
			var req dto.RequestEditMessage
			if len(env.Payload) == 0 {
				s.sendErr(sub, dtoWs.WsErrorPayload{
					Code:    dtoWs.ErrCodeInvalidPayload,
					Message: dtoWs.ErrCodeInvalidPayloadMsg,
				})
				continue
			}
			if err := json.Unmarshal(env.Payload, &req); err != nil {
				s.sendErr(sub, dtoWs.WsErrorPayload{
					Code:    dtoWs.ErrCodeInvalidPayload,
					Message: dtoWs.ErrCodeInvalidPayloadMsg,
				})
				continue
			}
			if req.ChatID <= 0 {
				s.sendErr(sub, dtoWs.WsErrorPayload{
					Code:    dtoWs.ErrCodeInvalidPayload,
					Message: dtoWs.ErrCodeInvalidPayloadMsg,
				})
				continue
			}
			if req.MessageID <= 0 {
				s.sendErr(sub, dtoWs.WsErrorPayload{
					Code:    dtoWs.ErrCodeInvalidPayload,
					Message: dtoWs.ErrCodeInvalidPayloadMsg,
				})
				continue
			}

			resp, err := s.messageService.EditMessage(ctx, userID, req.ChatID, &req)
			if err != nil {
				if errors.Is(err, domain.ErrMessageNotMember) {
					s.sendErr(sub, dtoWs.WsErrorPayload{
						Code:    dtoWs.ErrCodeNotMemberOfChat,
						Message: dtoWs.ErrCodeNotMemberOfChatMsg,
					})
					continue
				}
				s.sendErr(sub, dtoWs.WsErrorPayload{
					Code:    dtoWs.ErrCodeInternal,
					Message: dtoWs.ErrCodeInternalMsg,
				})
				continue
			}

			s.publishMessageEditPerViewer(ctx, req.ChatID, resp)

		case dtoWs.MessageDelete:
			var req dto.RequestDeleteMessage
			if len(env.Payload) == 0 {
				s.sendErr(sub, dtoWs.WsErrorPayload{
					Code:    dtoWs.ErrCodeInvalidPayload,
					Message: dtoWs.ErrCodeInvalidPayloadMsg,
				})
				continue
			}
			if err := json.Unmarshal(env.Payload, &req); err != nil {
				s.sendErr(sub, dtoWs.WsErrorPayload{
					Code:    dtoWs.ErrCodeInvalidPayload,
					Message: dtoWs.ErrCodeInvalidPayloadMsg,
				})
				continue
			}
			if req.ChatID <= 0 {
				s.sendErr(sub, dtoWs.WsErrorPayload{
					Code:    dtoWs.ErrCodeInvalidPayload,
					Message: dtoWs.ErrCodeInvalidPayloadMsg,
				})
				continue
			}
			if req.MessageID <= 0 {
				s.sendErr(sub, dtoWs.WsErrorPayload{
					Code:    dtoWs.ErrCodeInvalidPayload,
					Message: dtoWs.ErrCodeInvalidPayloadMsg,
				})
				continue
			}

			resp, err := s.messageService.DeleteMessage(ctx, userID, req.ChatID, &req)
			if err != nil {
				if errors.Is(err, domain.ErrMessageNotMember) {
					s.sendErr(sub, dtoWs.WsErrorPayload{
						Code:    dtoWs.ErrCodeNotMemberOfChat,
						Message: dtoWs.ErrCodeNotMemberOfChatMsg,
					})
					continue
				}
				s.sendErr(sub, dtoWs.WsErrorPayload{
					Code:    dtoWs.ErrCodeInternal,
					Message: dtoWs.ErrCodeInternalMsg,
				})
				continue
			}

			s.publishMessageDeletePerViewer(ctx, req.ChatID, resp)

		case dtoWs.PresenceTypingStart:
			s.handlePresenceTyping(ctx, userID, sub, env, true)

		case dtoWs.PresenceTypingStop:
			s.handlePresenceTyping(ctx, userID, sub, env, false)

		default:
			log.Debug("ws unknown message type", zap.String("type", string(env.Type)))
			s.sendErr(sub, dtoWs.WsErrorPayload{
				Code:    dtoWs.ErrCodeUnknownType,
				Message: dtoWs.ErrCodeUnknownTypeMsg,
			})
		}

		if env.Type != dtoWs.PresenceBackground {
			s.touchOnlineIfPresent(ctx, userID, sub)
		}
	}
}
