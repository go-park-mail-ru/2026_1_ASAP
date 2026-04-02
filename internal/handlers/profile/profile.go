package profile

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi"
	domain "github.com/go-park-mail-ru/2026_1_ASAP/internal/domain/profile"
	dtoApi "github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/api"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/media"
	dto "github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/profile"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/middleware"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/utils/response"
)

type ProfileServiceInterface interface {
	GetUserProfile(ctx context.Context, userID int64) (response *dto.ResponseGetProfile, err error)
	UpdateProfileBio(ctx context.Context, userID int64, request *dto.RequestUpdateBio) (response *dto.ResponseUpdateProfile, err error)
	UpdateProfileAvatar(ctx context.Context, userID int64, request *dto.RequestUpdateAvatar) (response *dto.ResponseUpdateProfile, err error)
	UpdateProfileBirthDate(ctx context.Context, userID int64, request *dto.RequestUpdateBirthDate) (response *dto.ResponseUpdateProfile, err error)
}

type ProfileHandler struct {
	profileService ProfileServiceInterface
}

func NewProfileHandler(profileService ProfileServiceInterface) *ProfileHandler {
	return &ProfileHandler{profileService: profileService}
}

// GetMyProfile godoc
// @Summary Получение профиля пользователя
// @Description Получение собственного профиля пользователя
// @Tags profile
// @Accept json
// @Produce json
// @Success 200 {object} dtoApi.ResponseGetProfileSuccessForSwagger
// @Failure 401 {object} dtoApi.ApiErrorResponse "Неавторизован"
// @Failure 500 {object} dtoApi.ApiErrorResponse "Внутреняя ошибка"
// @Router /api/v1/profile/me [get]
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

	profile, err := h.profileService.GetUserProfile(ctx, userId)
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

// GetProfile godoc
// @Summary Получение собственного профиля пользователя
// @Description Получение собственного профиля пользователя
// @Tags profile
// @Accept json
// @Produce json
// @Success 200 {object} dtoApi.ResponseGetProfileSuccessForSwagger
// @Failure 400 {object} dtoApi.ApiErrorResponse "Невалидный формат json"
// @Failure 401 {object} dtoApi.ApiErrorResponse "Неавторизован"
// @Failure 500 {object} dtoApi.ApiErrorResponse "Внутреняя ошибка"
// @Router /api/v1/profile/{id} [get]
func (h *ProfileHandler) GetUserProfile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	_, ok := ctx.Value(middleware.UserID).(int64)
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

	profileIDString := chi.URLParam(r, "id")
	profileID, err := strconv.ParseInt(profileIDString, 10, 64)
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
		response.Send(w, http.StatusBadRequest, resp)
		return
	}

	profile, err := h.profileService.GetUserProfile(ctx, profileID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			resp := dtoApi.ApiErrorResponse{
				Status: dtoApi.Error,
				Errors: []dtoApi.ApiError{
					{
						Code:    dtoApi.NotFound,
						Message: dtoApi.NotFoundMsg,
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

	resp := dtoApi.ApiSuccessResponse[dto.ResponseGetProfile]{
		Status: dtoApi.Success,
		Body:   *profile,
	}
	response.Send(w, http.StatusOK, resp)
}

// UpdateUserAvatar godoc
// @Summary Обновление аватара пользователя
// @Description Загружает новый аватар текущего пользователя
// @Tags profile
// @Accept multipart/form-data
// @Produce application/json
// @Param avatar formData file true "Файл аватара (image/jpeg|image/jpg|image/png|image/webp|image/gif), до 5MB"
// @Success 200 {object} dtoApi.ResponseUpdateProfileForSwagger
// @Failure 400 {object} dtoApi.ApiErrorResponse "Некорректный файл аватара"
// @Failure 401 {object} dtoApi.ApiErrorResponse "Неавторизован"
// @Failure 500 {object} dtoApi.ApiErrorResponse "Внутреняя ошибка"
// @Router /api/v1/profiles/me/avatar [patch]
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

	request := &dto.RequestUpdateAvatar{}
	fileInput := &media.FileInput{
		Body:        file,
		ContentType: header.Header.Get("Content-Type"),
		Size:        header.Size,
	}
	request.File = fileInput

	responseUpdate, err := h.profileService.UpdateProfileAvatar(ctx, userId, request)
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
			log.Println(err.Error())
			response.Send(w, http.StatusInternalServerError, resp)
			return
		}
	}

	resp := dtoApi.ApiSuccessResponse[dto.ResponseUpdateProfile]{
		Status: dtoApi.Success,
		Body:   *responseUpdate,
	}
	response.Send(w, http.StatusOK, resp)
}

