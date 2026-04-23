package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-park-mail-ru/2026_1_ASAP/config"
	authv1 "github.com/go-park-mail-ru/2026_1_ASAP/gen/go/auth/v1"
	sessionRepo "github.com/go-park-mail-ru/2026_1_ASAP/internal/auth/repository/sessions"
	userRepo "github.com/go-park-mail-ru/2026_1_ASAP/internal/auth/repository/user"
	grpcTransport "github.com/go-park-mail-ru/2026_1_ASAP/internal/auth/transport/grpc"
	authUsecase "github.com/go-park-mail-ru/2026_1_ASAP/internal/auth/usecase/auth"
	sessionUsecase "github.com/go-park-mail-ru/2026_1_ASAP/internal/auth/usecase/session"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

func main() {
	cfg, err := config.LoadAuthConfig()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("init logger: %v", err)
	}
	defer func() { _ = logger.Sync() }()

	ctx := context.Background()
	uRepo, err := userRepo.NewUserRepository(ctx, cfg.PostgresConfig, logger.Named("user_repo"))
	if err != nil {
		logger.Fatal("init profile repository", zap.Error(err))
	}
	defer uRepo.Close()

	sRepo := sessionRepo.NewSessionRepository(cfg.SessionConfig, cfg.RedisConfig, logger.Named("session_repo"))
	defer sRepo.Close()

	sSession := sessionUsecase.NewSessionService(sRepo, cfg.SessionConfig.SessionTTL)
	sAuth := authUsecase.NewAuthService(uRepo, sSession)
	authServer := grpcTransport.NewServer(sAuth, sSession)

	lis, err := net.Listen("tcp", cfg.ServerConfig.ServerInfo())
	if err != nil {
		logger.Fatal("listen", zap.Error(err))
	}

	grpcServer := grpc.NewServer()
	authv1.RegisterAuthServer(grpcServer, authServer)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-stop
		grpcServer.GracefulStop()
	}()

	logger.Info("auth grpc started", zap.String("addr", cfg.ServerConfig.ServerInfo()))
	if err := grpcServer.Serve(lis); err != nil {
		logger.Fatal("serve grpc", zap.Error(err))
	}
}
