package profile

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	domain "github.com/go-park-mail-ru/2026_1_ASAP/internal/domain/profile"
	dtoApi "github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/api"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/media"
	dto "github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/profile"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/middleware"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/utils/response"
)

type ProfileServiceInterface interface {
	GetUserProfile(ctx context.Context, request *dto.RequestGetProfile) (response *dto.ResponseGetProfile, err error)
	UpdateProfileBio(ctx context.Context, request *dto.RequestUpdateBio) (response *dto.ResponseUpdateProfile, err error)
	UpdateProfileAvatar(ctx context.Context, request *dto.RequestUpdateAvatar) (response *dto.ResponseUpdateProfile, err error)
}

type ProfileHandler struct {
	profileService ProfileServiceInterface
}

func NewProfileHandler(profileService ProfileServiceInterface) *ProfileHandler {
	return &ProfileHandler{profileService: profileService}
}

func (h *ProfileHandler) GetMyProfile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userId, ok := ctx.Value(middleware.UserID).(int64)
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
	request := &dto.RequestGetProfile{
		UserID: userId,
	}

	profile, err := h.profileService.GetUserProfile(ctx, request)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
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

	resp := dtoApi.ApiSuccessResponse[dto.ResponseGetProfile]{
		Status: dtoApi.Success,
		Body:   *profile,
	}
	response.Send(w, http.StatusOK, resp)
}

func (h *ProfileHandler) GetUserProfile(w http.ResponseWriter, r *http.Request) {
	// TODO
}

func (h *ProfileHandler) UpdateUserAvatar(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userId, ok := ctx.Value(middleware.UserID).(int64)
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

	request := &dto.RequestUpdateAvatar{UserID: userId}
	fileInput := &media.FileInput{
		Body:        file,
		ContentType: header.Header.Get("Content-Type"),
		Size:        header.Size,
	}
	request.File = fileInput

	responseUpdate, err := h.profileService.UpdateProfileAvatar(ctx, request)
	if err != nil {
		switch err {
		case domain.ErrEmptyAvatar:
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
		case domain.ErrInvalidAvatarType:
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

		case domain.ErrAvatarTooLarge:
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
			response.Send(w, http.StatusBadRequest, resp)
			return
		}
	}

	resp := dtoApi.ApiSuccessResponse[dto.ResponseUpdateProfile]{
		Status: dtoApi.Success,
		Body:   *responseUpdate,
	}
	response.Send(w, http.StatusOK, resp)
}

func (h *ProfileHandler) UpdateUserBio(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userId, ok := ctx.Value(middleware.UserID).(int64)
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

	request := &dto.RequestUpdateBio{UserID: userId}

	err := decoder.Decode(request)
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

	responseUpdate, err := h.profileService.UpdateProfileBio(ctx, request)
	if err != nil {
		// TODO
	}

	resp := dtoApi.ApiSuccessResponse[dto.ResponseUpdateProfile]{
		Status: dtoApi.Success,
		Body:   *responseUpdate,
	}
	response.Send(w, http.StatusOK, resp)
}
