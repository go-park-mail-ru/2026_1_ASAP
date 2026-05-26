package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	yoopayment "github.com/rvinnie/yookassa-sdk-go/yookassa/payment"
	"github.com/stretchr/testify/require"

	"github.com/go-park-mail-ru/2026_1_ASAP/internal/payment/domain"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/payment/dto"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/payment/usecase/mock"
)

func paymentPtr(s string) *string {
	return &s
}

func testPayment(status domain.PaymentStatus) *domain.Payment {
	return &domain.Payment{
		ID:               10,
		PaymentID:        "yoo-10",
		UserID:           77,
		Status:           status,
		Amount:           19900,
		SubscriptionDays: 30,
		PaymentURL:       paymentPtr("https://pay.local/10"),
		CreatedAt:        time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC),
		UpdatedAt:        time.Date(2026, 5, 2, 12, 0, 0, 0, time.UTC),
	}
}

func testYooPayment(id string, status yoopayment.Status) *yoopayment.Payment {
	return &yoopayment.Payment{
		ID:     id,
		Status: status,
		Confirmation: map[string]interface{}{
			"confirmation_url": "https://pay.local/" + id,
		},
	}
}

func assertPaymentResponse(t *testing.T, got *dto.ResponsePayment, want *domain.Payment) {
	t.Helper()
	require.NotNil(t, got)
	require.Equal(t, want.ID, got.ID)
	require.Equal(t, want.PaymentID, got.PaymentID)
	require.Equal(t, want.UserID, got.UserID)
	require.Equal(t, string(want.Status), got.Status)
	require.Equal(t, want.Amount, got.Amount)
	require.Equal(t, want.SubscriptionDays, got.SubscriptionDays)
	require.Equal(t, want.PaymentURL, got.PaymentURL)
	require.Equal(t, want.Message, got.Message)
	require.Equal(t, want.CreatedAt, got.CreatedAt)
	require.Equal(t, want.UpdatedAt, got.UpdatedAt)
}

func TestPaymentUseCase_CreatePayment_Positive(t *testing.T) {
	ctx := context.Background()

	type fields struct {
		repo         *mock.MockPaymentRepository
		subscription *mock.MockSubscriptionService
		yookassa     *mock.MockYooKassaClient
	}
	type args struct {
		req *dto.RequestCreatePayment
	}

	tests := []struct {
		prepare   func(f *fields)
		assert    func(t *testing.T, got *dto.ResponsePayment)
		name      string
		returnURL string
		args      args
	}{
		{
			name:      "returns existing open payment",
			returnURL: "https://app.local/payment/return",
			args: args{req: &dto.RequestCreatePayment{
				UserID:           77,
				Amount:           19900,
				SubscriptionDays: 30,
			}},
			prepare: func(f *fields) {
				f.repo.EXPECT().
					PaymentGetOpenPendingByUser(ctx, int64(77)).
					Return(testPayment(domain.PaymentStatusPending), nil)
			},
			assert: func(t *testing.T, got *dto.ResponsePayment) {
				t.Helper()
				assertPaymentResponse(t, got, testPayment(domain.PaymentStatusPending))
			},
		},
		{
			name:      "creates payment in yookassa and repository",
			returnURL: "https://app.local/payment/return",
			args: args{req: &dto.RequestCreatePayment{
				UserID:           77,
				Amount:           19900,
				SubscriptionDays: 30,
			}},
			prepare: func(f *fields) {
				f.repo.EXPECT().
					PaymentGetOpenPendingByUser(ctx, int64(77)).
					Return(nil, domain.ErrPaymentNotFound)
				f.yookassa.EXPECT().
					CreatePayment(ctx, gomock.Any()).
					DoAndReturn(func(_ context.Context, p *yoopayment.Payment) (*yoopayment.Payment, error) {
						require.Equal(t, "19900.00", p.Amount.Value)
						require.Equal(t, "RUB", p.Amount.Currency)
						require.Equal(t, "Subscription for user 77", p.Description)
						require.True(t, p.Capture)
						redirect, ok := p.Confirmation.(yoopayment.Redirect)
						require.True(t, ok)
						require.Equal(t, "https://app.local/payment/return", redirect.ReturnURL)
						return testYooPayment("yoo-10", yoopayment.Pending), nil
					})
				f.repo.EXPECT().
					PaymentCreate(ctx, gomock.Any()).
					DoAndReturn(func(_ context.Context, p *domain.Payment) (*domain.Payment, error) {
						require.Equal(t, "yoo-10", p.PaymentID)
						require.Equal(t, int64(77), p.UserID)
						require.Equal(t, domain.PaymentStatusPending, p.Status)
						require.Equal(t, int32(19900), p.Amount)
						require.Equal(t, int32(30), p.SubscriptionDays)
						require.Equal(t, paymentPtr("https://pay.local/yoo-10"), p.PaymentURL)
						return testPayment(domain.PaymentStatusPending), nil
					})
			},
			assert: func(t *testing.T, got *dto.ResponsePayment) {
				t.Helper()
				assertPaymentResponse(t, got, testPayment(domain.PaymentStatusPending))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				repo:         mock.NewMockPaymentRepository(ctrl),
				subscription: mock.NewMockSubscriptionService(ctrl),
				yookassa:     mock.NewMockYooKassaClient(ctrl),
			}
			if tt.prepare != nil {
				tt.prepare(&f)
			}
			u := NewPaymentUseCase(f.repo, f.subscription, f.yookassa, tt.returnURL)

			got, err := u.CreatePayment(ctx, tt.args.req)
			require.NoError(t, err)
			tt.assert(t, got)
		})
	}
}

