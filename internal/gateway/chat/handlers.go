package chat

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	chatv1 "github.com/go-park-mail-ru/2026_1_ASAP/gen/go/chat/v1"
	dtoApi "github.com/go-park-mail-ru/2026_1_ASAP/internal/gateway/dto/api"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/gateway/middleware"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/profile/dto/media"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/utils/response"
	"github.com/go-park-mail-ru/2026_1_ASAP/pkg/grpcerr"
	"google.golang.org/protobuf/types/known/emptypb"
)

type GatewayChatHandler struct {
	ChatService chatv1.ChatClient
}

func NewGatewayChatHandler(chat chatv1.ChatClient) *GatewayChatHandler {
	return &GatewayChatHandler{ChatService: chat}
}

// DTO types

type MessageInfoResponse struct {
	CreatedAt time.Time `json:"created_at"`
	Text      string    `json:"text"`
	SenderID  int64     `json:"sender_id"`
}

type ChatInfoResponse struct {
	Avatar      *string             `json:"avatar"`
	Description *string             `json:"description"`
	Type        string              `json:"type"`
	Title       string              `json:"title"`
	LastMessage MessageInfoResponse `json:"last_message"`
	ID          int64               `json:"id"`
	OwnerID     int64               `json:"owner_id"`
}

type CreateChatRequest struct {
	Title     string  `json:"title"`
	Type      string  `json:"type"`
	MembersID []int64 `json:"members_id"`
}

type UpdateTitleRequest struct {
	Title string `json:"title"`
}

type UpdateDescriptionRequest struct {
	Description string `json:"description"`
}

type AddMembersRequest struct {
	MembersID []int64 `json:"members_id"`
}

type JoinChannelRequest struct {
	UserId int64 `json:"user_id"`
	ChatId int64 `json:"chat_id"`
}

type ChatMembersResponse struct {
	MembersID []int64 `json:"members_id"`
}

type StickerResponse struct {
	Slug    *string `json:"slug,omitempty"`
	Emoji   *string `json:"emoji,omitempty"`
	FileURL string  `json:"file_url"`
	ID      int64   `json:"id"`
	PackID  int64   `json:"pack_id"`
	Width   *int32  `json:"width,omitempty"`
	Height  *int32  `json:"height,omitempty"`
}

type StickerPackResponse struct {
	Slug         *string           `json:"slug,omitempty"`
	ThumbnailURL *string           `json:"thumbnail_url,omitempty"`
	Name         string            `json:"name"`
	Title        string            `json:"title"`
	ID           int64             `json:"id"`
	Stickers     []StickerResponse `json:"stickers"`
}

type StickerPacksResponse struct {
	Packs []StickerPackResponse `json:"packs"`
}

// helpers

func userID(r *http.Request) (int64, bool) {
	uid, ok := r.Context().Value(middleware.UserID).(int64)
	return uid, ok
}

func chatIDParam(r *http.Request) (int64, error) {
	return strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
}

func sendUnauthorized(w http.ResponseWriter) {
	response.Send(w, http.StatusUnauthorized, dtoApi.ApiErrorResponse{
		Status: dtoApi.Error,
		Errors: []dtoApi.ApiError{{Code: dtoApi.Unauthorized, Message: dtoApi.UnauthorizedMsg}},
	})
}

func sendInvalidID(w http.ResponseWriter) {
	response.Send(w, http.StatusBadRequest, dtoApi.ApiErrorResponse{
		Status: dtoApi.Error,
		Errors: []dtoApi.ApiError{{Code: dtoApi.InvalidID, Message: dtoApi.InvalidIDMsg}},
	})
}

