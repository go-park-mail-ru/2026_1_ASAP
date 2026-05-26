//go:generate mockgen -destination=mock/payment_mock.go -package=mock github.com/go-park-mail-ru/2026_1_ASAP/gen/go/payment/v1 PaymentClient
package payment

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	paymentv1 "github.com/go-park-mail-ru/2026_1_ASAP/gen/go/payment/v1"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/gateway/middleware"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/gateway/payment/mock"
)

func paymentURLPtr(s string) *string {
	return &s
}

func gatewayPaymentDetails() *paymentv1.PaymentDetails {
	return &paymentv1.PaymentDetails{
		Id:               10,
		PaymentId:        "yoo-10",
		UserId:           77,
		Status:           "pending",
		Amount:           19900,
		SubscriptionDays: 30,
		PaymentUrl:       paymentURLPtr("https://pay.local/10"),
		Message:          paymentURLPtr("created"),
	}
}

func requestWithUser(method, target, body string, userID any) *http.Request {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	if userID != nil {
		req = req.WithContext(context.WithValue(req.Context(), middleware.UserID, userID))
	}
	return req
}

func TestPaymentDetailsHTTPBody(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		require.Nil(t, paymentDetailsHTTPBody(nil))
	})

	t.Run("with optional fields", func(t *testing.T) {
		body := paymentDetailsHTTPBody(gatewayPaymentDetails())
		require.Equal(t, int64(10), body["id"])
		require.Equal(t, "yoo-10", body["payment_id"])
		require.Equal(t, "https://pay.local/10", body["payment_url"])
		require.Equal(t, "created", body["message"])
	})

	t.Run("without optional fields", func(t *testing.T) {
		details := gatewayPaymentDetails()
		details.PaymentUrl = nil
		details.Message = nil
		body := paymentDetailsHTTPBody(details)
		require.NotContains(t, body, "payment_url")
		require.NotContains(t, body, "message")
	})
}

func TestPositiveGatewayPaymentHandler_CreatePayment(t *testing.T) {
	type fields struct {
		client *mock.MockPaymentClient
	}

	tests := []struct {
		prepare func(*fields)
		body    string
		name    string
		userID  int64
		want    int
	}{
		{
			name:   "creates payment",
			userID: 77,
			body:   `{"amount":19900,"subscription_days":30}`,
			prepare: func(f *fields) {
				f.client.EXPECT().
					CreatePayment(gomock.Any(), &paymentv1.RequestCreatePayment{
						UserId:           77,
						PaymentId:        "",
						Status:           "",
						Amount:           19900,
						SubscriptionDays: 30,
					}).
					Return(&paymentv1.ResponseCreatePayment{Payment: gatewayPaymentDetails()}, nil)
			},
			want: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{client: mock.NewMockPaymentClient(ctrl)}
			tt.prepare(&f)
			handler := NewGatewayPaymentHandler(f.client)

			w := httptest.NewRecorder()
			handler.CreatePayment(w, requestWithUser(http.MethodPost, "/payments", tt.body, tt.userID))
			require.Equal(t, tt.want, w.Code)
		})
	}
}

func TestNegativeGatewayPaymentHandler_CreatePayment(t *testing.T) {
	type fields struct {
		client *mock.MockPaymentClient
	}

	tests := []struct {
		prepare func(*fields)
		body    string
		name    string
		userID  any
		want    int
	}{
		{
			name:   "missing user",
			userID: nil,
			body:   `{"amount":19900,"subscription_days":30}`,
			want:   http.StatusUnauthorized,
		},
		{
			name:   "invalid json",
			userID: int64(77),
			body:   `{bad`,
			want:   http.StatusBadRequest,
		},
		{
			name:   "invalid amount",
			userID: int64(77),
			body:   `{"amount":0,"subscription_days":30}`,
			want:   http.StatusBadRequest,
		},
		{
			name:   "grpc error",
			userID: int64(77),
			body:   `{"amount":19900,"subscription_days":30}`,
			prepare: func(f *fields) {
				f.client.EXPECT().CreatePayment(gomock.Any(), gomock.Any()).Return(nil, status.Error(codes.Internal, "failed"))
			},
			want: http.StatusInternalServerError,
		},
		{
			name:   "nil payment in response",
			userID: int64(77),
			body:   `{"amount":19900,"subscription_days":30}`,
			prepare: func(f *fields) {
				f.client.EXPECT().CreatePayment(gomock.Any(), gomock.Any()).Return(&paymentv1.ResponseCreatePayment{}, nil)
			},
			want: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{client: mock.NewMockPaymentClient(ctrl)}
			if tt.prepare != nil {
				tt.prepare(&f)
			}
			handler := NewGatewayPaymentHandler(f.client)

			w := httptest.NewRecorder()
			handler.CreatePayment(w, requestWithUser(http.MethodPost, "/payments", tt.body, tt.userID))
			require.Equal(t, tt.want, w.Code)
		})
	}
}

