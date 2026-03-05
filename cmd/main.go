package main

import (
	"net/http"

	"github.com/go-chi/chi"

	authHandlers "github.com/go-park-mail-ru/2026_1_ASAP/internal/handlers/auth"
)

func main() {
	mux := chi.NewRouter()

	auth := authHandlers.NewAuthHandler()

	mux.Route("/api/v1/auth", func(mux chi.Router) {
		mux.Post("/login", auth.Login)
		mux.Post("/logout", auth.Logout)
		mux.Post("/register", auth.Register)
		mux.Get("/", auth.Root)
	})
	http.ListenAndServe(":8080", mux)
}
