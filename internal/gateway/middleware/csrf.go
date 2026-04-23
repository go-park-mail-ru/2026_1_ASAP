package middleware

import (
	"errors"
	"net/http"

	authv1 "github.com/go-park-mail-ru/2026_1_ASAP/gen/go/auth/v1"
	dtoApi "github.com/go-park-mail-ru/2026_1_ASAP/internal/gateway/dto/api"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/utils/csrf"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/utils/response"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	errCSRFSessionInvalid = errors.New("csrf: session invalid")
	errCSRFNotInSession   = errors.New("csrf: token not in session")
	errCSRFExpiredToken   = errors.New("csrf: token expired")
)

func mapGetCSRFGRPCError(err error) error {
	st, ok := status.FromError(err)
	if !ok {
		return err
	}
	switch st.Code() {
	case codes.NotFound:
		return errCSRFSessionInvalid
	case codes.FailedPrecondition:
		switch st.Message() {
		case "session expired":
			return errCSRFSessionInvalid
		case "csrf token not found":
			return errCSRFNotInSession
		case "csrf token expired":
			return errCSRFExpiredToken
		default:
			return err
		}
	default:
		return err
	}
}

func CSRFMiddleware(authClient authv1.AuthClient) func(http.Handler) http.Handler {
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
			resp, err := authClient.GetCSRFToken(r.Context(), &authv1.RequestGetCSRFToken{SessionId: cookie.Value})
			if err != nil {
				err = mapGetCSRFGRPCError(err)
				if errors.Is(err, errCSRFNotInSession) {
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
				if errors.Is(err, errCSRFExpiredToken) {
					newCsrfToken, tokenErr := csrf.GenerateToken()
					if tokenErr != nil {
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
					_, setErr := authClient.SetCSRFToken(r.Context(), &authv1.RequestSetCSRFToken{
						SessionId: cookie.Value,
						CsrfToken: newCsrfToken,
					})
					if setErr != nil {
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
				}
				if errors.Is(err, errCSRFSessionInvalid) {
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
			csrfTokenOrigin := resp.GetCsrfToken()
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
