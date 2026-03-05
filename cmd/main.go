package main

import (
	"net/http"

	"github.com/go-chi/chi"

	"log"

	config "github.com/go-park-mail-ru/2026_1_ASAP/config"
	authHandlers "github.com/go-park-mail-ru/2026_1_ASAP/internal/handlers/auth"
)

func main() {
	config, err := config.LoadConfigFromEnv()
	if err != nil {
		log.Fatalln(err.Error())
	}

	mux := chi.NewRouter()

	auth := authHandlers.NewAuthHandler()

	mux.Route("/api/v1/auth", func(mux chi.Router) {
		mux.Post("/login", auth.Login)
		mux.Post("/logout", auth.Logout)
		mux.Post("/register", auth.Register)
		mux.Get("/", auth.Root)
	})

	log.Printf("Server started at %s\n", config.ServerConfig.ServerInfo())
	err = http.ListenAndServe(config.ServerConfig.ServerInfo(), mux)
	if err != nil {
		log.Fatalf("Server failed: %w", err)
	}
}
