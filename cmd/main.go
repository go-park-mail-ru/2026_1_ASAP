package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/cors"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/middleware"
	httpSwagger "github.com/swaggo/http-swagger"

	config "github.com/go-park-mail-ru/2026_1_ASAP/config"
	_ "github.com/go-park-mail-ru/2026_1_ASAP/docs"
	authHandlers "github.com/go-park-mail-ru/2026_1_ASAP/internal/handlers/auth"
	chatHandlers "github.com/go-park-mail-ru/2026_1_ASAP/internal/handlers/chat"
	contactHandlers "github.com/go-park-mail-ru/2026_1_ASAP/internal/handlers/contacts"
	profileHandlers "github.com/go-park-mail-ru/2026_1_ASAP/internal/handlers/profile"
	wsHandlers "github.com/go-park-mail-ru/2026_1_ASAP/internal/handlers/ws"
	chatRepository "github.com/go-park-mail-ru/2026_1_ASAP/internal/repository/chat"
	contactRepository "github.com/go-park-mail-ru/2026_1_ASAP/internal/repository/contacts"
	mediaRepository "github.com/go-park-mail-ru/2026_1_ASAP/internal/repository/media"
	messageRepository "github.com/go-park-mail-ru/2026_1_ASAP/internal/repository/messages"
	sessionRepository "github.com/go-park-mail-ru/2026_1_ASAP/internal/repository/sessions"
	userRepository "github.com/go-park-mail-ru/2026_1_ASAP/internal/repository/user"
	authService "github.com/go-park-mail-ru/2026_1_ASAP/internal/services/auth"
	chatService "github.com/go-park-mail-ru/2026_1_ASAP/internal/services/chat"
	contactService "github.com/go-park-mail-ru/2026_1_ASAP/internal/services/contacts"
	messageService "github.com/go-park-mail-ru/2026_1_ASAP/internal/services/messages"
	profileService "github.com/go-park-mail-ru/2026_1_ASAP/internal/services/profile"

	"github.com/go-park-mail-ru/2026_1_ASAP/internal/services/session"
	"go.uber.org/zap"
)

