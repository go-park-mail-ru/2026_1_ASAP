package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-park-mail-ru/2026_1_ASAP/config"
	complaintv1 "github.com/go-park-mail-ru/2026_1_ASAP/gen/go/complaint/v1"
	mediav1 "github.com/go-park-mail-ru/2026_1_ASAP/gen/go/media/v1"
	analyticrepo "github.com/go-park-mail-ru/2026_1_ASAP/internal/complaint/repository/analytic"
	complaintrepo "github.com/go-park-mail-ru/2026_1_ASAP/internal/complaint/repository/complaint"
	complaintgrpc "github.com/go-park-mail-ru/2026_1_ASAP/internal/complaint/transport/grpc"
	grpcmedia "github.com/go-park-mail-ru/2026_1_ASAP/internal/complaint/transport/grpc/clients/media"
	analyticuc "github.com/go-park-mail-ru/2026_1_ASAP/internal/complaint/usecase/analytic"
	complaintuc "github.com/go-park-mail-ru/2026_1_ASAP/internal/complaint/usecase/complaint"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	cfg, err := config.LoadComplaintConfig()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("init logger: %v", err)
	}
	defer func() { _ = logger.Sync() }()

	ctx := context.Background()

	cRepo, err := complaintrepo.NewComplaintRepository(ctx, cfg.PostgresConfig, logger.Named("complaint_repo"))
	if err != nil {
		logger.Fatal("init complaint repository", zap.Error(err))
	}
	defer cRepo.Close()

	aRepo, err := analyticrepo.NewAnalyticRepository(ctx, cfg.PostgresConfig, logger.Named("analytic_repo"))
	if err != nil {
		logger.Fatal("init analytic repository", zap.Error(err))
	}
	defer aRepo.Close()

	mediaConn, err := grpc.NewClient(cfg.ComplaintMediaConfig.GRPCAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Fatal("dial media grpc", zap.String("addr", cfg.ComplaintMediaConfig.GRPCAddr), zap.Error(err))
	}
	defer func() { _ = mediaConn.Close() }()

	mediaRepo := grpcmedia.New(mediav1.NewMediaClient(mediaConn))
	complaintService := complaintuc.NewComplaintService(cRepo, mediaRepo)
	analyticService := analyticuc.NewAnalyticService(aRepo)
	srv := complaintgrpc.NewComplaintServer(complaintService, analyticService, logger.Named("complaint.transport"))

	lis, err := net.Listen("tcp", cfg.ServerConfig.ServerInfo())
	if err != nil {
		logger.Fatal("listen", zap.Error(err))
	}

	grpcServer := grpc.NewServer()
	complaintv1.RegisterComplaintServer(grpcServer, srv)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-stop
		grpcServer.GracefulStop()
	}()

	logger.Info("complaint grpc started", zap.String("addr", cfg.ServerConfig.ServerInfo()))
	if err := grpcServer.Serve(lis); err != nil {
		logger.Fatal("serve grpc", zap.Error(err))
	}
}
