package main

import (
	"net/http"

	"github.com/go-chi/chi"

	AuthHandlers "github.com/go-park-mail-ru/2026_1_ASAP/internal/handlers/auth"
)

func main() {
	mux := chi.NewRouter()

	auth := AuthHandlers.NewAuthHandler()

	mux.HandleFunc("/login", auth.Login)
	mux.HandleFunc("/logout", auth.Logout)
	mux.HandleFunc("/register", auth.Register)
	mux.HandleFunc("/", auth.Root)

	http.ListenAndServe(":8080", mux)
}