func sendChatError(w http.ResponseWriter, err error) {
	_, appCode, _ := grpcerr.Error(err)
	switch chatv1.ChatErrorCode(appCode) {
	case chatv1.ChatErrorCode_CHAT_ERROR_NOT_FOUND,
		chatv1.ChatErrorCode_CHAT_ERROR_CHAT_NOT_FOUND:
		response.Send(w, http.StatusNotFound, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{Code: dtoApi.NotFound, Message: dtoApi.CantFindChatMsg}},
		})
	case chatv1.ChatErrorCode_CHAT_ERROR_USER_NOT_FOUND:
		response.Send(w, http.StatusNotFound, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{Code: dtoApi.UserNotFound, Message: dtoApi.UserNotFoundMsg}},
		})
	case chatv1.ChatErrorCode_CHAT_ERROR_USER_NOT_MEMBER,
		chatv1.ChatErrorCode_CHAT_ERROR_NOT_MEMBER:
		response.Send(w, http.StatusForbidden, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{Code: dtoApi.NotMemberOfChat, Message: dtoApi.NotMemberOfChatMsg}},
		})
	case chatv1.ChatErrorCode_CHAT_ERROR_ONLY_OWNER_CAN_DELETE_CHAT:
		response.Send(w, http.StatusForbidden, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{Code: dtoApi.CantDeleteChat, Message: dtoApi.CantDeleteChatMsg}},
		})
	case chatv1.ChatErrorCode_CHAT_ERROR_DIALOG_EXISTS:
		response.Send(w, http.StatusConflict, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{Code: dtoApi.DialogAlreadyExists, Message: dtoApi.DialogAlreadyExistsMsg}},
		})
	case chatv1.ChatErrorCode_CHAT_ERROR_DIALOG_CANT_HAVE_CUSTOM_TITLE:
		response.Send(w, http.StatusBadRequest, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{Code: dtoApi.CantChangeTitle, Message: dtoApi.CantChangeTitleMsg}},
		})
	case chatv1.ChatErrorCode_CHAT_ERROR_DIALOG_CANT_HAVE_CUSTOM_AVATAR:
		response.Send(w, http.StatusBadRequest, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{Code: dtoApi.CantChangeAvatar, Message: dtoApi.CantChangeAvatarMsg}},
		})
	case chatv1.ChatErrorCode_CHAT_ERROR_CANT_ADD_MEMBER_TO_DIALOG:
		response.Send(w, http.StatusBadRequest, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{Code: dtoApi.CantAddMemberToDialog, Message: dtoApi.CantAddMemberToDialogMsg}},
		})
	case chatv1.ChatErrorCode_CHAT_ERROR_ONLY_OWNER_CAN_ADD_PEOPLE:
		response.Send(w, http.StatusForbidden, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{Code: dtoApi.OnlyOwnerCanAddMembers, Message: dtoApi.OnlyOwnerCanAddMembersMsg}},
		})
	case chatv1.ChatErrorCode_CHAT_ERROR_MEMBER_ALREADY_IN_CHAT:
		response.Send(w, http.StatusConflict, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{Code: dtoApi.MemberAlreadyInChat, Message: dtoApi.MemberAlreadyInChatMsg}},
		})
	case chatv1.ChatErrorCode_CHAT_ERROR_CANT_DELETE_MEMBER_FROM_DIALOG:
		response.Send(w, http.StatusBadRequest, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{Code: dtoApi.DeleteMemberFromDialog, Message: dtoApi.DeleteMemberFromDialogMsg}},
		})
	case chatv1.ChatErrorCode_CHAT_ERROR_ONLY_OWNER_CAN_DELETE_PEOPLE:
		response.Send(w, http.StatusForbidden, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{Code: dtoApi.OnlyOwnerCanDeleteMember, Message: dtoApi.OnlyOwnerCanDeleteMemberMsg}},
		})
	case chatv1.ChatErrorCode_CHAT_ERROR_CANT_DELETE_OWNER_OF_CHAT:
		response.Send(w, http.StatusBadRequest, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{Code: dtoApi.DeleteOwnerOfChat, Message: dtoApi.DeleteOwnerOfChatMsg}},
		})
	case chatv1.ChatErrorCode_CHAT_ERROR_MEMBER_NOT_FOUND:
		response.Send(w, http.StatusNotFound, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{Code: dtoApi.MemberNotFound, Message: dtoApi.MemberNotFoundMsg}},
		})
	case chatv1.ChatErrorCode_CHAT_ERROR_CANT_QUIT_DIALOG:
		response.Send(w, http.StatusBadRequest, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{Code: dtoApi.CantQuitDialog, Message: dtoApi.CantQuitDialogMsg}},
		})
	case chatv1.ChatErrorCode_CHAT_ERROR_OWNER_CANT_QUIT_GROUP:
		response.Send(w, http.StatusBadRequest, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{Code: dtoApi.OwnerCantQuitChat, Message: dtoApi.OwnerCantQuitChatMsg}},
		})
	case chatv1.ChatErrorCode_CHAT_ERROR_INVALID_INPUT:
		response.Send(w, http.StatusBadRequest, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{Code: dtoApi.InvalidJson, Message: dtoApi.InvalidJsonMsg}},
		})
	case chatv1.ChatErrorCode_CHAT_ERROR_ONLY_CHANNEL_CAN_BE_JOINED:
		response.Send(w, http.StatusBadRequest, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{Code: dtoApi.OnlyChannelCanBeJoined, Message: dtoApi.OnlyChannelCanBeJoinedMsg}},
		})
	case chatv1.ChatErrorCode_CHAT_ERROR_DIALOG_CANT_HAVE_CUSTOM_DESCRIPTION:
		response.Send(w, http.StatusBadRequest, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{Code: dtoApi.CantHaveCustomDescription, Message: dtoApi.CantHaveCustomDescriptionMsg}},
		})
	case chatv1.ChatErrorCode_CHAT_ERROR_ONLY_OWNER_CAN_CHANGE_DESCRIPTION:
		response.Send(w, http.StatusBadRequest, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{Code: dtoApi.OnlyOwnerCanChangeDescription, Message: dtoApi.OnlyOwnerCanChangeDescriptionMsg}},
		})
	default:
		response.Send(w, http.StatusInternalServerError, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{Code: dtoApi.InternalError, Message: dtoApi.InternalErrorMsg}},
		})
	}
}

