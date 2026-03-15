package middleware

import (
	"context"
	"net/http"

	dtoApi "github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/api"
	dtoSession "github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/session"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/utils/response"
	"github.com/google/uuid"
)

type ctxKey string

const UserID ctxKey = "userID"
const SessionID ctxKey = "sessionID"

type SessionService interface {
	CreateSession(userID uuid.UUID) (*dtoSession.SessionDTO, error)
	GetUserID(sessionID string) (uuid.UUID, error)
	DeleteSession(sessionID string) error
}

func AuthMiddleware(sessionService SessionService) func(http.Handler) http.Handler {
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
			userID, err := sessionService.GetUserID(cookie.Value)
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

			ctx := context.WithValue(r.Context(), UserID, userID)
			ctx = context.WithValue(ctx, SessionID, cookie.Value)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
