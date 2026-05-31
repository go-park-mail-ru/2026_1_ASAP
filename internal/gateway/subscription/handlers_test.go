//go:generate mockgen -destination=mock/subscription_mock.go -package=mock github.com/go-park-mail-ru/2026_1_ASAP/gen/go/subscription/v1 SubscriptionClient
package subscription

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	subscriptionv1 "github.com/go-park-mail-ru/2026_1_ASAP/gen/go/subscription/v1"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/gateway/middleware"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/gateway/subscription/mock"
	"github.com/go-park-mail-ru/2026_1_ASAP/pkg/grpcerr"
)

func subscriptionRequestWithUser(method, target string, userID any) *http.Request {
	req := httptest.NewRequest(method, target, nil)
	if userID != nil {
		req = req.WithContext(context.WithValue(req.Context(), middleware.UserID, userID))
	}
	return req
}

func TestPositiveSubscriptionHandler_GetSubscription(t *testing.T) {
	type fields struct {
		client *mock.MockSubscriptionClient
	}

	now := time.Date(2026, 5, 27, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		prepare func(*fields)
		name    string
		userID  int64
		want    int
	}{
		{
			name:   "gets subscription with timestamps",
			userID: 77,
			prepare: func(f *fields) {
				f.client.EXPECT().
					GetSubscription(gomock.Any(), &subscriptionv1.RequestGetSubscription{UserId: 77}).
					Return(&subscriptionv1.ResponseGetSubscription{
						UserId:  77,
						Active:  true,
						StartAt: timestamppb.New(now),
						EndAt:   timestamppb.New(now.Add(30 * 24 * time.Hour)),
					}, nil)
			},
			want: http.StatusOK,
		},
		{
			name:   "gets subscription without timestamps",
			userID: 77,
			prepare: func(f *fields) {
				f.client.EXPECT().
					GetSubscription(gomock.Any(), &subscriptionv1.RequestGetSubscription{UserId: 77}).
					Return(&subscriptionv1.ResponseGetSubscription{UserId: 77}, nil)
			},
			want: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{client: mock.NewMockSubscriptionClient(ctrl)}
			tt.prepare(&f)
			handler := NewSubscriptionHandler(f.client)

			w := httptest.NewRecorder()
			handler.GetSubscription(w, subscriptionRequestWithUser(http.MethodGet, "/subscription", tt.userID))
			require.Equal(t, tt.want, w.Code)
		})
	}
}

func TestNegativeSubscriptionHandler_GetSubscription(t *testing.T) {
	type fields struct {
		client *mock.MockSubscriptionClient
	}

	tests := []struct {
		prepare func(*fields)
		name    string
		userID  any
		want    int
	}{
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
					GetSubscription(gomock.Any(), gomock.Any()).
					Return(nil, grpcerr.New(codes.NotFound, int32(subscriptionv1.SubscriptionErrorCode_SUBSCRIPTION_ERROR_NOT_FOUND), "not found"))
			},
			want: http.StatusNotFound,
		},
		{
			name:   "expired",
			userID: int64(77),
			prepare: func(f *fields) {
				f.client.EXPECT().
					GetSubscription(gomock.Any(), gomock.Any()).
					Return(nil, grpcerr.New(codes.FailedPrecondition, int32(subscriptionv1.SubscriptionErrorCode_SUBSCRIPTION_ERROR_EXPIRED), "expired"))
			},
			want: http.StatusBadRequest,
		},
		{
			name:   "internal error",
			userID: int64(77),
			prepare: func(f *fields) {
				f.client.EXPECT().
					GetSubscription(gomock.Any(), gomock.Any()).
					Return(nil, grpcerr.New(codes.Internal, 999, "failed"))
			},
			want: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{client: mock.NewMockSubscriptionClient(ctrl)}
			if tt.prepare != nil {
				tt.prepare(&f)
			}
			handler := NewSubscriptionHandler(f.client)

			w := httptest.NewRecorder()
			handler.GetSubscription(w, subscriptionRequestWithUser(http.MethodGet, "/subscription", tt.userID))
			require.Equal(t, tt.want, w.Code)
		})
	}
}

func TestSubscriptionHandler_CancelSubscription(t *testing.T) {
	type fields struct {
		client *mock.MockSubscriptionClient
	}

	tests := []struct {
		prepare func(*fields)
		name    string
		userID  any
		want    int
	}{
		{
			name:   "cancels subscription",
			userID: int64(77),
			prepare: func(f *fields) {
				f.client.EXPECT().
					CancelSubscription(gomock.Any(), &subscriptionv1.RequestCancelSubscription{UserId: 77}).
					Return(&emptypb.Empty{}, nil)
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
					CancelSubscription(gomock.Any(), gomock.Any()).
					Return(nil, grpcerr.New(codes.NotFound, int32(subscriptionv1.SubscriptionErrorCode_SUBSCRIPTION_ERROR_NOT_FOUND), "not found"))
			},
			want: http.StatusNotFound,
		},
		{
			name:   "internal error",
			userID: int64(77),
			prepare: func(f *fields) {
				f.client.EXPECT().
					CancelSubscription(gomock.Any(), gomock.Any()).
					Return(nil, grpcerr.New(codes.Internal, 999, "failed"))
			},
			want: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{client: mock.NewMockSubscriptionClient(ctrl)}
			if tt.prepare != nil {
				tt.prepare(&f)
			}
			handler := NewSubscriptionHandler(f.client)

			w := httptest.NewRecorder()
			handler.CancelSubscription(w, subscriptionRequestWithUser(http.MethodDelete, "/subscription", tt.userID))
			require.Equal(t, tt.want, w.Code)
		})
	}
}
