package grpc

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	paymentv1 "github.com/go-park-mail-ru/2026_1_ASAP/gen/go/payment/v1"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/payment/domain"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/payment/dto"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/payment/transport/grpc/mock"
)

func paymentStringPtr(s string) *string {
	return &s
}

func responsePayment() *dto.ResponsePayment {
	return &dto.ResponsePayment{
		ID:               10,
		PaymentID:        "yoo-10",
		UserID:           77,
		Status:           "pending",
		Amount:           19900,
		SubscriptionDays: 30,
		PaymentURL:       paymentStringPtr("https://pay.local/10"),
		Message:          paymentStringPtr("created"),
		CreatedAt:        time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC),
		UpdatedAt:        time.Date(2026, 5, 2, 12, 0, 0, 0, time.UTC),
	}
}

func newTestPaymentServer(usecase PaymentUseCase) *PaymentServer {
	return &PaymentServer{PaymentUseCase: usecase, Logger: zap.NewNop()}
}

func requirePaymentDetails(t *testing.T, got *paymentv1.PaymentDetails, want *dto.ResponsePayment) {
	t.Helper()
	require.NotNil(t, got)
	require.Equal(t, want.ID, got.GetId())
	require.Equal(t, want.PaymentID, got.GetPaymentId())
	require.Equal(t, want.UserID, got.GetUserId())
	require.Equal(t, want.Status, got.GetStatus())
	require.Equal(t, want.Amount, got.GetAmount())
	require.Equal(t, want.SubscriptionDays, got.GetSubscriptionDays())
	require.Equal(t, *want.PaymentURL, got.GetPaymentUrl())
	require.Equal(t, *want.Message, got.GetMessage())
	require.True(t, timestamppb.New(want.CreatedAt).AsTime().Equal(got.GetCreatedAt().AsTime()))
	require.True(t, timestamppb.New(want.UpdatedAt).AsTime().Equal(got.GetUpdatedAt().AsTime()))
}

func TestPositivePaymentServer_CreatePayment(t *testing.T) {
	ctx := context.Background()

	type fields struct {
		usecase *mock.MockPaymentUseCase
	}
	type args struct {
		req *paymentv1.RequestCreatePayment
	}

	tests := []struct {
		prepare func(*fields)
		want    *dto.ResponsePayment
		name    string
		args    args
	}{
		{
			name: "creates payment",
			args: args{req: &paymentv1.RequestCreatePayment{
				UserId:           77,
				PaymentId:        "client-payment-id",
				Status:           "ignored",
				Amount:           19900,
				SubscriptionDays: 30,
			}},
			prepare: func(f *fields) {
				f.usecase.EXPECT().
					CreatePayment(ctx, &dto.RequestCreatePayment{
						UserID:           77,
						PaymentID:        "client-payment-id",
						Status:           "pending",
						Amount:           19900,
						SubscriptionDays: 30,
					}).
					Return(responsePayment(), nil)
			},
			want: responsePayment(),
		},
		{
			name: "nil usecase response",
			args: args{req: &paymentv1.RequestCreatePayment{
				UserId:           77,
				Amount:           19900,
				SubscriptionDays: 30,
			}},
			prepare: func(f *fields) {
				f.usecase.EXPECT().
					CreatePayment(ctx, &dto.RequestCreatePayment{
						UserID:           77,
						Status:           "pending",
						Amount:           19900,
						SubscriptionDays: 30,
					}).
					Return(nil, nil)
			},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{usecase: mock.NewMockPaymentUseCase(ctrl)}
			tt.prepare(&f)
			s := newTestPaymentServer(f.usecase)

			got, err := s.CreatePayment(ctx, tt.args.req)
			require.NoError(t, err)
			if tt.want == nil {
				require.Nil(t, got.GetPayment())
				return
			}
			requirePaymentDetails(t, got.GetPayment(), tt.want)
		})
	}
}

func TestNegativePaymentServer_CreatePayment(t *testing.T) {
	ctx := context.Background()

	type fields struct {
		usecase *mock.MockPaymentUseCase
	}
	type args struct {
		req *paymentv1.RequestCreatePayment
	}

	tests := []struct {
		prepare  func(*fields)
		name     string
		args     args
		wantCode codes.Code
	}{
		{
			name:     "nil request",
			args:     args{req: nil},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "invalid payment request",
			args: args{req: &paymentv1.RequestCreatePayment{
				UserId:           77,
				Amount:           0,
				SubscriptionDays: 30,
			}},
			prepare: func(f *fields) {
				f.usecase.EXPECT().CreatePayment(ctx, gomock.Any()).Return(nil, domain.ErrInvalidPaymentRequest)
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "duplicate payment",
			args: args{req: &paymentv1.RequestCreatePayment{
				UserId:           77,
				Amount:           19900,
				SubscriptionDays: 30,
			}},
			prepare: func(f *fields) {
				f.usecase.EXPECT().CreatePayment(ctx, gomock.Any()).Return(nil, domain.ErrDuplicatePayment)
			},
			wantCode: codes.AlreadyExists,
		},
		{
			name: "internal error",
			args: args{req: &paymentv1.RequestCreatePayment{
				UserId:           77,
				Amount:           19900,
				SubscriptionDays: 30,
			}},
			prepare: func(f *fields) {
				f.usecase.EXPECT().CreatePayment(ctx, gomock.Any()).Return(nil, errors.New("db down"))
			},
			wantCode: codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{usecase: mock.NewMockPaymentUseCase(ctrl)}
			if tt.prepare != nil {
				tt.prepare(&f)
			}
			s := newTestPaymentServer(f.usecase)

			got, err := s.CreatePayment(ctx, tt.args.req)
			require.Nil(t, got)
			require.Equal(t, tt.wantCode, status.Code(err))
		})
	}
}

