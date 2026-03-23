package hadlers

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
			response.Send(w, http.StatusUnauthorized, resp)
			return
		case errors.Is(err, domain.ErrContactExists):
			resp := dtoApi.ApiErrorResponse{
				Status: dtoApi.Error,
				Errors: []dtoApi.ApiError{
					{
						Code: "CONTACT_ALREADY_EXISTS",
						Message: "contact already exists",
					},
				},
			}
			response.Send(w, http.StatusUnauthorized, resp)
			return
		case errors.Is(err, domain.ErrCantCreateContactWithYourself):
			resp := dtoApi.ApiErrorResponse{
				Status: dtoApi.Error,
				Errors: []dtoApi.ApiError{
					{
						Code: "CANT_CREATE_CONTACT_WITH_YOURSELF",
						Message: "cant create contact with yourself",
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
	response.Send(w, http.StatusOK, resp)
}

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
			response.Send(w, http.StatusUnauthorized, resp)
			return
		case errors.Is(err, domain.ErrContactNotFound):
			resp := dtoApi.ApiErrorResponse{
				Status: dtoApi.Error,
				Errors: []dtoApi.ApiError{
					{
						Code: "CONTACT_NOT_FOUND",
						Message: "contact not found",
					},
				},
			}
			response.Send(w, http.StatusUnauthorized, resp)
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
			Body: string("User successful delete"),
		}
	response.Send(w, http.StatusOK, resp)
}
 