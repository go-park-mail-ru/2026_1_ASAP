package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/go-park-mail-ru/2026_1_ASAP/config"
	domainSession "github.com/go-park-mail-ru/2026_1_ASAP/internal/domain/session"
	domainUser "github.com/go-park-mail-ru/2026_1_ASAP/internal/domain/user"
	dtoApi "github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/api"
	dtoAuth "github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/auth"
	dtoSession "github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/session"
	dtoVK "github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/vkid"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/middleware"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/utils/loggerctx"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/utils/mapper"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/utils/response"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/utils/sanitize"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/utils/validation"
)

//go:generate go run github.com/golang/mock/mockgen@v1.6.0 -source=auth.go -destination=mock/auth_mock.go -package=mock
type AuthService interface {
	Register(ctx context.Context, request *dtoAuth.RequestRegistrate) (*dtoSession.SessionDTO, error)
	Login(ctx context.Context, request *dtoAuth.RequestLogin) (*dtoSession.SessionDTO, error)
	Logout(ctx context.Context, request *dtoAuth.RequestLogout) error
	AuthWithVKID(ctx context.Context, request *dtoVK.RequestAuth) (*dtoSession.SessionDTO, error)
}

type AuthHandler struct {
	AuthService AuthService
	VKIDConfig  config.VKIDConfig
	logger      *zap.Logger
}

func NewAuthHandler(authService AuthService, vkCfg config.VKIDConfig, logger *zap.Logger) *AuthHandler {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &AuthHandler{AuthService: authService, VKIDConfig: vkCfg, logger: logger.Named("vkid")}
}

func (authHandler *AuthHandler) Log(ctx context.Context) *zap.Logger {
	base := authHandler.logger
	if base == nil {
		return zap.NewNop()
	}
	return loggerctx.EnrichLoggerFromContext(ctx, base)
}

// Login godoc
// @Summary Логин пользователя
// @Description Аутентификация юзера с логином и паролем
// @Tags auth
// @Accept json
// @Produce json
// @Param request body dtoAuth.RequestLogin true "Запрос на логин"
// @Success 200 {object} dtoApi.ResponseLoginSuccessForSwagger
// @Failure 400 {object} dtoApi.ApiErrorResponse "Неправильный запрос или ошибка формата полей"
// @Failure 401 {object} dtoApi.ApiErrorResponse "Некорректные данные логины"
// @Router /api/v1/auth/login [post]
func (authHandler *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	ctx := r.Context()

	decoder := json.NewDecoder(r.Body)
	newRequestLogin := new(dtoAuth.RequestLogin)

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

	session, err := authHandler.AuthService.Login(ctx, newRequestLogin)
	if err != nil {
		if errors.Is(err, domainUser.ErrInvalidCredentials) {
			resp := dtoApi.ApiErrorResponse{
				Status: dtoApi.Error,
				Errors: []dtoApi.ApiError{
					{
						Code:    dtoApi.InvalidCredentials,
						Message: dtoApi.InvalidCredentialsMsg,
					},
				},
			}
			response.Send(w, http.StatusUnauthorized, resp)
			return
		}

		switch {
		case errors.Is(err, domainUser.ErrInvalidCredentials):

		case errors.Is(err, domainUser.ErrNotFound):
			resp := dtoApi.ApiErrorResponse{
				Status: dtoApi.Error,
				Errors: []dtoApi.ApiError{
					{
						Code:    dtoApi.InvalidCredentials,
						Message: dtoApi.InvalidCredentialsMsg,
					},
				},
			}
			response.Send(w, http.StatusUnauthorized, resp)
			return
		}
		resp := dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{
				{
					Code:    dtoApi.InternalError,
					Message: dtoApi.InternalErrorMsg,
				},
			},
		}
		authHandler.Log(ctx).Error("login: internal error", zap.Error(err))
		response.Send(w, http.StatusInternalServerError, resp)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    session.SessionID,
		Path:     "/",
		HttpOnly: true,
		Expires:  session.Expire,
		SameSite: http.SameSiteLaxMode,
		Secure:   true,
	})
	resp := dtoApi.ApiSuccessResponse[dtoAuth.ResponseLoginSuccess]{
		Status: dtoApi.Success,
		Body: dtoAuth.ResponseLoginSuccess{
			Login: sanitize.Text(newRequestLogin.Login),
		},
	}
	w.Header().Set("X-NEW-CSRF-TOKEN", session.CSRFToken)
	response.Send(w, http.StatusOK, resp)
}

