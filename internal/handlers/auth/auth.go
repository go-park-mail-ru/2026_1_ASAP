package handlers

import (
	"net/http"

	userService "github.com/go-park-mail-ru/2026_1_ASAP/internal/services/auth"
)

type AuthHandler struct {
	AuthService userService.UserServiceInterface
}

func NewAuthHandler() *AuthHandler {
	return &AuthHandler{}
}

func (api *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello from login page"))
}

func (api *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello from register page"))
}

func (api *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello from logout page"))
}

func (api *AuthHandler) Root(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello from root page"))
}
