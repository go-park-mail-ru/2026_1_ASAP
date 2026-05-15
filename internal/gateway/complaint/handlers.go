package complaint

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	complaintv1 "github.com/go-park-mail-ru/2026_1_ASAP/gen/go/complaint/v1"
	dtoApi "github.com/go-park-mail-ru/2026_1_ASAP/internal/gateway/dto/api"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/gateway/middleware"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/utils/response"
	"github.com/go-park-mail-ru/2026_1_ASAP/pkg/grpcerr"
)

type GatewayComplaintHandler struct {
	ComplaintService complaintv1.ComplaintClient
}

func NewGatewayComplaintHandler(complaintService complaintv1.ComplaintClient) *GatewayComplaintHandler {
	return &GatewayComplaintHandler{ComplaintService: complaintService}
}

type GatewayAnalyticHandler struct {
	ComplaintService complaintv1.ComplaintClient
}

func NewGatewayAnalyticHandler(complaintService complaintv1.ComplaintClient) *GatewayAnalyticHandler {
	return &GatewayAnalyticHandler{ComplaintService: complaintService}
}

type createComplaintRequest struct {
	Type     string `json:"type"`
	Feedback struct {
		FeedbackName  string `json:"feedback_name"`
		FeedbackEmail string `json:"feedback_email"`
	} `json:"feedback"`
	Body string `json:"body"`
}

type updateStatusRequest struct {
	ComplaintID int64  `json:"complaint_id"`
	Status      string `json:"status"`
}

type complaintDTO struct {
	ID            int64    `json:"id"`
	Type          string   `json:"type"`
	Status        string   `json:"status"`
	Feedback      feedback `json:"feedback"`
	Body          string   `json:"body"`
	UserID        int64    `json:"user_id,omitempty"`
	AttachmentURL *string  `json:"attachment_url,omitempty"`
	CreatedAt     anyTime  `json:"created_at"`
	UpdatedAt     anyTime  `json:"updated_at"`
}

type feedback struct {
	FeedbackName  string `json:"feedback_name"`
	FeedbackEmail string `json:"feedback_email"`
}

type complaintAnalyticDTO struct {
	CountStatus countStatusDTO `json:"count_status"`
	CountType   countTypeDTO   `json:"count_type"`
}

type countStatusDTO struct {
	CountStatusOpened int64 `json:"count_status_opened"`
	CountStatusInWork int64 `json:"count_status_in_work"`
	CountStatusClosed int64 `json:"count_status_closed"`
}

type countTypeDTO struct {
	CountBug     int64 `json:"count_type_bug"`
	CountUpgrade int64 `json:"count_type_upgrade"`
	CountProduct int64 `json:"count_type_product"`
}

type anyTime = string

func (h *GatewayComplaintHandler) CreateComplaintUnAuthorized(w http.ResponseWriter, r *http.Request) {
	h.createComplaint(w, r, nil)
}

func (h *GatewayComplaintHandler) CreateComplaintAuthorized(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	uid, ok := ctx.Value(middleware.UserID).(int64)
	if !ok {
		sendUnauthorized(w)
		return
	}

	h.createComplaint(w, r, &uid)
}

func (h *GatewayComplaintHandler) createComplaint(w http.ResponseWriter, r *http.Request, userID *int64) {
	defer r.Body.Close()
	ctx := r.Context()

	contentType := r.Header.Get("Content-Type")
	req := &complaintv1.RequestCreateComplaint{}
	if userID != nil {
		req.UserId = userID
	}
	if strings.HasPrefix(contentType, "multipart/form-data") {
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			sendInvalidJSON(w)
			return
		}
		payloadRaw := strings.TrimSpace(r.FormValue("payload"))
		if payloadRaw == "" {
			sendInvalidJSON(w)
			return
		}
		var payload createComplaintRequest
		if err := json.Unmarshal([]byte(payloadRaw), &payload); err != nil {
			sendInvalidJSON(w)
			return
		}
		req.Type = mapComplaintTypeFromString(payload.Type)
		req.Body = payload.Body
		req.Feedback = &complaintv1.Feedback{
			FeedbackName:  payload.Feedback.FeedbackName,
			FeedbackEmail: payload.Feedback.FeedbackEmail,
		}
		file, header, err := r.FormFile("attachment")
		if err == nil {
			defer file.Close()
			content, readErr := io.ReadAll(file)
			if readErr != nil {
				sendInternal(w)
				return
			}
			req.File = &complaintv1.ComplaintFile{
				Content: content,
				Type:    header.Header.Get("Content-Type"),
			}
		}
	} else {
		var body createComplaintRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			sendInvalidJSON(w)
			return
		}
		req.Type = mapComplaintTypeFromString(body.Type)
		req.Body = body.Body
		req.Feedback = &complaintv1.Feedback{
			FeedbackName:  body.Feedback.FeedbackName,
			FeedbackEmail: body.Feedback.FeedbackEmail,
		}
	}

	resp, err := h.ComplaintService.CreateComplaint(ctx, req)
	if err != nil {
		sendComplaintError(w, err)
		return
	}

	response.Send(w, http.StatusOK, dtoApi.ApiSuccessResponse[*complaintDTO]{
		Status: dtoApi.Success,
		Body:   mapComplaint(resp.GetComplaint()),
	})
}