func TestPaymentUseCase_CreatePayment_Negative(t *testing.T) {
	ctx := context.Background()

	type fields struct {
		repo         *mock.MockPaymentRepository
		subscription *mock.MockSubscriptionService
		yookassa     *mock.MockYooKassaClient
	}
	type args struct {
		req *dto.RequestCreatePayment
	}

	tests := []struct {
		prepare   func(f *fields)
		wantErr   error
		name      string
		returnURL string
		args      args
		wantText  string
	}{
		{
			name:      "nil request",
			returnURL: "https://app.local/payment/return",
			args:      args{req: nil},
			wantErr:   domain.ErrInvalidPaymentRequest,
		},
		{
			name:      "invalid amount",
			returnURL: "https://app.local/payment/return",
			args: args{req: &dto.RequestCreatePayment{
				UserID:           77,
				Amount:           0,
				SubscriptionDays: 30,
			}},
			wantErr: domain.ErrInvalidPaymentRequest,
		},
		{
			name: "return url unset",
			args: args{req: &dto.RequestCreatePayment{
				UserID:           77,
				Amount:           19900,
				SubscriptionDays: 30,
			}},
			wantErr: domain.ErrPaymentReturnURLUnset,
		},
		{
			name:      "open payment lookup error",
			returnURL: "https://app.local/payment/return",
			args: args{req: &dto.RequestCreatePayment{
				UserID:           77,
				Amount:           19900,
				SubscriptionDays: 30,
			}},
			prepare: func(f *fields) {
				f.repo.EXPECT().
					PaymentGetOpenPendingByUser(ctx, int64(77)).
					Return(nil, errors.New("db down"))
			},
			wantText: "payment get open pending",
		},
		{
			name:      "yookassa create error",
			returnURL: "https://app.local/payment/return",
			args: args{req: &dto.RequestCreatePayment{
				UserID:           77,
				Amount:           19900,
				SubscriptionDays: 30,
			}},
			prepare: func(f *fields) {
				f.repo.EXPECT().
					PaymentGetOpenPendingByUser(ctx, int64(77)).
					Return(nil, domain.ErrPaymentNotFound)
				f.yookassa.EXPECT().
					CreatePayment(ctx, gomock.Any()).
					Return(nil, errors.New("network down"))
			},
			wantText: "yookassa create payment",
		},
		{
			name:      "repository create error",
			returnURL: "https://app.local/payment/return",
			args: args{req: &dto.RequestCreatePayment{
				UserID:           77,
				Amount:           19900,
				SubscriptionDays: 30,
			}},
			prepare: func(f *fields) {
				f.repo.EXPECT().
					PaymentGetOpenPendingByUser(ctx, int64(77)).
					Return(nil, domain.ErrPaymentNotFound)
				f.yookassa.EXPECT().
					CreatePayment(ctx, gomock.Any()).
					Return(testYooPayment("yoo-10", yoopayment.Pending), nil)
				f.repo.EXPECT().
					PaymentCreate(ctx, gomock.Any()).
					Return(nil, errors.New("db down"))
			},
			wantText: "payment create",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				repo:         mock.NewMockPaymentRepository(ctrl),
				subscription: mock.NewMockSubscriptionService(ctrl),
				yookassa:     mock.NewMockYooKassaClient(ctrl),
			}
			if tt.prepare != nil {
				tt.prepare(&f)
			}
			u := NewPaymentUseCase(f.repo, f.subscription, f.yookassa, tt.returnURL)

			got, err := u.CreatePayment(ctx, tt.args.req)
			require.Nil(t, got)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				require.ErrorContains(t, err, tt.wantText)
			}
		})
	}
}

