package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	dtoApi "github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/api"
	dtoAuth "github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/auth"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/middleware"
	authService "github.com/go-park-mail-ru/2026_1_ASAP/internal/services/auth"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/utils/mapper"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/utils/response"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/utils/validation"
	"github.com/google/uuid"
)

type AuthHandler struct {
	AuthService authService.AuthServiceInterface
}

func NewAuthHandler(authService authService.AuthServiceInterface) *AuthHandler {
	return &AuthHandler{AuthService: authService}
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

	decoder := json.NewDecoder(r.Body)
	newRequestLogin := new(dtoAuth.RequestLogin)

	err := decoder.Decode(newRequestLogin)
	if err != nil {
		resp := dtoApi.ApiErrorResponse{
			Status: dtoApi.ERROR,
			Errors: []dtoApi.ApiError{
				{
					Code:    "INVALID_JSON",
					Message: "Invalid request body",
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
			Status: dtoApi.ERROR,
			Errors: apiErrors,
		}

		response.Send(w, http.StatusBadRequest, resp)
		return
	}

	session, err := authHandler.AuthService.Login(newRequestLogin)
	if err != nil {
		resp := dtoApi.ApiErrorResponse{
			Status: dtoApi.ERROR,
			Errors: []dtoApi.ApiError{
				{
					Code:    "INVALID_CREDENTIALS",
					Message: "Invalid credentials",
				},
			},
		}
		response.Send(w, http.StatusUnauthorized, resp)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    session.SessionID,
		Path:     "/",
		HttpOnly: true,
		Expires:  session.Expire,
	})
	resp := dtoApi.ApiSucessResponse[dtoAuth.ResponseLoginSuccess]{
		Status: dtoApi.SUCCESS,
		Body: dtoAuth.ResponseLoginSuccess{
			Login: newRequestLogin.Login,
		},
	}

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

	decoder := json.NewDecoder(r.Body)
	newRequestRegister := new(dtoAuth.RequestRegistrate)

	err := decoder.Decode(newRequestRegister)
	if err != nil {
		resp := dtoApi.ApiErrorResponse{
			Status: dtoApi.ERROR,
			Errors: []dtoApi.ApiError{
				{
					Code:    "INVALID_JSON",
					Message: "Invalid request body",
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
			Status: dtoApi.ERROR,
			Errors: apiErrors,
		}

		response.Send(w, http.StatusBadRequest, resp)
		return
	}

	errs = authHandler.AuthService.Register(newRequestRegister)
	if len(errs) > 0 {
		apiErrors := mapper.MapValidationErrorsToApiErrors(errs)
		resp := dtoApi.ApiErrorResponse{
			Status: dtoApi.ERROR,
			Errors: apiErrors,
		}

		response.Send(w, http.StatusConflict, resp)
		return
	}

	resp := dtoApi.ApiSucessResponse[dtoAuth.ResponseRegisterSuccess]{
		Status: dtoApi.SUCCESS,
		Body: dtoAuth.ResponseRegisterSuccess{
			Email: newRequestRegister.Email,
			Login: newRequestRegister.Login,
		},
	}
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
	sessionID, ok := r.Context().Value(middleware.SessionID).(string)
	if !ok {
		log.Println("here")
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
	newRequestLogout := &dtoAuth.RequestLogout{
		SessionID: sessionID,
	}
	err := authHandler.AuthService.Logout(newRequestLogout)
	if err != nil {
		resp := dtoApi.ApiErrorResponse{
			Status: dtoApi.ERROR,
			Errors: []dtoApi.ApiError{
				{
					Code:    "FAIL_LOGOUT",
					Message: "Failed to logout",
				},
			},
		}
		response.Send(w, http.StatusInternalServerError, resp)
		return
	}
	resp := dtoApi.ApiSucessResponse[dtoAuth.ResponseLogoutSuccess]{
		Status: dtoApi.SUCCESS,
		Body:   dtoAuth.ResponseLogoutSuccess{},
	}

	response.Send(w, http.StatusOK, resp)
}

func (authHandler *AuthHandler) Root(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(uuid.UUID)
	if !ok {
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
	w.Write([]byte(userID.String()))
}
