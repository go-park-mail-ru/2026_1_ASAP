package profile

import (
	"context"
	"errors"
	"net/http"

	domain "github.com/go-park-mail-ru/2026_1_ASAP/internal/domain/profile"
	dtoApi "github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/api"
	dto "github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/profile"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/middleware"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/utils/response"
)

type ProfileServiceInterface interface {
	GetUserProfile(ctx context.Context, request *dto.RequestGetProfile) (response *dto.ResponseGetProfile, err error)
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
