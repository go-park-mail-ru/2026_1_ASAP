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
	chatv1 "github.com/go-park-mail-ru/2026_1_ASAP/gen/go/chat/v1"
	mediav1 "github.com/go-park-mail-ru/2026_1_ASAP/gen/go/media/v1"
	profilev1 "github.com/go-park-mail-ru/2026_1_ASAP/gen/go/profile/v1"
	chatrepo "github.com/go-park-mail-ru/2026_1_ASAP/internal/chat/repository/chat"
	messagesrepo "github.com/go-park-mail-ru/2026_1_ASAP/internal/chat/repository/messages"
	chatgrpc "github.com/go-park-mail-ru/2026_1_ASAP/internal/chat/transport/grpc"
	grpcMedia "github.com/go-park-mail-ru/2026_1_ASAP/internal/chat/transport/grpc/clients/media"
	grpcProfile "github.com/go-park-mail-ru/2026_1_ASAP/internal/chat/transport/grpc/clients/profile"
	chatws "github.com/go-park-mail-ru/2026_1_ASAP/internal/chat/transport/ws"
	chatuc "github.com/go-park-mail-ru/2026_1_ASAP/internal/chat/usecase/chat"
	messagesuc "github.com/go-park-mail-ru/2026_1_ASAP/internal/chat/usecase/messages"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/metrics"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	cfg, err := config.LoadChatConfig()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("init logger: %v", err)
	}
	defer func() { _ = logger.Sync() }()

	ctx := context.Background()

	chatRepo, err := chatrepo.NewChatRepository(ctx, cfg.PostgresConfig, logger.Named("chat_repo"))
	if err != nil {
		logger.Fatal("init chat repository", zap.Error(err))
	}
	defer chatRepo.Close()

	msgRepo, err := messagesrepo.NewMessageRepository(ctx, cfg.PostgresConfig, logger.Named("messages_repo"))
	if err != nil {
		logger.Fatal("init messages repository", zap.Error(err))
	}
	defer msgRepo.Close()

	mediaConn, err := grpc.NewClient(cfg.ChatMediaConfig.GRPCAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("dial media grpc: %v", err)
	}
	defer func() { _ = mediaConn.Close() }()

	profileConn, err := grpc.NewClient(cfg.ChatProfileConfig.GRPCAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("dial profile grpc: %v", err)
	}
	defer func() { _ = profileConn.Close() }()

	mediaClient := grpcMedia.New(mediav1.NewMediaClient(mediaConn))
	profileClient := grpcProfile.New(profilev1.NewProfileClient(profileConn))

	chatService := chatuc.NewChatService(chatRepo, profileClient, mediaClient)
	messageService := messagesuc.NewMessageService(msgRepo, chatRepo)

	// gRPC server
	grpcSrv := chatgrpc.NewChatServer(chatService, messageService, logger.Named("chat.grpc"))

	lis, err := net.Listen("tcp", cfg.ServerConfig.ServerInfo())
	if err != nil {
		logger.Fatal("listen grpc", zap.Error(err))
	}

	grpcServer := grpc.NewServer(grpc.UnaryInterceptor(metrics.GRPCMetricsUnaryInterceptor("chat")))
	chatv1.RegisterChatServer(grpcServer, grpcSrv)
	metricsServer := &http.Server{
		Addr:    ":9104",
		Handler: promhttp.Handler(),
	}

	// WS HTTP server
	wsSrv := chatws.NewChatServer(logger.Named("chat.ws"), messageService, chatService)

	mux := http.NewServeMux()
	mux.Handle("/ws",
		chatws.RequestIDMiddleware()(
			chatws.AuthMiddleware()(
				http.HandlerFunc(wsSrv.SubscribeHandler),
			),
		),
	)

	wsAddr := cfg.WSServerConfig.ServerInfo()
	if wsAddr == ":" {
		wsAddr = ":8005"
	}
	httpServer := &http.Server{
		Addr:    wsAddr,
		Handler: mux,
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		logger.Info("chat ws started", zap.String("addr", wsAddr))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("ws http server", zap.Error(err))
		}
	}()
	go func() {
		if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("serve metrics http", zap.Error(err))
		}
	}()

	go func() {
		<-stop
		grpcServer.GracefulStop()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.AppConfig.ShutdownTime)
		defer cancel()
		_ = wsSrv.Shutdown(shutdownCtx)
		_ = httpServer.Shutdown(shutdownCtx)
		_ = metricsServer.Shutdown(shutdownCtx)
	}()

	logger.Info("chat grpc started", zap.String("addr", cfg.ServerConfig.ServerInfo()))
	if err := grpcServer.Serve(lis); err != nil {
		logger.Fatal("serve grpc", zap.Error(err))
	}
}
