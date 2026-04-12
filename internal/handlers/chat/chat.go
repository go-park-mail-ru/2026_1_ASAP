package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi"

	domain "github.com/go-park-mail-ru/2026_1_ASAP/internal/domain/chat"
	domainProfile "github.com/go-park-mail-ru/2026_1_ASAP/internal/domain/profile"
	dtoApi "github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/api"
	dto "github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/chat"
	dtoWs "github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/ws"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/middleware"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/utils/mapper"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/utils/response"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/utils/validation"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/media"
)

type ChatService interface {
	GetDialogName(ctx context.Context, chatID int64, userID int64) (string, error)
	GetAllChats(ctx context.Context, id int64) ([]dto.ChatInformationDTO, error)
	CreateChat(ctx context.Context, chatDTO dto.ChatCreate, ownerID int64) (*dto.ChatInformationDTO, error)
	GetChatByID(ctx context.Context, chatID, userID int64) (*dto.ChatInformationDTO, error)
	DeleteChat(ctx context.Context, userID, chatID int64) (error)
	UpdateChatAvatar(ctx context.Context, userID, chatID int64, request *dto.RequestUpdateAvatar) (*dto.ChatInformationDTO, error)
	UpdateChatTitle(ctx context.Context, userID, chatID int64, request *dto.RequestUpdateTitle) (*dto.ChatInformationDTO, error)
	AddMembersToChat(ctx context.Context, userID, chatID int64, request *dto.RequestAddMember) (error)
	DeleteMemberFromChat(ctx context.Context, userID, chatID int64, request *dto.RequestDeleteMember) (error)
	GetAllChatMembers(ctx context.Context, userID, chatID int64) (*dto.ResponseGetChatMembers, error) 
	QuitChat(ctx context.Context, userID, chatID int64) (error)
	GetChatMemberIDs(ctx context.Context, chatID int64) ([]int64, error)
}

type WsUserPublisher interface {
	PublishToUser(ctx context.Context, userID int64, message []byte)
}

type ChatsHandler struct {
	chatService ChatService
	wsPublisher WsUserPublisher
}

func NewChatHandler(chatService ChatService, wsPublisher WsUserPublisher) *ChatsHandler {
	return &ChatsHandler{
		chatService: chatService,
		wsPublisher: wsPublisher,
	}
}

func (h *ChatsHandler) wsPublishChatSnapshotToMembers(ctx context.Context, chatID int64, encode func(*dto.ChatInformationDTO) ([]byte, error)) {
	if h.wsPublisher == nil {
		return
	}
	memberIDs, err := h.chatService.GetChatMemberIDs(ctx, chatID)
	if err != nil {
		return
	}
	for _, uid := range memberIDs {
		info, errInfo := h.chatService.GetChatByID(ctx, chatID, uid)
		if errInfo != nil {
			continue
		}
		frame, errEnc := encode(info)
		if errEnc != nil {
			continue
		}
		h.wsPublisher.PublishToUser(ctx, uid, frame)
	}
}

