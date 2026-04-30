package main

import (
	"context"
	"errors"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-park-mail-ru/2026_1_ASAP/config"
	authv1 "github.com/go-park-mail-ru/2026_1_ASAP/gen/go/auth/v1"
	chatv1 "github.com/go-park-mail-ru/2026_1_ASAP/gen/go/chat/v1"
	complaintv1 "github.com/go-park-mail-ru/2026_1_ASAP/gen/go/complaint/v1"
	profilev1 "github.com/go-park-mail-ru/2026_1_ASAP/gen/go/profile/v1"
	gwauth "github.com/go-park-mail-ru/2026_1_ASAP/internal/gateway/auth"
	gwchat "github.com/go-park-mail-ru/2026_1_ASAP/internal/gateway/chat"
	gwcomplaint "github.com/go-park-mail-ru/2026_1_ASAP/internal/gateway/complaint"
	gwcontacts "github.com/go-park-mail-ru/2026_1_ASAP/internal/gateway/contacts"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/gateway/middleware"
	gwprofile "github.com/go-park-mail-ru/2026_1_ASAP/internal/gateway/profile"
	gwws "github.com/go-park-mail-ru/2026_1_ASAP/internal/gateway/ws"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

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
		zap.String("auth_grpc", cfg.Auth.GRPCAddr),
		zap.String("profile_grpc", cfg.Profile.GRPCAddr),
		zap.String("chat_grpc", cfg.Chat.GRPCAddr),
		zap.String("complaint_grpc", cfg.Complaint.GRPCAddr),
		zap.String("chat_ws", cfg.Chat.WSAddr),
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

	chatConn, err := grpc.NewClient(cfg.Chat.GRPCAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Fatal("dial chat grpc", zap.String("addr", cfg.Chat.GRPCAddr), zap.Error(err))
	}
	logger.Info("connected to chat grpc", zap.String("addr", cfg.Chat.GRPCAddr))
	defer func() { _ = chatConn.Close() }()

	complaintConn, err := grpc.NewClient(cfg.Complaint.GRPCAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Fatal("dial complaint grpc", zap.String("addr", cfg.Complaint.GRPCAddr), zap.Error(err))
	}
	logger.Info("connected to complaint grpc", zap.String("addr", cfg.Complaint.GRPCAddr))
	defer func() { _ = complaintConn.Close() }()

	authClient := authv1.NewAuthClient(authConn)
	profileClient := profilev1.NewProfileClient(profileConn)
	chatClient := chatv1.NewChatClient(chatConn)
	complaintClient := complaintv1.NewComplaintClient(complaintConn)

	authHandler := gwauth.NewGatewayAuthHandler(authClient)
	profileHandler := gwprofile.NewGatewayProfileHandler(authClient, profileClient)
	contactsHandler := gwcontacts.NewGatewayContactsHandler(profileClient)
	chatHandler := gwchat.NewGatewayChatHandler(chatClient)
	complaintHandler := gwcomplaint.NewGatewayComplaintHandler(complaintClient)
	analyticHandler := gwcomplaint.NewGatewayAnalyticHandler(complaintClient)
	accessLogger, _ := zap.NewProduction()

	authMiddleware := middleware.AuthMiddleware(authClient)
	csrfMiddleware := middleware.CSRFMiddleware(authClient)
	adminMiddleware := func(next http.Handler) http.Handler { return next }

	router := chi.NewRouter()
	router.Use(middleware.RequestIDMiddleware())
	router.Use(middleware.AccessMiddleware(accessLogger))

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

	router.Route("/api/v1/chats", func(mux chi.Router) {
		mux.With(authMiddleware, csrfMiddleware).Get("/", chatHandler.GetChats)
		mux.With(authMiddleware, csrfMiddleware).Post("/", chatHandler.CreateChat)
		mux.With(authMiddleware, csrfMiddleware).Get("/{id}", chatHandler.GetChatByID)
		mux.With(authMiddleware, csrfMiddleware).Delete("/{id}", chatHandler.DeleteChat)
		mux.With(authMiddleware, csrfMiddleware).Post("/{id}/avatar", chatHandler.UpdateChatAvatar)
		mux.With(authMiddleware, csrfMiddleware).Put("/{id}/title", chatHandler.UpdateChatTitle)
		mux.With(authMiddleware, csrfMiddleware).Post("/{id}/members", chatHandler.AddMembers)
		mux.With(authMiddleware, csrfMiddleware).Delete("/{id}/members/{member_id}", chatHandler.DeleteMember)
		mux.With(authMiddleware, csrfMiddleware).Get("/{id}/members", chatHandler.GetChatMembers)
		mux.With(authMiddleware, csrfMiddleware).Delete("/{id}/quit", chatHandler.QuitChat)
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
