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
	profilev1 "github.com/go-park-mail-ru/2026_1_ASAP/gen/go/profile/v1"
	gwauth "github.com/go-park-mail-ru/2026_1_ASAP/internal/gateway/auth"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/gateway/middleware"
	gwprofile "github.com/go-park-mail-ru/2026_1_ASAP/internal/gateway/profile"
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

	authClient := authv1.NewAuthClient(authConn)
	profileClient := profilev1.NewProfileClient(profileConn)

	authHandler := gwauth.NewGatewayAuthHandler(authClient)
	profileHandler := gwprofile.NewGatewayProfileHandler(authClient, profileClient)

	router := chi.NewRouter()
	router.Route("/api/v1/auth", func(r chi.Router) {
		r.Post("/login", authHandler.Login)
		r.Post("/register", authHandler.Register)
		r.With(middleware.AuthMiddleware(authClient), middleware.CSRFMiddleware(authClient)).Post("/logout", authHandler.Logout)
	})

	router.Get("/api/v1/profiles/search", profileHandler.SearchIdByLogin)

	router.Route("/api/v1", func(r chi.Router) {
		r.Use(middleware.AuthMiddleware(authClient))
		r.Get("/profile/me", profileHandler.GetMyProfile)
		r.Get("/profile/{id}", profileHandler.GetUserProfile)
		r.With(middleware.CSRFMiddleware(authClient)).Post("/profiles/me/avatar", profileHandler.UpdateUserAvatar)
		r.With(middleware.CSRFMiddleware(authClient)).Post("/profiles/me/bio", profileHandler.UpdateUserBio)
		r.With(middleware.CSRFMiddleware(authClient)).Post("/profiles/me/birth", profileHandler.UpdateUserBirthDate)
		r.With(middleware.CSRFMiddleware(authClient)).Post("/profiles/me/name", profileHandler.UpdateProfileName)
		r.With(middleware.CSRFMiddleware(authClient)).Delete("/profiles/me/avatar", profileHandler.DeleteUserAvatar)
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