func TestPaymentUseCase_GetPayment(t *testing.T) {
	ctx := context.Background()

	type fields struct {
		repo         *mock.MockPaymentRepository
		subscription *mock.MockSubscriptionService
		yookassa     *mock.MockYooKassaClient
	}
	type args struct {
		req *dto.RequestGetPayment
	}

	tests := []struct {
		prepare  func(f *fields)
		assert   func(t *testing.T, got *dto.ResponsePayment, err error)
		name     string
		args     args
		positive bool
	}{
		{
			name:     "returns payment",
			positive: true,
			args:     args{req: &dto.RequestGetPayment{ID: 10}},
			prepare: func(f *fields) {
				f.repo.EXPECT().PaymentGetByID(ctx, int64(10)).Return(testPayment(domain.PaymentStatusSucceeded), nil)
			},
			assert: func(t *testing.T, got *dto.ResponsePayment, err error) {
				t.Helper()
				require.NoError(t, err)
				assertPaymentResponse(t, got, testPayment(domain.PaymentStatusSucceeded))
			},
		},
		{
			name: "nil request",
			args: args{req: nil},
			assert: func(t *testing.T, got *dto.ResponsePayment, err error) {
				t.Helper()
				require.Nil(t, got)
				require.ErrorIs(t, err, domain.ErrInvalidPaymentRequest)
			},
		},
		{
			name: "repo error",
			args: args{req: &dto.RequestGetPayment{ID: 10}},
			prepare: func(f *fields) {
				f.repo.EXPECT().PaymentGetByID(ctx, int64(10)).Return(nil, domain.ErrPaymentNotFound)
			},
			assert: func(t *testing.T, got *dto.ResponsePayment, err error) {
				t.Helper()
				require.Nil(t, got)
				require.ErrorContains(t, err, "payment get")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				repo:         mock.NewMockPaymentRepository(ctrl),
				subscription: mock.NewMockSubscriptionService(ctrl),
				yookassa:     mock.NewMockYooKassaClient(ctrl),
			}
			if tt.prepare != nil {
				tt.prepare(&f)
			}
			u := NewPaymentUseCase(f.repo, f.subscription, f.yookassa, "https://app.local/payment/return")

			got, err := u.GetPayment(ctx, tt.args.req)
			tt.assert(t, got, err)
		})
	}
}

