package main

import (
	"log"
	"net/http"
	"time"

	"github.com/go-chi/cors"

	"github.com/go-chi/chi"
	httpSwagger "github.com/swaggo/http-swagger"

	config "github.com/go-park-mail-ru/2026_1_ASAP/config"
	_ "github.com/go-park-mail-ru/2026_1_ASAP/docs"
	authHandlers "github.com/go-park-mail-ru/2026_1_ASAP/internal/handlers/auth"
	chatHandlers "github.com/go-park-mail-ru/2026_1_ASAP/internal/handlers/chat"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/middleware"
	chatRepository "github.com/go-park-mail-ru/2026_1_ASAP/internal/repository/chat"
	sessionRepository "github.com/go-park-mail-ru/2026_1_ASAP/internal/repository/sessions"
	userRepository "github.com/go-park-mail-ru/2026_1_ASAP/internal/repository/user"
	authService "github.com/go-park-mail-ru/2026_1_ASAP/internal/services/auth"
	chatService "github.com/go-park-mail-ru/2026_1_ASAP/internal/services/chat"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/services/session"
)

// @title API Pulse App
// @version 1.0
// @description API веб-приложения Pulse
// @host pulseapp.space:8080
func main() {
	config, err := config.LoadConfigFromEnv()
	if err != nil {
		log.Fatalln(err.Error())
	}

	mux := chi.NewRouter()

	mux.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{
			"http://pulseapp.space",
			"http://pulseapp.space:8080",
			"http://212.233.96.180",
			"http://localhost",
			"http://localhost:8080",
		},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	sessionRepository := sessionRepository.NewSessionRepository()
	sessionService := session.NewSessionService(sessionRepository, config.SessionConfig.SessionTTL)
	userRepository := userRepository.NewMockUserRepository()
	chatRepository := chatRepository.NewMockRepository()
	userService := authService.NewAuthService(userRepository, sessionService)
	chatService := chatService.NewChatService(chatRepository, userRepository)
	auth := authHandlers.NewAuthHandler(userService)
	chatsHandler := chatHandlers.NewChatHandler(chatService)
	authMiddleware := middleware.AuthMiddleware(sessionService)

	mux.Route("/api/v1/auth", func(mux chi.Router) {
		mux.Post("/login", auth.Login)
		mux.Post("/register", auth.Register)
		mux.With(authMiddleware).Post("/logout", auth.Logout)
	})

	mux.Route("/api/v1/chats", func(mux chi.Router) {
		mux.With(authMiddleware).Get("/", chatsHandler.GetChats)
		mux.With(authMiddleware).Post("/", chatsHandler.ChatCreate)
		mux.With(authMiddleware).Get("/{id}", chatsHandler.GetChatByID)
	})

	mux.Get("/swagger/*", httpSwagger.Handler())

	log.Printf("Server started at %s\n", config.ServerConfig.ServerInfo())

	server := &http.Server{
		Addr:         config.ServerConfig.ServerInfo(),
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
	err = server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server failed: %v", err)
	}
}
