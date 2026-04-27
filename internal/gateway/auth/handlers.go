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
	"github.com/go-park-mail-ru/2026_1_ASAP/pkg/grpcerr"
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

	newRequestLogin := new(auth2.RequestLogin)
	if err := json.NewDecoder(r.Body).Decode(newRequestLogin); err != nil {
		response.Send(w, http.StatusBadRequest, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{Code: dtoApi.InvalidJson, Message: dtoApi.InvalidJsonMsg}},
		})
		return
	}

	if errs := validation.ValidationRequestLogin(newRequestLogin); len(errs) > 0 {
		response.Send(w, http.StatusBadRequest, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: mapper.MapValidationErrorsToApiErrors(errs),
		})
		return
	}

	session, err := h.AuthService.Login(ctx, &authv1.RequestLogin{
		Login:    newRequestLogin.Login,
		Password: newRequestLogin.Password,
	})
	if err != nil {
		_, appCode, _ := grpcerr.Error(err)
		switch authv1.AuthErrorCode(appCode) {
		case authv1.AuthErrorCode_AUTH_ERROR_INVALID_CREDENTIALS:
			response.Send(w, http.StatusUnauthorized, dtoApi.ApiErrorResponse{
				Status: dtoApi.Error,
				Errors: []dtoApi.ApiError{{Code: dtoApi.InvalidCredentials, Message: dtoApi.InvalidCredentialsMsg}},
			})
		case authv1.AuthErrorCode_AUTH_ERROR_INVALID_INPUT:
			response.Send(w, http.StatusBadRequest, dtoApi.ApiErrorResponse{
				Status: dtoApi.Error,
				Errors: []dtoApi.ApiError{{Code: dtoApi.InvalidJson, Message: dtoApi.InvalidJsonMsg}},
			})
		default:
			response.Send(w, http.StatusInternalServerError, dtoApi.ApiErrorResponse{
				Status: dtoApi.Error,
				Errors: []dtoApi.ApiError{{Code: dtoApi.InternalError, Message: dtoApi.InternalErrorMsg}},
			})
		}
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
		Body:   auth2.ResponseLoginSuccess{Login: session.GetLogin()},
	})
}

func (h *GatewayAuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	ctx := r.Context()

	newRequestRegister := new(auth2.RequestRegistrate)
	if err := json.NewDecoder(r.Body).Decode(newRequestRegister); err != nil {
		response.Send(w, http.StatusBadRequest, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{Code: dtoApi.InvalidJson, Message: dtoApi.InvalidJsonMsg}},
		})
		return
	}

	if errs := validation.ValidationRequestRegistrate(newRequestRegister); len(errs) > 0 {
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
		_, appCode, _ := grpcerr.Error(err)
		switch authv1.AuthErrorCode(appCode) {
		case authv1.AuthErrorCode_AUTH_ERROR_LOGIN_ALREADY_EXISTS:
			response.Send(w, http.StatusConflict, dtoApi.ApiErrorResponse{
				Status: dtoApi.Error,
				Errors: []dtoApi.ApiError{{Code: dtoApi.LoginAlreadyRegistered, Message: dtoApi.LoginAlreadyRegisteredMsg}},
			})
		case authv1.AuthErrorCode_AUTH_ERROR_EMAIL_ALREADY_EXISTS:
			response.Send(w, http.StatusConflict, dtoApi.ApiErrorResponse{
				Status: dtoApi.Error,
				Errors: []dtoApi.ApiError{{Code: dtoApi.EmailAlreadyRegistered, Message: dtoApi.EmailAlreadyRegisteredMsg}},
			})
		case authv1.AuthErrorCode_AUTH_ERROR_INVALID_INPUT:
			response.Send(w, http.StatusBadRequest, dtoApi.ApiErrorResponse{
				Status: dtoApi.Error,
				Errors: []dtoApi.ApiError{{Code: dtoApi.InvalidJson, Message: dtoApi.InvalidJsonMsg}},
			})
		default:
			response.Send(w, http.StatusInternalServerError, dtoApi.ApiErrorResponse{
				Status: dtoApi.Error,
				Errors: []dtoApi.ApiError{{Code: dtoApi.InternalError, Message: dtoApi.InternalErrorMsg}},
			})
		}
		return
	}

	response.Send(w, http.StatusOK, dtoApi.ApiSuccessResponse[auth2.ResponseRegisterSuccess]{
		Status: dtoApi.Success,
		Body:   auth2.ResponseRegisterSuccess{Login: resp.GetLogin(), Email: resp.GetEmail()},
	})
}

func (h *GatewayAuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	cookie, err := r.Cookie("session_id")
	if err != nil || cookie.Value == "" {
		response.Send(w, http.StatusUnauthorized, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{Code: dtoApi.Unauthorized, Message: dtoApi.UnauthorizedMsg}},
		})
		return
	}

	_, err = h.AuthService.Logout(ctx, &authv1.RequestLogout{SessionId: cookie.Value})
	if err != nil {
		_, appCode, _ := grpcerr.Error(err)
		switch authv1.AuthErrorCode(appCode) {
		case authv1.AuthErrorCode_AUTH_ERROR_SESSION_NOT_FOUND:
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

	response.Send(w, http.StatusOK, dtoApi.ApiSuccessResponse[auth2.ResponseLogoutSuccess]{
		Status: dtoApi.Success,
		Body:   auth2.ResponseLogoutSuccess{},
	})
}
