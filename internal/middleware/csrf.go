package middleware

import (
	"context"
	"errors"
	"net/http"

	domain "github.com/go-park-mail-ru/2026_1_ASAP/internal/domain/session"
	dtoApi "github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/api"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/utils/csrf"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/utils/response"
)

//go:generate mockgen -source=csrf.go -destination=mock/csrf_token_service_mock.go -package=mock
type CSRFTokenService interface {
	GetCSRFToken(ctx context.Context, sessionID string) (string, error)
	SetCSRFToken(ctx context.Context, sessionID string, token string) error
}

func CSRFMiddleware(CSRFTokenService CSRFTokenService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			csrfToken := r.Header.Get("X-CSRF-TOKEN")
			if csrfToken == "" {
				response.Send(w, http.StatusForbidden, dtoApi.ApiErrorResponse{
					Status: dtoApi.Error,
					Errors: []dtoApi.ApiError{
						{
							Code:    dtoApi.CSRFTokenMissing,
							Message: dtoApi.CSRFTokenMissingMsg,
						},
					},
				})
				return
			}
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
			csrfTokenOrigin, err := CSRFTokenService.GetCSRFToken(r.Context(), cookie.Value)
			if err != nil {
				if errors.Is(err, domain.ErrCSRFNotFound) {
					response.Send(w, http.StatusForbidden, dtoApi.ApiErrorResponse{
						Status: dtoApi.Error,
						Errors: []dtoApi.ApiError{
							{
								Code:    dtoApi.CSRFTokenNotInSession,
								Message: dtoApi.CSRFTokenNotInSessionMsg,
							},
						},
					})
					return
				}
				if errors.Is(err, domain.ErrCSRFExpired) {
					newCsrfToken, err := csrf.GenerateToken()
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
					if err := CSRFTokenService.SetCSRFToken(r.Context(), cookie.Value, newCsrfToken); err != nil {
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

					w.Header().Set("X-NEW-CSRF-TOKEN", newCsrfToken)
					response.Send(w, http.StatusForbidden, dtoApi.ApiErrorResponse{
						Status: dtoApi.Error,
						Errors: []dtoApi.ApiError{
							{
								Code:    dtoApi.CSRFTokenExpired,
								Message: dtoApi.CSRFTokenExpiredMsg,
							},
						},
					})
					return
				} else if errors.Is(err, domain.ErrNotFound) || errors.Is(err, domain.ErrExpired) {
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
			if csrfTokenOrigin != csrfToken {
				response.Send(w, http.StatusForbidden, dtoApi.ApiErrorResponse{
					Status: dtoApi.Error,
					Errors: []dtoApi.ApiError{
						{
							Code:    dtoApi.CSRFTokenMismatch,
							Message: dtoApi.CSRFTokenMismatchMsg,
						},
					},
				})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
