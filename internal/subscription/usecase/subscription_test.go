//go:generate mockgen -destination=mock/subscription_repo_mock.go -package=mock github.com/go-park-mail-ru/2026_1_ASAP/internal/subscription/usecase SubscriptionRepository
package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"github.com/go-park-mail-ru/2026_1_ASAP/internal/subscription/domain"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/subscription/dto"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/subscription/usecase/mock"
)

func TestPositiveSubscriptionUseCase_ActivateSubscription(t *testing.T) {
	type fields struct {
		subscriptionRepo *mock.MockSubscriptionRepository
	}

	type args struct {
		ctx     context.Context
		request *dto.RequestActivateSubscription
	}

	now := time.Now()
	endAt := now.Add(30 * 24 * time.Hour)

	tests := []struct {
		prepare func(*fields)
		want    *dto.ResponseActivateSubscription
		name    string
		args    args
	}{
		{
			name: "Successful activate subscription",
			prepare: func(f *fields) {
				f.subscriptionRepo.EXPECT().SubscriptionSet(gomock.Any(), gomock.AssignableToTypeOf(&domain.Subscription{})).
					DoAndReturn(func(_ context.Context, sub *domain.Subscription) (*domain.Subscription, error) {
						return &domain.Subscription{
							UserID:  sub.UserID,
							Active:  true,
							StartAt: sub.StartAt,
							EndAt:   sub.EndAt,
						}, nil
					})
			},
			args: args{
				ctx: context.Background(),
				request: &dto.RequestActivateSubscription{
					UserID: 100,
					Days:   30,
				},
			},
			want: &dto.ResponseActivateSubscription{
				UserID: 100,
				EndAt:  endAt,
			},
		},
		{
			name: "Successful activate subscription with 7 days",
			prepare: func(f *fields) {
				f.subscriptionRepo.EXPECT().SubscriptionSet(gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, sub *domain.Subscription) (*domain.Subscription, error) {
						return &domain.Subscription{
							UserID:  sub.UserID,
							Active:  true,
							StartAt: sub.StartAt,
							EndAt:   sub.EndAt,
						}, nil
					})
			},
			args: args{
				ctx: context.Background(),
				request: &dto.RequestActivateSubscription{
					UserID: 200,
					Days:   7,
				},
			},
			want: &dto.ResponseActivateSubscription{
				UserID: 200,
				EndAt:  time.Now().Add(7 * 24 * time.Hour),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				subscriptionRepo: mock.NewMockSubscriptionRepository(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &SubscriptionUseCase{
				subscriptionRepository: f.subscriptionRepo,
			}

			result, err := s.ActivateSubscription(tt.args.ctx, tt.args.request)
			require.NoError(t, err)
			require.Equal(t, tt.want.UserID, result.UserID)
		})
	}
}

func TestNegativeSubscriptionUseCase_ActivateSubscription(t *testing.T) {
	type fields struct {
		subscriptionRepo *mock.MockSubscriptionRepository
	}

	type args struct {
		ctx     context.Context
		request *dto.RequestActivateSubscription
	}

	tests := []struct {
		wantErr    error
		prepare    func(*fields)
		name       string
		args       args
		wantAnyErr bool
	}{
		{
			name: "Nil request",
			args: args{
				ctx:     context.Background(),
				request: nil,
			},
			wantErr: errors.New("invalid request"),
		},
		{
			name: "Invalid user ID (zero)",
			args: args{
				ctx: context.Background(),
				request: &dto.RequestActivateSubscription{
					UserID: 0,
					Days:   30,
				},
			},
			wantErr: errors.New("invalid request"),
		},
		{
			name: "Invalid days (zero)",
			args: args{
				ctx: context.Background(),
				request: &dto.RequestActivateSubscription{
					UserID: 100,
					Days:   0,
				},
			},
			wantErr: errors.New("invalid request"),
		},
		{
			name: "Invalid days (negative)",
			args: args{
				ctx: context.Background(),
				request: &dto.RequestActivateSubscription{
					UserID: 100,
					Days:   -5,
				},
			},
			wantErr: errors.New("invalid request"),
		},
		{
			name: "Repository error",
			prepare: func(f *fields) {
				f.subscriptionRepo.EXPECT().SubscriptionSet(gomock.Any(), gomock.Any()).Return(nil, errors.New("db error"))
			},
			args: args{
				ctx: context.Background(),
				request: &dto.RequestActivateSubscription{
					UserID: 100,
					Days:   30,
				},
			},
			wantAnyErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				subscriptionRepo: mock.NewMockSubscriptionRepository(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &SubscriptionUseCase{
				subscriptionRepository: f.subscriptionRepo,
			}

			result, err := s.ActivateSubscription(tt.args.ctx, tt.args.request)
			require.Nil(t, result)
			if tt.wantAnyErr {
				require.Error(t, err)
			} else {
				require.EqualError(t, err, tt.wantErr.Error())
			}
		})
	}
}

