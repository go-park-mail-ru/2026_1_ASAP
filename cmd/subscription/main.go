package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-park-mail-ru/2026_1_ASAP/config"
	subscriptionv1 "github.com/go-park-mail-ru/2026_1_ASAP/gen/go/subscription/v1"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/metrics"
	subrepo "github.com/go-park-mail-ru/2026_1_ASAP/internal/subscription/repository/subscription"
	subgrpc "github.com/go-park-mail-ru/2026_1_ASAP/internal/subscription/transport/grpc"
	subuc "github.com/go-park-mail-ru/2026_1_ASAP/internal/subscription/usecase"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

func main() {
	cfg, err := config.LoadSubscriptionConfig()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("init logger: %v", err)
	}
	defer func() { _ = logger.Sync() }()

	ctx := context.Background()
	repo, err := subrepo.NewSubscriptionRepository(ctx, cfg.PostgresConfig, logger.Named("subscription_repo"))
	if err != nil {
		logger.Fatal("init subscription repository", zap.Error(err))
	}
	defer repo.Close()

	svc := subuc.NewSubscriptionUseCase(repo)
	srv := &subgrpc.SubscriptionServer{SubscriptionUseCase: svc}

	lis, err := net.Listen("tcp", cfg.ServerConfig.ServerInfo())
	if err != nil {
		logger.Fatal("listen", zap.Error(err))
	}

	grpcServer := grpc.NewServer(grpc.UnaryInterceptor(metrics.GRPCMetricsUnaryInterceptor("subscription")))
	subscriptionv1.RegisterSubscriptionServer(grpcServer, srv)
	metricsServer := &http.Server{
		Addr:    ":9111",
		Handler: promhttp.Handler(),
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

	logger.Info("subscription grpc started", zap.String("addr", cfg.ServerConfig.ServerInfo()))
	if err := grpcServer.Serve(lis); err != nil {
		logger.Fatal("serve grpc", zap.Error(err))
	}
}
