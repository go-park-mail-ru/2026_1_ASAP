package ws

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
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
}

type subscriber struct {
	chatID    int64
	conn      *websocket.Conn
	msgs      chan []byte
	closeSlow func()
}

type ChatServer struct {
	subscriberMessageBuffer int

	subscribers         map[*subscriber]struct{}
	subscribersByChatId map[int64]map[*subscriber]struct{}
	mu                  sync.RWMutex

	messageService MessagesServiceInterface
	chatService    ChatServiceInterface
}

func NewChatServer(messageService MessagesServiceInterface, chatService ChatServiceInterface) *ChatServer {
	return &ChatServer{
		subscribers:             make(map[*subscriber]struct{}),
		subscribersByChatId:     make(map[int64]map[*subscriber]struct{}),
		messageService:          messageService,
		chatService:             chatService,
		subscriberMessageBuffer: 16,
	}
}

func (s *ChatServer) addSubscriber(sub *subscriber) {
	s.mu.Lock()
	s.subscribers[sub] = struct{}{}
	chatHub := s.subscribersByChatId[sub.chatID]
	if chatHub == nil {
		chatHub = make(map[*subscriber]struct{})
		s.subscribersByChatId[sub.chatID] = chatHub
	}
	chatHub[sub] = struct{}{}
	s.mu.Unlock()
}

func (s *ChatServer) removeSubscriber(sub *subscriber) {
	s.mu.Lock()
	delete(s.subscribers, sub)
	if chatHub := s.subscribersByChatId[sub.chatID]; chatHub != nil {
		delete(chatHub, sub)
		if len(chatHub) == 0 {
			delete(s.subscribersByChatId, sub.chatID)
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

func (s *ChatServer) publishMessageToChat(chatID int64, message []byte) {
	s.mu.RLock()
	chatHub := s.subscribersByChatId[chatID]
	if chatHub == nil || len(chatHub) == 0 {
		s.mu.RUnlock()
		return
	}
	subs := make([]*subscriber, 0, len(chatHub))
	for sub := range chatHub {
		subs = append(subs, sub)
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

	chatIDStr := r.URL.Query().Get("chatID")
	chatID, err := strconv.ParseInt(chatIDStr, 10, 64)
	if err != nil || chatID <= 0 {
		resp := dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{
				{
					Code:    dtoApi.InvalidID,
					Message: dtoApi.InvalidIDMsg,
				},
			},
		}
		response.Send(w, http.StatusBadRequest, resp)
		return
	}

	isMember, err := s.chatService.IsMember(reqCtx, userID, chatID)
	if err != nil {
		resp := dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{
				{
					Code:    dtoApi.InternalError,
					Message: dtoApi.InternalErrorMsg,
				},
			},
		}
		response.Send(w, http.StatusInternalServerError, resp)
		return
	}

	if !isMember {
		resp := dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{
				{
					Code:    dtoApi.NotMemberOfChat,
					Message: dtoApi.NotMemberOfChatMsg,
				},
			},
		}
		response.Send(w, http.StatusForbidden, resp)
		return
	}

	sub := &subscriber{
		chatID: chatID,
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

	go s.readClientMessages(ctx, cancel, wsConn, userID, chatID, sub)

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

func (s *ChatServer) readClientMessages(ctx context.Context, cancel context.CancelFunc, wsConn *websocket.Conn, userID, chatID int64, sub *subscriber) {
	defer cancel()
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

			resp, err := s.messageService.SendMessage(ctx, userID, chatID, &req)
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

			s.publishMessageToChat(chatID, out)
		default:
			s.sendErr(sub, dtoWs.WsErrorPayload{
				Code:    dtoWs.ErrCodeUnknownType,
				Message: dtoWs.ErrCodeUnknownTypeMsg,
			})
		}
	}
}
