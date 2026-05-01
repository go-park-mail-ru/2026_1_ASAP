package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-park-mail-ru/2026_1_ASAP/config"
	searchv1 "github.com/go-park-mail-ru/2026_1_ASAP/gen/go/search/v1"
	searchpg "github.com/go-park-mail-ru/2026_1_ASAP/internal/search/repository/postgres"
	searchgrpc "github.com/go-park-mail-ru/2026_1_ASAP/internal/search/transport/grpc"
	searchuc "github.com/go-park-mail-ru/2026_1_ASAP/internal/search/usecase/search"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

func main() {
	cfg, err := config.LoadSearchConfig()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("init logger: %v", err)
	}
	defer func() { _ = logger.Sync() }()

	ctx := context.Background()
	repo, err := searchpg.NewRepository(ctx, cfg.PostgresConfig, logger.Named("search_repo"))
	if err != nil {
		logger.Fatal("init search repository", zap.Error(err))
	}
	defer repo.Close()

	svc := searchuc.NewService(repo)
	srv := searchgrpc.NewSearchServer(svc, logger.Named("search.transport"))

	lis, err := net.Listen("tcp", cfg.ServerConfig.ServerInfo())
	if err != nil {
		logger.Fatal("listen", zap.Error(err))
	}

	grpcServer := grpc.NewServer()
	searchv1.RegisterSearchServer(grpcServer, srv)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-stop
		grpcServer.GracefulStop()
	}()

	logger.Info("search grpc started", zap.String("addr", cfg.ServerConfig.ServerInfo()))
	if err := grpcServer.Serve(lis); err != nil {
		logger.Fatal("serve grpc", zap.Error(err))
	}
}
