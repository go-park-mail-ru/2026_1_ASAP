package ws

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/coder/websocket"
	domainChat "github.com/go-park-mail-ru/2026_1_ASAP/internal/domain/chat"
	dtoApi "github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/api"
	dto "github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/message"
	dtoWs "github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/ws"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/middleware"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/utils/response"
)

type MessagesServiceInterface interface {
	SendMessage(ctx context.Context, userID int64, chatID int64, req *dto.RequestSendMessage) (*dto.ResponseSendMessage, error)
	GetMessagesByChatId(ctx context.Context, userID int64, chatID int64, req *dto.RequestGetMessages) (*dto.ResponseGetMessages, error)
}

type ChatServiceInterface interface {
	IsMember(ctx context.Context, userID int64, chatID int64) (bool, error)
	GetChatMemberIDs(ctx context.Context, chatID int64) ([]int64, error)
}

type subscriber struct {
	userID    int64
	conn      *websocket.Conn
	msgs      chan []byte
	cancel    context.CancelFunc
	closeSlow func()
}

type ChatServer struct {
	subscriberMessageBuffer int

	subscribers         map[*subscriber]struct{}
	subscribersByUserID map[int64]map[*subscriber]struct{}
	mu                  sync.RWMutex
	wg                  sync.WaitGroup

	messageService MessagesServiceInterface
	chatService    ChatServiceInterface
}

func NewChatServer(messageService MessagesServiceInterface, chatService ChatServiceInterface) *ChatServer {
	return &ChatServer{
		subscribers:             make(map[*subscriber]struct{}),
		subscribersByUserID:     make(map[int64]map[*subscriber]struct{}),
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

// publishMessageNewToChatMembers шлёт message.New всем онлайн-участникам чата (по user id).
func (s *ChatServer) publishMessageNewToChatMembers(ctx context.Context, chatID int64, message []byte) {
	memberIDs, err := s.chatService.GetChatMemberIDs(ctx, chatID)
	if err != nil || len(memberIDs) == 0 {
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

func (s *ChatServer) SubscribeHandler(w http.ResponseWriter, r *http.Request) {
	reqCtx := r.Context()
	userID, ok := reqCtx.Value(middleware.UserID).(int64)
	if !ok {
		resp := dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{
				{
					Code:    dtoApi.Unauthorized,
					Message: dtoApi.UnauthorizedMsg,
				},
			},
		}
		response.Send(w, http.StatusUnauthorized, resp)
		return
	}

	sub := &subscriber{
		userID: userID,
		msgs:   make(chan []byte, s.subscriberMessageBuffer),
	}

	wsConn, err := websocket.Accept(w, r, &websocket.AcceptOptions{})
	if err != nil {
		return
	}

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

	defer wsConn.CloseNow()

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
	for {
		select {
		case msg, ok := <-sub.msgs:
			if !ok {
				return
			}
			ctxWithTimeout, timeoutCancel := context.WithTimeout(ctx, 1*time.Second)
			writeErr := wsConn.Write(ctxWithTimeout, websocket.MessageText, msg)
			timeoutCancel()
			if writeErr != nil {
				if errors.Is(writeErr, context.DeadlineExceeded) {
					return
				}
				if websocket.CloseStatus(writeErr) == websocket.StatusGoingAway ||
					websocket.CloseStatus(writeErr) == websocket.StatusNormalClosure {
					return
				}
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
	for {
		frameType, data, err := wsConn.Read(ctx)
		if err != nil {
			return
		}
		if frameType != websocket.MessageText {
			continue
		}

		var env dtoWs.WsRequest
		if err := json.Unmarshal(data, &env); err != nil || env.Type == "" {
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
				switch {
				case errors.Is(err, domainChat.ErrMessageEmpty):
					s.sendErr(sub, dtoWs.WsErrorPayload{
						Code:    dtoWs.ErrCodeEmptyText,
						Message: dtoWs.ErrCodeEmptyTextMsg,
					})
				case errors.Is(err, domainChat.ErrMessageTooLong):
					s.sendErr(sub, dtoWs.WsErrorPayload{
						Code:    dtoWs.ErrCodeMessageTooLong,
						Message: dtoWs.ErrCodeMessageTooLongMsg,
					})
				case errors.Is(err, domainChat.ErrMessageNotMember):
					s.sendErr(sub, dtoWs.WsErrorPayload{
						Code:    dtoWs.ErrCodeNotMemberOfChat,
						Message: dtoWs.ErrCodeNotMemberOfChatMsg,
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
				if errors.Is(err, domainChat.ErrMessageNotMember) {
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
				s.sendErr(sub, dtoWs.WsErrorPayload{
					Code:    dtoWs.ErrCodeInternal,
					Message: dtoWs.ErrCodeInternalMsg,
				})
				continue
			}
			s.enqueueToSubscriber(sub, out)
		default:
			s.sendErr(sub, dtoWs.WsErrorPayload{
				Code:    dtoWs.ErrCodeUnknownType,
				Message: dtoWs.ErrCodeUnknownTypeMsg,
			})
		}
	}
}