func mapChatType(t chatv1.ChatType) string {
	switch t {
	case chatv1.ChatType_GROUP:
		return "group"
	case chatv1.ChatType_CHANNEL:
		return "channel"
	default:
		return "dialog"
	}
}

func parseChatType(s string) (chatv1.ChatType, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "group":
		return chatv1.ChatType_GROUP, nil
	case "channel":
		return chatv1.ChatType_CHANNEL, nil
	case "dialog":
		return chatv1.ChatType_DIALOG, nil
	default:
		return chatv1.ChatType_DIALOG, fmt.Errorf("invalid chat type: use dialog, group or channel")
	}
}

func mapChatInfo(c *chatv1.ChatInformation) ChatInfoResponse {
	out := ChatInfoResponse{
		ID:      c.GetId(),
		Type:    mapChatType(c.GetType()),
		Title:   c.GetTitle(),
		OwnerID: c.GetOwnerId(),
	}
	if a := c.GetAvatar(); a != "" {
		v := a
		out.Avatar = &v
	}
	if d := c.GetDescription(); d != "" {
		v := d
		out.Description = &v
	}
	if lm := c.GetLastMessage(); lm != nil {
		out.LastMessage = MessageInfoResponse{
			SenderID:  lm.GetSenderId(),
			Text:      lm.GetText(),
			CreatedAt: lm.GetCreatedAt().AsTime(),
		}
	}
	return out
}

func mapStickerPacks(resp *chatv1.ResponseGetStickerPacks) StickerPacksResponse {
	if resp == nil {
		return StickerPacksResponse{}
	}
	packs := make([]StickerPackResponse, 0, len(resp.GetPacks()))
	for _, pack := range resp.GetPacks() {
		item := StickerPackResponse{
			ID:       pack.GetId(),
			Name:     pack.GetName(),
			Title:    pack.GetTitle(),
			Stickers: make([]StickerResponse, 0, len(pack.GetStickers())),
		}
		if slug := pack.GetSlug(); slug != "" {
			item.Slug = &slug
		}
		if thumbnail := pack.GetThumbnailUrl(); thumbnail != "" {
			item.ThumbnailURL = &thumbnail
		}
		for _, sticker := range pack.GetStickers() {
			stickerItem := StickerResponse{
				ID:      sticker.GetId(),
				PackID:  sticker.GetPackId(),
				FileURL: sticker.GetFileUrl(),
			}
			if slug := sticker.GetSlug(); slug != "" {
				stickerItem.Slug = &slug
			}
			if emoji := sticker.GetEmoji(); emoji != "" {
				stickerItem.Emoji = &emoji
			}
			if sticker.Width != nil {
				width := sticker.GetWidth()
				stickerItem.Width = &width
			}
			if sticker.Height != nil {
				height := sticker.GetHeight()
				stickerItem.Height = &height
			}
			item.Stickers = append(item.Stickers, stickerItem)
		}
		packs = append(packs, item)
	}
	return StickerPacksResponse{Packs: packs}
}

