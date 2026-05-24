package grpc

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	subscriptionv1 "github.com/go-park-mail-ru/2026_1_ASAP/gen/go/subscription/v1"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/subscription/domain"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/subscription/dto"
	"github.com/go-park-mail-ru/2026_1_ASAP/pkg/grpcerr"
)

type SubscriptionUseCase interface {
	ActivateSubscription(ctx context.Context, request *dto.RequestActivateSubscription) (*dto.ResponseActivateSubscription, error)
	CancelSubscription(ctx context.Context, request *dto.RequestCancelSubscription) (*dto.ResponseCancelSubscription, error)
	GetSubscription(ctx context.Context, request *dto.RequestGetSubscription) (*dto.ResponseGetSubscription, error)
}

type SubscriptionServer struct {
	subscriptionv1.UnimplementedSubscriptionServer

	SubscriptionUseCase SubscriptionUseCase
}

func (s SubscriptionServer) ActivateSubscription(ctx context.Context, subscription *subscriptionv1.RequestActivateSubscription) (*subscriptionv1.ResponseActivateSubscription, error) {
	resp, err := s.SubscriptionUseCase.ActivateSubscription(ctx, &dto.RequestActivateSubscription{
		UserID: subscription.UserId,
		Days:   subscription.Days,
	})
	if err != nil {
		return nil, mapDomainErr(err)
	}

	return &subscriptionv1.ResponseActivateSubscription{
		UserId: resp.UserID,
		EndAt:  timestamppb.New(resp.EndAt),
	}, nil
}

func (s SubscriptionServer) GetSubscription(ctx context.Context, subscription *subscriptionv1.RequestGetSubscription) (*subscriptionv1.ResponseGetSubscription, error) {
	resp, err := s.SubscriptionUseCase.GetSubscription(ctx, &dto.RequestGetSubscription{
		UserID: subscription.UserId,
	})
	if err != nil {
		return nil, mapDomainErr(err)
	}

	return &subscriptionv1.ResponseGetSubscription{
		UserId:  resp.UserID,
		EndAt:   timestamppb.New(resp.EndAt),
		StartAt: timestamppb.New(resp.StartAt),
		Active:  resp.Active,
	}, nil
}

func (s SubscriptionServer) CancelSubscription(ctx context.Context, subscription *subscriptionv1.RequestCancelSubscription) (*emptypb.Empty, error) {
	_, err := s.SubscriptionUseCase.CancelSubscription(ctx, &dto.RequestCancelSubscription{
		UserID: subscription.UserId,
	})
	if err != nil {
		return nil, mapDomainErr(err)
	}

	return &emptypb.Empty{}, nil
}

func mapDomainErr(err error) error {
	switch {
	case errors.Is(err, domain.ErrSubscriptionNotFound):
		return grpcerr.New(codes.NotFound, int32(subscriptionv1.SubscriptionErrorCode_SUBSCRIPTION_ERROR_NOT_FOUND), "subscription not found")
	case errors.Is(err, domain.ErrSubscriptionExpired):
		return grpcerr.New(codes.FailedPrecondition, int32(subscriptionv1.SubscriptionErrorCode_SUBSCRIPTION_ERROR_EXPIRED), "subscription expired")
	}
	return errors.New("unknown error")
}
