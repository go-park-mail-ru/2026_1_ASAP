package middleware

import (
	"context"
	"net/http"

	dtoApi "github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/api"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/services/session"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/utils/response"
)

func AuthMiddleware(sessionService session.SessionServiceInterface) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("session_id")
			if err != nil {
				response.Send(w, http.StatusUnauthorized, dtoApi.ApiResponse{
					Status: dtoApi.ERROR,
					Error: []dtoApi.ApiError{
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
				response.Send(w, http.StatusUnauthorized, dtoApi.ApiResponse{
					Status: dtoApi.ERROR,
					Error: []dtoApi.ApiError{
						{
							Code:    "UNAUTHORIZED",
							Message: "Unauthorized",
						},
					},
				})
				return
			}

			ctx := context.WithValue(r.Context(), "userID", userID)
			ctx = context.WithValue(ctx, "sessionID", cookie.Value)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
