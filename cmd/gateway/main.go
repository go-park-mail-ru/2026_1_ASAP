package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-park-mail-ru/2026_1_ASAP/config"
	authv1 "github.com/go-park-mail-ru/2026_1_ASAP/gen/go/auth/v1"
	chatv1 "github.com/go-park-mail-ru/2026_1_ASAP/gen/go/chat/v1"
	profilev1 "github.com/go-park-mail-ru/2026_1_ASAP/gen/go/profile/v1"
	gwauth "github.com/go-park-mail-ru/2026_1_ASAP/internal/gateway/auth"
	gwchat "github.com/go-park-mail-ru/2026_1_ASAP/internal/gateway/chat"
	gwcontacts "github.com/go-park-mail-ru/2026_1_ASAP/internal/gateway/contacts"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/gateway/middleware"
	gwprofile "github.com/go-park-mail-ru/2026_1_ASAP/internal/gateway/profile"
	gwws "github.com/go-park-mail-ru/2026_1_ASAP/internal/gateway/ws"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	cfg, err := config.LoadGatewayConfig(os.Getenv("CONFIG_PATH"))
	if err != nil {
		log.Fatalf("load gateway config: %v", err)
	}

	authConn, err := grpc.NewClient(cfg.Auth.GRPCAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("dial auth grpc: %v", err)
	}
	defer func() { _ = authConn.Close() }()

	profileConn, err := grpc.NewClient(cfg.Profile.GRPCAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("dial profile grpc: %v", err)
	}
	defer func() { _ = profileConn.Close() }()

	chatConn, err := grpc.NewClient(cfg.Chat.GRPCAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("dial chat grpc: %v", err)
	}
	defer func() { _ = chatConn.Close() }()

	authClient := authv1.NewAuthClient(authConn)
	profileClient := profilev1.NewProfileClient(profileConn)
	chatClient := chatv1.NewChatClient(chatConn)

	authHandler := gwauth.NewGatewayAuthHandler(authClient)
	profileHandler := gwprofile.NewGatewayProfileHandler(authClient, profileClient)
	contactsHandler := gwcontacts.NewGatewayContactsHandler(profileClient)
	chatHandler := gwchat.NewGatewayChatHandler(chatClient)
	accessLogger, _ := zap.NewProduction()

	authMiddleware := middleware.AuthMiddleware(authClient)
	csrfMiddleware := middleware.CSRFMiddleware(authClient)

	router := chi.NewRouter()
	router.Use(middleware.RequestIDMiddleware())
	router.Use(middleware.AccessMiddleware(accessLogger))

	router.Route("/api/v1/auth", func(mux chi.Router) {
		mux.Post("/login", authHandler.Login)
		mux.Post("/vk", authHandler.VkIDLogin)
		mux.Post("/register", authHandler.Register)
		mux.With(authMiddleware, csrfMiddleware).Post("/logout", authHandler.Logout)
	})

	router.Route("/api/v1/contacts", func(mux chi.Router) {
		mux.With(authMiddleware, csrfMiddleware).Get("/", contactsHandler.GetContacts)
		mux.With(authMiddleware, csrfMiddleware).Post("/", contactsHandler.CreateContact)
		mux.With(authMiddleware, csrfMiddleware).Delete("/{contact_user_id}", contactsHandler.DeleteContact)
	})

	router.With(authMiddleware).Handle("/api/v1/ws", gwws.NewProxy(cfg.Chat.WSAddr))

	router.Route("/api/v1/profiles", func(mux chi.Router) {
		mux.With(authMiddleware, csrfMiddleware).Get("/me", profileHandler.GetMyProfile)
		mux.With(authMiddleware, csrfMiddleware).Post("/me/bio", profileHandler.UpdateUserBio)
		mux.With(authMiddleware, csrfMiddleware).Post("/me/avatar", profileHandler.UpdateUserAvatar)
		mux.With(authMiddleware, csrfMiddleware).Delete("/me/avatar", profileHandler.DeleteUserAvatar)
		mux.With(authMiddleware, csrfMiddleware).Get("/search", profileHandler.SearchIdByLogin)
		mux.With(authMiddleware, csrfMiddleware).Get("/{id}", profileHandler.GetUserProfile)
		mux.With(authMiddleware, csrfMiddleware).Post("/me/birth", profileHandler.UpdateUserBirthDate)
		mux.With(authMiddleware, csrfMiddleware).Post("/me/name", profileHandler.UpdateProfileName)
	})

	router.Route("/api/v1/chats", func(mux chi.Router) {
		mux.With(authMiddleware, csrfMiddleware).Get("/", chatHandler.GetChats)
		mux.With(authMiddleware, csrfMiddleware).Post("/", chatHandler.CreateChat)
		mux.With(authMiddleware, csrfMiddleware).Get("/{id}", chatHandler.GetChatByID)
		mux.With(authMiddleware, csrfMiddleware).Delete("/{id}", chatHandler.DeleteChat)
		mux.With(authMiddleware, csrfMiddleware).Post("/{id}/avatar", chatHandler.UpdateChatAvatar)
		mux.With(authMiddleware, csrfMiddleware).Put("/{id}/title", chatHandler.UpdateChatTitle)
		mux.With(authMiddleware, csrfMiddleware).Post("/{id}/members", chatHandler.AddMembers)
		mux.With(authMiddleware, csrfMiddleware).Delete("/{id}/members/{member_id}", chatHandler.DeleteMember)
		mux.With(authMiddleware, csrfMiddleware).Get("/{id}/members", chatHandler.GetChatMembers)
		mux.With(authMiddleware, csrfMiddleware).Delete("/{id}/quit", chatHandler.QuitChat)
	})

	server := &http.Server{
		Addr:         cfg.Server.Host + ":" + cfg.Server.Port,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("gateway listen: %v", err)
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = server.Shutdown(shutdownCtx)
}
