package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi"
	dtoApi "github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/api"
	dto "github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/chat"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/middleware"
	chatService "github.com/go-park-mail-ru/2026_1_ASAP/internal/services/chat"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/utils/mapper"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/utils/response"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/utils/validation"
	"github.com/google/uuid"
)

type ChatsHandler struct {
	chatService chatService.ChatServiceInterface
}

func NewChatHandler(chatService chatService.ChatServiceInterface) *ChatsHandler {
	return &ChatsHandler{
		chatService: chatService,
	}
}

func (h *ChatsHandler) GetChats(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserID).(uuid.UUID)
	if !ok {
		resp := dtoApi.ApiErrorResponse{
			Status: dtoApi.ERROR,
			Errors: []dtoApi.ApiError{
				{
					Code:    "UNAUTHORIZED",
					Message: "User not authorized",
				},
			},
		}
		response.Send(w, http.StatusUnauthorized, resp)
		return
	}

	chats, err := h.chatService.GetAllChats(userID)
	if err != nil {
		resp := dtoApi.ApiErrorResponse{
			Status: dtoApi.ERROR,
			Errors: []dtoApi.ApiError{
				{
					Code:    "INTERNAL",
					Message: "Cant get all chats",
				},
			},
		}
		response.Send(w, http.StatusInternalServerError, resp)
		return
	}

	resp := dtoApi.ApiSucessResponse[[]dto.ChatInformationDTO]{
		Status: dtoApi.SUCCESS,
		Body:   chats,
	}
	response.Send(w, http.StatusOK, resp)
}

func (h *ChatsHandler) ChatCreate(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	userID, ok := r.Context().Value(middleware.UserID).(uuid.UUID)
	if !ok {
		resp := dtoApi.ApiErrorResponse{
			Status: dtoApi.ERROR,
			Errors: []dtoApi.ApiError{
				{
					Code:    "UNAUTHORIZED",
					Message: "User not authorized",
				},
			},
		}
		response.Send(w, http.StatusUnauthorized, resp)
		return
	}

	decoder := json.NewDecoder(r.Body)
	var req dto.ChatCreate
	err := decoder.Decode(&req)
	if err != nil {
		resp := dtoApi.ApiErrorResponse{
			Status: dtoApi.ERROR,
			Errors: []dtoApi.ApiError{
				{
					Code:    "INVALID_JSON",
					Message: "Invalid 1request body",
				},
			},
		}
		response.Send(w, http.StatusBadRequest, resp)
		return
	}

	errs := validation.ValidationChatCreate(&req)
	if errs != nil {
		apiErrors := mapper.MapValidationErrorsToApiErrors(errs)
		resp := dtoApi.ApiErrorResponse{
			Status: dtoApi.ERROR,
			Errors: apiErrors,
		}

		response.Send(w, http.StatusBadRequest, resp)
		return
	}

	creater := false
	for _, id := range req.MembersID {
		if id == userID {
			creater = true
			break
		}
	}

	if !creater {
		req.MembersID = append(req.MembersID, userID)
	}

	createdChat, err := h.chatService.CreateChat(req)
	if err != nil {
		resp := dtoApi.ApiErrorResponse{
			Status: dtoApi.ERROR,
			Errors: []dtoApi.ApiError{
				{
					Code:    "CREATE_FAILED",
					Message: "Failed to create chat",
				},
			},
		}
		response.Send(w, http.StatusInternalServerError, resp)
		return
	}

	resp := dtoApi.ApiSucessResponse[*dto.ChatInformationDTO]{
		Status: dtoApi.SUCCESS,
		Body:   createdChat,
	}
	response.Send(w, http.StatusCreated, resp)
}

func (h *ChatsHandler) GetChatByID(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserID).(uuid.UUID)
	if !ok {
		resp := dtoApi.ApiErrorResponse{
			Status: dtoApi.ERROR,
			Errors: []dtoApi.ApiError{
				{
					Code:    "UNAUTHORIZED",
					Message: "User not authorized",
				},
			},
		}
		response.Send(w, http.StatusUnauthorized, resp)
		return
	}

	chatIDurl := chi.URLParam(r, "id")
	chatID, err := uuid.Parse(chatIDurl)
	if err != nil {
		resp := dtoApi.ApiErrorResponse{
			Status: dtoApi.ERROR,
			Errors: []dtoApi.ApiError{
				{
					Code:    "INVALID_ID",
					Message: "Invalid chat id",
				},
			},
		}
		response.Send(w, http.StatusUnauthorized, resp)
		return
	}

	chat, err := h.chatService.GetChatByID(chatID, userID)
	if err != nil {
		resp := dtoApi.ApiErrorResponse{
			Status: dtoApi.ERROR,
			Errors: []dtoApi.ApiError{
				{
					Code:    "ACCESS_DENIED",
					Message: "You don't have access to this chat",
				},
			},
		}
		response.Send(w, http.StatusForbidden, resp)
		return
	}

	resp := dtoApi.ApiSucessResponse[*dto.ChatInformationDTO]{
		Status: dtoApi.SUCCESS,
		Body:   chat,
	}
	response.Send(w, http.StatusOK, resp)
}
