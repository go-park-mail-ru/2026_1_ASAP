package auth

import (
	"encoding/json"
	"net/http"

	authv1 "github.com/go-park-mail-ru/2026_1_ASAP/gen/go/auth/v1"
	auth2 "github.com/go-park-mail-ru/2026_1_ASAP/internal/auth/dto/auth"
	dtoApi "github.com/go-park-mail-ru/2026_1_ASAP/internal/gateway/dto/api"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/gateway/mapper"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/utils/response"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/utils/validation"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type GatewayAuthHandler struct {
	AuthService authv1.AuthClient
}

func NewGatewayAuthHandler(authService authv1.AuthClient) *GatewayAuthHandler {
	return &GatewayAuthHandler{
		AuthService: authService,
	}
}

func (h *GatewayAuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	ctx := r.Context()

	decoder := json.NewDecoder(r.Body)
	newRequestLogin := new(auth2.RequestLogin)

	err := decoder.Decode(newRequestLogin)
	if err != nil {
		resp := dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{
				{
					Code:    dtoApi.InvalidJson,
					Message: dtoApi.InvalidJsonMsg,
				},
			},
		}
		response.Send(w, http.StatusBadRequest, resp)
		return
	}

	errs := validation.ValidationRequestLogin(newRequestLogin)
	if len(errs) > 0 {
		apiErrors := mapper.MapValidationErrorsToApiErrors(errs)
		resp := dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: apiErrors,
		}

		response.Send(w, http.StatusBadRequest, resp)
		return
	}

	session, err := h.AuthService.Login(ctx, &authv1.RequestLogin{
		Login:    newRequestLogin.Login,
		Password: newRequestLogin.Password,
	})
	if err != nil {
		st, ok := status.FromError(err)
		if ok {
			switch st.Code() {
			case codes.Unauthenticated, codes.NotFound:
				response.Send(w, http.StatusUnauthorized, dtoApi.ApiErrorResponse{
					Status: dtoApi.Error,
					Errors: []dtoApi.ApiError{{
						Code:    dtoApi.InvalidCredentials,
						Message: dtoApi.InvalidCredentialsMsg,
					}},
				})
				return
			case codes.InvalidArgument:
				response.Send(w, http.StatusBadRequest, dtoApi.ApiErrorResponse{
					Status: dtoApi.Error,
					Errors: []dtoApi.ApiError{{
						Code:    dtoApi.InvalidJson,
						Message: dtoApi.InvalidJsonMsg,
					}},
				})
				return
			}
		}

		response.Send(w, http.StatusInternalServerError, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{
				Code:    dtoApi.InternalError,
				Message: dtoApi.InternalErrorMsg,
			}},
		})
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    session.GetSession().GetSessionId(),
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   true,
	})

	w.Header().Set("X-NEW-CSRF-TOKEN", session.GetSession().GetCsrfToken())

	response.Send(w, http.StatusOK, dtoApi.ApiSuccessResponse[auth2.ResponseLoginSuccess]{
		Status: dtoApi.Success,
		Body: auth2.ResponseLoginSuccess{
			Login: session.GetLogin(),
		},
	})
}

func (h *GatewayAuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	ctx := r.Context()

	decoder := json.NewDecoder(r.Body)
	newRequestRegister := new(auth2.RequestRegistrate)

	err := decoder.Decode(newRequestRegister)
	if err != nil {
		response.Send(w, http.StatusBadRequest, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{
				Code:    dtoApi.InvalidJson,
				Message: dtoApi.InvalidJsonMsg,
			}},
		})
		return
	}

	errs := validation.ValidationRequestRegistrate(newRequestRegister)
	if len(errs) > 0 {
		response.Send(w, http.StatusBadRequest, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: mapper.MapValidationErrorsToApiErrors(errs),
		})
		return
	}

	resp, err := h.AuthService.Register(ctx, &authv1.RequestRegister{
		Login:    newRequestRegister.Login,
		Email:    newRequestRegister.Email,
		Password: newRequestRegister.Password,
	})
	if err != nil {
		st, ok := status.FromError(err)
		if ok {
			switch st.Code() {
			case codes.AlreadyExists:
				response.Send(w, http.StatusConflict, dtoApi.ApiErrorResponse{
					Status: dtoApi.Error,
					Errors: []dtoApi.ApiError{{
						Code:    dtoApi.LoginAlreadyRegistered,
						Message: dtoApi.LoginAlreadyRegisteredMsg,
					}},
				})
				return
			case codes.InvalidArgument:
				response.Send(w, http.StatusBadRequest, dtoApi.ApiErrorResponse{
					Status: dtoApi.Error,
					Errors: []dtoApi.ApiError{{
						Code:    dtoApi.InvalidJson,
						Message: dtoApi.InvalidJsonMsg,
					}},
				})
				return
			}
		}

		response.Send(w, http.StatusInternalServerError, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{
				Code:    dtoApi.InternalError,
				Message: dtoApi.InternalErrorMsg,
			}},
		})
		return
	}

	response.Send(w, http.StatusOK, dtoApi.ApiSuccessResponse[auth2.ResponseRegisterSuccess]{
		Status: dtoApi.Success,
		Body: auth2.ResponseRegisterSuccess{
			Login: resp.GetLogin(),
			Email: resp.GetEmail(),
		},
	})
}

func (h *GatewayAuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	cookie, err := r.Cookie("session_id")
	if err != nil || cookie.Value == "" {
		response.Send(w, http.StatusUnauthorized, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{
				Code:    dtoApi.Unauthorized,
				Message: dtoApi.UnauthorizedMsg,
			}},
		})
		return
	}

	_, err = h.AuthService.Logout(ctx, &authv1.RequestLogout{
		SessionId: cookie.Value,
	})
	if err != nil {
		st, ok := status.FromError(err)
		if ok {
			switch st.Code() {
			case codes.Unauthenticated, codes.NotFound:
				response.Send(w, http.StatusUnauthorized, dtoApi.ApiErrorResponse{
					Status: dtoApi.Error,
					Errors: []dtoApi.ApiError{{
						Code:    dtoApi.Unauthorized,
						Message: dtoApi.UnauthorizedMsg,
					}},
				})
				return
			}
		}

		response.Send(w, http.StatusInternalServerError, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{
				Code:    dtoApi.InternalError,
				Message: dtoApi.InternalErrorMsg,
			}},
		})
		return
	}

	response.Send(w, http.StatusOK, dtoApi.ApiSuccessResponse[auth2.ResponseLogoutSuccess]{
		Status: dtoApi.Success,
		Body:   auth2.ResponseLogoutSuccess{},
	})
}
