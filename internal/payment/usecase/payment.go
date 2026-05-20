package usecase

import (
	"context"
	"errors"
	"fmt"

	yoocommon "github.com/rvinnie/yookassa-sdk-go/yookassa/common"
	yoopayment "github.com/rvinnie/yookassa-sdk-go/yookassa/payment"

	"github.com/go-park-mail-ru/2026_1_ASAP/internal/payment/domain"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/payment/dto"
)

type PaymentRepository interface {
	PaymentCreate(ctx context.Context, p *domain.Payment) (*domain.Payment, error)
	PaymentGetByID(ctx context.Context, id int64) (*domain.Payment, error)
	PaymentGetByPaymentID(ctx context.Context, paymentID string) (*domain.Payment, error)
	PaymentGetOpenPendingByUser(ctx context.Context, userID int64) (*domain.Payment, error)
	PaymentUpdate(ctx context.Context, p *domain.Payment) (*domain.Payment, error)
}

type SubscriptionService interface {
	Activate(ctx context.Context, userID int64, days int64) error
}

type YooKassaClient interface {
	CreatePayment(ctx context.Context, payment *yoopayment.Payment) (*yoopayment.Payment, error)
	FindPayment(ctx context.Context, id string) (*yoopayment.Payment, error)
}

type PaymentUseCase struct {
	repo         PaymentRepository
	subscription SubscriptionService
	yookassa     YooKassaClient
	returnURL    string
}

func NewPaymentUseCase(repo PaymentRepository, subscription SubscriptionService, yookassa YooKassaClient, returnURL string) *PaymentUseCase {
	return &PaymentUseCase{
		repo:         repo,
		subscription: subscription,
		yookassa:     yookassa,
		returnURL:    returnURL,
	}
}

func (u *PaymentUseCase) CreatePayment(ctx context.Context, req *dto.RequestCreatePayment) (*dto.ResponsePayment, error) {
	if err := validateCreate(req); err != nil {
		return nil, err
	}
	if u.returnURL == "" {
		return nil, domain.ErrPaymentReturnURLUnset
	}

	if open, err := u.repo.PaymentGetOpenPendingByUser(ctx, req.UserID); err == nil && open != nil {
		return responseFromDomain(open), nil
	} else if err != nil && !errors.Is(err, domain.ErrPaymentNotFound) {
		return nil, fmt.Errorf("payment get open pending: %w", err)
	}

	yooPayment, err := u.yookassa.CreatePayment(ctx, &yoopayment.Payment{
		Amount: &yoocommon.Amount{
			Value:    fmt.Sprintf("%d.00", req.Amount),
			Currency: "RUB",
		},
		Description: fmt.Sprintf("Subscription for user %d", req.UserID),
		Capture:     true,
		Confirmation: yoopayment.Redirect{
			Type:      yoopayment.TypeRedirect,
			ReturnURL: u.returnURL,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("yookassa create payment: %w", err)
	}

	var paymentURL *string
	if url := yooKassaConfirmationURL(yooPayment); url != "" {
		paymentURL = &url
	}

	p := &domain.Payment{
		PaymentID:        yooPayment.ID,
		UserID:           req.UserID,
		Status:           mapYooKassaStatus(yooPayment.Status),
		Amount:           req.Amount,
		SubscriptionDays: req.SubscriptionDays,
		PaymentURL:       paymentURL,
	}

	created, err := u.repo.PaymentCreate(ctx, p)
	if err != nil {
		return nil, fmt.Errorf("payment create: %w", err)
	}

	return responseFromDomain(created), nil
}

func (u *PaymentUseCase) SyncOpenPayment(ctx context.Context, req *dto.RequestSyncOpenPayment) (*dto.ResponsePayment, error) {
	if req == nil || req.UserID <= 0 {
		return nil, domain.ErrInvalidPaymentRequest
	}

	p, err := u.repo.PaymentGetOpenPendingByUser(ctx, req.UserID)
	if err != nil {
		return nil, err
	}

	remote, err := u.yookassa.FindPayment(ctx, p.PaymentID)
	if err != nil {
		return nil, fmt.Errorf("yookassa find payment: %w", err)
	}

	out, err := u.persistRemoteAndMaybeActivate(ctx, p, remote)
	if err != nil {
		return nil, fmt.Errorf("payment sync: %w", err)
	}
	return out, nil
}

func (u *PaymentUseCase) GetPayment(ctx context.Context, req *dto.RequestGetPayment) (*dto.ResponsePayment, error) {
	if req == nil || req.ID <= 0 {
		return nil, domain.ErrInvalidPaymentRequest
	}

	p, err := u.repo.PaymentGetByID(ctx, req.ID)
	if err != nil {
		return nil, fmt.Errorf("payment get: %w", err)
	}

	return responseFromDomain(p), nil
}

func validateCreate(req *dto.RequestCreatePayment) error {
	if req == nil || req.UserID <= 0 {
		return domain.ErrInvalidPaymentRequest
	}
	if req.Amount <= 0 {
		return domain.ErrInvalidPaymentRequest
	}
	if req.SubscriptionDays <= 0 {
		return domain.ErrInvalidPaymentRequest
	}
	return nil
}

func yooKassaConfirmationURL(p *yoopayment.Payment) string {
	if p == nil || p.Confirmation == nil {
		return ""
	}
	m, ok := p.Confirmation.(map[string]interface{})
	if !ok {
		return ""
	}
	u, _ := m["confirmation_url"].(string)
	return u
}

func mapYooKassaStatus(s yoopayment.Status) domain.PaymentStatus {
	switch s {
	case yoopayment.Pending:
		return domain.PaymentStatusPending
	case yoopayment.WaitingForCapture:
		return domain.PaymentStatusWaitingForCapture
	case yoopayment.Succeeded:
		return domain.PaymentStatusSucceeded
	case yoopayment.Canceled:
		return domain.PaymentStatusCanceled
	default:
		return domain.PaymentStatus(string(s))
	}
}

func responseFromDomain(p *domain.Payment) *dto.ResponsePayment {
	if p == nil {
		return nil
	}
	return &dto.ResponsePayment{
		ID:               p.ID,
		PaymentID:        p.PaymentID,
		UserID:           p.UserID,
		Status:           string(p.Status),
		Amount:           p.Amount,
		SubscriptionDays: p.SubscriptionDays,
		PaymentURL:       p.PaymentURL,
		Message:          p.Message,
		CreatedAt:        p.CreatedAt,
		UpdatedAt:        p.UpdatedAt,
	}
}
