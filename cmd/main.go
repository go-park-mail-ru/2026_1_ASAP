package main

import (
	"context"
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
	contactHandlers "github.com/go-park-mail-ru/2026_1_ASAP/internal/handlers/contacts"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/middleware"
	chatRepository "github.com/go-park-mail-ru/2026_1_ASAP/internal/repository/chat"
	contactRepository "github.com/go-park-mail-ru/2026_1_ASAP/internal/repository/contacts"
	sessionRepository "github.com/go-park-mail-ru/2026_1_ASAP/internal/repository/sessions"
	userRepository "github.com/go-park-mail-ru/2026_1_ASAP/internal/repository/user"
	authService "github.com/go-park-mail-ru/2026_1_ASAP/internal/services/auth"
	chatService "github.com/go-park-mail-ru/2026_1_ASAP/internal/services/chat"
	contactService "github.com/go-park-mail-ru/2026_1_ASAP/internal/services/contacts"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/services/session"
)

// @title API Pulse App
// @version 1.0
// @description API веб-приложения Pulse
// @host pulseapp.space:8080
func main() {
	cfg, err := config.LoadConfigFromEnv()
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

	sessRepo := sessionRepository.NewSessionRepository(cfg.SessionConfig, cfg.RedisConfig)
	sessionServ := session.NewSessionService(sessRepo, cfg.SessionConfig.SessionTTL)
	userRepo, err := userRepository.NewUserRepository(context.Background(), cfg.PostgresConfig)
	if err != nil {
		log.Fatalln(err.Error())
	}


	chatRepo, err := chatRepository.NewChatRepository(context.Background(), cfg.PostgresConfig)
	if err != nil {
		log.Fatalln(err.Error())
	}
	contactRepo, err := contactRepository.NewContactsRepository(context.Background(), cfg.PostgresConfig)
	if err != nil {
		log.Fatalln(err.Error())
	}
	authServ := authService.NewAuthService(userRepo, sessionServ)
	chatServ := chatService.NewChatService(chatRepo, userRepo)
	contactServ := contactService.NewContactService(contactRepo, userRepo)
	auth := authHandlers.NewAuthHandler(authServ)
	chatsHandler := chatHandlers.NewChatHandler(chatServ)
	authMiddleware := middleware.AuthMiddleware(sessionServ)
	contactsHandler := contactHandlers.NewContactHandler(contactServ)
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

	mux.Route("/api/v1/contacts", func(mux chi.Router) {
		mux.With(authMiddleware).Get("/", contactsHandler.GetContacts)
		mux.With(authMiddleware).Post("/", contactsHandler.CreateContact)
		mux.With(authMiddleware).Delete("/{contact_user_id}", contactsHandler.DeleteContact)
	})

	mux.Get("/swagger/*", httpSwagger.Handler())

	log.Printf("Server started at %s\n", cfg.ServerConfig.ServerInfo())

	server := &http.Server{
		Addr:         cfg.ServerConfig.ServerInfo(),
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
