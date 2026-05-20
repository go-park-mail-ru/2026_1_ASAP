package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	yoopayment "github.com/rvinnie/yookassa-sdk-go/yookassa/payment"

	"github.com/go-park-mail-ru/2026_1_ASAP/internal/payment/domain"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/payment/dto"
)

const maxYooKassaWebhookBody = 256 << 10

func (u *PaymentUseCase) HandleYooKassaWebhook(ctx context.Context, raw []byte) error {
	if len(raw) == 0 || len(raw) > maxYooKassaWebhookBody {
		return nil
	}

	var envelope struct {
		Type   string          `json:"type"`
		Event  string          `json:"event"`
		Object json.RawMessage `json:"object"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil
	}
	if envelope.Type != "notification" || !strings.HasPrefix(envelope.Event, "payment.") {
		return nil
	}

	var obj struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(envelope.Object, &obj); err != nil || obj.ID == "" {
		return nil
	}

	p, err := u.repo.PaymentGetByPaymentID(ctx, obj.ID)
	if err != nil {
		if errors.Is(err, domain.ErrPaymentNotFound) {
			return nil
		}
		return err
	}

	remote, err := u.yookassa.FindPayment(ctx, obj.ID)
	if err != nil {
		return err
	}

	_, err = u.persistRemoteAndMaybeActivate(ctx, p, remote)
	return err
}

func (u *PaymentUseCase) persistRemoteAndMaybeActivate(ctx context.Context, p *domain.Payment, remote *yoopayment.Payment) (*dto.ResponsePayment, error) {
	prev := p.Status
	p.Status = mapYooKassaStatus(remote.Status)
	if url := yooKassaConfirmationURL(remote); url != "" {
		p.PaymentURL = &url
	}

	updated, err := u.repo.PaymentUpdate(ctx, p)
	if err != nil {
		return nil, err
	}

	if prev != domain.PaymentStatusSucceeded && updated.Status == domain.PaymentStatusSucceeded {
		if err := u.subscription.Activate(ctx, updated.UserID, int64(updated.SubscriptionDays)); err != nil {
			return nil, fmt.Errorf("subscription activate: %w", err)
		}
	}

	return responseFromDomain(updated), nil
}
