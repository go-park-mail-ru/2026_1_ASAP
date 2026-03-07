package handlers

import (
	"encoding/json"
	"net/http"

	dtoAuth "github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/auth"
	authService "github.com/go-park-mail-ru/2026_1_ASAP/internal/services/auth"
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
		// TODO: Обработать ошибку
		return
	}

	authHandler.AuthService.Register(newRequestRegister)

	// TODO: Написать функцию отдачи ответов
	w.Write([]byte("Hello from register page"))
}

func (authHandler *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello from logout page"))
}

func (authHandler *AuthHandler) Root(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello from root page"))
}
