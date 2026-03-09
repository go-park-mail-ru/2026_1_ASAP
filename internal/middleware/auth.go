package middleware

import (
	"context"
	"net/http"

	dtoApi "github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/api"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/services/session"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/utils/response"
)

type ctxKey string

const UserID ctxKey = "userID"
const SessionID ctxKey = "sessionID"

func AuthMiddleware(sessionService session.SessionServiceInterface) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("session_id")
			if err != nil {
				response.Send(w, http.StatusUnauthorized, dtoApi.ApiErrorResponse{
					Status: dtoApi.ERROR,
					Errors: []dtoApi.ApiError{
						{
							Code:    "UNAUTHORIZED",
							Message: "Unauthorized",
						},
					},
				})
				return
			}
			userID, err := sessionService.GetUserID(cookie.Value)
			if err != nil {
				response.Send(w, http.StatusUnauthorized, dtoApi.ApiErrorResponse{
					Status: dtoApi.ERROR,
					Errors: []dtoApi.ApiError{
						{
							Code:    "UNAUTHORIZED",
							Message: "Unauthorized",
						},
					},
				})
				return
			}

			ctx := context.WithValue(r.Context(), UserID, userID)
			ctx = context.WithValue(ctx, SessionID, cookie.Value)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