func (h *ChatsHandler) wsPublishChatDeleted(ctx context.Context, memberIDs []int64, chatID int64) {
	if h.wsPublisher == nil {
		return
	}
	frame, err := dtoWs.EncodeChatDeleted(chatID)
	if err != nil {
		return
	}
	for _, uid := range memberIDs {
		h.wsPublisher.PublishToUser(ctx, uid, frame)
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

	h.wsPublishChatSnapshotToMembers(ctx, createdChat.ID, dtoWs.EncodeChatNew)
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

	memberIDs, _ := h.chatService.GetChatMemberIDs(ctx, chatID)

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

	h.wsPublishChatDeleted(ctx, memberIDs, chatID)
}

// UpdateChatAvatar обновляет аватар чата
// @Summary Обновление аватара чата
// @Description Обновляет аватар чата. Поддерживаются форматы: JPEG, PNG, GIF, WebP. Максимальный размер файла: 5MB. 
// @Description Только участники чата могут обновлять аватар.
// @Description Для диалогов обновление аватара запрещено.
// @Tags chats
// @Accept multipart/form-data
// @Produce json
// @Param id path int true "ID чата" minimum(1)
// @Param avatar formData file true "Файл аватара" 
// @Security BearerAuth
// @Success 200 {object} dtoApi.ApiSuccessResponse[dto.ChatInformationDTO] "Аватар успешно обновлён"
// @Failure 400 {object} dtoApi.ApiErrorResponse "Неверный запрос (неверный ID, файл не найден, пустой файл, неверный формат, файл слишком большой, попытка обновить аватар диалога)"
// @Failure 401 {object} dtoApi.ApiErrorResponse "Пользователь не авторизован"
// @Failure 403 {object} dtoApi.ApiErrorResponse "Пользователь не является участником чата"
// @Failure 500 {object} dtoApi.ApiErrorResponse "Внутренняя ошибка сервера"
// @Router /api/v1/chats/{id}/avatar [post]
func (h *ChatsHandler) UpdateChatAvatar(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := ctx.Value(middleware.UserID).(int64)
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

	file, header, err := r.FormFile("avatar")
	if err != nil {
		resp := dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{
				{
					Code:    dtoApi.FileNotFound,
					Message: dtoApi.FileNotFoundMsg,
				},
			},
		}
		response.Send(w, http.StatusBadRequest, resp)
		return
	}

	defer file.Close()

	request := &dto.RequestUpdateAvatar{}
	fileInput := &media.FileInput{
		Body:        file,
		ContentType: header.Header.Get("Content-Type"),
		Size:        header.Size,
	}
	request.File = fileInput

	responseUpdate, err := h.chatService.UpdateChatAvatar(ctx, userID, chatID, request)
	if err != nil {
		switch {
		case errors.Is(err, domainProfile.ErrEmptyAvatar):
			resp := dtoApi.ApiErrorResponse{
				Status: dtoApi.Error,
				Errors: []dtoApi.ApiError{
					{
						Code:    dtoApi.EmptyFile,
						Message: dtoApi.EmptyFileMsg,
					},
				},
			}
			response.Send(w, http.StatusBadRequest, resp)
			return
		case errors.Is(err, domain.ErrNotMember):
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
		case errors.Is(err, domain.ErrDialogCannotHaveCustomAvatar):
        	resp := dtoApi.ApiErrorResponse{
            	Status: dtoApi.Error,
            	Errors: []dtoApi.ApiError{
                	{
                    	Code:    dtoApi.CantChangeAvatar,
                    	Message: dtoApi.CantChangeAvatarMsg,
                	},
            	},
        	}
        	response.Send(w, http.StatusBadRequest, resp)
        	return
		case errors.Is(err, domainProfile.ErrInvalidAvatarType):
			resp := dtoApi.ApiErrorResponse{
				Status: dtoApi.Error,
				Errors: []dtoApi.ApiError{
					{
						Code:    dtoApi.InvalidFileFormat,
						Message: dtoApi.InvalidFileFormatMsg,
					},
				},
			}
			response.Send(w, http.StatusBadRequest, resp)
			return

		case errors.Is(err, domainProfile.ErrAvatarTooLarge):
			resp := dtoApi.ApiErrorResponse{
				Status: dtoApi.Error,
				Errors: []dtoApi.ApiError{
					{
						Code:    dtoApi.FileTooLarge,
						Message: dtoApi.FileTooLargeMsg,
					},
				},
			}
			response.Send(w, http.StatusBadRequest, resp)
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

	resp := dtoApi.ApiSuccessResponse[*dto.ChatInformationDTO]{
		Status: dtoApi.Success,
		Body:   responseUpdate,
	}
	response.Send(w, http.StatusOK, resp)

	h.wsPublishChatSnapshotToMembers(ctx, chatID, dtoWs.EncodeChatUpdated)
}

// UpdateChatTitle обновляет название чата
// @Summary Обновление названия чата
// @Description Обновляет название чата.
// @Description Для диалогов (личных чатов) изменение названия запрещено.
// @Tags chats
// @Accept json
// @Produce json
// @Param id path int true "ID чата" minimum(1)
// @Param request body dto.RequestUpdateTitle true "Новое название чата"
// @Security BearerAuth
// @Success 200 {object} dtoApi.ApiSuccessResponse[dto.ChatInformationDTO] "Название успешно обновлено"
// @Failure 400 {object} dtoApi.ApiErrorResponse "Неверный запрос (неверный ID, неверный JSON, ошибка валидации, попытка изменить название диалога)"
// @Failure 401 {object} dtoApi.ApiErrorResponse "Пользователь не авторизован"
// @Failure 403 {object} dtoApi.ApiErrorResponse "Пользователь не является участником чата или не имеет прав администратора"
// @Failure 404 {object} dtoApi.ApiErrorResponse "Чат не найден"
// @Failure 500 {object} dtoApi.ApiErrorResponse "Внутренняя ошибка сервера"
// @Router /api/v1/chats/{id}/title [post]
func (h *ChatsHandler) UpdateChatTitle(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	ctx := r.Context()
	userID, ok := ctx.Value(middleware.UserID).(int64)
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

	decoder := json.NewDecoder(r.Body)
	request := &dto.RequestUpdateTitle{}
	err = decoder.Decode(request)
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

	errs := validation.ValidationRequestTitle(request)
	if errs != nil {
		apiErrors := mapper.MapValidationErrorsToApiErrors(errs)
		resp := dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: apiErrors,
		}

		response.Send(w, http.StatusBadRequest, resp)
		return
	}

	responseUpdate, err := h.chatService.UpdateChatTitle(ctx, userID, chatID, request)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrNotMember):
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
		case errors.Is(err, domain.ErrChatNotFound):
			resp := dtoApi.ApiErrorResponse{
            	Status: dtoApi.Error,
            	Errors: []dtoApi.ApiError{
                	{
                    	Code:    dtoApi.CantFindChat,
                    	Message: dtoApi.CantFindChatMsg,
                	},
            	},
        	}
        	response.Send(w, http.StatusNotFound, resp)
        	return
		case errors.Is(err, domain.ErrDialogCannotHaveCustomTitle):
			resp := dtoApi.ApiErrorResponse{
            	Status: dtoApi.Error,
            	Errors: []dtoApi.ApiError{
                	{
                    	Code:    dtoApi.CantChangeTitle,
                    	Message: dtoApi.CantChangeTitleMsg,
                	},
            	},
        	}
        	response.Send(w, http.StatusBadRequest, resp)
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

	resp := dtoApi.ApiSuccessResponse[*dto.ChatInformationDTO]{
		Status: dtoApi.Success,
		Body:   responseUpdate,
	}
	response.Send(w, http.StatusOK, resp)

	h.wsPublishChatSnapshotToMembers(ctx, chatID, dtoWs.EncodeChatUpdated)
}

