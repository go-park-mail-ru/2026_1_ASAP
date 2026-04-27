package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-park-mail-ru/2026_1_ASAP/config"
	mediav1 "github.com/go-park-mail-ru/2026_1_ASAP/gen/go/media/v1"
	mediarepo "github.com/go-park-mail-ru/2026_1_ASAP/internal/media/repository"
	mediagrpc "github.com/go-park-mail-ru/2026_1_ASAP/internal/media/transport/grpc"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

func main() {
	cfg, err := config.LoadMediaConfig()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("init logger: %v", err)
	}
	defer func() { _ = logger.Sync() }()

	ctx := context.Background()
	mRepo, err := mediarepo.NewMediaRepository(ctx, cfg.S3Config, logger.Named("media_repo"))
	if err != nil {
		logger.Fatal("init media repository", zap.Error(err))
	}
	defer mRepo.Close()

	mediaSrv := mediagrpc.NewMediaServer(mRepo, logger.Named("media_grpc"))

	lis, err := net.Listen("tcp", cfg.ServerConfig.ServerInfo())
	if err != nil {
		logger.Fatal("listen", zap.Error(err))
	}

	grpcServer := grpc.NewServer()
	mediav1.RegisterMediaServer(grpcServer, mediaSrv)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-stop
		grpcServer.GracefulStop()
	}()

	logger.Info("media grpc started", zap.String("addr", cfg.ServerConfig.ServerInfo()))
	if err := grpcServer.Serve(lis); err != nil {
		logger.Fatal("serve grpc", zap.Error(err))
	}
}
