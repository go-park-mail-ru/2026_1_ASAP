package contacts

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	profilev1 "github.com/go-park-mail-ru/2026_1_ASAP/gen/go/profile/v1"
	dtoApi "github.com/go-park-mail-ru/2026_1_ASAP/internal/gateway/dto/api"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/gateway/middleware"
	contactdto "github.com/go-park-mail-ru/2026_1_ASAP/internal/profile/dto/contact"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/utils/response"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type GatewayContactsHandler struct {
	Profile profilev1.ProfileClient
}

func NewGatewayContactsHandler(p profilev1.ProfileClient) *GatewayContactsHandler {
	return &GatewayContactsHandler{Profile: p}
}

func contactItemToResponse(c *profilev1.ContactItem) *contactdto.ContactResponse {
	if c == nil {
		return nil
	}
	out := &contactdto.ContactResponse{
		UserID:        c.GetUserId(),
		ContactUserID: c.GetContactUserId(),
		FirstName:     c.GetFirstName(),
		IsOnline:      c.GetIsOnline(),
	}
	if c.LastName != nil {
		ln := c.GetLastName()
		out.LastName = &ln
	}
	if u := c.GetContactAvatarUrl(); u != "" {
		out.ContactAvatarUrl = &u
	}
	if t := c.GetCreatedAt(); t != nil {
		out.CreatedAt = t.AsTime()
	} else {
		out.CreatedAt = time.Time{}
	}
	return out
}

// GetContacts GET /api/v1/contacts/
func (h *GatewayContactsHandler) GetContacts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	uid, ok := ctx.Value(middleware.UserID).(int64)
	if !ok {
		unauthorized(w)
		return
	}
	resp, err := h.Profile.ListContacts(ctx, &profilev1.RequestListContacts{UserId: uid})
	if err != nil {
		internal(w)
		return
	}
	list := make([]contactdto.ContactResponse, 0)
	if resp != nil {
		for _, c := range resp.GetContacts() {
			if d := contactItemToResponse(c); d != nil {
				list = append(list, *d)
			}
		}
	}
	response.Send(w, http.StatusOK, dtoApi.ApiSuccessResponse[[]contactdto.ContactResponse]{
		Status: dtoApi.Success,
		Body:   list,
	})
}

// CreateContact POST /api/v1/contacts/
func (h *GatewayContactsHandler) CreateContact(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	ctx := r.Context()
	uid, ok := ctx.Value(middleware.UserID).(int64)
	if !ok {
		unauthorized(w)
		return
	}
	var body contactdto.AddContactRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Send(w, http.StatusBadRequest, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{
				Code:    dtoApi.InvalidJson,
				Message: dtoApi.InvalidJsonMsg,
			}},
		})
		return
	}
	if body.ContactUserID <= 0 {
		response.Send(w, http.StatusBadRequest, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{
				Code:    dtoApi.InvalidID,
				Message: dtoApi.InvalidIDMsg,
			}},
		})
		return
	}
	req := &profilev1.RequestAddContact{
		UserId:        uid,
		ContactUserId: body.ContactUserID,
		FirstName:     body.FirstName,
	}
	if body.LastName != nil {
		req.LastName = body.LastName
	}
	resp, err := h.Profile.AddContact(ctx, req)
	if err != nil {
		contactCreateErr(w, err)
		return
	}
	var b *contactdto.ContactResponse
	if resp != nil && resp.GetContact() != nil {
		b = contactItemToResponse(resp.GetContact())
	}
	response.Send(w, http.StatusCreated, dtoApi.ApiSuccessResponse[*contactdto.ContactResponse]{
		Status: dtoApi.Success,
		Body:   b,
	})
}

// DeleteContact DELETE /api/v1/contacts/{contact_user_id}
func (h *GatewayContactsHandler) DeleteContact(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	uid, ok := ctx.Value(middleware.UserID).(int64)
	if !ok {
		unauthorized(w)
		return
	}
	s := chi.URLParam(r, "contact_user_id")
	cid, err := strconv.ParseInt(s, 10, 64)
	if err != nil || cid < 1 {
		response.Send(w, http.StatusBadRequest, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{
				Code:    dtoApi.InvalidID,
				Message: dtoApi.InvalidIDMsg,
			}},
		})
		return
	}
	_, err = h.Profile.DeleteContact(ctx, &profilev1.RequestDeleteContact{
		UserId:        uid,
		ContactUserId: cid,
	})
	if err != nil {
		contactDeleteErr(w, err)
		return
	}
	response.Send(w, http.StatusOK, dtoApi.ApiSuccessResponse[any]{
		Status: dtoApi.Success,
		Body:   "contact deleted",
	})
}

func unauthorized(w http.ResponseWriter) {
	response.Send(w, http.StatusUnauthorized, dtoApi.ApiErrorResponse{
		Status: dtoApi.Error,
		Errors: []dtoApi.ApiError{{
			Code:    dtoApi.Unauthorized,
			Message: dtoApi.UnauthorizedMsg,
		}},
	})
}

func contactCreateErr(w http.ResponseWriter, err error) {
	st, ok := status.FromError(err)
	if !ok {
		internal(w)
		return
	}
	switch st.Code() {
	case codes.AlreadyExists:
		response.Send(w, http.StatusConflict, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{
				Code:    dtoApi.ContactAlreadyExists,
				Message: dtoApi.ContactAlreadyExistsMsg,
			}},
		})
	case codes.NotFound:
		response.Send(w, http.StatusNotFound, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{
				Code:    dtoApi.NotFound,
				Message: dtoApi.NotFoundMsg,
			}},
		})
	case codes.InvalidArgument:
		response.Send(w, http.StatusBadRequest, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{
				Code:    dtoApi.ContactWithYourself,
				Message: dtoApi.ContactWithYourselfMsg,
			}},
		})
	default:
		internal(w)
	}
}

func contactDeleteErr(w http.ResponseWriter, err error) {
	st, ok := status.FromError(err)
	if !ok {
		internal(w)
		return
	}
	if st.Code() == codes.NotFound {
		response.Send(w, http.StatusNotFound, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{
				Code:    dtoApi.ContactNotFound,
				Message: dtoApi.ContactNotFoundMsg,
			}},
		})
		return
	}
	internal(w)
}

func internal(w http.ResponseWriter) {
	response.Send(w, http.StatusInternalServerError, dtoApi.ApiErrorResponse{
		Status: dtoApi.Error,
		Errors: []dtoApi.ApiError{{
			Code:    dtoApi.InternalError,
			Message: dtoApi.InternalErrorMsg,
		}},
	})
}
