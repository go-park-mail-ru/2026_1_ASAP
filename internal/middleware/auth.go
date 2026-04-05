package middleware

import (
	"context"
	"errors"
	"net/http"

	domainSession "github.com/go-park-mail-ru/2026_1_ASAP/internal/domain/session"
	dtoApi "github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/api"
	dtoSession "github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/session"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/utils/response"
)

type ctxKey string

const UserID ctxKey = "userID"
const SessionID ctxKey = "sessionID"

type SessionService interface {
	CreateSession(ctx context.Context, userID int64) (*dtoSession.SessionDTO, error)
	GetUserID(ctx context.Context, sessionID string) (int64, error)
	DeleteSession(ctx context.Context, sessionID string) error
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

			userID, err := sessionService.GetUserID(r.Context(), cookie.Value)
			if err != nil {
				if errors.Is(err, domainSession.ErrNotFound) {
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
				response.Send(w, http.StatusUnauthorized, dtoApi.ApiErrorResponse{
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
			ctx := context.WithValue(r.Context(), UserID, userID)
			ctx = context.WithValue(ctx, SessionID, cookie.Value)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