func TestPaymentUseCase_SyncOpenPayment_Positive(t *testing.T) {
	ctx := context.Background()

	type fields struct {
		repo         *mock.MockPaymentRepository
		subscription *mock.MockSubscriptionService
		yookassa     *mock.MockYooKassaClient
	}
	type args struct {
		req *dto.RequestSyncOpenPayment
	}

	tests := []struct {
		prepare func(f *fields)
		assert  func(t *testing.T, got *dto.ResponsePayment)
		name    string
		args    args
	}{
		{
			name: "syncs succeeded payment and activates subscription",
			args: args{req: &dto.RequestSyncOpenPayment{UserID: 77}},
			prepare: func(f *fields) {
				open := testPayment(domain.PaymentStatusPending)
				updated := testPayment(domain.PaymentStatusSucceeded)
				f.repo.EXPECT().PaymentGetOpenPendingByUser(ctx, int64(77)).Return(open, nil)
				f.yookassa.EXPECT().FindPayment(ctx, "yoo-10").Return(testYooPayment("yoo-10", yoopayment.Succeeded), nil)
				f.repo.EXPECT().
					PaymentUpdate(ctx, gomock.Any()).
					DoAndReturn(func(_ context.Context, p *domain.Payment) (*domain.Payment, error) {
						require.Equal(t, domain.PaymentStatusSucceeded, p.Status)
						require.Equal(t, paymentPtr("https://pay.local/yoo-10"), p.PaymentURL)
						return updated, nil
					})
				f.subscription.EXPECT().Activate(ctx, int64(77), int64(30)).Return(nil)
			},
			assert: func(t *testing.T, got *dto.ResponsePayment) {
				t.Helper()
				assertPaymentResponse(t, got, testPayment(domain.PaymentStatusSucceeded))
			},
		},
		{
			name: "syncs already succeeded payment without reactivation",
			args: args{req: &dto.RequestSyncOpenPayment{UserID: 77}},
			prepare: func(f *fields) {
				open := testPayment(domain.PaymentStatusSucceeded)
				updated := testPayment(domain.PaymentStatusSucceeded)
				f.repo.EXPECT().PaymentGetOpenPendingByUser(ctx, int64(77)).Return(open, nil)
				f.yookassa.EXPECT().FindPayment(ctx, "yoo-10").Return(testYooPayment("yoo-10", yoopayment.Succeeded), nil)
				f.repo.EXPECT().PaymentUpdate(ctx, gomock.Any()).Return(updated, nil)
			},
			assert: func(t *testing.T, got *dto.ResponsePayment) {
				t.Helper()
				assertPaymentResponse(t, got, testPayment(domain.PaymentStatusSucceeded))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				repo:         mock.NewMockPaymentRepository(ctrl),
				subscription: mock.NewMockSubscriptionService(ctrl),
				yookassa:     mock.NewMockYooKassaClient(ctrl),
			}
			if tt.prepare != nil {
				tt.prepare(&f)
			}
			u := NewPaymentUseCase(f.repo, f.subscription, f.yookassa, "https://app.local/payment/return")

			got, err := u.SyncOpenPayment(ctx, tt.args.req)
			require.NoError(t, err)
			tt.assert(t, got)
		})
	}
}

