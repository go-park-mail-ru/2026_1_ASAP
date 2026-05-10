package transport

import (
	"context"

	"github.com/go-park-mail-ru/2026_1_ASAP/internal/subscription/dto"
)

type SubscriptionUseCase interface {
	ActivateSubscription(ctx context.Context, request *dto.RequestActivateSubscription) (*dto.ResponseActivateSubscription, error)
	CancelSubscription(ctx context.Context, request *dto.RequestCancelSubscription) (*dto.ResponseCancelSubscription, error)
	GetSubscription(ctx context.Context, request *dto.RequestGetSubscription) (*dto.ResponseGetSubscription, error)
}
