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
	paymentv1 "github.com/go-park-mail-ru/2026_1_ASAP/gen/go/payment/v1"
	subscriptionv1 "github.com/go-park-mail-ru/2026_1_ASAP/gen/go/subscription/v1"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/metrics"
	payrepo "github.com/go-park-mail-ru/2026_1_ASAP/internal/payment/repository/payment"
	paygrpc "github.com/go-park-mail-ru/2026_1_ASAP/internal/payment/transport/grpc"
	paysub "github.com/go-park-mail-ru/2026_1_ASAP/internal/payment/transport/subscription"
	payuc "github.com/go-park-mail-ru/2026_1_ASAP/internal/payment/usecase"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rvinnie/yookassa-sdk-go/yookassa"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	cfg, err := config.LoadPaymentConfig()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("init logger: %v", err)
	}
	defer func() { _ = logger.Sync() }()

	ctx := context.Background()
	repo, err := payrepo.NewPaymentRepository(ctx, cfg.PostgresConfig, logger.Named("payment_repo"))
	if err != nil {
		logger.Fatal("init payment repository", zap.Error(err))
	}
	defer repo.Close()

	yooClient := yookassa.NewClient(cfg.ShopID, cfg.SecretKey)
	yooHandler := yookassa.NewPaymentHandler(yooClient)

	subConn, err := grpc.NewClient(cfg.SubscriptionGRPC.GRPCAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Fatal("dial subscription grpc", zap.String("addr", cfg.SubscriptionGRPC.GRPCAddr), zap.Error(err))
	}
	defer func() { _ = subConn.Close() }()
	logger.Info("connected to subscription grpc", zap.String("addr", cfg.SubscriptionGRPC.GRPCAddr))

	subClient := subscriptionv1.NewSubscriptionClient(subConn)
	subAdapter := paysub.New(subClient)

	svc := payuc.NewPaymentUseCase(repo, subAdapter, yooHandler, cfg.ReturnURL)
	srv := &paygrpc.PaymentServer{
		PaymentUseCase: svc,
		Logger:         logger.Named("payment_grpc"),
	}

	lis, err := net.Listen("tcp", cfg.ServerConfig.ServerInfo())
	if err != nil {
		logger.Fatal("listen", zap.Error(err))
	}

	grpcServer := grpc.NewServer(grpc.UnaryInterceptor(metrics.GRPCMetricsUnaryInterceptor("payment")))
	paymentv1.RegisterPaymentServer(grpcServer, srv)
	metricsServer := &http.Server{
		Addr:    ":9112",
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

	logger.Info("payment grpc started", zap.String("addr", cfg.ServerConfig.ServerInfo()))
	if err := grpcServer.Serve(lis); err != nil {
		logger.Fatal("serve grpc", zap.Error(err))
	}
}