// UpdateUserBio godoc
// @Summary Обновление bio пользователя
// @Description Обновляет bio текущего авторизованного пользователя
// @Tags profile
// @Accept json
// @Produce application/json
// @Param request body dto.RequestUpdateBio true "Запрос на обновление био (поле bio)"
// @Success 200 {object} dtoApi.ResponseUpdateProfileForSwagger
// @Failure 400 {object} dtoApi.ApiErrorResponse "Невалидный json или bio не может быть пустым"
// @Failure 401 {object} dtoApi.ApiErrorResponse "Неавторизован"
// @Router /api/v1/profiles/me/bio [patch]
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

	request := &dto.RequestUpdateBio{}

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

	responseUpdate, err := h.profileService.UpdateProfileBio(ctx, userId, request)
	if err != nil {
		if errors.Is(err, domain.ErrEmptyBio) {
			resp := dtoApi.ApiErrorResponse{
				Status: dtoApi.Error,
				Errors: []dtoApi.ApiError{
					{
						Code:    dtoApi.EmptyBIO,
						Message: dtoApi.EmptyBIOMsg,
					},
				},
			}
			response.Send(w, http.StatusBadRequest, resp)
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

	resp := dtoApi.ApiSuccessResponse[dto.ResponseUpdateProfile]{
		Status: dtoApi.Success,
		Body:   *responseUpdate,
	}
	response.Send(w, http.StatusOK, resp)
}

// UpdateProfileBirthDate godoc
// @Summary Обновление даты рождения пользователя
// @Description Устанавливает дату рождения текущего авторизованного пользователя. Ожидается строка в формате YYYY-MM-DD (RFC 3339, только дата), не в будущем.
// @Tags profile
// @Accept json
// @Produce application/json
// @Param request body dto.RequestUpdateBirthDate true "Тело запроса: birth_date — дата в формате YYYY-MM-DD"
// @Success 200 {object} dtoApi.ResponseUpdateProfileForSwagger
// @Failure 400 {object} dtoApi.ApiErrorResponse "Невалидный JSON, отсутствует birth_date, неверный формат даты или дата в будущем"
// @Failure 401 {object} dtoApi.ApiErrorResponse "Неавторизован"
// @Failure 500 {object} dtoApi.ApiErrorResponse "Внутренняя ошибка"
// @Router /api/v1/profiles/me/birth [post]
func (h *ProfileHandler) UpdateProfileBirthDate(w http.ResponseWriter, r *http.Request) {
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

	request := &dto.RequestUpdateBirthDate{}
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

	responseUpdate, err := h.profileService.UpdateProfileBirthDate(ctx, userId, request)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrInvalidBirthDateFormat):
			resp := dtoApi.ApiErrorResponse{
				Status: dtoApi.Error,
				Errors: []dtoApi.ApiError{
					{
						Code:    dtoApi.InvalidDateFormat,
						Message: dtoApi.InvalidDateFormatMsg,
					},
				},
			}
			response.Send(w, http.StatusBadRequest, resp)
			return
		case errors.Is(err, domain.ErrInvalidBirthDate):
			resp := dtoApi.ApiErrorResponse{
				Status: dtoApi.Error,
				Errors: []dtoApi.ApiError{
					{
						Code:    dtoApi.InvalidDate,
						Message: dtoApi.InvalidDateMsg,
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

	resp := dtoApi.ApiSuccessResponse[dto.ResponseUpdateProfile]{
		Status: dtoApi.Success,
		Body:   *responseUpdate,
	}
	response.Send(w, http.StatusOK, resp)
}
