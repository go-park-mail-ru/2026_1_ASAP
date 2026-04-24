package middleware

import (
	"context"
	"net/http"

	authv1 "github.com/go-park-mail-ru/2026_1_ASAP/gen/go/auth/v1"
	dtoApi "github.com/go-park-mail-ru/2026_1_ASAP/internal/gateway/dto/api"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/utils/response"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

//go:generate go run github.com/golang/mock/mockgen@v1.6.0 -source=auth.go -destination=mock/session_service_mock.go -package=mock

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
				st, ok := status.FromError(err)
				if ok {
					if st.Code() == codes.NotFound || st.Code() == codes.FailedPrecondition {
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
