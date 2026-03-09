package main

import (
	"net/http"

	"github.com/go-chi/chi"

	"log"

	config "github.com/go-park-mail-ru/2026_1_ASAP/config"
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

func main() {
	config, err := config.LoadConfigFromEnv()
	if err != nil {
		log.Fatalln(err.Error())
	}

	mux := chi.NewRouter()
	sessionRepository := sessionRepository.NewSessionRepository()
	sessionService := session.NewSessionService(sessionRepository, config.SessionConfig.SessionTTL)
	userRepository := userRepository.NewUserRepository()
	chatRepository := chatRepository.NewChatRepository()
	userService := authService.NewAuthService(userRepository, sessionService)
	chatService := chatService.NewChatService(chatRepository)
	auth := authHandlers.NewAuthHandler(userService)
	chatsHandler := chatHandlers.NewChatHandler(chatService)
	authMiddleware := middleware.AuthMiddleware(sessionService)

	mux.Route("/api/v1/auth", func(mux chi.Router) {
		mux.Post("/login", auth.Login)
		mux.Post("/register", auth.Register)
		mux.With(authMiddleware).Post("/logout", auth.Logout)
	})

	mux.Route("api/v1/chats", func(mux chi.Router) {
		mux.With(authMiddleware).Get("/", chatsHandler.GetChats)
		mux.With(authMiddleware).Post("/", chatsHandler.ChatCreate)
		mux.With(authMiddleware).Get("/{id}", chatsHandler.GetChatByID)
	})

	log.Printf("Server started at %s\n", config.ServerConfig.ServerInfo())
	err = http.ListenAndServe(config.ServerConfig.ServerInfo(), mux)
	if err != nil {
		log.Fatalf("Server failed: %w", err)
	}
}