// AddMembersToChat добавляет участников в чат
// @Summary Добавление участников в чат
// @Description Добавляет одного или нескольких участников в групповой чат.
// @Description Только владелец чата может добавлять новых участников.
// @Description Для диалогов (личных чатов) добавление участников запрещено.
// @Tags chats
// @Accept json
// @Produce json
// @Param id path int true "ID чата" minimum(1)
// @Param request body dto.RequestAddMember true "Список ID пользователей для добавления"
// @Security BearerAuth
// @Success 200 {object} dtoApi.ApiSuccessResponse[any] "Участники успешно добавлены"
// @Failure 400 {object} dtoApi.ApiErrorResponse "Неверный запрос (неверный ID, неверный JSON, ошибка валидации, попытка добавить в диалог, пользователь уже в чате)"
// @Failure 401 {object} dtoApi.ApiErrorResponse "Пользователь не авторизован"
// @Failure 403 {object} dtoApi.ApiErrorResponse "Пользователь не является участником чата или не имеет прав владельца"
// @Failure 404 {object} dtoApi.ApiErrorResponse "Чат не найден"
// @Failure 500 {object} dtoApi.ApiErrorResponse "Внутренняя ошибка сервера"
// @Router /api/v1/chats/{id}/members [post]
func (h *ChatsHandler) AddMembersToChat(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	ctx := r.Context()
	userID, ok := ctx.Value(middleware.UserID).(int64)
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

	decoder := json.NewDecoder(r.Body)
	request := &dto.RequestAddMember{}
	err = decoder.Decode(request)
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

	errs := validation.ValidationRequestAddMember(request)
	if errs != nil {
		apiErrors := mapper.MapValidationErrorsToApiErrors(errs)
		resp := dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: apiErrors,
		}

		response.Send(w, http.StatusBadRequest, resp)
		return
	}

	err = h.chatService.AddMembersToChat(ctx, userID, chatID, request)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrNotMember):
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
		case errors.Is(err, domain.ErrChatNotFound):
			resp := dtoApi.ApiErrorResponse{
            	Status: dtoApi.Error,
            	Errors: []dtoApi.ApiError{
                	{
                    	Code:    dtoApi.CantFindChat,
                    	Message: dtoApi.CantFindChatMsg,
                	},
            	},
        	}
        	response.Send(w, http.StatusNotFound, resp)
        	return
		case errors.Is(err, domain.ErrCantAddMemberToDialog):
			resp := dtoApi.ApiErrorResponse{
            	Status: dtoApi.Error,
            	Errors: []dtoApi.ApiError{
                	{
                    	Code:    dtoApi.CantAddMemberToDialog,
                    	Message: dtoApi.CantAddMemberToDialogMsg,
                	},
            	},
        	}
        	response.Send(w, http.StatusBadRequest, resp)
        	return
		case errors.Is(err, domain.ErrOnlyOwnerCanAddPeople):
			resp := dtoApi.ApiErrorResponse{
            	Status: dtoApi.Error,
            	Errors: []dtoApi.ApiError{
                	{
                    	Code:    dtoApi.OnlyOwnerCanAddMembers,
                    	Message: dtoApi.OnlyOwnerCanAddMembersMsg,
                	},
            	},
        	}
        	response.Send(w, http.StatusForbidden, resp)
        	return
		case errors.Is(err, domain.ErrMemberAlreadyInChat):
			resp := dtoApi.ApiErrorResponse{
            	Status: dtoApi.Error,
            	Errors: []dtoApi.ApiError{
                	{
                    	Code:    dtoApi.MemberAlreadyInChat,
                    	Message: dtoApi.MemberAlreadyInChatMsg,
                	},
            	},
        	}
        	response.Send(w, http.StatusBadRequest, resp)
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
		Body:   "Members added to chat successfuly",
	}
	response.Send(w, http.StatusOK, resp)
}

