package middleware

import (
	"context"
	"net/http"

	dtoApi "github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/api"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/utils/response"
)

//go:generate go run github.com/golang/mock/mockgen@v1.6.0 -source=admin.go -destination=mock/admin_service_mock.go -package=mock
type AdminService interface {
	IsAdmin(ctx context.Context, userID int64) (bool, error)
}

func AdminMiddleware(adminService AdminService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, ok := r.Context().Value(UserID).(int64)
			if !ok {
				response.Send(w, http.StatusUnauthorized, dtoApi.ApiErrorResponse{
					Status: dtoApi.Error,
					Errors: []dtoApi.ApiError{
						{
							Code:    dtoApi.Unauthorized,
							Message: dtoApi.UnauthorizedMsg,
						},
					},
				})
				return
			}

			isAdmin, err := adminService.IsAdmin(r.Context(), userID)
			if err != nil {
				response.Send(w, http.StatusInternalServerError, dtoApi.ApiErrorResponse{
					Status: dtoApi.Error,
					Errors: []dtoApi.ApiError{
						{
							Code:    dtoApi.InternalError,
							Message: dtoApi.InternalErrorMsg,
						},
					},
				})
				return
			}
			if !isAdmin {
				response.Send(w, http.StatusForbidden, dtoApi.ApiErrorResponse{
					Status: dtoApi.Error,
					Errors: []dtoApi.ApiError{
						{
							Code:    dtoApi.AccessDenied,
							Message: dtoApi.AccessDeniedMsg,
						},
					},
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
