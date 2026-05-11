package subscription

import (
	"net/http"
	"time"

	subscriptionv1 "github.com/go-park-mail-ru/2026_1_ASAP/gen/go/subscription/v1"
	dtoApi "github.com/go-park-mail-ru/2026_1_ASAP/internal/gateway/dto/api"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/gateway/middleware"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/subscription/dto"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/utils/response"
	"github.com/go-park-mail-ru/2026_1_ASAP/pkg/grpcerr"
)

type SubscriptionHandler struct {
	SubscriptionClient subscriptionv1.SubscriptionClient
}

func NewSubscriptionHandler(subscriptionClient subscriptionv1.SubscriptionClient) *SubscriptionHandler {
	return &SubscriptionHandler{
		SubscriptionClient: subscriptionClient,
	}
}

func (h SubscriptionHandler) GetSubscription(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	uid, ok := ctx.Value(middleware.UserID).(int64)
	if !ok {
		response.Send(w, http.StatusUnauthorized, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{Code: dtoApi.Unauthorized, Message: dtoApi.UnauthorizedMsg}},
		})
		return
	}

	resp, err := h.SubscriptionClient.GetSubscription(ctx, &subscriptionv1.RequestGetSubscription{
		UserId: uid,
	})
	if err != nil {
		sendSubcriptionError(w, err)
		return
	}

	startAt := time.Time{}
	if ts := resp.GetStartAt(); ts != nil {
		startAt = ts.AsTime()
	}
	endAt := time.Time{}
	if ts := resp.GetEndAt(); ts != nil {
		endAt = ts.AsTime()
	}

	response.Send(w, http.StatusOK, dtoApi.ApiSuccessResponse[*dto.ResponseGetSubscription]{
		Status: dtoApi.Success,
		Body: &dto.ResponseGetSubscription{
			UserID:  resp.UserId,
			Active:  resp.Active,
			StartAt: startAt,
			EndAt:   endAt,
		},
	})
}

func (h SubscriptionHandler) CancelSubscription(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	uid, ok := ctx.Value(middleware.UserID).(int64)
	if !ok {
		response.Send(w, http.StatusUnauthorized, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{Code: dtoApi.Unauthorized, Message: dtoApi.UnauthorizedMsg}},
		})
		return
	}

	if _, err := h.SubscriptionClient.CancelSubscription(ctx, &subscriptionv1.RequestCancelSubscription{
		UserId: uid,
	}); err != nil {
		sendSubcriptionError(w, err)
		return
	}

	response.Send(w, http.StatusOK, dtoApi.ApiSuccessResponse[*dto.ResponseCancelSubscription]{
		Status: dtoApi.Success,
		Body: &dto.ResponseCancelSubscription{
			UserID: uid,
		},
	})
}

func sendSubcriptionError(w http.ResponseWriter, err error) {
	_, appCode, _ := grpcerr.Error(err)
	switch subscriptionv1.SubscriptionErrorCode(appCode) {
	case subscriptionv1.SubscriptionErrorCode_SUBSCRIPTION_ERROR_NOT_FOUND:
		response.Send(w, http.StatusNotFound, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{Code: dtoApi.SubscriptionNotFound, Message: dtoApi.SubscriptionNotFoundMsg}},
		})
	case subscriptionv1.SubscriptionErrorCode_SUBSCRIPTION_ERROR_EXPIRED:
		response.Send(w, http.StatusBadRequest, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{Code: dtoApi.SubscriptionExpired, Message: dtoApi.SubscriptionExpiredMsg}},
		})
	default:
		response.Send(w, http.StatusInternalServerError, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{Code: dtoApi.InternalError, Message: dtoApi.InternalErrorMsg}},
		})
	}
}