func TestPositiveSubscriptionUseCase_CancelSubscription(t *testing.T) {
	type fields struct {
		subscriptionRepo *mock.MockSubscriptionRepository
	}

	type args struct {
		ctx     context.Context
		request *dto.RequestCancelSubscription
	}

	now := time.Now()
	startAt := now.Add(-30 * 24 * time.Hour)
	endAt := now.Add(30 * 24 * time.Hour)

	tests := []struct {
		prepare func(*fields)
		want    *dto.ResponseCancelSubscription
		name    string
		args    args
	}{
		{
			name: "Successful cancel subscription",
			prepare: func(f *fields) {
				f.subscriptionRepo.EXPECT().SubscriptionGet(gomock.Any(), int64(100)).Return(&domain.Subscription{
					UserID:  100,
					Active:  true,
					StartAt: startAt,
					EndAt:   endAt,
				}, nil)

				f.subscriptionRepo.EXPECT().SubscriptionSet(gomock.Any(), &domain.Subscription{
					UserID:  100,
					Active:  false,
					StartAt: startAt,
					EndAt:   endAt,
				}).Return(&domain.Subscription{
					UserID:  100,
					Active:  false,
					StartAt: startAt,
					EndAt:   endAt,
				}, nil)
			},
			args: args{
				ctx: context.Background(),
				request: &dto.RequestCancelSubscription{
					UserID: 100,
				},
			},
			want: &dto.ResponseCancelSubscription{
				UserID: 100,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				subscriptionRepo: mock.NewMockSubscriptionRepository(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &SubscriptionUseCase{
				subscriptionRepository: f.subscriptionRepo,
			}

			result, err := s.CancelSubscription(tt.args.ctx, tt.args.request)
			require.NoError(t, err)
			require.Equal(t, tt.want.UserID, result.UserID)
		})
	}
}

func TestNegativeSubscriptionUseCase_CancelSubscription(t *testing.T) {
	type fields struct {
		subscriptionRepo *mock.MockSubscriptionRepository
	}

	type args struct {
		ctx     context.Context
		request *dto.RequestCancelSubscription
	}

	tests := []struct {
		wantErr    error
		prepare    func(*fields)
		name       string
		args       args
		wantAnyErr bool
	}{
		{
			name: "Nil request",
			args: args{
				ctx:     context.Background(),
				request: nil,
			},
			wantErr: errors.New("invalid request"),
		},
		{
			name: "Invalid user ID (zero)",
			args: args{
				ctx: context.Background(),
				request: &dto.RequestCancelSubscription{
					UserID: 0,
				},
			},
			wantErr: errors.New("invalid request"),
		},
		{
			name: "Subscription not found",
			prepare: func(f *fields) {
				f.subscriptionRepo.EXPECT().SubscriptionGet(gomock.Any(), int64(100)).Return(nil, domain.ErrSubscriptionNotFound)
			},
			args: args{
				ctx: context.Background(),
				request: &dto.RequestCancelSubscription{
					UserID: 100,
				},
			},
			wantErr: domain.ErrSubscriptionNotFound,
		},
		{
			name: "Repository Get error",
			prepare: func(f *fields) {
				f.subscriptionRepo.EXPECT().SubscriptionGet(gomock.Any(), int64(100)).Return(nil, errors.New("db error"))
			},
			args: args{
				ctx: context.Background(),
				request: &dto.RequestCancelSubscription{
					UserID: 100,
				},
			},
			wantAnyErr: true,
		},
		{
			name: "Repository Set error after get",
			prepare: func(f *fields) {
				f.subscriptionRepo.EXPECT().SubscriptionGet(gomock.Any(), int64(100)).Return(&domain.Subscription{
					UserID: 100,
					Active: true,
				}, nil)

				f.subscriptionRepo.EXPECT().SubscriptionSet(gomock.Any(), gomock.Any()).Return(nil, errors.New("update failed"))
			},
			args: args{
				ctx: context.Background(),
				request: &dto.RequestCancelSubscription{
					UserID: 100,
				},
			},
			wantAnyErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				subscriptionRepo: mock.NewMockSubscriptionRepository(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &SubscriptionUseCase{
				subscriptionRepository: f.subscriptionRepo,
			}

			result, err := s.CancelSubscription(tt.args.ctx, tt.args.request)
			require.Nil(t, result)
			if tt.wantAnyErr {
				require.Error(t, err)
			} else {
				require.EqualError(t, err, tt.wantErr.Error())
			}
		})
	}
}

func TestPositiveSubscriptionUseCase_GetSubscription(t *testing.T) {
	type fields struct {
		subscriptionRepo *mock.MockSubscriptionRepository
	}

	type args struct {
		ctx     context.Context
		request *dto.RequestGetSubscription
	}

	now := time.Now()
	activeStartAt := now.Add(-30 * 24 * time.Hour)
	activeEndAt := now.Add(30 * 24 * time.Hour)

	expiredEndAt := now.Add(-1 * time.Hour)

	tests := []struct {
		prepare func(*fields)
		want    *dto.ResponseGetSubscription
		name    string
		args    args
	}{
		{
			name: "Successful get active subscription",
			prepare: func(f *fields) {
				f.subscriptionRepo.EXPECT().SubscriptionGet(gomock.Any(), int64(100)).Return(&domain.Subscription{
					UserID:  100,
					Active:  true,
					StartAt: activeStartAt,
					EndAt:   activeEndAt,
				}, nil)
			},
			args: args{
				ctx: context.Background(),
				request: &dto.RequestGetSubscription{
					UserID: 100,
				},
			},
			want: &dto.ResponseGetSubscription{
				UserID:  100,
				Active:  true,
				StartAt: activeStartAt,
				EndAt:   activeEndAt,
			},
		},
		{
			name: "Get subscription that expired - auto deactivate",
			prepare: func(f *fields) {
				f.subscriptionRepo.EXPECT().SubscriptionGet(gomock.Any(), int64(200)).Return(&domain.Subscription{
					UserID:  200,
					Active:  true,
					StartAt: activeStartAt,
					EndAt:   expiredEndAt,
				}, nil)

				f.subscriptionRepo.EXPECT().SubscriptionSet(gomock.Any(), &domain.Subscription{
					UserID:  200,
					Active:  false,
					StartAt: activeStartAt,
					EndAt:   expiredEndAt,
				}).Return(&domain.Subscription{
					UserID:  200,
					Active:  false,
					StartAt: activeStartAt,
					EndAt:   expiredEndAt,
				}, nil)
			},
			args: args{
				ctx: context.Background(),
				request: &dto.RequestGetSubscription{
					UserID: 200,
				},
			},
			want: &dto.ResponseGetSubscription{
				UserID:  200,
				Active:  false,
				StartAt: activeStartAt,
				EndAt:   expiredEndAt,
			},
		},
		{
			name: "Get inactive subscription",
			prepare: func(f *fields) {
				f.subscriptionRepo.EXPECT().SubscriptionGet(gomock.Any(), int64(300)).Return(&domain.Subscription{
					UserID:  300,
					Active:  false,
					StartAt: activeStartAt,
					EndAt:   activeEndAt,
				}, nil)
			},
			args: args{
				ctx: context.Background(),
				request: &dto.RequestGetSubscription{
					UserID: 300,
				},
			},
			want: &dto.ResponseGetSubscription{
				UserID:  300,
				Active:  false,
				StartAt: activeStartAt,
				EndAt:   activeEndAt,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				subscriptionRepo: mock.NewMockSubscriptionRepository(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &SubscriptionUseCase{
				subscriptionRepository: f.subscriptionRepo,
			}

			result, err := s.GetSubscription(tt.args.ctx, tt.args.request)
			require.NoError(t, err)
			require.Equal(t, tt.want.UserID, result.UserID)
			require.Equal(t, tt.want.Active, result.Active)
		})
	}
}

func TestNegativeSubscriptionUseCase_GetSubscription(t *testing.T) {
	type fields struct {
		subscriptionRepo *mock.MockSubscriptionRepository
	}

	type args struct {
		ctx     context.Context
		request *dto.RequestGetSubscription
	}

	tests := []struct {
		wantErr    error
		prepare    func(*fields)
		name       string
		args       args
		wantAnyErr bool
	}{
		{
			name: "Nil request",
			args: args{
				ctx:     context.Background(),
				request: nil,
			},
			wantErr: errors.New("invalid request"),
		},
		{
			name: "Invalid user ID (zero)",
			args: args{
				ctx: context.Background(),
				request: &dto.RequestGetSubscription{
					UserID: 0,
				},
			},
			wantErr: errors.New("invalid request"),
		},
		{
			name: "Subscription not found",
			prepare: func(f *fields) {
				f.subscriptionRepo.EXPECT().SubscriptionGet(gomock.Any(), int64(100)).Return(nil, domain.ErrSubscriptionNotFound)
			},
			args: args{
				ctx: context.Background(),
				request: &dto.RequestGetSubscription{
					UserID: 100,
				},
			},
			wantErr: domain.ErrSubscriptionNotFound,
		},
		{
			name: "Repository Get error",
			prepare: func(f *fields) {
				f.subscriptionRepo.EXPECT().SubscriptionGet(gomock.Any(), int64(100)).Return(nil, errors.New("db error"))
			},
			args: args{
				ctx: context.Background(),
				request: &dto.RequestGetSubscription{
					UserID: 100,
				},
			},
			wantAnyErr: true,
		},
		{
			name: "Nil subscription returned",
			prepare: func(f *fields) {
				f.subscriptionRepo.EXPECT().SubscriptionGet(gomock.Any(), int64(100)).Return(nil, nil)
			},
			args: args{
				ctx: context.Background(),
				request: &dto.RequestGetSubscription{
					UserID: 100,
				},
			},
			wantErr: domain.ErrSubscriptionNotFound,
		},
		{
			name: "Expired subscription update fails",
			prepare: func(f *fields) {
				expiredEndAt := time.Now().Add(-1 * time.Hour)
				f.subscriptionRepo.EXPECT().SubscriptionGet(gomock.Any(), int64(100)).Return(&domain.Subscription{
					UserID:  100,
					Active:  true,
					StartAt: time.Now().Add(-30 * 24 * time.Hour),
					EndAt:   expiredEndAt,
				}, nil)

				f.subscriptionRepo.EXPECT().SubscriptionSet(gomock.Any(), gomock.Any()).Return(nil, errors.New("update failed"))
			},
			args: args{
				ctx: context.Background(),
				request: &dto.RequestGetSubscription{
					UserID: 100,
				},
			},
			wantErr: domain.ErrSubscriptionExpired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				subscriptionRepo: mock.NewMockSubscriptionRepository(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &SubscriptionUseCase{
				subscriptionRepository: f.subscriptionRepo,
			}

			result, err := s.GetSubscription(tt.args.ctx, tt.args.request)
			require.Nil(t, result)
			if tt.wantAnyErr {
				require.Error(t, err)
			} else {
				require.EqualError(t, err, tt.wantErr.Error())
			}
		})
	}
}

func TestNewSubscriptionUseCase(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mock.NewMockSubscriptionRepository(ctrl)
	uc := NewSubscriptionUseCase(mockRepo)

	require.NotNil(t, uc)
	require.Equal(t, mockRepo, uc.subscriptionRepository)
}