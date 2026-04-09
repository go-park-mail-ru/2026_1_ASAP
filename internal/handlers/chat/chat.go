package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi"

	domain "github.com/go-park-mail-ru/2026_1_ASAP/internal/domain/chat"
	dtoApi "github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/api"
	dto "github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/chat"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/middleware"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/utils/mapper"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/utils/response"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/utils/validation"
)

type ChatService interface {
	GetDialogName(ctx context.Context, chatID int64, userID int64) (string, error)
	GetAllChats(ctx context.Context, id int64) ([]dto.ChatInformationDTO, error)
	CreateChat(ctx context.Context, chatDTO dto.ChatCreate, ownerID int64) (*dto.ChatInformationDTO, error)
	GetChatByID(ctx context.Context, chatID, userID int64) (*dto.ChatInformationDTO, error)
	DeleteChat(ctx context.Context, userID, chatID int64) (error)
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
	ctx := r.Context()
	userID, ok := r.Context().Value(middleware.UserID).(int64)
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

	chats, err := h.chatService.GetAllChats(ctx, userID)
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
	ctx := r.Context()
	userID, ok := r.Context().Value(middleware.UserID).(int64)
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

	createdChat, err := h.chatService.CreateChat(ctx, req, userID)
	if err != nil {
		if errors.Is(err, domain.ErrDialogAlreadyExists) {
			resp := dtoApi.ApiErrorResponse{
				Status: dtoApi.Error,
				Errors: []dtoApi.ApiError{
					{
						Code:    dtoApi.DialogAlreadyExists,
						Message: dtoApi.DialogAlreadyExistsMsg,
					},
				},
			}
			response.Send(w, http.StatusConflict, resp)
			return
		}
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
	ctx := r.Context()
	userID, ok := r.Context().Value(middleware.UserID).(int64)
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
	chatID, err := strconv.ParseInt(chatIDurl, 10, 64)
	if err != nil || chatID < 1 {
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

	chat, err := h.chatService.GetChatByID(ctx, chatID, userID)
	if err != nil {
		if errors.Is(err, domain.ErrNotMember) {
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
		resp := dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{
				{
					Code: dtoApi.InternalError,
					Message: dtoApi.InternalErrorMsg,
				},
			},
		}
		response.Send(w, http.StatusInternalServerError, resp)
		return
	}

	resp := dtoApi.ApiSuccessResponse[*dto.ChatInformationDTO]{
		Status: dtoApi.Success,
		Body:   chat,
	}
	response.Send(w, http.StatusOK, resp)
}

// DeleteChat godoc
// @Summary Удалить чат
// @Description Удаляет чат. Только владелец чата может выполнить это действие. Чат удаляется полностью для всех участников вместе со всеми сообщениями.
// @Tags chats
// @Accept json
// @Produce json
// @Param id path int true "ID чата"
// @Success 200 {object} dtoApi.ApiSuccessResponse[string] "Чат успешно удален"
// @Failure 400 {object} dtoApi.ApiErrorResponse "Некорректный ID чата"
// @Failure 401 {object} dtoApi.ApiErrorResponse "Пользователь не авторизован"
// @Failure 403 {object} dtoApi.ApiErrorResponse "Нет прав на удаление чата (не владелец или не участник)"
// @Failure 500 {object} dtoApi.ApiErrorResponse "Внутренняя ошибка сервера"
// @Router /api/v1/chats/{id} [delete]
func (h *ChatsHandler) DeleteChat(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := r.Context().Value(middleware.UserID).(int64)
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
	chatID, err := strconv.ParseInt(chatIDurl, 10, 64)
	if err != nil || chatID < 1 {
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

	err = h.chatService.DeleteChat(ctx, userID, chatID)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrOnlyOwnerCanDeleteChat):
			resp := dtoApi.ApiErrorResponse{
				Status: dtoApi.Error,
				Errors: []dtoApi.ApiError{
					{
						Code: dtoApi.CantDeleteChat,
						Message: dtoApi.CantDeleteChatMsg,
					},
				},
			}
			response.Send(w, http.StatusForbidden, resp)
			return
		case errors.Is(err, domain.ErrNotMember):
			resp := dtoApi.ApiErrorResponse{
				Status: dtoApi.Error,
				Errors: []dtoApi.ApiError{
					{
						Code: dtoApi.NotMemberOfChat,
						Message: dtoApi.NotMemberOfChatMsg,
					},
				},
			}
			response.Send(w, http.StatusForbidden, resp)
			return
		default:
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
	}
	resp := dtoApi.ApiSuccessResponse[any]{
			Status: dtoApi.Success,
			Body: string("Chat successful delete"),
		}
	response.Send(w, http.StatusOK, resp)

}