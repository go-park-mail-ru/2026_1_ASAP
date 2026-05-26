package payment

import (
	"io"
	"net/http"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	paymentv1 "github.com/go-park-mail-ru/2026_1_ASAP/gen/go/payment/v1"
	dtoApi "github.com/go-park-mail-ru/2026_1_ASAP/internal/gateway/dto/api"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/gateway/jsonbody"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/gateway/middleware"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/utils/response"
)

const maxYooKassaWebhookBody = 256 << 10

type GatewayPaymentHandler struct {
	PaymentClient paymentv1.PaymentClient
}

func NewGatewayPaymentHandler(client paymentv1.PaymentClient) *GatewayPaymentHandler {
	return &GatewayPaymentHandler{PaymentClient: client}
}

func paymentDetailsHTTPBody(p *paymentv1.PaymentDetails) map[string]any {
	if p == nil {
		return nil
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
	return body
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
	if err := jsonbody.Decode(r.Body, &req); err != nil {
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
	body := paymentDetailsHTTPBody(p)
	if body == nil {
		response.Send(w, http.StatusInternalServerError, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{Code: dtoApi.InternalError, Message: dtoApi.InternalErrorMsg}},
		})
		return
	}

	response.Send(w, http.StatusOK, dtoApi.ApiSuccessResponse[map[string]any]{
		Status: dtoApi.Success,
		Body:   body,
	})
}

func (h *GatewayPaymentHandler) SyncOpenPayment(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := ctx.Value(middleware.UserID).(int64)
	if !ok || userID <= 0 {
		response.Send(w, http.StatusUnauthorized, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{Code: dtoApi.Unauthorized, Message: dtoApi.UnauthorizedMsg}},
		})
		return
	}

	resp, err := h.PaymentClient.SyncOpenPayment(ctx, &paymentv1.RequestSyncOpenPayment{UserId: userID})
	if err != nil {
		if st, ok := status.FromError(err); ok && st.Code() == codes.NotFound {
			response.Send(w, http.StatusNotFound, dtoApi.ApiErrorResponse{
				Status: dtoApi.Error,
				Errors: []dtoApi.ApiError{{Code: dtoApi.NotFound, Message: dtoApi.NotFoundMsg}},
			})
			return
		}
		response.Send(w, http.StatusInternalServerError, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{Code: dtoApi.InternalError, Message: dtoApi.InternalErrorMsg}},
		})
		return
	}

	body := paymentDetailsHTTPBody(resp.GetPayment())
	if body == nil {
		response.Send(w, http.StatusInternalServerError, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{Code: dtoApi.InternalError, Message: dtoApi.InternalErrorMsg}},
		})
		return
	}

	response.Send(w, http.StatusOK, dtoApi.ApiSuccessResponse[map[string]any]{
		Status: dtoApi.Success,
		Body:   body,
	})
}

func (h *GatewayPaymentHandler) YooKassaWebhook(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	body, err := io.ReadAll(io.LimitReader(r.Body, maxYooKassaWebhookBody+1))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if len(body) > maxYooKassaWebhookBody {
		w.WriteHeader(http.StatusRequestEntityTooLarge)
		return
	}

	_, err = h.PaymentClient.ProcessYooKassaWebhook(r.Context(), &paymentv1.ProcessYooKassaWebhookRequest{RawBody: body})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
