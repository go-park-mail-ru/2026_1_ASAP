package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/google/uuid"

	dtoApi "github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/api"
	dto "github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/chat"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/middleware"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/utils/mapper"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/utils/response"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/utils/validation"
)

type ChatService interface {
	GetAllChats(userID uuid.UUID) ([]dto.ChatInformationDTO, error)
	CreateChat(chatDTO dto.ChatCreate, ownerID uuid.UUID) (*dto.ChatInformationDTO, error)
	GetChatByID(chatID, userID uuid.UUID) (*dto.ChatInformationDTO, error)
}

type ChatsHandler struct {
	chatService ChatService
}

func NewChatHandler(chatService ChatService) *ChatsHandler {
	return &ChatsHandler{
		chatService: chatService,
	}
}

// GetChats godoc
// @Summary Получить список чатов
// @Description Возвращает все чаты пользователя
// @Tags chats
// @Accept json
// @Produce json
// @Success 200 {object} dtoApi.ResponseGetChatsSuccessForSwagger
// @Failure 401 {object} dtoApi.ApiErrorResponse "Пользователь не авторизован"
// @Failure 500 {object} dtoApi.ApiErrorResponse "Невозможно получить чаты"
// @Router /api/v1/chats [get]
func (h *ChatsHandler) GetChats(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserID).(uuid.UUID)
	if !ok {

	}

	chats, err := h.chatService.GetAllChats(userID)
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

	resp := dtoApi.ApiSuccessResponse[[]dto.ChatInformationDTO]{
		Status: dtoApi.Success,
		Body:   chats,
	}
	response.Send(w, http.StatusOK, resp)
}

// ChatCreate godoc
// @Summary Создать чат
// @Description Создаёт новый чат
// @Tags chats
// @Accept json
// @Produce json
// @Param request body dto.ChatCreate true "Запрос на создание чата"
// @Success 201 {object} dtoApi.ResponseCreateChatSuccessForSwagger
// @Failure 400 {object} dtoApi.ApiErrorResponse "Некорретный запрос или ошибка формата полей"
// @Failure 401 {object} dtoApi.ApiErrorResponse "Пользователь неавторизован"
// @Failure 500 {object} dtoApi.ApiErrorResponse "Ошибка создания чата"
// @Router /api/v1/chats [post]
func (h *ChatsHandler) ChatCreate(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	userID, ok := r.Context().Value(middleware.UserID).(uuid.UUID)
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

	decoder := json.NewDecoder(r.Body)
	var req dto.ChatCreate
	err := decoder.Decode(&req)
	if err != nil {
		resp := dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{
				{
					Code:    dtoApi.InvalidJson,
					Message: dtoApi.InvalidJsonMsg,
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
			Status: dtoApi.Error,
			Errors: apiErrors,
		}

		response.Send(w, http.StatusBadRequest, resp)
		return
	}

	createdChat, err := h.chatService.CreateChat(req, userID)
	if err != nil {
		resp := dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{
				{
					Code:    dtoApi.CreateFailed,
					Message: dtoApi.CreateFailedMsg,
				},
			},
		}
		response.Send(w, http.StatusInternalServerError, resp)
		return
	}

	resp := dtoApi.ApiSuccessResponse[*dto.ChatInformationDTO]{
		Status: dtoApi.Success,
		Body:   createdChat,
	}
	response.Send(w, http.StatusCreated, resp)
}

// GetChatByID godoc
// @Summary Получить чат по ID
// @Description Возвращает информацию о чате
// @Tags chats
// @Accept json
// @Produce json
// @Param chat_id path string true "ID чата"
// @Success 200 {object} dtoApi.ResponseGetChatByIDSuccessForSwagger
// @Failure 400 {object} dtoApi.ApiErrorResponse "Некорректный ID чата"
// @Failure 401 {object} dtoApi.ApiErrorResponse "Пользователь неавторизован"
// @Failure 404 {object} dtoApi.ApiErrorResponse "Чат не найден"
// @Failure 500 {object} dtoApi.ApiErrorResponse "Внутреняя ошибка"
// @Router /api/v1/chats/{chat_id} [get]
func (h *ChatsHandler) GetChatByID(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserID).(uuid.UUID)
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

	chatIDurl := chi.URLParam(r, "id")
	chatID, err := uuid.Parse(chatIDurl)
	if err != nil {
		resp := dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{
				{
					Code:    dtoApi.InvalidID,
					Message: dtoApi.InvalidIDMsg,
				},
			},
		}
		response.Send(w, http.StatusUnauthorized, resp)
		return
	}

	chat, err := h.chatService.GetChatByID(chatID, userID)
	if err != nil {
		resp := dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{
				{
					Code:    dtoApi.AccessDenied,
					Message: dtoApi.AccessDeniedMsg,
				},
			},
		}
		response.Send(w, http.StatusForbidden, resp)
		return
	}

	resp := dtoApi.ApiSuccessResponse[*dto.ChatInformationDTO]{
		Status: dtoApi.Success,
		Body:   chat,
	}
	response.Send(w, http.StatusOK, resp)
}
