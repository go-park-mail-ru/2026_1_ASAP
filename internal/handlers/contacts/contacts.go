package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi"

	domain "github.com/go-park-mail-ru/2026_1_ASAP/internal/domain/contacts"
	domainUser "github.com/go-park-mail-ru/2026_1_ASAP/internal/domain/user"
	dtoApi "github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/api"
	dto "github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/contacts"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/middleware"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/utils/mapper"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/utils/response"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/utils/validation"
)

type ContactService interface {
	GetContacts(ctx context.Context, userID int64) ([]*dto.ContactResponse, error)
	AddContact(ctx context.Context, contactRequest dto.AddContactRequest, userID int64) (*dto.ContactResponse, error)
	DeleteContact(ctx context.Context, contactRequest dto.DeleteContactRequest, userID int64) (error)
}

type ContactHandler struct {
	contactService ContactService
}

func NewContactHandler(contactService ContactService) *ContactHandler {
	return &ContactHandler{
		contactService: contactService,
	}
}

// GetContacts godoc
// @Summary Получить список контактов
// @Description Возвращает все контакты текущего пользователя
// @Tags contacts
// @Accept json
// @Produce json
// @Success 200 {object} dtoApi.ResponseGetContactsSuccessForSwagger "Успешное получение списка контактов"
// @Failure 401 {object} dtoApi.ApiErrorResponse "Пользователь не авторизован"
// @Failure 500 {object} dtoApi.ApiErrorResponse "Внутренняя ошибка сервера"
// @Router /api/v1/contacts [get]
func (h *ContactHandler) GetContacts(w http.ResponseWriter, r *http.Request) {
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

	contacts, err := h.contactService.GetContacts(ctx, userID)
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

	resp := dtoApi.ApiSuccessResponse[[]*dto.ContactResponse]{
		Status: dtoApi.Success,
		Body: contacts,
	}
	response.Send(w, http.StatusOK, resp)
}

// CreateContact godoc
// @Summary Создать контакт
// @Description Добавляет нового контакта в список контактов текущего пользователя
// @Tags contacts
// @Accept json
// @Produce json
// @Param request body dto.AddContactRequest true "Запрос на создание контакта"
// @Success 200 {object} dtoApi.ResponseCreateContactSuccessForSwagger "Контакт успешно создан"
// @Failure 400 {object} dtoApi.ApiErrorResponse "Некорректный запрос или ошибка валидации"
// @Failure 401 {object} dtoApi.ApiErrorResponse "Пользователь не авторизован"
// @Failure 404 {object} dtoApi.ApiErrorResponse "Пользователь не найден"
// @Failure 409 {object} dtoApi.ApiErrorResponse "Контакт уже существует"
// @Failure 500 {object} dtoApi.ApiErrorResponse "Внутренняя ошибка сервера"
// @Router /api/v1/contacts [post]
func (h *ContactHandler) CreateContact(w http.ResponseWriter, r *http.Request) {
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
	var req dto.AddContactRequest
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

	errs := validation.ValidationContactCreate(&req)
	if len(errs) > 0 {
		apiErrors := mapper.MapValidationErrorsToApiErrors(errs)
		resp := dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: apiErrors,
		}

		response.Send(w, http.StatusBadRequest, resp)
		return
	}

	createdContact, err := h.contactService.AddContact(ctx, req, userID)
	if err != nil {
		switch {
		case errors.Is(err, domainUser.ErrNotFound):
			resp := dtoApi.ApiErrorResponse{
				Status: dtoApi.Error,
				Errors: []dtoApi.ApiError{
					{
						Code:    dtoApi.InvalidCredentials,
						Message: dtoApi.InvalidCredentialsMsg,
					},
				},
			}
			response.Send(w, http.StatusNotFound, resp)
			return
		case errors.Is(err, domain.ErrContactExists):
			resp := dtoApi.ApiErrorResponse{
				Status: dtoApi.Error,
				Errors: []dtoApi.ApiError{
					{
						Code: dtoApi.ContactAlreadyExists,
						Message: dtoApi.ContactAlreadyExistsMsg,
					},
				},
			}
			response.Send(w, http.StatusConflict, resp)
			return
		case errors.Is(err, domain.ErrCantCreateContactWithYourself):
			resp := dtoApi.ApiErrorResponse{
				Status: dtoApi.Error,
				Errors: []dtoApi.ApiError{
					{
						Code: dtoApi.ContactWithYourself,
						Message: dtoApi.ContactWithYourselfMsg,
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
	resp := dtoApi.ApiSuccessResponse[*dto.ContactResponse]{
			Status: dtoApi.Success,
			Body: createdContact,
		}
	response.Send(w, http.StatusCreated, resp)
}

// DeleteContact godoc
// @Summary Удалить контакт
// @Description Удаляет контакт из списка контактов текущего пользователя
// @Tags contacts
// @Accept json
// @Produce json
// @Param contact_user_id path int true "ID пользователя, которого нужно удалить из контактов"
// @Success 200 {object} dtoApi.ResponseDeleteContactSuccessForSwagger "Контакт успешно удален"
// @Failure 400 {object} dtoApi.ApiErrorResponse "Некорректный ID контакта"
// @Failure 401 {object} dtoApi.ApiErrorResponse "Пользователь не авторизован"
// @Failure 404 {object} dtoApi.ApiErrorResponse "Контакт не найден"
// @Failure 500 {object} dtoApi.ApiErrorResponse "Внутренняя ошибка сервера"
// @Router /api/v1/contacts/{contact_user_id} [delete]
func(h *ContactHandler) DeleteContact(w http.ResponseWriter, r *http.Request) {
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

	contactUserIDurl := chi.URLParam(r, "contact_user_id")
	contactUserID, err := strconv.ParseInt(contactUserIDurl, 10, 64)
	if err != nil || contactUserID < 1 {
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

	deletedContact := &dto.DeleteContactRequest{
		ContactUserID: contactUserID,
	}
	err = h.contactService.DeleteContact(ctx, *deletedContact, userID)
	if err != nil {
		switch {
		case errors.Is(err, domainUser.ErrNotFound):
			resp := dtoApi.ApiErrorResponse{
				Status: dtoApi.Error,
				Errors: []dtoApi.ApiError{
					{
						Code:    dtoApi.InvalidCredentials,
						Message: dtoApi.InvalidCredentialsMsg,
					},
				},
			}
			response.Send(w, http.StatusNotFound, resp)
			return
		case errors.Is(err, domain.ErrContactNotFound):
			resp := dtoApi.ApiErrorResponse{
				Status: dtoApi.Error,
				Errors: []dtoApi.ApiError{
					{
						Code: dtoApi.ContactNotFound,
						Message: dtoApi.ContactNotFoundMsg,
					},
				},
			}
			response.Send(w, http.StatusNotFound, resp)
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
	resp := dtoApi.ApiSuccessResponse[any]{
			Status: dtoApi.Success,
			Body: string("Contact successful delete"),
		}
	response.Send(w, http.StatusOK, resp)
}
 