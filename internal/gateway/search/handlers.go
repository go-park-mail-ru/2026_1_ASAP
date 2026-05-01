package search

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	searchv1 "github.com/go-park-mail-ru/2026_1_ASAP/gen/go/search/v1"
	dtoApi "github.com/go-park-mail-ru/2026_1_ASAP/internal/gateway/dto/api"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/gateway/middleware"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/utils/response"
	"github.com/go-park-mail-ru/2026_1_ASAP/pkg/grpcerr"
)

type GatewaySearchHandler struct {
	Search searchv1.SearchClient
}

func NewGatewaySearchHandler(client searchv1.SearchClient) *GatewaySearchHandler {
	return &GatewaySearchHandler{Search: client}
}

type searchListResponse[T any] struct {
	NextBeforeID int64 `json:"next_before_id"`
	Items        []T   `json:"items"`
}

type searchMessageItemJSON struct {
	MessageID   int64                `json:"message_id"`
	ChatID      int64                `json:"chat_id"`
	SenderID    int64                `json:"sender_id"`
	TextPreview string               `json:"text_preview"`
	CreatedAt   time.Time            `json:"created_at"`
	Highlights  *searchHighlightJSON `json:"highlights,omitempty"`
}

type searchHighlightJSON struct {
	Fragment string `json:"fragment"`
}

type searchUserItemJSON struct {
	UserID      int64      `json:"user_id"`
	DisplayName string     `json:"display_name"`
	IsOnline    bool       `json:"is_online"`
	Login       *string    `json:"login,omitempty"`
	AvatarURL   *string    `json:"avatar_url,omitempty"`
	LastSeenAt  *time.Time `json:"last_seen_at,omitempty"`
}

type searchChatItemJSON struct {
	ChatID             int64      `json:"chat_id"`
	Type               string     `json:"type"`
	Title              string     `json:"title"`
	AvatarURL          *string    `json:"avatar_url,omitempty"`
	LastMessagePreview *string    `json:"last_message_preview,omitempty"`
	LastMessageAt      *time.Time `json:"last_message_at,omitempty"`
	UnreadCount        int64      `json:"unread_count"`
}

func (h *GatewaySearchHandler) SearchMessages(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	uid, ok := ctx.Value(middleware.UserID).(int64)
	if !ok {
		sendUnauthorized(w)
		return
	}
	q := r.URL.Query().Get("q")
	chatID, err := strconv.ParseInt(r.URL.Query().Get("chat_id"), 10, 64)
	if err != nil || chatID <= 0 {
		sendBadRequest(w, "invalid chat_id")
		return
	}
	beforeID, _ := strconv.ParseInt(r.URL.Query().Get("before_id"), 10, 64)
	limit, _ := strconv.ParseInt(r.URL.Query().Get("limit"), 10, 32)

	resp, err := h.Search.SearchMessagesInChat(ctx, &searchv1.SearchMessagesInChatRequest{
		UserId:   uid,
		ChatId:   chatID,
		Query:    q,
		Limit:    int32(limit),
		BeforeId: beforeID,
	})
	if err != nil {
		sendSearchError(w, err)
		return
	}

	items := make([]searchMessageItemJSON, 0, len(resp.GetMessages()))
	for _, m := range resp.GetMessages() {
		item := searchMessageItemJSON{
			MessageID:   m.GetMessageId(),
			ChatID:      m.GetChatId(),
			SenderID:    m.GetSenderId(),
			TextPreview: m.GetTextPreview(),
		}
		if t := m.GetCreatedAt(); t != nil {
			item.CreatedAt = t.AsTime()
		}
		if hl := m.GetHighlights(); hl != nil && hl.GetFragment() != "" {
			item.Highlights = &searchHighlightJSON{Fragment: hl.GetFragment()}
		}
		items = append(items, item)
	}
	response.Send(w, http.StatusOK, dtoApi.ApiSuccessResponse[searchListResponse[searchMessageItemJSON]]{
		Status: dtoApi.Success,
		Body: searchListResponse[searchMessageItemJSON]{
			Items:        items,
			NextBeforeID: resp.GetNextBeforeId(),
		},
	})
}

func (h *GatewaySearchHandler) SearchUsers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	uid, ok := ctx.Value(middleware.UserID).(int64)
	if !ok {
		sendUnauthorized(w)
		return
	}
	q := r.URL.Query().Get("q")
	scope := mapSearchScope(r.URL.Query().Get("scope"))
	beforeID, _ := strconv.ParseInt(r.URL.Query().Get("before_id"), 10, 64)
	limit, _ := strconv.ParseInt(r.URL.Query().Get("limit"), 10, 32)

	resp, err := h.Search.SearchContacts(ctx, &searchv1.SearchContactsRequest{
		UserId:   uid,
		Query:    q,
		Scope:    scope,
		Limit:    int32(limit),
		BeforeId: beforeID,
	})
	if err != nil {
		sendSearchError(w, err)
		return
	}

	items := make([]searchUserItemJSON, 0, len(resp.GetContacts()))
	for _, c := range resp.GetContacts() {
		item := searchUserItemJSON{
			UserID:      c.GetUserId(),
			DisplayName: c.GetDisplayName(),
			IsOnline:    c.GetIsOnline(),
		}
		if c.Login != nil {
			ln := c.GetLogin()
			item.Login = &ln
		}
		if c.AvatarUrl != nil {
			av := c.GetAvatarUrl()
			item.AvatarURL = &av
		}
		if ts := c.GetLastSeenAt(); ts != nil {
			t := ts.AsTime()
			item.LastSeenAt = &t
		}
		items = append(items, item)
	}
	response.Send(w, http.StatusOK, dtoApi.ApiSuccessResponse[searchListResponse[searchUserItemJSON]]{
		Status: dtoApi.Success,
		Body: searchListResponse[searchUserItemJSON]{
			Items:        items,
			NextBeforeID: resp.GetNextBeforeId(),
		},
	})
}

