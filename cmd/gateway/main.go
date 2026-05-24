package main

import (
	"context"
	"errors"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/go-park-mail-ru/2026_1_ASAP/config"
	authv1 "github.com/go-park-mail-ru/2026_1_ASAP/gen/go/auth/v1"
	chatv1 "github.com/go-park-mail-ru/2026_1_ASAP/gen/go/chat/v1"
	complaintv1 "github.com/go-park-mail-ru/2026_1_ASAP/gen/go/complaint/v1"
	mediav1 "github.com/go-park-mail-ru/2026_1_ASAP/gen/go/media/v1"
	paymentv1 "github.com/go-park-mail-ru/2026_1_ASAP/gen/go/payment/v1"
	profilev1 "github.com/go-park-mail-ru/2026_1_ASAP/gen/go/profile/v1"
	searchv1 "github.com/go-park-mail-ru/2026_1_ASAP/gen/go/search/v1"
	subscriptionv1 "github.com/go-park-mail-ru/2026_1_ASAP/gen/go/subscription/v1"
	gwauth "github.com/go-park-mail-ru/2026_1_ASAP/internal/gateway/auth"
	gwchat "github.com/go-park-mail-ru/2026_1_ASAP/internal/gateway/chat"
	gwcomplaint "github.com/go-park-mail-ru/2026_1_ASAP/internal/gateway/complaint"
	gwcontacts "github.com/go-park-mail-ru/2026_1_ASAP/internal/gateway/contacts"
	gwmessage "github.com/go-park-mail-ru/2026_1_ASAP/internal/gateway/message"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/gateway/middleware"
	gwpayment "github.com/go-park-mail-ru/2026_1_ASAP/internal/gateway/payment"
	gwprofile "github.com/go-park-mail-ru/2026_1_ASAP/internal/gateway/profile"
	searchgw "github.com/go-park-mail-ru/2026_1_ASAP/internal/gateway/search"
	gwsubscription "github.com/go-park-mail-ru/2026_1_ASAP/internal/gateway/subscription"
	gwws "github.com/go-park-mail-ru/2026_1_ASAP/internal/gateway/ws"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/metrics"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/encoding/gzip"
)

const grpcMaxMessageBytes = 64 << 20

