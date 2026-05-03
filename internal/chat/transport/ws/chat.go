package ws

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/google/uuid"
	"go.uber.org/zap"

	domain "github.com/go-park-mail-ru/2026_1_ASAP/internal/chat/domain/chat"
	dto "github.com/go-park-mail-ru/2026_1_ASAP/internal/chat/dto/message"
	dtoWs "github.com/go-park-mail-ru/2026_1_ASAP/internal/chat/dto/ws"
	chatdto "github.com/go-park-mail-ru/2026_1_ASAP/internal/chat/dto/chat"
	"github.com/go-park-mail-ru/2026_1_ASAP/pkg/loggerctx"
)

const wsWriteTimeout = 3 * time.Second

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
	GetMessagesByChatId(ctx context.Context, userID int64, chatID int64, req *dto.RequestGetMessages) (*dto.ResponseGetMessages, error)
	EditMessage(ctx context.Context, userID, chatID int64, req *dto.RequestEditMessage) (*dto.ResponseSendMessage, error)
	DeleteMessage(ctx context.Context, userID, chatID int64, req *dto.RequestDeleteMessage) (*dto.ResponseDeleteMessage, error)
}

type ChatServiceInterface interface {
	GetChatMemberIDs(ctx context.Context, chatID int64) ([]int64, error)
	GetChatByID(ctx context.Context, chatID, userID int64) (*chatdto.ChatInformationDTO, error)
}

type subscriber struct {
	conn      *websocket.Conn
	msgs      chan []byte
	cancel    context.CancelFunc
	closeSlow func()
	userID    int64
}

type ChatServer struct {
	messageService          MessagesServiceInterface
	chatService             ChatServiceInterface
	subscribers             map[*subscriber]struct{}
	subscribersByUserID     map[int64]map[*subscriber]struct{}
	logger                  *zap.Logger
	wg                      sync.WaitGroup
	subscriberMessageBuffer int
	mu                      sync.RWMutex
}

func NewChatServer(logger *zap.Logger, messageService MessagesServiceInterface, chatService ChatServiceInterface) *ChatServer {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &ChatServer{
		subscribers:             make(map[*subscriber]struct{}),
		subscribersByUserID:     make(map[int64]map[*subscriber]struct{}),
		logger:                  logger,
		messageService:          messageService,
		chatService:             chatService,
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

func (s *ChatServer) publishMessageNewToChatMembers(ctx context.Context, chatID int64, message []byte) {
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
	defer s.removeSubscriber(sub)

	defer func() {
		_ = wsConn.CloseNow()
	}()

	ctx, cancel := context.WithCancel(reqCtx)
	defer cancel()
	sub.cancel = cancel

	s.wg.Add(1)
	defer s.wg.Done()

	go s.readClientMessages(ctx, wsConn, userID, sub)
	s.writeClientMessages(ctx, wsConn, sub)
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

			out, err := dtoWs.EncodeMessageNew(resp)
			if err != nil {
				log.Error("ws encode message new", zap.Error(err))
				s.sendErr(sub, dtoWs.WsErrorPayload{
					Code:    dtoWs.ErrCodeInternal,
					Message: dtoWs.ErrCodeInternalMsg,
				})
				continue
			}

			s.publishMessageNewToChatMembers(ctx, req.ChatID, out)

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

			out, err := dtoWs.EncodeMessageEdit(resp)
			if err != nil {
				log.Error("ws encode message edit", zap.Error(err))
				s.sendErr(sub, dtoWs.WsErrorPayload{
					Code:    dtoWs.ErrCodeInternal,
					Message: dtoWs.ErrCodeInternalMsg,
				})
				continue
			}

			s.publishMessageNewToChatMembers(ctx, req.ChatID, out)

			chatInfo, err := s.chatService.GetChatByID(ctx, req.ChatID, userID)
			if err != nil {
				log.Warn("ws get chat after edit message", zap.Int64("chat_id", req.ChatID), zap.Error(err))
				continue
			}

			outChat, err := dtoWs.EncodeChatUpdated(chatInfo)
			if err != nil {
				log.Error("ws encode chat update", zap.Error(err))
				continue
			}

			s.publishMessageNewToChatMembers(ctx, req.ChatID, outChat)
		
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

			out, err := dtoWs.EncodeMessageDelete(resp)
			if err != nil {
				log.Error("ws encode message delete", zap.Error(err))
				s.sendErr(sub, dtoWs.WsErrorPayload{
					Code:    dtoWs.ErrCodeInternal,
					Message: dtoWs.ErrCodeInternalMsg,
				})
				continue
			}
			
			s.publishMessageNewToChatMembers(ctx, req.ChatID, out)

			chatInfo, err := s.chatService.GetChatByID(ctx, req.ChatID, userID)
			if err != nil {
				log.Warn("ws get chat after delete message", zap.Int64("chat_id", req.ChatID), zap.Error(err))
				continue
			}

			outChat, err := dtoWs.EncodeChatUpdated(chatInfo)
			if err != nil {
				log.Error("ws encode chat update", zap.Error(err))
				continue
			}

			s.publishMessageNewToChatMembers(ctx, req.ChatID, outChat)
		
		default:
			log.Debug("ws unknown message type", zap.String("type", string(env.Type)))
			s.sendErr(sub, dtoWs.WsErrorPayload{
				Code:    dtoWs.ErrCodeUnknownType,
				Message: dtoWs.ErrCodeUnknownTypeMsg,
			})
		}
	}
}