func TestPositivePaymentServer_GetPayment(t *testing.T) {
	ctx := context.Background()

	type fields struct {
		usecase *mock.MockPaymentUseCase
	}
	type args struct {
		req *paymentv1.RequestGetPayment
	}

	tests := []struct {
		prepare func(*fields)
		want    *dto.ResponsePayment
		name    string
		args    args
	}{
		{
			name: "gets payment",
			args: args{req: &paymentv1.RequestGetPayment{Id: 10}},
			prepare: func(f *fields) {
				f.usecase.EXPECT().
					GetPayment(ctx, &dto.RequestGetPayment{ID: 10}).
					Return(responsePayment(), nil)
			},
			want: responsePayment(),
		},
		{
			name: "nil usecase response",
			args: args{req: &paymentv1.RequestGetPayment{Id: 10}},
			prepare: func(f *fields) {
				f.usecase.EXPECT().
					GetPayment(ctx, &dto.RequestGetPayment{ID: 10}).
					Return(nil, nil)
			},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{usecase: mock.NewMockPaymentUseCase(ctrl)}
			tt.prepare(&f)
			s := newTestPaymentServer(f.usecase)

			got, err := s.GetPayment(ctx, tt.args.req)
			require.NoError(t, err)
			if tt.want == nil {
				require.Nil(t, got.GetPayment())
				return
			}
			requirePaymentDetails(t, got.GetPayment(), tt.want)
		})
	}
}

func TestNegativePaymentServer_GetPayment(t *testing.T) {
	ctx := context.Background()

	type fields struct {
		usecase *mock.MockPaymentUseCase
	}
	type args struct {
		req *paymentv1.RequestGetPayment
	}

	tests := []struct {
		prepare  func(*fields)
		name     string
		args     args
		wantCode codes.Code
	}{
		{
			name:     "nil request",
			args:     args{req: nil},
			wantCode: codes.InvalidArgument,
		},
		{
			name:     "zero id",
			args:     args{req: &paymentv1.RequestGetPayment{}},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "not found",
			args: args{req: &paymentv1.RequestGetPayment{Id: 10}},
			prepare: func(f *fields) {
				f.usecase.EXPECT().GetPayment(ctx, &dto.RequestGetPayment{ID: 10}).Return(nil, domain.ErrPaymentNotFound)
			},
			wantCode: codes.NotFound,
		},
		{
			name: "internal error",
			args: args{req: &paymentv1.RequestGetPayment{Id: 10}},
			prepare: func(f *fields) {
				f.usecase.EXPECT().GetPayment(ctx, gomock.Any()).Return(nil, errors.New("db down"))
			},
			wantCode: codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{usecase: mock.NewMockPaymentUseCase(ctrl)}
			if tt.prepare != nil {
				tt.prepare(&f)
			}
			s := newTestPaymentServer(f.usecase)

			got, err := s.GetPayment(ctx, tt.args.req)
			require.Nil(t, got)
			require.Equal(t, tt.wantCode, status.Code(err))
		})
	}
}

func TestPositivePaymentServer_SyncOpenPayment(t *testing.T) {
	ctx := context.Background()

	type fields struct {
		usecase *mock.MockPaymentUseCase
	}
	type args struct {
		req *paymentv1.RequestSyncOpenPayment
	}

	tests := []struct {
		prepare func(*fields)
		want    *dto.ResponsePayment
		name    string
		args    args
	}{
		{
			name: "syncs open payment",
			args: args{req: &paymentv1.RequestSyncOpenPayment{UserId: 77}},
			prepare: func(f *fields) {
				f.usecase.EXPECT().
					SyncOpenPayment(ctx, &dto.RequestSyncOpenPayment{UserID: 77}).
					Return(responsePayment(), nil)
			},
			want: responsePayment(),
		},
		{
			name: "nil usecase response",
			args: args{req: &paymentv1.RequestSyncOpenPayment{UserId: 77}},
			prepare: func(f *fields) {
				f.usecase.EXPECT().
					SyncOpenPayment(ctx, &dto.RequestSyncOpenPayment{UserID: 77}).
					Return(nil, nil)
			},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{usecase: mock.NewMockPaymentUseCase(ctrl)}
			tt.prepare(&f)
			s := newTestPaymentServer(f.usecase)

			got, err := s.SyncOpenPayment(ctx, tt.args.req)
			require.NoError(t, err)
			if tt.want == nil {
				require.Nil(t, got.GetPayment())
				return
			}
			requirePaymentDetails(t, got.GetPayment(), tt.want)
		})
	}
}