func (h *GatewayComplaintHandler) GetComplaint(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if _, ok := ctx.Value(middleware.UserID).(int64); !ok {
		sendUnauthorized(w)
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		sendInvalidID(w)
		return
	}
	resp, err := h.ComplaintService.GetComplaint(ctx, &complaintv1.RequestGetComplaint{ComplaintId: id})
	if err != nil {
		sendComplaintError(w, err)
		return
	}

	response.Send(w, http.StatusOK, dtoApi.ApiSuccessResponse[*complaintDTO]{
		Status: dtoApi.Success,
		Body:   mapComplaint(resp.GetComplaint()),
	})
}

func (h *GatewayComplaintHandler) GetMyComplaints(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	uid, ok := ctx.Value(middleware.UserID).(int64)
	if !ok {
		sendUnauthorized(w)
		return
	}

	resp, err := h.ComplaintService.GetComplaintsByUser(ctx, &complaintv1.RequestGetComplaintsByUser{UserId: uid})
	if err != nil {
		sendComplaintError(w, err)
		return
	}

	response.Send(w, http.StatusOK, dtoApi.ApiSuccessResponse[[]*complaintDTO]{
		Status: dtoApi.Success,
		Body:   mapComplaints(resp.GetComplaints()),
	})
}

func (h *GatewayComplaintHandler) GetAllComplaints(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if _, ok := ctx.Value(middleware.UserID).(int64); !ok {
		sendUnauthorized(w)
		return
	}

	resp, err := h.ComplaintService.GetAllComplaints(ctx, &complaintv1.RequestGetAllComplaints{})
	if err != nil {
		sendComplaintError(w, err)
		return
	}

	response.Send(w, http.StatusOK, dtoApi.ApiSuccessResponse[[]*complaintDTO]{
		Status: dtoApi.Success,
		Body:   mapComplaints(resp.GetComplaints()),
	})
}

func (h *GatewayComplaintHandler) UpdateComplaintStatus(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	ctx := r.Context()
	if _, ok := ctx.Value(middleware.UserID).(int64); !ok {
		sendUnauthorized(w)
		return
	}

	var body updateStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		sendInvalidJSON(w)
		return
	}
	if body.ComplaintID <= 0 {
		sendInvalidID(w)
		return
	}

	resp, err := h.ComplaintService.UpdateComplaintStatus(ctx, &complaintv1.RequestUpdateComplaintStatus{
		ComplaintId: body.ComplaintID,
		Status:      mapComplaintStatusFromString(body.Status),
	})
	if err != nil {
		sendComplaintError(w, err)
		return
	}

	response.Send(w, http.StatusOK, dtoApi.ApiSuccessResponse[*complaintDTO]{
		Status: dtoApi.Success,
		Body:   mapComplaint(resp.GetComplaint()),
	})
}

func (h *GatewayAnalyticHandler) GetUserComplaintAnalytic(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	uid, ok := ctx.Value(middleware.UserID).(int64)
	if !ok {
		sendUnauthorized(w)
		return
	}

	resp, err := h.ComplaintService.GetUserComplaintAnalytic(ctx, &complaintv1.RequestGetUserComplaintAnalytic{UserId: uid})
	if err != nil {
		sendComplaintError(w, err)
		return
	}

	response.Send(w, http.StatusOK, dtoApi.ApiSuccessResponse[complaintAnalyticDTO]{
		Status: dtoApi.Success,
		Body: complaintAnalyticDTO{
			CountStatus: countStatusDTO{
				CountStatusOpened: resp.GetCountStatus().GetCountStatusOpened(),
				CountStatusInWork: resp.GetCountStatus().GetCountStatusInWork(),
				CountStatusClosed: resp.GetCountStatus().GetCountStatusClosed(),
			},
			CountType: countTypeDTO{
				CountBug:     resp.GetCountType().GetCountBug(),
				CountUpgrade: resp.GetCountType().GetCountUpgrade(),
				CountProduct: resp.GetCountType().GetCountProduct(),
			},
		},
	})
}

