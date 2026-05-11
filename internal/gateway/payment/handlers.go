package payment

import (
	"encoding/json"
	"net/http"

	paymentv1 "github.com/go-park-mail-ru/2026_1_ASAP/gen/go/payment/v1"
	dtoApi "github.com/go-park-mail-ru/2026_1_ASAP/internal/gateway/dto/api"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/gateway/middleware"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/utils/response"
)

type GatewayPaymentHandler struct {
	PaymentClient paymentv1.PaymentClient
}

func NewGatewayPaymentHandler(client paymentv1.PaymentClient) *GatewayPaymentHandler {
	return &GatewayPaymentHandler{PaymentClient: client}
}

type CreatePaymentRequest struct {
	Amount           int32  `json:"amount"`
	SubscriptionDays int32  `json:"subscription_days"`
	Description      string `json:"description,omitempty"`
}

func (h *GatewayPaymentHandler) CreatePayment(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	ctx := r.Context()
	userID, ok := ctx.Value(middleware.UserID).(int64)
	if !ok || userID <= 0 {
		response.Send(w, http.StatusUnauthorized, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{Code: dtoApi.Unauthorized, Message: dtoApi.UnauthorizedMsg}},
		})
		return
	}

	var req CreatePaymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Send(w, http.StatusBadRequest, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{Code: dtoApi.InvalidJson, Message: dtoApi.InvalidJsonMsg}},
		})
		return
	}

	if req.Amount <= 0 || req.SubscriptionDays <= 0 {
		response.Send(w, http.StatusBadRequest, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{
				Code:    dtoApi.InvalidJson,
				Message: dtoApi.InvalidJsonMsg,
			}},
		})
		return
	}

	resp, err := h.PaymentClient.CreatePayment(ctx, &paymentv1.RequestCreatePayment{
		UserId:           userID,
		PaymentId:        "",
		Status:           "",
		Amount:           req.Amount,
		SubscriptionDays: req.SubscriptionDays,
	})
	if err != nil {
		response.Send(w, http.StatusInternalServerError, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{Code: dtoApi.InternalError, Message: dtoApi.InternalErrorMsg}},
		})
		return
	}

	p := resp.GetPayment()
	if p == nil {
		response.Send(w, http.StatusInternalServerError, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{Code: dtoApi.InternalError, Message: dtoApi.InternalErrorMsg}},
		})
		return
	}

	body := map[string]any{
		"id":                p.GetId(),
		"payment_id":        p.GetPaymentId(),
		"user_id":           p.GetUserId(),
		"status":            p.GetStatus(),
		"amount":            p.GetAmount(),
		"subscription_days": p.GetSubscriptionDays(),
	}
	if u := p.GetPaymentUrl(); u != "" {
		body["payment_url"] = u
	}
	if m := p.GetMessage(); m != "" {
		body["message"] = m
	}

	response.Send(w, http.StatusOK, dtoApi.ApiSuccessResponse[map[string]any]{
		Status: dtoApi.Success,
		Body:   body,
	})
}
