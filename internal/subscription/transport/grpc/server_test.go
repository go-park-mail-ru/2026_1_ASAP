//go:generate mockgen -destination=mock/subscription_usecase_mock.go -package=mock github.com/go-park-mail-ru/2026_1_ASAP/internal/subscription/transport/grpc SubscriptionUseCase
package grpc

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	subscriptionv1 "github.com/go-park-mail-ru/2026_1_ASAP/gen/go/subscription/v1"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/subscription/domain"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/subscription/dto"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/subscription/transport/grpc/mock"
)

func TestPositiveSubscriptionServer_ActivateSubscription(t *testing.T) {
	type fields struct {
		subscriptionUseCase *mock.MockSubscriptionUseCase
	}

	type args struct {
		ctx context.Context
		req *subscriptionv1.RequestActivateSubscription
	}

	now := time.Now()
	endAt := now.Add(30 * 24 * time.Hour)

	tests := []struct {
		prepare func(*fields)
		want    *subscriptionv1.ResponseActivateSubscription
		name    string
		args    args
	}{
		{
			name: "Successful activate subscription",
			prepare: func(f *fields) {
				f.subscriptionUseCase.EXPECT().ActivateSubscription(gomock.Any(), &dto.RequestActivateSubscription{
					UserID: 100,
					Days:   30,
				}).Return(&dto.ResponseActivateSubscription{
					UserID: 100,
					EndAt:  endAt,
				}, nil)
			},
			args: args{
				ctx: context.Background(),
				req: &subscriptionv1.RequestActivateSubscription{
					UserId: 100,
					Days:   30,
				},
			},
			want: &subscriptionv1.ResponseActivateSubscription{
				UserId: 100,
				EndAt:  timestamppb.New(endAt),
			},
		},
		{
			name: "Successful activate subscription with 7 days",
			prepare: func(f *fields) {
				endAt7 := time.Now().Add(7 * 24 * time.Hour)
				f.subscriptionUseCase.EXPECT().ActivateSubscription(gomock.Any(), &dto.RequestActivateSubscription{
					UserID: 200,
					Days:   7,
				}).Return(&dto.ResponseActivateSubscription{
					UserID: 200,
					EndAt:  endAt7,
				}, nil)
			},
			args: args{
				ctx: context.Background(),
				req: &subscriptionv1.RequestActivateSubscription{
					UserId: 200,
					Days:   7,
				},
			},
			want: &subscriptionv1.ResponseActivateSubscription{
				UserId: 200,
				EndAt:  timestamppb.New(time.Now().Add(7 * 24 * time.Hour)),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				subscriptionUseCase: mock.NewMockSubscriptionUseCase(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &SubscriptionServer{
				SubscriptionUseCase: f.subscriptionUseCase,
			}

			resp, err := s.ActivateSubscription(tt.args.ctx, tt.args.req)
			require.NoError(t, err)
			require.Equal(t, tt.want.GetUserId(), resp.GetUserId())
		})
	}
}

func TestNegativeSubscriptionServer_ActivateSubscription(t *testing.T) {
	type fields struct {
		subscriptionUseCase *mock.MockSubscriptionUseCase
	}

	type args struct {
		ctx context.Context
		req *subscriptionv1.RequestActivateSubscription
	}

	tests := []struct {
		wantCode   codes.Code
		wantErrMsg string
		prepare    func(*fields)
		name       string
		args       args
	}{
		{
			name: "UseCase returns error",
			prepare: func(f *fields) {
				f.subscriptionUseCase.EXPECT().ActivateSubscription(gomock.Any(), gomock.Any()).Return(nil, errors.New("internal error"))
			},
			args: args{
				ctx: context.Background(),
				req: &subscriptionv1.RequestActivateSubscription{
					UserId: 100,
					Days:   30,
				},
			},
			wantErrMsg: "unknown error",
		},
		{
			name: "UseCase returns subscription not found",
			prepare: func(f *fields) {
				f.subscriptionUseCase.EXPECT().ActivateSubscription(gomock.Any(), gomock.Any()).Return(nil, domain.ErrSubscriptionNotFound)
			},
			args: args{
				ctx: context.Background(),
				req: &subscriptionv1.RequestActivateSubscription{
					UserId: 100,
					Days:   30,
				},
			},
			wantCode: codes.NotFound,
		},
		{
			name: "UseCase returns subscription expired",
			prepare: func(f *fields) {
				f.subscriptionUseCase.EXPECT().ActivateSubscription(gomock.Any(), gomock.Any()).Return(nil, domain.ErrSubscriptionExpired)
			},
			args: args{
				ctx: context.Background(),
				req: &subscriptionv1.RequestActivateSubscription{
					UserId: 100,
					Days:   30,
				},
			},
			wantCode: codes.FailedPrecondition,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				subscriptionUseCase: mock.NewMockSubscriptionUseCase(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &SubscriptionServer{
				SubscriptionUseCase: f.subscriptionUseCase,
			}

			_, err := s.ActivateSubscription(tt.args.ctx, tt.args.req)
			require.Error(t, err)

			if tt.wantErrMsg != "" {
				require.EqualError(t, err, tt.wantErrMsg)
			} else {
				st, ok := status.FromError(err)
				require.True(t, ok)
				require.Equal(t, tt.wantCode, st.Code())
			}
		})
	}
}

func TestPositiveSubscriptionServer_GetSubscription(t *testing.T) {
	type fields struct {
		subscriptionUseCase *mock.MockSubscriptionUseCase
	}

	type args struct {
		ctx context.Context
		req *subscriptionv1.RequestGetSubscription
	}

	now := time.Now()
	startAt := now.Add(-30 * 24 * time.Hour)
	endAt := now.Add(30 * 24 * time.Hour)

	tests := []struct {
		prepare func(*fields)
		want    *subscriptionv1.ResponseGetSubscription
		name    string
		args    args
	}{
		{
			name: "Successful get active subscription",
			prepare: func(f *fields) {
				f.subscriptionUseCase.EXPECT().GetSubscription(gomock.Any(), &dto.RequestGetSubscription{
					UserID: 100,
				}).Return(&dto.ResponseGetSubscription{
					UserID:  100,
					Active:  true,
					StartAt: startAt,
					EndAt:   endAt,
				}, nil)
			},
			args: args{
				ctx: context.Background(),
				req: &subscriptionv1.RequestGetSubscription{
					UserId: 100,
				},
			},
			want: &subscriptionv1.ResponseGetSubscription{
				UserId:  100,
				Active:  true,
				StartAt: timestamppb.New(startAt),
				EndAt:   timestamppb.New(endAt),
			},
		},
		{
			name: "Successful get inactive subscription",
			prepare: func(f *fields) {
				f.subscriptionUseCase.EXPECT().GetSubscription(gomock.Any(), &dto.RequestGetSubscription{
					UserID: 200,
				}).Return(&dto.ResponseGetSubscription{
					UserID:  200,
					Active:  false,
					StartAt: startAt,
					EndAt:   endAt,
				}, nil)
			},
			args: args{
				ctx: context.Background(),
				req: &subscriptionv1.RequestGetSubscription{
					UserId: 200,
				},
			},
			want: &subscriptionv1.ResponseGetSubscription{
				UserId:  200,
				Active:  false,
				StartAt: timestamppb.New(startAt),
				EndAt:   timestamppb.New(endAt),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				subscriptionUseCase: mock.NewMockSubscriptionUseCase(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &SubscriptionServer{
				SubscriptionUseCase: f.subscriptionUseCase,
			}

			resp, err := s.GetSubscription(tt.args.ctx, tt.args.req)
			require.NoError(t, err)
			require.Equal(t, tt.want.GetUserId(), resp.GetUserId())
			require.Equal(t, tt.want.GetActive(), resp.GetActive())
		})
	}
}

func TestNegativeSubscriptionServer_GetSubscription(t *testing.T) {
	type fields struct {
		subscriptionUseCase *mock.MockSubscriptionUseCase
	}

	type args struct {
		ctx context.Context
		req *subscriptionv1.RequestGetSubscription
	}

	tests := []struct {
		wantCode   codes.Code
		wantErrMsg string
		prepare    func(*fields)
		name       string
		args       args
	}{
		{
			name: "UseCase returns error",
			prepare: func(f *fields) {
				f.subscriptionUseCase.EXPECT().GetSubscription(gomock.Any(), gomock.Any()).Return(nil, errors.New("internal error"))
			},
			args: args{
				ctx: context.Background(),
				req: &subscriptionv1.RequestGetSubscription{
					UserId: 100,
				},
			},
			wantErrMsg: "unknown error",
		},
		{
			name: "UseCase returns subscription not found",
			prepare: func(f *fields) {
				f.subscriptionUseCase.EXPECT().GetSubscription(gomock.Any(), gomock.Any()).Return(nil, domain.ErrSubscriptionNotFound)
			},
			args: args{
				ctx: context.Background(),
				req: &subscriptionv1.RequestGetSubscription{
					UserId: 100,
				},
			},
			wantCode: codes.NotFound,
		},
		{
			name: "UseCase returns subscription expired",
			prepare: func(f *fields) {
				f.subscriptionUseCase.EXPECT().GetSubscription(gomock.Any(), gomock.Any()).Return(nil, domain.ErrSubscriptionExpired)
			},
			args: args{
				ctx: context.Background(),
				req: &subscriptionv1.RequestGetSubscription{
					UserId: 100,
				},
			},
			wantCode: codes.FailedPrecondition,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				subscriptionUseCase: mock.NewMockSubscriptionUseCase(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &SubscriptionServer{
				SubscriptionUseCase: f.subscriptionUseCase,
			}

			_, err := s.GetSubscription(tt.args.ctx, tt.args.req)
			require.Error(t, err)

			if tt.wantErrMsg != "" {
				require.EqualError(t, err, tt.wantErrMsg)
			} else {
				st, ok := status.FromError(err)
				require.True(t, ok)
				require.Equal(t, tt.wantCode, st.Code())
			}
		})
	}
}

func TestPositiveSubscriptionServer_CancelSubscription(t *testing.T) {
	type fields struct {
		subscriptionUseCase *mock.MockSubscriptionUseCase
	}

	type args struct {
		ctx context.Context
		req *subscriptionv1.RequestCancelSubscription
	}

	tests := []struct {
		prepare func(*fields)
		want    *emptypb.Empty
		name    string
		args    args
	}{
		{
			name: "Successful cancel subscription",
			prepare: func(f *fields) {
				f.subscriptionUseCase.EXPECT().CancelSubscription(gomock.Any(), &dto.RequestCancelSubscription{
					UserID: 100,
				}).Return(&dto.ResponseCancelSubscription{
					UserID: 100,
				}, nil)
			},
			args: args{
				ctx: context.Background(),
				req: &subscriptionv1.RequestCancelSubscription{
					UserId: 100,
				},
			},
			want: &emptypb.Empty{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				subscriptionUseCase: mock.NewMockSubscriptionUseCase(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &SubscriptionServer{
				SubscriptionUseCase: f.subscriptionUseCase,
			}

			resp, err := s.CancelSubscription(tt.args.ctx, tt.args.req)
			require.NoError(t, err)
			require.NotNil(t, resp)
		})
	}
}

func TestNegativeSubscriptionServer_CancelSubscription(t *testing.T) {
	type fields struct {
		subscriptionUseCase *mock.MockSubscriptionUseCase
	}

	type args struct {
		ctx context.Context
		req *subscriptionv1.RequestCancelSubscription
	}

	tests := []struct {
		wantCode   codes.Code
		wantErrMsg string
		prepare    func(*fields)
		name       string
		args       args
	}{
		{
			name: "UseCase returns error",
			prepare: func(f *fields) {
				f.subscriptionUseCase.EXPECT().CancelSubscription(gomock.Any(), gomock.Any()).Return(nil, errors.New("internal error"))
			},
			args: args{
				ctx: context.Background(),
				req: &subscriptionv1.RequestCancelSubscription{
					UserId: 100,
				},
			},
			wantErrMsg: "unknown error",
		},
		{
			name: "UseCase returns subscription not found",
			prepare: func(f *fields) {
				f.subscriptionUseCase.EXPECT().CancelSubscription(gomock.Any(), gomock.Any()).Return(nil, domain.ErrSubscriptionNotFound)
			},
			args: args{
				ctx: context.Background(),
				req: &subscriptionv1.RequestCancelSubscription{
					UserId: 100,
				},
			},
			wantCode: codes.NotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				subscriptionUseCase: mock.NewMockSubscriptionUseCase(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &SubscriptionServer{
				SubscriptionUseCase: f.subscriptionUseCase,
			}

			_, err := s.CancelSubscription(tt.args.ctx, tt.args.req)
			require.Error(t, err)

			if tt.wantErrMsg != "" {
				require.EqualError(t, err, tt.wantErrMsg)
			} else {
				st, ok := status.FromError(err)
				require.True(t, ok)
				require.Equal(t, tt.wantCode, st.Code())
			}
		})
	}
}

func TestMapDomainErr(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantCode codes.Code
		wantMsg  string
	}{
		{
			name:     "Subscription not found",
			err:      domain.ErrSubscriptionNotFound,
			wantCode: codes.NotFound,
			wantMsg:  "rpc error: code = NotFound desc = subscription not found",
		},
		{
			name:     "Subscription expired",
			err:      domain.ErrSubscriptionExpired,
			wantCode: codes.FailedPrecondition,
			wantMsg:  "rpc error: code = FailedPrecondition desc = subscription expired",
		},
		{
			name:     "Unknown error",
			err:      errors.New("some error"),
			wantCode: codes.Unknown,
			wantMsg:  "unknown error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := mapDomainErr(tt.err)
			require.Error(t, err)
			require.EqualError(t, err, tt.wantMsg)

			// Для проверки grpc статуса (кроме unknown error)
			if tt.wantCode != codes.Unknown {
				st, ok := status.FromError(err)
				require.True(t, ok)
				require.Equal(t, tt.wantCode, st.Code())
			}
		})
	}
}