// @title API Pulse App
// @version 1.0
// @description API веб-приложения Pulse
// @host pulseapp.space:8080
func main() {
	appContext, done := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer done()

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	zapCfg := zap.NewProductionConfig()
	zapCfg.Level = zap.NewAtomicLevelAt(cfg.AppConfig.LogLevel)
	logger, err := zapCfg.Build()
	if err != nil {
		log.Fatalf("zap: %v", err)
	}
	defer func() { _ = logger.Sync() }()

	appLogger := logger.Named("app")
	repositoryLogger := logger.Named("repository")
	handlerLogger := logger.Named("handler")
	// Repositories
	sessRepo := sessionRepository.NewSessionRepository(cfg.SessionConfig, cfg.RedisConfig, repositoryLogger.Named("session_repo"))
	userRepo, err := userRepository.NewUserRepository(context.Background(), cfg.PostgresConfig, repositoryLogger.Named("user_repo"))
	if err != nil {
		appLogger.Fatal(err.Error())
	}

	chatRepo, err := chatRepository.NewChatRepository(context.Background(), cfg.PostgresConfig, repositoryLogger.Named("chat_repo"))
	if err != nil {
		appLogger.Fatal(err.Error())
	}
	contactRepo, err := contactRepository.NewContactsRepository(context.Background(), cfg.PostgresConfig, repositoryLogger.Named("contacts_repo"))
	if err != nil {
		appLogger.Fatal(err.Error())
	}

	mediaRepo, err := mediaRepository.NewMediaRepository(context.Background(), cfg.S3Config, repositoryLogger.Named("media_repo"))
	if err != nil {
		appLogger.Fatal(err.Error())
	}

	messageRepo, err := messageRepository.NewMessageRepository(context.Background(), cfg.PostgresConfig, repositoryLogger.Named("message_repo"))
	if err != nil {
		appLogger.Fatal(err.Error())

	}
	// Services
	sessionServ := session.NewSessionService(sessRepo, cfg.SessionConfig.SessionTTL)
	authServ := authService.NewAuthService(userRepo, sessionServ)
	chatServ := chatService.NewChatService(chatRepo, userRepo, mediaRepo)
	contactServ := contactService.NewContactService(contactRepo, userRepo)
	profileServ := profileService.NewProfileService(userRepo, mediaRepo)
	messageServ := messageService.NewMessageService(messageRepo, chatRepo)

	// Handlers
	ws := wsHandlers.NewChatServer(handlerLogger.Named("ws"), messageServ, chatServ)
	chatsHandler := chatHandlers.NewChatHandler(chatServ, ws)
	contactsHandler := contactHandlers.NewContactHandler(contactServ)
	profileHandlers := profileHandlers.NewProfileHandler(profileServ)
	auth := authHandlers.NewAuthHandler(authServ)

	//Middleware
	requestIDMiddleware := middleware.RequestIDMiddleware()
	accessMiddleware := middleware.AccessMiddleware(logger.Named("access"))
	authMiddleware := middleware.AuthMiddleware(sessionServ)
	csrfMiddleware := middleware.CSRFMiddleware(sessionServ)

	mux := chi.NewRouter()

	mux.Use(requestIDMiddleware)
	mux.Use(accessMiddleware)
	mux.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{
			"http://pulseapp.space",
			"http://pulseapp.space:8080",
			"http://212.233.96.180",
			"http://localhost",
			"http://localhost:8080",
			"http://127.0.0.1",
			"http://127.0.0.1:8080",
			"http://0.0.0.0:80",
			"http://0.0.0.0",
		},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{
			"Accept",
			"Authorization",
			"Content-Type",
			"X-CSRF-TOKEN",
			"X-NEW-CSRF-TOKEN",
		},
		ExposedHeaders:   []string{"Link", "X-CSRF-TOKEN", "X-NEW-CSRF-TOKEN"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	mux.Route("/api/v1/auth", func(mux chi.Router) {
		mux.Post("/login", auth.Login)
		mux.Post("/register", auth.Register)
		mux.With(authMiddleware, csrfMiddleware).Post("/logout", auth.Logout)
	})

	mux.Route("/api/v1/chats", func(mux chi.Router) {
		mux.With(authMiddleware, csrfMiddleware).Get("/", chatsHandler.GetChats)
		mux.With(authMiddleware, csrfMiddleware).Post("/", chatsHandler.ChatCreate)
		mux.With(authMiddleware, csrfMiddleware).Get("/{id}", chatsHandler.GetChatByID)
		mux.With(authMiddleware, csrfMiddleware).Post("/{id}/avatar", chatsHandler.UpdateChatAvatar)
		mux.With(authMiddleware, csrfMiddleware).Post("/{id}/members", chatsHandler.AddMembersToChat)
		mux.With(authMiddleware, csrfMiddleware).Delete("/{id}/members", chatsHandler.DeleteMemberFromChat)
		mux.With(authMiddleware, csrfMiddleware).Post("/{id}/quit", chatsHandler.QuitChat)
		mux.With(authMiddleware, csrfMiddleware).Post("/{id}/title", chatsHandler.UpdateChatTitle)
		mux.With(authMiddleware, csrfMiddleware).Delete("/{id}", chatsHandler.DeleteChat)
		mux.With(authMiddleware, csrfMiddleware).Get("/{id}/members", chatsHandler.GetChatMembers)
	})

	mux.Route("/api/v1/contacts", func(mux chi.Router) {
		mux.With(authMiddleware, csrfMiddleware).Get("/", contactsHandler.GetContacts)
		mux.With(authMiddleware, csrfMiddleware).Post("/", contactsHandler.CreateContact)
		mux.With(authMiddleware, csrfMiddleware).Delete("/{contact_user_id}", contactsHandler.DeleteContact)
	})

	mux.Route("/api/v1/profiles", func(mux chi.Router) {
		mux.With(authMiddleware, csrfMiddleware).Get("/me", profileHandlers.GetMyProfile)
		mux.With(authMiddleware, csrfMiddleware).Post("/me/bio", profileHandlers.UpdateUserBio)
		mux.With(authMiddleware, csrfMiddleware).Post("/me/avatar", profileHandlers.UpdateUserAvatar)
		mux.With(authMiddleware, csrfMiddleware).Delete("/me/avatar", profileHandlers.DeleteUserAvatat)
		mux.With(authMiddleware, csrfMiddleware).Get("/{id}", profileHandlers.GetUserProfile)
		mux.With(authMiddleware, csrfMiddleware).Post("/me/birth", profileHandlers.UpdateProfileBirthDate)
		mux.With(authMiddleware, csrfMiddleware).Get("/search", profileHandlers.SearchIdByLogin)
		mux.With(authMiddleware, csrfMiddleware).Post("/me/name", profileHandlers.UpdateProfileName)
	})

	mux.With(authMiddleware).Get("/api/v1/ws", ws.SubscribeHandler)

	mux.Get("/swagger/*", httpSwagger.Handler())

	server := &http.Server{
		Addr:         cfg.ServerConfig.ServerInfo(),
		Handler:      mux,
		ReadTimeout:  120 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			appLogger.Fatal("listen", zap.Error(err))
		}
	}()

	<-appContext.Done()

	ctx, cancel := context.WithTimeout(context.Background(), cfg.AppConfig.ShutdownTime)
	defer cancel()

	if err := ws.Shutdown(ctx); err != nil {
		appLogger.Error("WS shutdown", zap.Error(err))
	}

	if err := server.Shutdown(ctx); err != nil {
		appLogger.Error("HTTP shutdown", zap.Error(err))
	}

	sessRepo.Close()
	userRepo.Close()
	chatRepo.Close()
	contactRepo.Close()
	mediaRepo.Close()
	appLogger.Info("Graceful shutdown complete")
}