func (h *GatewaySearchHandler) SearchChats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	uid, ok := ctx.Value(middleware.UserID).(int64)
	if !ok {
		sendUnauthorized(w)
		return
	}
	q := r.URL.Query().Get("q")
	kinds, err := parseChatKinds(r.URL.Query().Get("type"))
	if err != nil {
		sendBadRequest(w, err.Error())
		return
	}
	beforeID, _ := strconv.ParseInt(r.URL.Query().Get("before_id"), 10, 64)
	limit, _ := strconv.ParseInt(r.URL.Query().Get("limit"), 10, 32)

	resp, err := h.Search.SearchChats(ctx, &searchv1.SearchChatsRequest{
		UserId:   uid,
		Query:    q,
		Kinds:    kinds,
		Limit:    int32(limit),
		BeforeId: beforeID,
	})
	if err != nil {
		sendSearchError(w, err)
		return
	}

	items := make([]searchChatItemJSON, 0, len(resp.GetChats()))
	for _, c := range resp.GetChats() {
		item := searchChatItemJSON{
			ChatID:      c.GetChatId(),
			Type:        mapSearchChatType(c.GetType()),
			Title:       c.GetTitle(),
			UnreadCount: c.GetUnreadCount(),
		}
		if u := c.GetAvatarUrl(); u != "" {
			item.AvatarURL = &u
		}
		if p := c.GetLastMessagePreview(); p != "" {
			item.LastMessagePreview = &p
		}
		if ts := c.GetLastMessageAt(); ts != nil {
			t := ts.AsTime()
			item.LastMessageAt = &t
		}
		items = append(items, item)
	}
	response.Send(w, http.StatusOK, dtoApi.ApiSuccessResponse[searchListResponse[searchChatItemJSON]]{
		Status: dtoApi.Success,
		Body: searchListResponse[searchChatItemJSON]{
			Items:        items,
			NextBeforeID: resp.GetNextBeforeId(),
		},
	})
}

func sendUnauthorized(w http.ResponseWriter) {
	response.Send(w, http.StatusUnauthorized, dtoApi.ApiErrorResponse{
		Status: dtoApi.Error,
		Errors: []dtoApi.ApiError{{Code: dtoApi.Unauthorized, Message: dtoApi.UnauthorizedMsg}},
	})
}

func sendBadRequest(w http.ResponseWriter, msg string) {
	response.Send(w, http.StatusBadRequest, dtoApi.ApiErrorResponse{
		Status: dtoApi.Error,
		Errors: []dtoApi.ApiError{{Code: dtoApi.InvalidJson, Message: dtoApi.ErrorMessage(msg)}},
	})
}

func sendSearchError(w http.ResponseWriter, err error) {
	code, appCode, _ := grpcerr.Error(err)
	switch searchv1.SearchErrorCode(appCode) {
	case searchv1.SearchErrorCode_SEARCH_ERROR_INVALID_INPUT:
		response.Send(w, http.StatusBadRequest, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{Code: dtoApi.InvalidJson, Message: dtoApi.ErrorMessage(err.Error())}},
		})
	case searchv1.SearchErrorCode_SEARCH_ERROR_NOT_FOUND:
		response.Send(w, http.StatusNotFound, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{Code: dtoApi.NotFound, Message: dtoApi.NotFoundMsg}},
		})
	case searchv1.SearchErrorCode_SEARCH_ERROR_PERMISSION_DENIED:
		response.Send(w, http.StatusForbidden, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{Code: dtoApi.AccessDenied, Message: dtoApi.ErrorMessage(err.Error())}},
		})
	default:
		_ = code
		response.Send(w, http.StatusInternalServerError, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{Code: dtoApi.InternalError, Message: dtoApi.InternalErrorMsg}},
		})
	}
}

func mapSearchChatType(t searchv1.ChatType) string {
	switch t {
	case searchv1.ChatType_CHAT_TYPE_GROUP:
		return "group"
	case searchv1.ChatType_CHAT_TYPE_CHANNEL:
		return "channel"
	default:
		return "dialog"
	}
}

func parseChatKinds(raw string) ([]searchv1.SearchChatKind, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	parts := strings.Split(raw, ",")
	seen := make(map[searchv1.SearchChatKind]struct{}, len(parts))
	kinds := make([]searchv1.SearchChatKind, 0, len(parts))
	for _, p := range parts {
		switch strings.ToLower(strings.TrimSpace(p)) {
		case "dialog":
			seen[searchv1.SearchChatKind_SEARCH_CHAT_KIND_DIALOG] = struct{}{}
		case "group":
			seen[searchv1.SearchChatKind_SEARCH_CHAT_KIND_GROUP] = struct{}{}
		case "channel":
			seen[searchv1.SearchChatKind_SEARCH_CHAT_KIND_CHANNEL] = struct{}{}
		default:
			return nil, fmt.Errorf("invalid type filter: use dialog,group,channel")
		}
	}
	for k := range seen {
		kinds = append(kinds, k)
	}
	return kinds, nil
}

func mapSearchScope(scope string) searchv1.SearchContactScope {
	switch scope {
	case "contacts":
		return searchv1.SearchContactScope_SEARCH_CONTACT_SCOPE_CONTACTS
	default:
		return searchv1.SearchContactScope_SEARCH_CONTACT_SCOPE_LOCAL
	}
}
