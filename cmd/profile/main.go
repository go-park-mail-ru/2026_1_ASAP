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
	profilev1 "github.com/go-park-mail-ru/2026_1_ASAP/gen/go/profile/v1"
	contactrepo "github.com/go-park-mail-ru/2026_1_ASAP/internal/profile/repository/contacts"
	profilerepo "github.com/go-park-mail-ru/2026_1_ASAP/internal/profile/repository/profile"
	grpcTransport "github.com/go-park-mail-ru/2026_1_ASAP/internal/profile/transport/grpc"
	grpcMedia "github.com/go-park-mail-ru/2026_1_ASAP/internal/profile/transport/grpc/clients/media"

	contactuc "github.com/go-park-mail-ru/2026_1_ASAP/internal/profile/usecase/contact"
	profileuc "github.com/go-park-mail-ru/2026_1_ASAP/internal/profile/usecase/profile"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	cfg, err := config.LoadProfileConfig()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("init logger: %v", err)
	}
	defer func() { _ = logger.Sync() }()

	ctx := context.Background()
	pRepo, err := profilerepo.NewUserRepository(ctx, cfg.PostgresConfig, logger.Named("profile_repo"))
	if err != nil {
		logger.Fatal("init profile repository", zap.Error(err))
	}
	defer pRepo.Close()

	mediaConn, err := grpc.NewClient(cfg.ProfileMediaConfig.GRPCAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("dial profile grpc: %v", err)
	}
	defer func() { _ = mediaConn.Close() }()

	mRepo := grpcMedia.New(mediav1.NewMediaClient(mediaConn))
	cRepo, err := contactrepo.NewContactsRepository(ctx, cfg.PostgresConfig, logger.Named("contact_repo"))
	if err != nil {
		logger.Fatal("init contact repository", zap.Error(err))
	}
	defer cRepo.Close()

	contactService := contactuc.NewContactService(cRepo, pRepo)
	svc := profileuc.NewProfileService(pRepo, mRepo)
	srv := grpcTransport.NewProfileServer(svc, contactService, logger.Named("profile.transport"))

	lis, err := net.Listen("tcp", cfg.ServerConfig.ServerInfo())
	if err != nil {
		logger.Fatal("listen", zap.Error(err))
	}

	grpcServer := grpc.NewServer()
	profilev1.RegisterProfileServer(grpcServer, srv)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-stop
		grpcServer.GracefulStop()
	}()

	logger.Info("profile grpc started", zap.String("addr", cfg.ServerConfig.ServerInfo()))
	if err := grpcServer.Serve(lis); err != nil {
		logger.Fatal("serve grpc", zap.Error(err))
	}
}
