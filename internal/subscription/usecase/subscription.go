package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-park-mail-ru/2026_1_ASAP/internal/subscription/domain"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/subscription/dto"
)

const HoursInDay = 24

type SubscriptionRepository interface {
	SubscriptionGet(ctx context.Context, user_id int64) (*domain.Subscription, error)
	SubscriptionSet(ctx context.Context, subscription *domain.Subscription) (*domain.Subscription, error)
}

type SubscriptionUseCase struct {
	subscriptionRepository SubscriptionRepository
}

func NewSubscriptionUseCase(subscriptionRepository SubscriptionRepository) *SubscriptionUseCase {
	return &SubscriptionUseCase{
		subscriptionRepository: subscriptionRepository,
	}
}

func (s SubscriptionUseCase) ActivateSubscription(ctx context.Context, request *dto.RequestActivateSubscription) (*dto.ResponseActivateSubscription, error) {
	if request == nil || request.Days <= 0 || request.UserID <= 0 {
		return nil, errors.New("invalid request")
	}

	startAt := time.Now()
	endAt := time.Now().Add(time.Duration(request.Days) * HoursInDay * time.Hour)
	sub := &domain.Subscription{
		UserID:  request.UserID,
		StartAt: startAt,
		EndAt:   endAt,
		Active:  true,
	}

	subscription, err := s.subscriptionRepository.SubscriptionSet(ctx, sub)
	if err != nil {
		return nil, fmt.Errorf("subscription activate: %w", err)
	}

	return &dto.ResponseActivateSubscription{
		UserID: subscription.UserID,
		EndAt:  subscription.EndAt,
	}, nil
}

func (s SubscriptionUseCase) CancelSubscription(ctx context.Context, request *dto.RequestCancelSubscription) (*dto.ResponseCancelSubscription, error) {
	if request == nil || request.UserID <= 0 {
		return nil, errors.New("invalid request")
	}

	subscription, err := s.subscriptionRepository.SubscriptionGet(ctx, request.UserID)
	if err != nil {
		if errors.Is(err, domain.ErrSubscriptionNotFound) {
			return nil, domain.ErrSubscriptionNotFound
		}
		return nil, fmt.Errorf("subscription cancel get: %w", err)
	}

	subscription.Active = false
	if _, err := s.subscriptionRepository.SubscriptionSet(ctx, subscription); err != nil {
		return nil, fmt.Errorf("subscription cancel set: %w", err)
	}

	return &dto.ResponseCancelSubscription{UserID: subscription.UserID}, nil
}

func (s SubscriptionUseCase) GetSubscription(ctx context.Context, request *dto.RequestGetSubscription) (*dto.ResponseGetSubscription, error) {
	if request == nil || request.UserID <= 0 {
		return nil, errors.New("invalid request")
	}

	subscription, err := s.subscriptionRepository.SubscriptionGet(ctx, request.UserID)
	if err != nil {
		if errors.Is(err, domain.ErrSubscriptionNotFound) {
			return nil, domain.ErrSubscriptionNotFound
		}
		return nil, fmt.Errorf("subscription get: %w", err)
	}
	if subscription == nil {
		return nil, domain.ErrSubscriptionNotFound
	}

	now := time.Now()
	if subscription.Active && !now.Before(subscription.EndAt) {
		subscription.Active = false
		updated, err := s.subscriptionRepository.SubscriptionSet(ctx, subscription)
		if err != nil {
			return nil, domain.ErrSubscriptionExpired
		}
		subscription = updated
	}

	active := subscription.Active && now.Before(subscription.EndAt)

	return &dto.ResponseGetSubscription{
		UserID:  subscription.UserID,
		Active:  active,
		StartAt: subscription.StartAt,
		EndAt:   subscription.EndAt,
	}, nil
}