// Handlers

func (h *GatewayChatHandler) GetChats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	uid, ok := userID(r)
	if !ok {
		sendUnauthorized(w)
		return
	}

	resp, err := h.ChatService.GetChats(ctx, &chatv1.RequestGetUserChats{UserId: uid})
	if err != nil {
		sendChatError(w, err)
		return
	}

	chats := make([]ChatInfoResponse, 0, len(resp.GetChats()))
	for _, c := range resp.GetChats() {
		chats = append(chats, mapChatInfo(c))
	}
	response.Send(w, http.StatusOK, dtoApi.ApiSuccessResponse[[]ChatInfoResponse]{
		Status: dtoApi.Success,
		Body:   chats,
	})
}

func (h *GatewayChatHandler) CreateChat(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	ctx := r.Context()
	uid, ok := userID(r)
	if !ok {
		sendUnauthorized(w)
		return
	}

	var req CreateChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Send(w, http.StatusBadRequest, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{Code: dtoApi.InvalidJson, Message: dtoApi.InvalidJsonMsg}},
		})
		return
	}

	chatType, err := parseChatType(req.Type)
	if err != nil {
		response.Send(w, http.StatusBadRequest, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{Code: dtoApi.InvalidJson, Message: dtoApi.ErrorMessage(err.Error())}},
		})
		return
	}
	if (chatType == chatv1.ChatType_GROUP || chatType == chatv1.ChatType_CHANNEL) && strings.TrimSpace(req.Title) == "" {
		response.Send(w, http.StatusBadRequest, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{Code: dtoApi.InvalidJson, Message: dtoApi.ErrorMessage("title is required for group/channel")}},
		})
		return
	}

	resp, err := h.ChatService.Create(ctx, &chatv1.RequestChatCreate{
		UserId:    uid,
		Type:      chatType,
		Title:     req.Title,
		MembersId: req.MembersID,
	})
	if err != nil {
		sendChatError(w, err)
		return
	}

	response.Send(w, http.StatusOK, dtoApi.ApiSuccessResponse[ChatInfoResponse]{
		Status: dtoApi.Success,
		Body:   mapChatInfo(resp),
	})
}

func (h *GatewayChatHandler) GetChatByID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	uid, ok := userID(r)
	if !ok {
		sendUnauthorized(w)
		return
	}

	chatID, err := chatIDParam(r)
	if err != nil {
		sendInvalidID(w)
		return
	}

	resp, err := h.ChatService.GetChatByID(ctx, &chatv1.RequestGetChatByID{UserId: uid, ChatId: chatID})
	if err != nil {
		sendChatError(w, err)
		return
	}

	response.Send(w, http.StatusOK, dtoApi.ApiSuccessResponse[ChatInfoResponse]{
		Status: dtoApi.Success,
		Body:   mapChatInfo(resp),
	})
}

func (h *GatewayChatHandler) DeleteChat(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	uid, ok := userID(r)
	if !ok {
		sendUnauthorized(w)
		return
	}

	chatID, err := chatIDParam(r)
	if err != nil {
		sendInvalidID(w)
		return
	}

	if _, err := h.ChatService.DeleteChat(ctx, &chatv1.RequestDeleteChat{UserId: uid, ChatId: chatID}); err != nil {
		sendChatError(w, err)
		return
	}

	response.Send(w, http.StatusOK, dtoApi.ApiSuccessResponse[struct{}]{
		Status: dtoApi.Success,
		Body:   struct{}{},
	})
}