// Register godoc
// @Summary Регистрация пользователя
// @Description Создаёт нового пользователя с email и login
// @Tags auth
// @Accept json
// @Produce json
// @Param request body dtoAuth.RequestRegistrate true "Запрос на регистрацию"
// @Success 200 {object} dtoApi.ResponseRegisterSuccessForSwagger
// @Failure 400 {object} dtoApi.ApiErrorResponse "Неправильный запрос или ошибка формата полей"
// @Failure 409 {object} dtoApi.ApiErrorResponse "пользователь уже существует"
// @Router /api/v1/auth/register [post]
func (authHandler *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	ctx := r.Context()

	decoder := json.NewDecoder(r.Body)
	newRequestRegister := new(dtoAuth.RequestRegistrate)

	err := decoder.Decode(newRequestRegister)
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

	errs := validation.ValidationRequestRegistrate(newRequestRegister)
	if len(errs) > 0 {
		apiErrors := mapper.MapValidationErrorsToApiErrors(errs)
		resp := dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: apiErrors,
		}

		response.Send(w, http.StatusBadRequest, resp)
		return
	}

	session, err := authHandler.AuthService.Register(ctx, newRequestRegister)
	if err != nil {
		switch {
		case errors.Is(err, domainUser.ErrLoginAlreadyExists):
			resp := dtoApi.ApiErrorResponse{
				Status: dtoApi.Error,
				Errors: []dtoApi.ApiError{
					{
						Code:    dtoApi.LoginAlreadyRegistered,
						Message: dtoApi.LoginAlreadyRegisteredMsg,
					},
				},
			}
			response.Send(w, http.StatusConflict, resp)
			return
		case errors.Is(err, domainUser.ErrEmailAlreadyExists):
			resp := dtoApi.ApiErrorResponse{
				Status: dtoApi.Error,
				Errors: []dtoApi.ApiError{
					{
						Code:    dtoApi.EmailAlreadyRegistered,
						Message: dtoApi.EmailAlreadyRegisteredMsg,
					},
				},
			}
			response.Send(w, http.StatusConflict, resp)
			return
		}
		resp := dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{
				{
					Code:    dtoApi.InternalError,
					Message: dtoApi.InternalErrorMsg,
				},
			},
		}
		authHandler.Log(ctx).Error("register: internal error", zap.Error(err))
		response.Send(w, http.StatusInternalServerError, resp)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    session.SessionID,
		Path:     "/",
		HttpOnly: true,
		Expires:  session.Expire,
		SameSite: http.SameSiteLaxMode,
		Secure:   true,
	})

	resp := dtoApi.ApiSuccessResponse[dtoAuth.ResponseRegisterSuccess]{
		Status: dtoApi.Success,
		Body: dtoAuth.ResponseRegisterSuccess{
			Email: newRequestRegister.Email,
			Login: sanitize.Text(newRequestRegister.Login),
		},
	}
	w.Header().Set("X-NEW-CSRF-TOKEN", session.CSRFToken)
	response.Send(w, http.StatusOK, resp)
}

// Logout godoc
// @Summary Выход пользователя
// @Description Завершение сессии пользователя
// @Tags auth
// @Accept json
// @Produce json
// @Success 200 {object} dtoApi.ResponseLogoutSuccessForSwagger
// @Failure 401 {object} dtoApi.ApiErrorResponse "Неавторизован"
// @Failure 500 {object} dtoApi.ApiErrorResponse "Ошибка выхода"
// @Router /api/v1/auth/logout [post]
func (authHandler *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sessionID, ok := r.Context().Value(middleware.SessionID).(string)
	if !ok {
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
	newRequestLogout := &dtoAuth.RequestLogout{
		SessionID: sessionID,
	}
	err := authHandler.AuthService.Logout(ctx, newRequestLogout)
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
		resp := dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{
				{
					Code:    dtoApi.InternalError,
					Message: dtoApi.InternalErrorMsg,
				},
			},
		}
		response.Send(w, http.StatusInternalServerError, resp)
		return
	}
	resp := dtoApi.ApiSuccessResponse[dtoAuth.ResponseLogoutSuccess]{
		Status: dtoApi.Success,
		Body:   dtoAuth.ResponseLogoutSuccess{},
	}

	response.Send(w, http.StatusOK, resp)
}