func main() {
	logger, err := zap.NewProduction()
	if err != nil {
		panic(err)
	}
	defer func() { _ = logger.Sync() }()

	cfg, err := config.LoadGatewayConfig("")
	if err != nil {
		logger.Fatal("load gateway config", zap.Error(err))
	}
	logger.Info(
		"gateway config loaded",
		zap.String("http_addr", cfg.Server.Host+":"+cfg.Server.Port),
		zap.Bool("session_cookie_secure", cfg.SessionCookie.Secure),
		zap.String("auth_grpc", cfg.Auth.GRPCAddr),
		zap.String("profile_grpc", cfg.Profile.GRPCAddr),
		zap.String("chat_grpc", cfg.Chat.GRPCAddr),
		zap.String("complaint_grpc", cfg.Complaint.GRPCAddr),
		zap.String("chat_ws", cfg.Chat.WSAddr),
		zap.String("search_grpc", cfg.Search.GRPCAddr),
		zap.String("subscription_grpc", cfg.Subscription.GRPCAddr),
		zap.String("payment_grpc", cfg.Payment.GRPCAddr),
	)

	authConn, err := grpc.NewClient(cfg.Auth.GRPCAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Fatal("dial auth grpc", zap.String("addr", cfg.Auth.GRPCAddr), zap.Error(err))
	}
	logger.Info("connected to auth grpc", zap.String("addr", cfg.Auth.GRPCAddr))
	defer func() { _ = authConn.Close() }()

	profileConn, err := grpc.NewClient(cfg.Profile.GRPCAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Fatal("dial profile grpc", zap.String("addr", cfg.Profile.GRPCAddr), zap.Error(err))
	}
	logger.Info("connected to profile grpc", zap.String("addr", cfg.Profile.GRPCAddr))
	defer func() { _ = profileConn.Close() }()

	grpcDialOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(grpcMaxMessageBytes),
			grpc.MaxCallSendMsgSize(grpcMaxMessageBytes),
			grpc.UseCompressor(gzip.Name),
		),
	}

	chatConn, err := grpc.NewClient(cfg.Chat.GRPCAddr, grpcDialOpts...)
	if err != nil {
		logger.Fatal("dial chat grpc", zap.String("addr", cfg.Chat.GRPCAddr), zap.Error(err))
	}
	logger.Info("connected to chat grpc", zap.String("addr", cfg.Chat.GRPCAddr))
	defer func() { _ = chatConn.Close() }()

	mediaConn, err := grpc.NewClient(cfg.Media.GRPCAddr, grpcDialOpts...)
	if err != nil {
		logger.Fatal("dial media grpc", zap.String("addr", cfg.Media.GRPCAddr), zap.Error(err))
	}
	logger.Info("connected to media grpc", zap.String("addr", cfg.Media.GRPCAddr))
	defer func() { _ = mediaConn.Close() }()

	complaintConn, err := grpc.NewClient(cfg.Complaint.GRPCAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Fatal("dial complaint grpc", zap.String("addr", cfg.Complaint.GRPCAddr), zap.Error(err))
	}
	logger.Info("connected to complaint grpc", zap.String("addr", cfg.Complaint.GRPCAddr))
	defer func() { _ = complaintConn.Close() }()

	searchConn, err := grpc.NewClient(cfg.Search.GRPCAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Fatal("dial search grpc", zap.String("addr", cfg.Search.GRPCAddr), zap.Error(err))
	}
	logger.Info("connected to search grpc", zap.String("addr", cfg.Search.GRPCAddr))
	defer func() { _ = searchConn.Close() }()

	subscriptionConn, err := grpc.NewClient(cfg.Subscription.GRPCAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Fatal("dial subscription grpc", zap.String("addr", cfg.Subscription.GRPCAddr), zap.Error(err))
	}
	logger.Info("connected to subscription grpc", zap.String("addr", cfg.Subscription.GRPCAddr))
	defer func() { _ = subscriptionConn.Close() }()

	paymentConn, err := grpc.NewClient(cfg.Payment.GRPCAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Fatal("dial payment grpc", zap.String("addr", cfg.Payment.GRPCAddr), zap.Error(err))
	}
	logger.Info("connected to payment grpc", zap.String("addr", cfg.Payment.GRPCAddr))
	defer func() { _ = paymentConn.Close() }()

	authClient := authv1.NewAuthClient(authConn)
	profileClient := profilev1.NewProfileClient(profileConn)
	chatClient := chatv1.NewChatClient(chatConn)
	mediaClient := mediav1.NewMediaClient(mediaConn)
	complaintClient := complaintv1.NewComplaintClient(complaintConn)
	searchClient := searchv1.NewSearchClient(searchConn)
	subscriptionClient := subscriptionv1.NewSubscriptionClient(subscriptionConn)
	paymentClient := paymentv1.NewPaymentClient(paymentConn)

	authHandler := gwauth.NewGatewayAuthHandler(authClient, cfg.SessionCookie)
	profileHandler := gwprofile.NewGatewayProfileHandler(authClient, profileClient)
	contactsHandler := gwcontacts.NewGatewayContactsHandler(profileClient)
	chatHandler := gwchat.NewGatewayChatHandler(chatClient)
	messageHandler := gwmessage.NewGatewayMessageHandler(chatClient, mediaClient, cfg.PublicBaseURL)
	complaintHandler := gwcomplaint.NewGatewayComplaintHandler(complaintClient)
	analyticHandler := gwcomplaint.NewGatewayAnalyticHandler(complaintClient)
	searchHandler := searchgw.NewGatewaySearchHandler(searchClient)
	subscriptionHandler := gwsubscription.NewSubscriptionHandler(subscriptionClient)
	paymentHandler := gwpayment.NewGatewayPaymentHandler(paymentClient)
	accessLogger, _ := zap.NewProduction()

	authMiddleware := middleware.AuthMiddleware(authClient)
	csrfMiddleware := middleware.CSRFMiddleware(authClient)
	adminMiddleware := func(next http.Handler) http.Handler { return next }

	router := chi.NewRouter()
	router.Use(middleware.RequestIDMiddleware())
	router.Use(metrics.HTTPMetricsMiddleware())
	router.Use(middleware.AccessMiddleware(accessLogger))
	router.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{
			"http://pulseapp.space",
			"http://pulseapp.space:8080",
			"http://212.233.96.180",
			"http://localhost",
			"http://localhost:3000",
			"http://localhost:8080",
			"http://127.0.0.1",
			"http://127.0.0.1:3000",
			"http://127.0.0.1:8080",
			"http://0.0.0.0:80",
			"http://0.0.0.0",

			"https://pulseapp.space",
			"https://pulseapp.space:8080",
			"https://212.233.96.180",
			"https://localhost",
			"https://localhost:3000",
			"https://localhost:8080",
			"https://127.0.0.1",
			"https://127.0.0.1:3000",
			"https://127.0.0.1:8080",
			"https://0.0.0.0:80",
			"https://0.0.0.0",
		},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{
			"Accept",
			"Authorization",
			"Content-Type",
			"X-CSRF-TOKEN",
			"X-NEW-CSRF-TOKEN",
		},
		ExposedHeaders:   []string{"Link", "X-CSRF-TOKEN", "X-NEW-CSRF-TOKEN"},
		AllowCredentials: true,
		MaxAge:           300,
	}))
	router.Route("/api/v1/auth", func(mux chi.Router) {
		mux.Post("/login", authHandler.Login)
		mux.Post("/vk", authHandler.VkIDLogin)
		mux.Post("/register", authHandler.Register)
		mux.With(authMiddleware, csrfMiddleware).Post("/logout", authHandler.Logout)
	})

	router.Route("/api/v1/contacts", func(mux chi.Router) {
		mux.With(authMiddleware, csrfMiddleware).Get("/", contactsHandler.GetContacts)
		mux.With(authMiddleware, csrfMiddleware).Post("/", contactsHandler.CreateContact)
		mux.With(authMiddleware, csrfMiddleware).Delete("/{contact_user_id}", contactsHandler.DeleteContact)
	})

	router.With(authMiddleware).Handle("/api/v1/ws", gwws.NewProxy(cfg.Chat.WSAddr))

	router.Route("/api/v1/profiles", func(mux chi.Router) {
		mux.With(authMiddleware, csrfMiddleware).Get("/me", profileHandler.GetMyProfile)
		mux.With(authMiddleware, csrfMiddleware).Post("/me/bio", profileHandler.UpdateUserBio)
		mux.With(authMiddleware, csrfMiddleware).Post("/me/avatar", profileHandler.UpdateUserAvatar)
		mux.With(authMiddleware, csrfMiddleware).Delete("/me/avatar", profileHandler.DeleteUserAvatar)
		mux.With(authMiddleware, csrfMiddleware).Get("/search", profileHandler.SearchIdByLogin)
		mux.With(authMiddleware, csrfMiddleware).Get("/{id}", profileHandler.GetUserProfile)
		mux.With(authMiddleware, csrfMiddleware).Post("/me/birth", profileHandler.UpdateUserBirthDate)
		mux.With(authMiddleware, csrfMiddleware).Post("/me/name", profileHandler.UpdateProfileName)
	})

	router.Route("/api/v1/messages", func(mux chi.Router) {
		mux.With(authMiddleware, csrfMiddleware).Post("/attachments/upload", messageHandler.UploadAttachment)
		mux.With(authMiddleware).Get("/attachments/*", messageHandler.DownloadAttachment)
	})

	router.Route("/api/v1/chats", func(mux chi.Router) {
		mux.With(authMiddleware, csrfMiddleware).Get("/", chatHandler.GetChats)
		mux.With(authMiddleware, csrfMiddleware).Post("/", chatHandler.CreateChat)
		mux.With(authMiddleware, csrfMiddleware).Get("/{id}", chatHandler.GetChatByID)
		mux.With(authMiddleware, csrfMiddleware).Delete("/{id}", chatHandler.DeleteChat)
		mux.With(authMiddleware, csrfMiddleware).Post("/{id}/avatar", chatHandler.UpdateChatAvatar)
		mux.With(authMiddleware, csrfMiddleware).Post("/{id}/title", chatHandler.UpdateChatTitle)
		mux.With(authMiddleware, csrfMiddleware).Post("/{id}/members", chatHandler.AddMembers)
		mux.With(authMiddleware, csrfMiddleware).Delete("/{id}/members/{member_id}", chatHandler.DeleteMember)
		mux.With(authMiddleware, csrfMiddleware).Get("/{id}/members", chatHandler.GetChatMembers)
		mux.With(authMiddleware, csrfMiddleware).Delete("/{id}/quit", chatHandler.QuitChat)
		mux.With(authMiddleware, csrfMiddleware).Post("/{id}/join", chatHandler.JoinChannel)
		mux.With(authMiddleware, csrfMiddleware).Post("/{id}/description", chatHandler.UpdateChatDescription)
	})

	router.Route("/api/v1/sticker-packs", func(mux chi.Router) {
		mux.With(authMiddleware, csrfMiddleware).Get("/", chatHandler.GetStickerPacks)
	})

	router.Route("/api/v1/complaints", func(mux chi.Router) {
		mux.Post("/createUnauthorized", complaintHandler.CreateComplaintUnAuthorized)
		mux.With(authMiddleware, csrfMiddleware).Post("/create", complaintHandler.CreateComplaintAuthorized)
		mux.With(authMiddleware, csrfMiddleware).Get("/my", complaintHandler.GetMyComplaints)
		mux.With(authMiddleware, csrfMiddleware).Get("/{id}", complaintHandler.GetComplaint)
		mux.With(authMiddleware, csrfMiddleware, adminMiddleware).Post("/update", complaintHandler.UpdateComplaintStatus)
		mux.With(authMiddleware, csrfMiddleware, adminMiddleware).Get("/all", complaintHandler.GetAllComplaints)
	})

	router.Route("/api/v1/analytics", func(mux chi.Router) {
		mux.With(authMiddleware, csrfMiddleware).Get("/complaint", analyticHandler.GetUserComplaintAnalytic)
	})

	router.Route("/api/v1/search", func(mux chi.Router) {
		mux.With(authMiddleware, csrfMiddleware).Get("/messages", searchHandler.SearchMessages)
		mux.With(authMiddleware, csrfMiddleware).Get("/users", searchHandler.SearchUsers)
		mux.With(authMiddleware, csrfMiddleware).Get("/chats", searchHandler.SearchChats)
	})

	router.Route("/api/v1/subscription", func(mux chi.Router) {
		mux.With(authMiddleware, csrfMiddleware).Get("/", subscriptionHandler.GetSubscription)
		mux.With(authMiddleware, csrfMiddleware).Delete("/", subscriptionHandler.CancelSubscription)
	})

	router.Post("/api/v1/payment/webhooks/yookassa", paymentHandler.YooKassaWebhook)

	router.Route("/api/v1/payment", func(mux chi.Router) {
		mux.With(authMiddleware, csrfMiddleware).Post("/", paymentHandler.CreatePayment)
		mux.With(authMiddleware, csrfMiddleware).Post("/sync", paymentHandler.SyncOpenPayment)
	})

	router.Handle("/metrics", promhttp.Handler())

	server := &http.Server{
		Addr:         cfg.Server.Host + ":" + cfg.Server.Port,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		logger.Info("gateway http server listening", zap.String("addr", server.Addr))
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatal("gateway listen", zap.Error(err))
		}
	}()

	<-ctx.Done()
	logger.Info("shutdown signal received, stopping gateway")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("gateway shutdown error", zap.Error(err))
		return
	}
	logger.Info("gateway stopped gracefully")
}
