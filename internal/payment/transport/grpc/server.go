package grpc

import (
	"context"
	"errors"

	paymentv1 "github.com/go-park-mail-ru/2026_1_ASAP/gen/go/payment/v1"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/payment/domain"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/payment/dto"
	"github.com/go-park-mail-ru/2026_1_ASAP/pkg/loggerctx"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type PaymentUseCase interface {
	CreatePayment(ctx context.Context, request *dto.RequestCreatePayment) (*dto.ResponsePayment, error)
	GetPayment(ctx context.Context, request *dto.RequestGetPayment) (*dto.ResponsePayment, error)
	SyncOpenPayment(ctx context.Context, request *dto.RequestSyncOpenPayment) (*dto.ResponsePayment, error)
	HandleYooKassaWebhook(ctx context.Context, rawBody []byte) error
}

type PaymentServer struct {
	paymentv1.UnimplementedPaymentServer

	PaymentUseCase PaymentUseCase
	Logger         *zap.Logger
}

func (s *PaymentServer) CreatePayment(ctx context.Context, req *paymentv1.RequestCreatePayment) (*paymentv1.ResponseCreatePayment, error) {
	if req == nil {
		s.Log(ctx).Warn("CreatePayment: nil request")
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	dtoReq := &dto.RequestCreatePayment{
		UserID:           req.GetUserId(),
		PaymentID:        req.GetPaymentId(),
		Status:           "pending",
		Amount:           req.GetAmount(),
		SubscriptionDays: req.GetSubscriptionDays(),
	}

	s.Log(ctx).Info("CreatePayment: request",
		zap.Int64("user_id", dtoReq.UserID),
		zap.String("payment_id", dtoReq.PaymentID),
		zap.Int32("amount", dtoReq.Amount),
		zap.Int32("subscription_days", dtoReq.SubscriptionDays),
	)

	resp, err := s.PaymentUseCase.CreatePayment(ctx, dtoReq)
	if err != nil {
		s.Log(ctx).Error("CreatePayment: failed",
			zap.Int64("user_id", dtoReq.UserID),
			zap.String("payment_id", dtoReq.PaymentID),
			zap.Error(err),
		)
		return nil, mapPaymentErr(err)
	}

	if resp != nil {
		s.Log(ctx).Info("CreatePayment: success",
			zap.Int64("payment_pk", resp.ID),
			zap.String("payment_id", resp.PaymentID),
			zap.String("status", resp.Status),
		)
	}

	return &paymentv1.ResponseCreatePayment{
		Payment: paymentDetailsToProto(resp),
	}, nil
}

func (s *PaymentServer) GetPayment(ctx context.Context, req *paymentv1.RequestGetPayment) (*paymentv1.ResponseGetPayment, error) {
	if req == nil || req.GetId() <= 0 {
		s.Log(ctx).Warn("GetPayment: invalid request",
			zap.Int64("id", func() int64 {
				if req == nil {
					return 0
				}
				return req.GetId()
			}()),
		)
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	s.Log(ctx).Info("GetPayment: request",
		zap.Int64("id", req.GetId()),
	)

	resp, err := s.PaymentUseCase.GetPayment(ctx, &dto.RequestGetPayment{
		ID: req.GetId(),
	})
	if err != nil {
		s.Log(ctx).Error("GetPayment: failed",
			zap.Int64("id", req.GetId()),
			zap.Error(err),
		)
		return nil, mapPaymentErr(err)
	}

	if resp != nil {
		s.Log(ctx).Info("GetPayment: success",
			zap.Int64("payment_pk", resp.ID),
			zap.String("payment_id", resp.PaymentID),
			zap.String("status", resp.Status),
		)
	}

	return &paymentv1.ResponseGetPayment{
		Payment: paymentDetailsToProto(resp),
	}, nil
}

func (s *PaymentServer) SyncOpenPayment(ctx context.Context, req *paymentv1.RequestSyncOpenPayment) (*paymentv1.ResponseGetPayment, error) {
	if req == nil || req.GetUserId() <= 0 {
		s.Log(ctx).Warn("SyncOpenPayment: invalid request")
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	s.Log(ctx).Info("SyncOpenPayment: request", zap.Int64("user_id", req.GetUserId()))

	resp, err := s.PaymentUseCase.SyncOpenPayment(ctx, &dto.RequestSyncOpenPayment{
		UserID: req.GetUserId(),
	})
	if err != nil {
		s.Log(ctx).Error("SyncOpenPayment: failed", zap.Int64("user_id", req.GetUserId()), zap.Error(err))
		return nil, mapPaymentErr(err)
	}

	if resp != nil {
		s.Log(ctx).Info("SyncOpenPayment: success",
			zap.Int64("payment_pk", resp.ID),
			zap.String("payment_id", resp.PaymentID),
			zap.String("status", resp.Status),
		)
	}

	return &paymentv1.ResponseGetPayment{
		Payment: paymentDetailsToProto(resp),
	}, nil
}

func (s *PaymentServer) ProcessYooKassaWebhook(ctx context.Context, req *paymentv1.ProcessYooKassaWebhookRequest) (*emptypb.Empty, error) {
	if req == nil || len(req.GetRawBody()) == 0 {
		return &emptypb.Empty{}, nil
	}

	if err := s.PaymentUseCase.HandleYooKassaWebhook(ctx, req.GetRawBody()); err != nil {
		s.Log(ctx).Error("ProcessYooKassaWebhook: failed", zap.Error(err))
		return nil, status.Error(codes.Internal, "webhook processing failed")
	}

	return &emptypb.Empty{}, nil
}

func paymentDetailsToProto(d *dto.ResponsePayment) *paymentv1.PaymentDetails {
	if d == nil {
		return nil
	}
	out := &paymentv1.PaymentDetails{
		Id:               d.ID,
		PaymentId:        d.PaymentID,
		UserId:           d.UserID,
		Status:           d.Status,
		Amount:           d.Amount,
		SubscriptionDays: d.SubscriptionDays,
		CreatedAt:        timestamppb.New(d.CreatedAt),
		UpdatedAt:        timestamppb.New(d.UpdatedAt),
	}
	if d.PaymentURL != nil {
		u := *d.PaymentURL
		out.PaymentUrl = &u
	}
	if d.Message != nil {
		m := *d.Message
		out.Message = &m
	}
	return out
}

func mapPaymentErr(err error) error {
	switch {
	case errors.Is(err, domain.ErrPaymentNotFound):
		return status.Error(codes.NotFound, "payment not found")
	case errors.Is(err, domain.ErrDuplicatePayment):
		return status.Error(codes.AlreadyExists, "payment already exists")
	case errors.Is(err, domain.ErrInvalidPaymentRequest):
		return status.Error(codes.InvalidArgument, "invalid payment request")
	case errors.Is(err, domain.ErrPaymentReturnURLUnset):
		return status.Error(codes.FailedPrecondition, "payment return url is not configured")
	default:
		return status.Error(codes.Internal, "internal error")
	}
}

// Log returns request-scoped logger enriched with context (if present).
func (s *PaymentServer) Log(ctx context.Context) *zap.Logger {
	return loggerctx.EnrichLoggerFromContext(ctx, s.Logger)
}