func (authHandler *AuthHandler) VkIDLogin(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := authHandler.Log(ctx)

	var request dtoVK.RequestVKID
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		log.Warn("vkid: invalid request json", zap.Error(err))
		response.Send(w, http.StatusBadRequest, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{
				{
					Code:    dtoApi.InvalidJson,
					Message: dtoApi.InvalidJsonMsg,
				},
			},
		})
		return
	}

	queryParams := url.Values{
		"grant_type":    []string{"authorization_code"},
		"code_verifier": []string{request.CodeVerifier},
		"redirect_uri":  []string{authHandler.VKIDConfig.RedirectURI},
		"code":          []string{request.Code},
		"client_id":     []string{authHandler.VKIDConfig.ClientID},
		"device_id":     []string{request.DeviceID},
		"state":         []string{request.State},
	}

	tokenCtx, cancelToken := context.WithTimeout(ctx, 5*time.Second)
	defer cancelToken()

	req, err := http.NewRequestWithContext(
		tokenCtx,
		http.MethodPost,
		authHandler.VKIDConfig.AuthURL,
		strings.NewReader(queryParams.Encode()),
	)
	if err != nil {
		log.Error("vkid: build token request", zap.Error(err))
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
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	tokenResp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Error("vkid: token exchange request", zap.Error(err))
		response.Send(w, http.StatusBadGateway, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{
				{
					Code:    dtoApi.VKIDFailed,
					Message: dtoApi.VKIDFailedMsg,
				},
			},
		})
		return
	}
	defer tokenResp.Body.Close()

	if tokenResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(tokenResp.Body)
		log.Warn("vkid: token exchange failed",
			zap.Int("status", tokenResp.StatusCode),
			zap.String("body", truncateForLog(body, 512)),
		)
		response.Send(w, http.StatusUnauthorized, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{
				{
					Code:    dtoApi.VKIDFailed,
					Message: dtoApi.VKIDFailedMsg,
				},
			},
		})
		return
	}

	var vkIDCallback dtoVK.CallbackResponseFromVKID
	if err := json.NewDecoder(tokenResp.Body).Decode(&vkIDCallback); err != nil {
		log.Error("vkid: decode token response", zap.Error(err))
		response.Send(w, http.StatusBadGateway, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{
				{
					Code:    dtoApi.VKIDFailed,
					Message: dtoApi.VKIDFailedMsg,
				},
			},
		})
		return
	}

	queryParams = url.Values{
		"client_id": []string{authHandler.VKIDConfig.ClientID},
		"id_token":  []string{vkIDCallback.IDToken},
	}

	infoCtx, cancelInfo := context.WithTimeout(ctx, 5*time.Second)
	defer cancelInfo()

	req, err = http.NewRequestWithContext(
		infoCtx,
		http.MethodPost,
		authHandler.VKIDConfig.PublicInfoURL,
		strings.NewReader(queryParams.Encode()),
	)
	if err != nil {
		log.Error("vkid: build public_info request", zap.Error(err))
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
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	infoResp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Error("vkid: public_info request", zap.Error(err))
		response.Send(w, http.StatusBadGateway, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{
				{
					Code:    dtoApi.VKIDFailed,
					Message: dtoApi.VKIDFailedMsg,
				},
			},
		})
		return
	}
	defer infoResp.Body.Close()

	if infoResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(infoResp.Body)
		log.Warn("vkid: public_info non-200",
			zap.Int("status", infoResp.StatusCode),
			zap.String("body", truncateForLog(body, 512)),
		)
		response.Send(w, http.StatusUnauthorized, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{
				{
					Code:    dtoApi.VKIDFailed,
					Message: dtoApi.VKIDFailedMsg,
				},
			},
		})
		return
	}

	publicInfoRaw, err := io.ReadAll(infoResp.Body)
	if err != nil {
		log.Error("vkid: read public_info body", zap.Error(err))
		response.Send(w, http.StatusBadGateway, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{
				{
					Code:    dtoApi.VKIDFailed,
					Message: dtoApi.VKIDFailedMsg,
				},
			},
		})
		return
	}
	log.Debug("vkid: public_info raw", zap.ByteString("body", publicInfoRaw))

	authRequest, err := dtoVK.RequestAuthFromPublicInfoJSON(publicInfoRaw, vkIDCallback.UserID)
	if err != nil {
		log.Error("vkid: parse public_info user payload", zap.Error(err))
		response.Send(w, http.StatusBadGateway, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{
				{
					Code:    dtoApi.VKIDFailed,
					Message: dtoApi.VKIDFailedMsg,
				},
			},
		})
		return
	}

	session, err := authHandler.AuthService.AuthWithVKID(ctx, authRequest)
	if err != nil {
		log.Error("vkid: AuthWithVKID", zap.Error(err))
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

	log.Info("vkid: login ok", zap.String("user_id", session.UserID))

	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    session.SessionID,
		Path:     "/",
		HttpOnly: true,
		Expires:  session.Expire,
		SameSite: http.SameSiteLaxMode,
		Secure:   true,
	})

	res := dtoApi.ApiSuccessResponse[dtoAuth.ResponseLoginSuccess]{
		Status: dtoApi.Success,
		Body: dtoAuth.ResponseLoginSuccess{
			Login: session.UserID,
		},
	}

	w.Header().Set("X-NEW-CSRF-TOKEN", session.CSRFToken)
	response.Send(w, http.StatusOK, res)
}

func truncateForLog(b []byte, max int) string {
	if len(b) <= max {
		return string(b)
	}
	return string(b[:max]) + "…"
}