func TestNegativePaymentServer_SyncOpenPayment(t *testing.T) {
	ctx := context.Background()

	type fields struct {
		usecase *mock.MockPaymentUseCase
	}
	type args struct {
		req *paymentv1.RequestSyncOpenPayment
	}

	tests := []struct {
		prepare  func(*fields)
		name     string
		args     args
		wantCode codes.Code
	}{
		{
			name:     "nil request",
			args:     args{req: nil},
			wantCode: codes.InvalidArgument,
		},
		{
			name:     "zero user id",
			args:     args{req: &paymentv1.RequestSyncOpenPayment{}},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "return url unset",
			args: args{req: &paymentv1.RequestSyncOpenPayment{UserId: 77}},
			prepare: func(f *fields) {
				f.usecase.EXPECT().SyncOpenPayment(ctx, gomock.Any()).Return(nil, domain.ErrPaymentReturnURLUnset)
			},
			wantCode: codes.FailedPrecondition,
		},
		{
			name: "internal error",
			args: args{req: &paymentv1.RequestSyncOpenPayment{UserId: 77}},
			prepare: func(f *fields) {
				f.usecase.EXPECT().SyncOpenPayment(ctx, gomock.Any()).Return(nil, errors.New("db down"))
			},
			wantCode: codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{usecase: mock.NewMockPaymentUseCase(ctrl)}
			if tt.prepare != nil {
				tt.prepare(&f)
			}
			s := newTestPaymentServer(f.usecase)

			got, err := s.SyncOpenPayment(ctx, tt.args.req)
			require.Nil(t, got)
			require.Equal(t, tt.wantCode, status.Code(err))
		})
	}
}

func TestPaymentServer_ProcessYooKassaWebhook(t *testing.T) {
	ctx := context.Background()

	type fields struct {
		usecase *mock.MockPaymentUseCase
	}
	type args struct {
		req *paymentv1.ProcessYooKassaWebhookRequest
	}

	tests := []struct {
		prepare  func(*fields)
		name     string
		args     args
		wantCode codes.Code
	}{
		{
			name: "nil request is ignored",
			args: args{req: nil},
		},
		{
			name: "empty body is ignored",
			args: args{req: &paymentv1.ProcessYooKassaWebhookRequest{}},
		},
		{
			name: "processes webhook",
			args: args{req: &paymentv1.ProcessYooKassaWebhookRequest{RawBody: []byte(`{"event":"payment.succeeded"}`)}},
			prepare: func(f *fields) {
				f.usecase.EXPECT().
					HandleYooKassaWebhook(ctx, []byte(`{"event":"payment.succeeded"}`)).
					Return(nil)
			},
		},
		{
			name: "webhook error",
			args: args{req: &paymentv1.ProcessYooKassaWebhookRequest{RawBody: []byte(`{"event":"payment.succeeded"}`)}},
			prepare: func(f *fields) {
				f.usecase.EXPECT().HandleYooKassaWebhook(ctx, gomock.Any()).Return(errors.New("failed"))
			},
			wantCode: codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{usecase: mock.NewMockPaymentUseCase(ctrl)}
			if tt.prepare != nil {
				tt.prepare(&f)
			}
			s := newTestPaymentServer(f.usecase)

			got, err := s.ProcessYooKassaWebhook(ctx, tt.args.req)
			if tt.wantCode != codes.OK {
				require.Nil(t, got)
				require.Equal(t, tt.wantCode, status.Code(err))
				return
			}
			require.NoError(t, err)
			require.NotNil(t, got)
		})
	}
}

func TestPaymentServer_Helpers(t *testing.T) {
	t.Run("payment details nil", func(t *testing.T) {
		require.Nil(t, paymentDetailsToProto(nil))
	})

	t.Run("payment details without optional fields", func(t *testing.T) {
		resp := responsePayment()
		resp.PaymentURL = nil
		resp.Message = nil
		got := paymentDetailsToProto(resp)

		require.NotNil(t, got)
		require.Empty(t, got.GetPaymentUrl())
		require.Empty(t, got.GetMessage())
		require.Nil(t, got.PaymentUrl)
		require.Nil(t, got.Message)
	})

	t.Run("map payment errors", func(t *testing.T) {
		tests := []struct {
			err  error
			name string
			want codes.Code
		}{
			{name: "not found", err: domain.ErrPaymentNotFound, want: codes.NotFound},
			{name: "duplicate", err: domain.ErrDuplicatePayment, want: codes.AlreadyExists},
			{name: "invalid", err: domain.ErrInvalidPaymentRequest, want: codes.InvalidArgument},
			{name: "return url unset", err: domain.ErrPaymentReturnURLUnset, want: codes.FailedPrecondition},
			{name: "internal", err: errors.New("db down"), want: codes.Internal},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				require.Equal(t, tt.want, status.Code(mapPaymentErr(tt.err)))
			})
		}
	})

	t.Run("nil logger", func(t *testing.T) {
		s := &PaymentServer{}
		require.NotNil(t, s.Log(context.Background()))
	})
}
