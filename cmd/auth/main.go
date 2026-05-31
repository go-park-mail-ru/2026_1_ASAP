package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/go-park-mail-ru/2026_1_ASAP/config"
	authv1 "github.com/go-park-mail-ru/2026_1_ASAP/gen/go/auth/v1"
	profilev1 "github.com/go-park-mail-ru/2026_1_ASAP/gen/go/profile/v1"
	sessionRepo "github.com/go-park-mail-ru/2026_1_ASAP/internal/auth/repository/sessions"
	userRepo "github.com/go-park-mail-ru/2026_1_ASAP/internal/auth/repository/user"
	grpcTransport "github.com/go-park-mail-ru/2026_1_ASAP/internal/auth/transport/grpc"
	grpcProfile "github.com/go-park-mail-ru/2026_1_ASAP/internal/auth/transport/grpc/clients/profile"
	authUsecase "github.com/go-park-mail-ru/2026_1_ASAP/internal/auth/usecase/auth"
	sessionUsecase "github.com/go-park-mail-ru/2026_1_ASAP/internal/auth/usecase/session"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/metrics"
)

func main() {
	cfg, err := config.LoadAuthConfig()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	profileConn, err := grpc.NewClient(cfg.AuthProfileConfig.GRPCAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("dial profile grpc: %v", err)
	}
	defer func() { _ = profileConn.Close() }()

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

	profileService := grpcProfile.NewProfileAdapter(profilev1.NewProfileClient(profileConn))

	sSession := sessionUsecase.NewSessionService(sRepo, cfg.SessionConfig.SessionTTL)
	sAuth := authUsecase.NewAuthService(uRepo, sSession, profileService)
	authServer := grpcTransport.NewServer(sAuth, sSession, cfg.VKIDConfig, logger.Named("auth.transport"))

	lis, err := net.Listen("tcp", cfg.ServerConfig.ServerInfo())
	if err != nil {
		logger.Fatal("listen", zap.Error(err))
	}

	grpcServer := grpc.NewServer(grpc.UnaryInterceptor(metrics.GRPCMetricsUnaryInterceptor("auth")))
	authv1.RegisterAuthServer(grpcServer, authServer)
	metricsServer := &http.Server{
		Addr:              ":9101",
		Handler:           promhttp.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("serve metrics http", zap.Error(err))
		}
	}()
	go func() {
		<-stop
		grpcServer.GracefulStop()
		_ = metricsServer.Shutdown(context.Background())
	}()

	logger.Info("auth grpc started", zap.String("addr", cfg.ServerConfig.ServerInfo()))
	if err := grpcServer.Serve(lis); err != nil {
		logger.Fatal("serve grpc", zap.Error(err))
	}
}