func (h *GatewayChatHandler) UpdateChatAvatar(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	uid, ok := userID(r)
	if !ok {
		sendUnauthorized(w)
		return
	}

	chatID, err := chatIDParam(r)
	if err != nil {
		sendInvalidID(w)
		return
	}

	file, header, err := r.FormFile("avatar")
	if err != nil {
		response.Send(w, http.StatusBadRequest, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{Code: dtoApi.FileNotFound, Message: dtoApi.FileNotFoundMsg}},
		})
		return
	}
	fileInput, err := media.FileInputFromMultipart(file, header)
	if err != nil {
		if errors.Is(err, media.ErrEmptyFile) {
			response.Send(w, http.StatusBadRequest, dtoApi.ApiErrorResponse{
				Status: dtoApi.Error,
				Errors: []dtoApi.ApiError{{Code: dtoApi.EmptyFile, Message: dtoApi.EmptyFileMsg}},
			})
			return
		}
		if errors.Is(err, media.ErrFileTooLarge) {
			response.Send(w, http.StatusBadRequest, dtoApi.ApiErrorResponse{
				Status: dtoApi.Error,
				Errors: []dtoApi.ApiError{{Code: dtoApi.FileTooLarge, Message: dtoApi.FileTooLargeMsg}},
			})
			return
		}
		response.Send(w, http.StatusInternalServerError, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{Code: dtoApi.InternalError, Message: dtoApi.InternalErrorMsg}},
		})
		return
	}

	content, err := io.ReadAll(fileInput.Body)
	if err != nil {
		response.Send(w, http.StatusInternalServerError, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{Code: dtoApi.InternalError, Message: dtoApi.InternalErrorMsg}},
		})
		return
	}

	resp, err := h.ChatService.UpdateChatAvatar(ctx, &chatv1.RequestUpdateAvatar{
		UserId:  uid,
		ChatId:  chatID,
		Content: content,
		Type:    fileInput.ContentType,
	})
	if err != nil {
		sendChatError(w, err)
		return
	}

	response.Send(w, http.StatusOK, dtoApi.ApiSuccessResponse[ChatInfoResponse]{
		Status: dtoApi.Success,
		Body:   mapChatInfo(resp),
	})
}

func (h *GatewayChatHandler) UpdateChatTitle(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	ctx := r.Context()
	uid, ok := userID(r)
	if !ok {
		sendUnauthorized(w)
		return
	}

	chatID, err := chatIDParam(r)
	if err != nil {
		sendInvalidID(w)
		return
	}

	var req UpdateTitleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Send(w, http.StatusBadRequest, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{Code: dtoApi.InvalidJson, Message: dtoApi.InvalidJsonMsg}},
		})
		return
	}

	resp, err := h.ChatService.UpdateChatTitle(ctx, &chatv1.RequestUpdateTitle{
		UserId: uid,
		ChatId: chatID,
		Title:  req.Title,
	})
	if err != nil {
		sendChatError(w, err)
		return
	}

	response.Send(w, http.StatusOK, dtoApi.ApiSuccessResponse[ChatInfoResponse]{
		Status: dtoApi.Success,
		Body:   mapChatInfo(resp),
	})
}

func (h *GatewayChatHandler) UpdateChatDescription(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	ctx := r.Context()
	uid, ok := userID(r)
	if !ok {
		sendUnauthorized(w)
		return
	}

	chatID, err := chatIDParam(r)
	if err != nil {
		sendInvalidID(w)
		return
	}

	var req UpdateDescriptionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Send(w, http.StatusBadRequest, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{Code: dtoApi.InvalidJson, Message: dtoApi.InvalidJsonMsg}},
		})
		return
	}

	resp, err := h.ChatService.UpdateChatDescription(ctx, &chatv1.RequestUpdateDescription{
		UserId:      uid,
		ChatId:      chatID,
		Description: req.Description,
	})
	if err != nil {
		sendChatError(w, err)
		return
	}

	response.Send(w, http.StatusOK, dtoApi.ApiSuccessResponse[ChatInfoResponse]{
		Status: dtoApi.Success,
		Body:   mapChatInfo(resp),
	})
}

