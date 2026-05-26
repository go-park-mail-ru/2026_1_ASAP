package middleware

import (
	"context"
	"net/http"

	authv1 "github.com/go-park-mail-ru/2026_1_ASAP/gen/go/auth/v1"
	dtoApi "github.com/go-park-mail-ru/2026_1_ASAP/internal/gateway/dto/api"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/utils/response"
	"github.com/go-park-mail-ru/2026_1_ASAP/pkg/grpcerr"
)

//go:generate mockgen -source=auth.go -destination=mock/session_service_mock.go -package=mock

const (
	UserID    ctxKey = "userID"
	SessionID ctxKey = "sessionID"
)

func AuthMiddleware(sessionService authv1.AuthClient) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("session_id")
			if err != nil {
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

			sess, err := sessionService.ValidateSession(r.Context(), &authv1.RequestValidateSession{
				SessionId: cookie.Value,
			})
			if err != nil {
				_, appCode, _ := grpcerr.Error(err)
				switch authv1.AuthErrorCode(appCode) {
				case authv1.AuthErrorCode_AUTH_ERROR_SESSION_NOT_FOUND,
					authv1.AuthErrorCode_AUTH_ERROR_SESSION_EXPIRED:
					response.Send(w, http.StatusUnauthorized, dtoApi.ApiErrorResponse{
						Status: dtoApi.Error,
						Errors: []dtoApi.ApiError{{Code: dtoApi.Unauthorized, Message: dtoApi.UnauthorizedMsg}},
					})
				default:
					response.Send(w, http.StatusInternalServerError, dtoApi.ApiErrorResponse{
						Status: dtoApi.Error,
						Errors: []dtoApi.ApiError{{Code: dtoApi.InternalError, Message: dtoApi.InternalErrorMsg}},
					})
				}
				return
			}
			if sess == nil {
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
			ctx := context.WithValue(r.Context(), UserID, sess.GetUserId())
			ctx = context.WithValue(ctx, SessionID, cookie.Value)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
