package handlers

import (
	"net/http"
)

type AuthHandler struct {
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