func TestPaymentUseCase_SyncOpenPayment_Negative(t *testing.T) {
	ctx := context.Background()

	type fields struct {
		repo         *mock.MockPaymentRepository
		subscription *mock.MockSubscriptionService
		yookassa     *mock.MockYooKassaClient
	}
	type args struct {
		req *dto.RequestSyncOpenPayment
	}

	tests := []struct {
		prepare  func(f *fields)
		wantErr  error
		name     string
		args     args
		wantText string
	}{
		{
			name:    "invalid request",
			args:    args{req: &dto.RequestSyncOpenPayment{UserID: 0}},
			wantErr: domain.ErrInvalidPaymentRequest,
		},
		{
			name: "open payment error",
			args: args{req: &dto.RequestSyncOpenPayment{UserID: 77}},
			prepare: func(f *fields) {
				f.repo.EXPECT().PaymentGetOpenPendingByUser(ctx, int64(77)).Return(nil, domain.ErrPaymentNotFound)
			},
			wantErr: domain.ErrPaymentNotFound,
		},
		{
			name: "find payment error",
			args: args{req: &dto.RequestSyncOpenPayment{UserID: 77}},
			prepare: func(f *fields) {
				f.repo.EXPECT().PaymentGetOpenPendingByUser(ctx, int64(77)).Return(testPayment(domain.PaymentStatusPending), nil)
				f.yookassa.EXPECT().FindPayment(ctx, "yoo-10").Return(nil, errors.New("network down"))
			},
			wantText: "yookassa find payment",
		},
		{
			name: "update error",
			args: args{req: &dto.RequestSyncOpenPayment{UserID: 77}},
			prepare: func(f *fields) {
				f.repo.EXPECT().PaymentGetOpenPendingByUser(ctx, int64(77)).Return(testPayment(domain.PaymentStatusPending), nil)
				f.yookassa.EXPECT().FindPayment(ctx, "yoo-10").Return(testYooPayment("yoo-10", yoopayment.Succeeded), nil)
				f.repo.EXPECT().PaymentUpdate(ctx, gomock.Any()).Return(nil, errors.New("db down"))
			},
			wantText: "payment sync",
		},
		{
			name: "activation error",
			args: args{req: &dto.RequestSyncOpenPayment{UserID: 77}},
			prepare: func(f *fields) {
				f.repo.EXPECT().PaymentGetOpenPendingByUser(ctx, int64(77)).Return(testPayment(domain.PaymentStatusPending), nil)
				f.yookassa.EXPECT().FindPayment(ctx, "yoo-10").Return(testYooPayment("yoo-10", yoopayment.Succeeded), nil)
				f.repo.EXPECT().PaymentUpdate(ctx, gomock.Any()).Return(testPayment(domain.PaymentStatusSucceeded), nil)
				f.subscription.EXPECT().Activate(ctx, int64(77), int64(30)).Return(errors.New("subscription down"))
			},
			wantText: "subscription activate",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				repo:         mock.NewMockPaymentRepository(ctrl),
				subscription: mock.NewMockSubscriptionService(ctrl),
				yookassa:     mock.NewMockYooKassaClient(ctrl),
			}
			if tt.prepare != nil {
				tt.prepare(&f)
			}
			u := NewPaymentUseCase(f.repo, f.subscription, f.yookassa, "https://app.local/payment/return")

			got, err := u.SyncOpenPayment(ctx, tt.args.req)
			require.Nil(t, got)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				require.ErrorContains(t, err, tt.wantText)
			}
		})
	}
}

