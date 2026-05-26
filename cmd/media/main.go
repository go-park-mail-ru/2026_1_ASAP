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
	_ "google.golang.org/grpc/encoding/gzip"

	"github.com/go-park-mail-ru/2026_1_ASAP/config"
	mediav1 "github.com/go-park-mail-ru/2026_1_ASAP/gen/go/media/v1"
	mediarepo "github.com/go-park-mail-ru/2026_1_ASAP/internal/media/repository"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/media/speechkit"
	mediagrpc "github.com/go-park-mail-ru/2026_1_ASAP/internal/media/transport/grpc"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/media/vision"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/metrics"
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

	capybaraDetector := vision.NewDetector(cfg.CapybaraDetectorConfig, logger.Named("capybara"))
	if cfg.CapybaraDetectorConfig.Enabled {
		warmCtx, warmCancel := context.WithTimeout(ctx, 2*time.Minute)
		if warmErr := capybaraDetector.Warmup(warmCtx); warmErr != nil {
			logger.Warn("capybara worker warmup failed", zap.Error(warmErr))
		} else {
			logger.Info("capybara worker ready")
		}
		warmCancel()
	}
	mRepo.SetCapybaraDetector(&vision.ClassifierAdapter{Detector: capybaraDetector})

	stt := speechkit.NewClient(speechkit.Config{
		APIKey: cfg.SpeechKitConfig.APIKey,
		Lang:   cfg.SpeechKitConfig.Lang,
	})
	mediaSrv := mediagrpc.NewMediaServer(mRepo, stt, logger.Named("media_grpc"))

	lis, err := net.Listen("tcp", cfg.ServerConfig.ServerInfo())
	if err != nil {
		logger.Fatal("listen", zap.Error(err))
	}

	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(metrics.GRPCMetricsUnaryInterceptor("media")),
		grpc.MaxRecvMsgSize(64<<20),
		grpc.MaxSendMsgSize(64<<20),
	)
	mediav1.RegisterMediaServer(grpcServer, mediaSrv)
	metricsServer := &http.Server{
		Addr:              ":9103",
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

	logger.Info("media grpc started", zap.String("addr", cfg.ServerConfig.ServerInfo()))
	if err := grpcServer.Serve(lis); err != nil {
		logger.Fatal("serve grpc", zap.Error(err))
	}
}