func (h *GatewayChatHandler) AddMembers(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	ctx := r.Context()
	uid, ok := userID(r)
	if !ok {
		sendUnauthorized(w)
		return
	}

	chatID, err := chatIDParam(r)
	if err != nil {
		sendInvalidID(w)
		return
	}

	var req AddMembersRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Send(w, http.StatusBadRequest, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{Code: dtoApi.InvalidJson, Message: dtoApi.InvalidJsonMsg}},
		})
		return
	}

	if _, err := h.ChatService.AddMembersToChat(ctx, &chatv1.RequestAddMembersToChat{
		UserId:    uid,
		ChatId:    chatID,
		MembersId: req.MembersID,
	}); err != nil {
		sendChatError(w, err)
		return
	}

	response.Send(w, http.StatusOK, dtoApi.ApiSuccessResponse[struct{}]{
		Status: dtoApi.Success,
		Body:   struct{}{},
	})
}

func (h *GatewayChatHandler) JoinChannel(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	uid, ok := userID(r)
	if !ok {
		sendUnauthorized(w)
		return
	}

	chatID, err := chatIDParam(r)
	if err != nil {
		sendInvalidID(w)
		return
	}

	if _, err := h.ChatService.JoinChannel(ctx, &chatv1.RequestJoinChannel{
		UserId: uid,
		ChatId: chatID,
	}); err != nil {
		sendChatError(w, err)
		return
	}

	response.Send(w, http.StatusOK, dtoApi.ApiSuccessResponse[struct{}]{
		Status: dtoApi.Success,
		Body:   struct{}{},
	})
}

func (h *GatewayChatHandler) DeleteMember(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	uid, ok := userID(r)
	if !ok {
		sendUnauthorized(w)
		return
	}

	chatID, err := chatIDParam(r)
	if err != nil {
		sendInvalidID(w)
		return
	}

	memberID, err := strconv.ParseInt(chi.URLParam(r, "member_id"), 10, 64)
	if err != nil {
		sendInvalidID(w)
		return
	}

	if _, err := h.ChatService.DeleteMemberFromChat(ctx, &chatv1.RequestDeleteMemberFromChat{
		UserId:   uid,
		ChatId:   chatID,
		MemberId: memberID,
	}); err != nil {
		sendChatError(w, err)
		return
	}

	response.Send(w, http.StatusOK, dtoApi.ApiSuccessResponse[struct{}]{
		Status: dtoApi.Success,
		Body:   struct{}{},
	})
}

func (h *GatewayChatHandler) GetChatMembers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	uid, ok := userID(r)
	if !ok {
		sendUnauthorized(w)
		return
	}

	chatID, err := chatIDParam(r)
	if err != nil {
		sendInvalidID(w)
		return
	}

	resp, err := h.ChatService.GetChatMembers(ctx, &chatv1.RequestChatMembers{UserId: uid, ChatId: chatID})
	if err != nil {
		sendChatError(w, err)
		return
	}

	response.Send(w, http.StatusOK, dtoApi.ApiSuccessResponse[ChatMembersResponse]{
		Status: dtoApi.Success,
		Body:   ChatMembersResponse{MembersID: resp.GetMembersId()},
	})
}

func (h *GatewayChatHandler) QuitChat(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	uid, ok := userID(r)
	if !ok {
		sendUnauthorized(w)
		return
	}

	chatID, err := chatIDParam(r)
	if err != nil {
		sendInvalidID(w)
		return
	}

	if _, err := h.ChatService.QuitChat(ctx, &chatv1.RequestQuitChat{UserId: uid, ChatId: chatID}); err != nil {
		sendChatError(w, err)
		return
	}

	response.Send(w, http.StatusOK, dtoApi.ApiSuccessResponse[struct{}]{
		Status: dtoApi.Success,
		Body:   struct{}{},
	})
}

func (h *GatewayChatHandler) GetStickerPacks(w http.ResponseWriter, r *http.Request) {
	resp, err := h.ChatService.GetStickerPacks(r.Context(), &emptypb.Empty{})
	if err != nil {
		sendChatError(w, err)
		return
	}

	response.Send(w, http.StatusOK, dtoApi.ApiSuccessResponse[StickerPacksResponse]{
		Status: dtoApi.Success,
		Body:   mapStickerPacks(resp),
	})
}
