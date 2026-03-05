package handlers

import (
	"net/http"

	authService "github.com/go-park-mail-ru/2026_1_ASAP/internal/services/auth"
)

type AuthHandler struct {
	AuthService authService.UserServiceInterface
}

func NewAuthHandler(authService authService.UserServiceInterface) *AuthHandler {
	return &AuthHandler{AuthService: authService}
}

func (authHeandler *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello from login page"))
}

func (authHeandler *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello from register page"))
}

func (authHeandler *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello from logout page"))
}

func (authHeandler *AuthHandler) Root(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello from root page"))
}