func TestPaymentUseCase_HandleYooKassaWebhook(t *testing.T) {
	ctx := context.Background()

	type fields struct {
		repo         *mock.MockPaymentRepository
		subscription *mock.MockSubscriptionService
		yookassa     *mock.MockYooKassaClient
	}
	type args struct {
		raw []byte
	}

	tests := []struct {
		prepare  func(f *fields)
		wantErr  error
		name     string
		args     args
		wantText string
	}{
		{
			name: "empty body is ignored",
			args: args{raw: nil},
		},
		{
			name: "invalid json is ignored",
			args: args{raw: []byte(`{bad`)},
		},
		{
			name: "non payment event is ignored",
			args: args{raw: []byte(`{"type":"notification","event":"refund.succeeded","object":{"id":"yoo-10"}}`)},
		},
		{
			name: "unknown payment is ignored",
			args: args{raw: []byte(`{"type":"notification","event":"payment.succeeded","object":{"id":"yoo-10"}}`)},
			prepare: func(f *fields) {
				f.repo.EXPECT().PaymentGetByPaymentID(ctx, "yoo-10").Return(nil, domain.ErrPaymentNotFound)
			},
		},
		{
			name: "repo error is returned",
			args: args{raw: []byte(`{"type":"notification","event":"payment.succeeded","object":{"id":"yoo-10"}}`)},
			prepare: func(f *fields) {
				f.repo.EXPECT().PaymentGetByPaymentID(ctx, "yoo-10").Return(nil, errors.New("db down"))
			},
			wantText: "db down",
		},
		{
			name: "find error is returned",
			args: args{raw: []byte(`{"type":"notification","event":"payment.succeeded","object":{"id":"yoo-10"}}`)},
			prepare: func(f *fields) {
				f.repo.EXPECT().PaymentGetByPaymentID(ctx, "yoo-10").Return(testPayment(domain.PaymentStatusPending), nil)
				f.yookassa.EXPECT().FindPayment(ctx, "yoo-10").Return(nil, errors.New("network down"))
			},
			wantText: "network down",
		},
		{
			name: "succeeded payment activates subscription",
			args: args{raw: []byte(`{"type":"notification","event":"payment.succeeded","object":{"id":"yoo-10"}}`)},
			prepare: func(f *fields) {
				f.repo.EXPECT().PaymentGetByPaymentID(ctx, "yoo-10").Return(testPayment(domain.PaymentStatusPending), nil)
				f.yookassa.EXPECT().FindPayment(ctx, "yoo-10").Return(testYooPayment("yoo-10", yoopayment.Succeeded), nil)
				f.repo.EXPECT().PaymentUpdate(ctx, gomock.Any()).Return(testPayment(domain.PaymentStatusSucceeded), nil)
				f.subscription.EXPECT().Activate(ctx, int64(77), int64(30)).Return(nil)
			},
		},
		{
			name: "oversized body is ignored",
			args: args{raw: make([]byte, maxYooKassaWebhookBody+1)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				repo:         mock.NewMockPaymentRepository(ctrl),
				subscription: mock.NewMockSubscriptionService(ctrl),
				yookassa:     mock.NewMockYooKassaClient(ctrl),
			}
			if tt.prepare != nil {
				tt.prepare(&f)
			}
			u := NewPaymentUseCase(f.repo, f.subscription, f.yookassa, "https://app.local/payment/return")

			err := u.HandleYooKassaWebhook(ctx, tt.args.raw)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else if tt.wantText != "" {
				require.ErrorContains(t, err, tt.wantText)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestPaymentUseCase_Helpers(t *testing.T) {
	t.Run("validate create", func(t *testing.T) {
		tests := []struct {
			req     *dto.RequestCreatePayment
			name    string
			wantErr bool
		}{
			{name: "valid", req: &dto.RequestCreatePayment{UserID: 1, Amount: 100, SubscriptionDays: 30}},
			{name: "nil", req: nil, wantErr: true},
			{name: "invalid user", req: &dto.RequestCreatePayment{UserID: 0, Amount: 100, SubscriptionDays: 30}, wantErr: true},
			{name: "invalid amount", req: &dto.RequestCreatePayment{UserID: 1, Amount: 0, SubscriptionDays: 30}, wantErr: true},
			{name: "invalid days", req: &dto.RequestCreatePayment{UserID: 1, Amount: 100, SubscriptionDays: 0}, wantErr: true},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := validateCreate(tt.req)
				if tt.wantErr {
					require.ErrorIs(t, err, domain.ErrInvalidPaymentRequest)
				} else {
					require.NoError(t, err)
				}
			})
		}
	})

	t.Run("yookassa status mapping", func(t *testing.T) {
		tests := []struct {
			in   yoopayment.Status
			want domain.PaymentStatus
			name string
		}{
			{name: "pending", in: yoopayment.Pending, want: domain.PaymentStatusPending},
			{name: "waiting", in: yoopayment.WaitingForCapture, want: domain.PaymentStatusWaitingForCapture},
			{name: "succeeded", in: yoopayment.Succeeded, want: domain.PaymentStatusSucceeded},
			{name: "canceled", in: yoopayment.Canceled, want: domain.PaymentStatusCanceled},
			{name: "custom", in: yoopayment.Status("custom"), want: domain.PaymentStatus("custom")},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				require.Equal(t, tt.want, mapYooKassaStatus(tt.in))
			})
		}
	})

	t.Run("confirmation url", func(t *testing.T) {
		tests := []struct {
			p    *yoopayment.Payment
			name string
			want string
		}{
			{name: "nil payment", p: nil, want: ""},
			{name: "nil confirmation", p: &yoopayment.Payment{}, want: ""},
			{name: "unsupported confirmation", p: &yoopayment.Payment{Confirmation: yoopayment.Redirect{}}, want: ""},
			{name: "map confirmation", p: &yoopayment.Payment{Confirmation: map[string]interface{}{"confirmation_url": "https://pay.local"}}, want: "https://pay.local"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				require.Equal(t, tt.want, yooKassaConfirmationURL(tt.p))
			})
		}
	})

	t.Run("response from nil domain", func(t *testing.T) {
		require.Nil(t, responseFromDomain(nil))
	})
}