// DeleteMemberFromChat удаляет участника из чата
// @Summary Удаление участника из чата
// @Description Удаляет участника из группового чата.
// @Description Только владелец чата может удалять других участников.
// @Description Нельзя удалить владельца чата.
// @Description Для диалогов (личных чатов) удаление участников запрещено.
// @Tags chats
// @Accept json
// @Produce json
// @Param id path int true "ID чата" minimum(1)
// @Param request body dto.RequestDeleteMember true "ID пользователя для удаления"
// @Security BearerAuth
// @Success 200 {object} dtoApi.ApiSuccessResponse[any] "Участник успешно удалён из чата"
// @Failure 400 {object} dtoApi.ApiErrorResponse "Неверный запрос (неверный ID, неверный JSON, ошибка валидации, попытка удалить из диалога)"
// @Failure 401 {object} dtoApi.ApiErrorResponse "Пользователь не авторизован"
// @Failure 403 {object} dtoApi.ApiErrorResponse "Нет прав для удаления (не участник чата, не владелец, попытка удалить владельца)"
// @Failure 404 {object} dtoApi.ApiErrorResponse "Чат не найден или участник не найден"
// @Failure 500 {object} dtoApi.ApiErrorResponse "Внутренняя ошибка сервера"
// @Router /api/v1/chats/{id}/members [delete]
func (h *ChatsHandler) DeleteMemberFromChat(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	ctx := r.Context()
	userID, ok := ctx.Value(middleware.UserID).(int64)
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

	decoder := json.NewDecoder(r.Body)
	request := &dto.RequestDeleteMember{}
	err = decoder.Decode(request)
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

	errs := validation.ValidationRequestDeleteMember(request)
	if errs != nil {
		apiErrors := mapper.MapValidationErrorsToApiErrors(errs)
		resp := dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: apiErrors,
		}

		response.Send(w, http.StatusBadRequest, resp)
		return
	}

	err = h.chatService.DeleteMemberFromChat(ctx, userID, chatID, request)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrNotMember):
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
		case errors.Is(err, domain.ErrChatNotFound):
			resp := dtoApi.ApiErrorResponse{
            	Status: dtoApi.Error,
            	Errors: []dtoApi.ApiError{
                	{
                    	Code:    dtoApi.CantFindChat,
                    	Message: dtoApi.CantFindChatMsg,
                	},
            	},
        	}
        	response.Send(w, http.StatusNotFound, resp)
        	return
		case errors.Is(err, domain.ErrCantDeleteMemberFromDialog):
			resp := dtoApi.ApiErrorResponse{
            	Status: dtoApi.Error,
            	Errors: []dtoApi.ApiError{
                	{
                    	Code:    dtoApi.DeleteMemberFromDialog,
                    	Message: dtoApi.DeleteMemberFromDialogMsg,
                	},
            	},
        	}
        	response.Send(w, http.StatusBadRequest, resp)
        	return
		case errors.Is(err, domain.ErrOnlyOwnerCanDeletePeople):
			resp := dtoApi.ApiErrorResponse{
            	Status: dtoApi.Error,
            	Errors: []dtoApi.ApiError{
                	{
                    	Code:    dtoApi.OnlyOwnerCanDeleteMember,
                    	Message: dtoApi.OnlyOwnerCanDeleteMemberMsg,
                	},
            	},
        	}
        	response.Send(w, http.StatusForbidden, resp)
        	return
		case errors.Is(err, domain.ErrCantDeleteOwnerOfChat):
			resp := dtoApi.ApiErrorResponse{
            	Status: dtoApi.Error,
            	Errors: []dtoApi.ApiError{
                	{
                    	Code:    dtoApi.DeleteOwnerOfChat,
                    	Message: dtoApi.DeleteOwnerOfChatMsg,
                	},
            	},
        	}
        	response.Send(w, http.StatusForbidden, resp)
        	return
		case errors.Is(err, domain.ErrMemberNotFound):
			resp := dtoApi.ApiErrorResponse{
            	Status: dtoApi.Error,
            	Errors: []dtoApi.ApiError{
                	{
                    	Code:    dtoApi.MemberNotFound,
                    	Message: dtoApi.MemberNotFoundMsg,
                	},
            	},
        	}
        	response.Send(w, http.StatusNotFound, resp)
        	return
		case errors.Is(err, domain.ErrUserNotMember):
			resp := dtoApi.ApiErrorResponse{
            	Status: dtoApi.Error,
            	Errors: []dtoApi.ApiError{
                	{
                    	Code:    dtoApi.UserNotMemberOfChat,
                    	Message: dtoApi.UserNotMemberOfChatMsg,
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
		Body:   "Members deleted from chat successfuly",
	}
	response.Send(w, http.StatusOK, resp)

	h.wsPublishChatDeleted(ctx, []int64{request.MemberId}, chatID)
	h.wsPublishChatSnapshotToMembers(ctx, chatID, dtoWs.EncodeChatUpdated)
}

// GetChatMembers получает список участников чата
// @Summary Получение списка участников чата
// @Description Возвращает список ID всех участников чата.
// @Description Только участники чата могут просматривать список участников.
// @Description Для диалогов (личных чатов) возвращает ID обоих участников.
// @Description Для групп возвращает ID всех участников, включая владельца.
// @Tags chats
// @Accept json
// @Produce json
// @Param id path int true "ID чата" minimum(1)
// @Security BearerAuth
// @Success 200 {object} dtoApi.ApiSuccessResponse[dto.ResponseGetChatMembers] "Список участников успешно получен"
// @Failure 400 {object} dtoApi.ApiErrorResponse "Неверный запрос (неверный ID чата)"
// @Failure 401 {object} dtoApi.ApiErrorResponse "Пользователь не авторизован"
// @Failure 403 {object} dtoApi.ApiErrorResponse "Пользователь не является участником чата"
// @Failure 500 {object} dtoApi.ApiErrorResponse "Внутренняя ошибка сервера"
// @Router /api/v1/chats/{id}/members [get]
func (h *ChatsHandler) GetChatMembers(w http.ResponseWriter, r *http.Request) {
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

	members, err := h.chatService.GetAllChatMembers(ctx, userID, chatID)
	if err != nil {
		if errors.Is(err, domain.ErrNotMember) {
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

	resp := dtoApi.ApiSuccessResponse[*dto.ResponseGetChatMembers]{
		Status: dtoApi.Success,
		Body:   members,
	}
	response.Send(w, http.StatusOK, resp)
}

// QuitChat выход пользователя из чата
// @Summary Выход из чата
// @Description Позволяет пользователю выйти из группового чата.
// @Description Владелец чата не может выйти.
// @Description Выход из диалогов (личных чатов) запрещён.
// @Description При успешном выходе пользователь удаляется из списка участников чата.
// @Tags chats
// @Accept json
// @Produce json
// @Param id path int true "ID чата" minimum(1)
// @Security BearerAuth
// @Success 200 {object} dtoApi.ApiSuccessResponse[any] "Пользователь успешно покинул чат"
// @Failure 400 {object} dtoApi.ApiErrorResponse "Неверный запрос (неверный ID чата, попытка выйти из диалога)"
// @Failure 401 {object} dtoApi.ApiErrorResponse "Пользователь не авторизован"
// @Failure 403 {object} dtoApi.ApiErrorResponse "Нет прав для выхода (пользователь не участник чата или является владельцем)"
// @Failure 404 {object} dtoApi.ApiErrorResponse "Чат не найден или пользователь не найден в чате"
// @Failure 500 {object} dtoApi.ApiErrorResponse "Внутренняя ошибка сервера"
// @Router /api/v1/chats/{id}/quit [post]
func (h *ChatsHandler) QuitChat(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := ctx.Value(middleware.UserID).(int64)
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

	err = h.chatService.QuitChat(ctx, userID, chatID)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrNotMember):
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
		case errors.Is(err, domain.ErrChatNotFound):
			resp := dtoApi.ApiErrorResponse{
            	Status: dtoApi.Error,
            	Errors: []dtoApi.ApiError{
                	{
                    	Code:    dtoApi.CantFindChat,
                    	Message: dtoApi.CantFindChatMsg,
                	},
            	},
        	}
        	response.Send(w, http.StatusNotFound, resp)
        	return
		case errors.Is(err, domain.ErrCantQuitDialog):
			resp := dtoApi.ApiErrorResponse{
            	Status: dtoApi.Error,
            	Errors: []dtoApi.ApiError{
                	{
                    	Code:    dtoApi.CantQuitDialog,
                    	Message: dtoApi.CantQuitDialogMsg,
                	},
            	},
        	}
        	response.Send(w, http.StatusBadRequest, resp)
        	return
		case errors.Is(err, domain.ErrOwnerCantQuitGroup):
			resp := dtoApi.ApiErrorResponse{
            	Status: dtoApi.Error,
            	Errors: []dtoApi.ApiError{
                	{
                    	Code:    dtoApi.OwnerCantQuitChat,
                    	Message: dtoApi.OwnerCantQuitChatMsg,
                	},
            	},
        	}
        	response.Send(w, http.StatusForbidden, resp)
        	return
		case errors.Is(err, domain.ErrMemberNotFound):
			resp := dtoApi.ApiErrorResponse{
            	Status: dtoApi.Error,
            	Errors: []dtoApi.ApiError{
                	{
                    	Code:    dtoApi.MemberNotFound,
                    	Message: dtoApi.MemberNotFoundMsg,
                	},
            	},
        	}
        	response.Send(w, http.StatusNotFound, resp)
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
		Body:   "You are leave chat successfuly",
	}
	response.Send(w, http.StatusOK, resp)

	h.wsPublishChatDeleted(ctx, []int64{userID}, chatID)
	h.wsPublishChatSnapshotToMembers(ctx, chatID, dtoWs.EncodeChatUpdated)
}