func TestGatewayPaymentHandler_SyncOpenPayment(t *testing.T) {
	type fields struct {
		client *mock.MockPaymentClient
	}

	tests := []struct {
		prepare func(*fields)
		name    string
		userID  any
		want    int
	}{
		{
			name:   "syncs payment",
			userID: int64(77),
			prepare: func(f *fields) {
				f.client.EXPECT().
					SyncOpenPayment(gomock.Any(), &paymentv1.RequestSyncOpenPayment{UserId: 77}).
					Return(&paymentv1.ResponseGetPayment{Payment: gatewayPaymentDetails()}, nil)
			},
			want: http.StatusOK,
		},
		{
			name:   "missing user",
			userID: nil,
			want:   http.StatusUnauthorized,
		},
		{
			name:   "not found",
			userID: int64(77),
			prepare: func(f *fields) {
				f.client.EXPECT().
					SyncOpenPayment(gomock.Any(), gomock.Any()).
					Return(nil, status.Error(codes.NotFound, "missing"))
			},
			want: http.StatusNotFound,
		},
		{
			name:   "grpc error",
			userID: int64(77),
			prepare: func(f *fields) {
				f.client.EXPECT().
					SyncOpenPayment(gomock.Any(), gomock.Any()).
					Return(nil, status.Error(codes.Internal, "failed"))
			},
			want: http.StatusInternalServerError,
		},
		{
			name:   "nil payment in response",
			userID: int64(77),
			prepare: func(f *fields) {
				f.client.EXPECT().
					SyncOpenPayment(gomock.Any(), gomock.Any()).
					Return(&paymentv1.ResponseGetPayment{}, nil)
			},
			want: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{client: mock.NewMockPaymentClient(ctrl)}
			if tt.prepare != nil {
				tt.prepare(&f)
			}
			handler := NewGatewayPaymentHandler(f.client)

			w := httptest.NewRecorder()
			handler.SyncOpenPayment(w, requestWithUser(http.MethodPost, "/payments/sync", "", tt.userID))
			require.Equal(t, tt.want, w.Code)
		})
	}
}

func TestGatewayPaymentHandler_YooKassaWebhook(t *testing.T) {
	type fields struct {
		client *mock.MockPaymentClient
	}

	tests := []struct {
		prepare func(*fields)
		body    []byte
		name    string
		want    int
	}{
		{
			name: "processes webhook",
			body: []byte(`{"event":"payment.succeeded"}`),
			prepare: func(f *fields) {
				f.client.EXPECT().
					ProcessYooKassaWebhook(gomock.Any(), &paymentv1.ProcessYooKassaWebhookRequest{RawBody: []byte(`{"event":"payment.succeeded"}`)}).
					Return(&emptypb.Empty{}, nil)
			},
			want: http.StatusOK,
		},
		{
			name: "too large",
			body: bytes.Repeat([]byte("a"), maxYooKassaWebhookBody+1),
			want: http.StatusRequestEntityTooLarge,
		},
		{
			name: "grpc error",
			body: []byte(`{"event":"payment.succeeded"}`),
			prepare: func(f *fields) {
				f.client.EXPECT().ProcessYooKassaWebhook(gomock.Any(), gomock.Any()).Return(nil, status.Error(codes.Internal, "failed"))
			},
			want: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{client: mock.NewMockPaymentClient(ctrl)}
			if tt.prepare != nil {
				tt.prepare(&f)
			}
			handler := NewGatewayPaymentHandler(f.client)

			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(tt.body))
			handler.YooKassaWebhook(w, req)
			require.Equal(t, tt.want, w.Code)
		})
	}
}