func mapComplaintTypeFromString(t string) complaintv1.ComplaintType {
	switch strings.ToLower(strings.TrimSpace(t)) {
	case "bug":
		return complaintv1.ComplaintType_COMPLAINT_TYPE_BUG
	case "product":
		return complaintv1.ComplaintType_COMPLAINT_TYPE_PRODUCT
	case "upgrade":
		return complaintv1.ComplaintType_COMPLAINT_TYPE_UPGRADE
	default:
		return complaintv1.ComplaintType_COMPLAINT_TYPE_UNSPECIFIED
	}
}

func mapComplaintStatusFromString(s string) complaintv1.ComplaintStatus {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "new":
		return complaintv1.ComplaintStatus_COMPLAINT_STATUS_NEW
	case "in_progress":
		return complaintv1.ComplaintStatus_COMPLAINT_STATUS_IN_PROGRESS
	case "closed":
		return complaintv1.ComplaintStatus_COMPLAINT_STATUS_CLOSED
	default:
		return complaintv1.ComplaintStatus_COMPLAINT_STATUS_UNSPECIFIED
	}
}

func mapComplaint(c *complaintv1.ComplaintItem) *complaintDTO {
	if c == nil {
		return nil
	}
	var fb feedback
	if c.GetFeedback() != nil {
		fb = feedback{
			FeedbackName:  c.GetFeedback().GetFeedbackName(),
			FeedbackEmail: c.GetFeedback().GetFeedbackEmail(),
		}
	}
	out := &complaintDTO{
		ID:        c.GetId(),
		Type:      c.GetType(),
		Status:    c.GetStatus(),
		Feedback:  fb,
		Body:      c.GetBody(),
		UserID:    c.GetUserId(),
		CreatedAt: c.GetCreatedAt().AsTime().Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: c.GetUpdatedAt().AsTime().Format("2006-01-02T15:04:05Z07:00"),
	}
	if c.AttachmentUrl != nil {
		out.AttachmentURL = c.AttachmentUrl
	}
	return out
}

func mapComplaints(list []*complaintv1.ComplaintItem) []*complaintDTO {
	result := make([]*complaintDTO, 0, len(list))
	for _, item := range list {
		result = append(result, mapComplaint(item))
	}
	return result
}

func sendComplaintError(w http.ResponseWriter, err error) {
	_, appCode, _ := grpcerr.Error(err)
	switch complaintv1.ComplaintErrorCode(appCode) {
	case complaintv1.ComplaintErrorCode_COMPLAINT_ERROR_NOT_FOUND:
		response.Send(w, http.StatusNotFound, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{Code: dtoApi.NotFound, Message: dtoApi.NotFoundMsg}},
		})
	case complaintv1.ComplaintErrorCode_COMPLAINT_ERROR_INVALID_INPUT:
		response.Send(w, http.StatusBadRequest, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{Code: dtoApi.InvalidJson, Message: dtoApi.InvalidJsonMsg}},
		})
	default:
		sendInternal(w)
	}
}

func sendUnauthorized(w http.ResponseWriter) {
	response.Send(w, http.StatusUnauthorized, dtoApi.ApiErrorResponse{
		Status: dtoApi.Error,
		Errors: []dtoApi.ApiError{{Code: dtoApi.Unauthorized, Message: dtoApi.UnauthorizedMsg}},
	})
}

func sendInvalidID(w http.ResponseWriter) {
	response.Send(w, http.StatusBadRequest, dtoApi.ApiErrorResponse{
		Status: dtoApi.Error,
		Errors: []dtoApi.ApiError{{Code: dtoApi.InvalidID, Message: dtoApi.InvalidIDMsg}},
	})
}

func sendInvalidJSON(w http.ResponseWriter) {
	response.Send(w, http.StatusBadRequest, dtoApi.ApiErrorResponse{
		Status: dtoApi.Error,
		Errors: []dtoApi.ApiError{{Code: dtoApi.InvalidJson, Message: dtoApi.InvalidJsonMsg}},
	})
}

func sendInternal(w http.ResponseWriter) {
	response.Send(w, http.StatusInternalServerError, dtoApi.ApiErrorResponse{
		Status: dtoApi.Error,
		Errors: []dtoApi.ApiError{{Code: dtoApi.InternalError, Message: dtoApi.InternalErrorMsg}},
	})
}
