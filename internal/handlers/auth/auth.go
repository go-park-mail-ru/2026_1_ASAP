package handlers

import (
	"encoding/json"
	"net/http"

	dtoApi "github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/api"
	dtoAuth "github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/auth"
	authService "github.com/go-park-mail-ru/2026_1_ASAP/internal/services/auth"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/utils/mapper"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/utils/response"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/utils/validation"
)

type AuthHandler struct {
	AuthService authService.AuthServiceInterface
}

func NewAuthHandler(authService authService.AuthServiceInterface) *AuthHandler {
	return &AuthHandler{AuthService: authService}
}

func (authHandler *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello from login page"))
}

func (authHandler *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	decoder := json.NewDecoder(r.Body)
	newRequestRegister := new(dtoAuth.RequestRegistrate)

	err := decoder.Decode(newRequestRegister)
	if err != nil {
		resp := dtoApi.ApiResponse{
			Status: dtoApi.ERROR,
			Error: []dtoApi.ApiError{
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
		resp := dtoApi.ApiResponse{
			Status: dtoApi.ERROR,
			Error:  apiErrors,
		}

		response.Send(w, http.StatusBadRequest, resp)
		return
	}

	errs = authHandler.AuthService.Register(newRequestRegister)
	if len(errs) > 0 {
		apiErrors := mapper.MapValidationErrorsToApiErrors(errs)
		resp := dtoApi.ApiResponse{
			Status: dtoApi.ERROR,
			Error:  apiErrors,
		}

		response.Send(w, http.StatusConflict, resp)
		return
	}

	resp := dtoApi.ApiResponse{
		Status: dtoApi.SUCCESS,
		Body: dtoAuth.ResponseRegisterSuccess{
			Email: newRequestRegister.Email,
			Login: newRequestRegister.Login,
		},
	}
	response.Send(w, http.StatusOK, resp)
}

func (authHandler *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello from logout page"))
}

func (authHandler *AuthHandler) Root(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello from root page"))
}
