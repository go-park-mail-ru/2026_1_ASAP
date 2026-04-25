package complaint

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi"

	dtoApi "github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/api"
	dtoComplaint "github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/complaint"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/media"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/middleware"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/utils/response"
)

type ComplaintServiceInterface interface {
	CreateAuthrozied(ctx context.Context, userID int64, request dtoComplaint.RequestCreateComplaint) (*dtoComplaint.ResponseCreateComplaint, error)
	GetComplaint(ctx context.Context, request dtoComplaint.RequestGetComplaint) (dtoComplaint.ResponseGetComplaint, error)
	GetComplaintsByUser(ctx context.Context, userID int64) (dtoComplaint.ResponseGetComplaints, error)
	CreateUnAuthrozied(ctx context.Context, request dtoComplaint.RequestCreateComplaint) (*dtoComplaint.ResponseCreateComplaint, error)
	UpdateComplaintStatus(ctx context.Context, request dtoComplaint.RequestUpdateComplaintStatus) (dtoComplaint.ResponseGetComplaint, error)
	GetAllComplaints(ctx context.Context) (dtoComplaint.ResponseGetComplaints, error)
}

type ComplaintHandler struct {
	ComplaintService ComplaintServiceInterface
}

func NewComplaintHandler(complaintService ComplaintServiceInterface) *ComplaintHandler {
	return &ComplaintHandler{
		ComplaintService: complaintService,
	}
}

func (h *ComplaintHandler) CreateComplaintUnAuthorized(w http.ResponseWriter, r *http.Request) {
	var req dtoComplaint.RequestCreateComplaint
	contentType := r.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "multipart/form-data") {
		resp := dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{
				{
					Code:    dtoApi.InvalidJson,
					Message: dtoApi.InvalidJsonMsg,
				},
			},
		}
		response.Send(w, http.StatusBadRequest, resp)
		return
	}

	if err := r.ParseMultipartForm(media.MaxAvatarBytes + 1024); err != nil {
		resp := dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{
				{
					Code:    dtoApi.InvalidJson,
					Message: dtoApi.InvalidJsonMsg,
				},
			},
		}
		response.Send(w, http.StatusBadRequest, resp)
		return
	}

	rawJSON := r.FormValue("payload")
	if rawJSON == "" {
		rawJSON = r.FormValue("json")
	}
	if rawJSON == "" {
		resp := dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{
				{
					Code:    dtoApi.InvalidJson,
					Message: dtoApi.InvalidJsonMsg,
				},
			},
		}
		response.Send(w, http.StatusBadRequest, resp)
		return
	}

	if err := json.Unmarshal([]byte(rawJSON), &req); err != nil {
		resp := dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{
				{
					Code:    dtoApi.InvalidJson,
					Message: dtoApi.InvalidJsonMsg,
				},
			},
		}
		response.Send(w, http.StatusBadRequest, resp)
		return
	}

	fileField := "attachment"
	file, header, err := r.FormFile(fileField)
	if err != nil {
		if errors.Is(err, http.ErrMissingFile) {
			fileField = "file"
			file, header, err = r.FormFile(fileField)
		}
	}
	if err != nil {
		if !errors.Is(err, http.ErrMissingFile) {
			resp := dtoApi.ApiErrorResponse{
				Status: dtoApi.Error,
				Errors: []dtoApi.ApiError{
					{
						Code:    dtoApi.FileNotFound,
						Message: dtoApi.FileNotFoundMsg,
					},
				},
			}
			response.Send(w, http.StatusBadRequest, resp)
			return
		}
	} else {
		fileInput, err := media.FileInputFromMultipart(file, header)
		if err != nil {
			switch {
			case errors.Is(err, media.ErrEmptyFile):
				resp := dtoApi.ApiErrorResponse{
					Status: dtoApi.Error,
					Errors: []dtoApi.ApiError{
						{
							Code:    dtoApi.EmptyFile,
							Message: dtoApi.EmptyFileMsg,
						},
					},
				}
				response.Send(w, http.StatusBadRequest, resp)
				return
			case errors.Is(err, media.ErrFileTooLarge):
				resp := dtoApi.ApiErrorResponse{
					Status: dtoApi.Error,
					Errors: []dtoApi.ApiError{
						{
							Code:    dtoApi.FileTooLarge,
							Message: dtoApi.FileTooLargeMsg,
						},
					},
				}
				response.Send(w, http.StatusBadRequest, resp)
				return
			default:
				resp := dtoApi.ApiErrorResponse{
					Status: dtoApi.Error,
					Errors: []dtoApi.ApiError{
						{
							Code:    dtoApi.InternalError,
							Message: dtoApi.InternalErrorMsg,
						},
					},
				}
				log.Println(err.Error())
				response.Send(w, http.StatusInternalServerError, resp)
				return
			}
		}
		req.File = fileInput
	}

	complaint, err := h.ComplaintService.CreateUnAuthrozied(r.Context(), req)
	if err != nil {
		resp := dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{
				{
					Code:    dtoApi.InternalError,
					Message: dtoApi.InternalErrorMsg,
				},
			},
		}
		log.Println(err.Error())
		response.Send(w, http.StatusInternalServerError, resp)
		return
	}

	resp := dtoApi.ApiSuccessResponse[dtoComplaint.ResponseCreateComplaint]{
		Status: dtoApi.Success,
		Body:   *complaint,
	}
	response.Send(w, http.StatusOK, resp)
}

func (h *ComplaintHandler) CreateComplaintAuthorized(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userId, ok := ctx.Value(middleware.UserID).(int64)
	if !ok {
		resp := dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{
				{
					Code:    dtoApi.Unauthorized,
					Message: dtoApi.UnauthorizedMsg,
				},
			},
		}
		response.Send(w, http.StatusUnauthorized, resp)
		return
	}

	var req dtoComplaint.RequestCreateComplaint
	contentType := r.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "multipart/form-data") {
		resp := dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{
				{
					Code:    dtoApi.InvalidJson,
					Message: dtoApi.InvalidJsonMsg,
				},
			},
		}
		response.Send(w, http.StatusBadRequest, resp)
		return
	}

	if err := r.ParseMultipartForm(media.MaxAvatarBytes + 1024); err != nil {
		resp := dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{
				{
					Code:    dtoApi.InvalidJson,
					Message: dtoApi.InvalidJsonMsg,
				},
			},
		}
		response.Send(w, http.StatusBadRequest, resp)
		return
	}

	rawJSON := r.FormValue("payload")
	if rawJSON == "" {
		rawJSON = r.FormValue("json")
	}
	if rawJSON == "" {
		resp := dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{
				{
					Code:    dtoApi.InvalidJson,
					Message: dtoApi.InvalidJsonMsg,
				},
			},
		}
		response.Send(w, http.StatusBadRequest, resp)
		return
	}

	if err := json.Unmarshal([]byte(rawJSON), &req); err != nil {
		resp := dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{
				{
					Code:    dtoApi.InvalidJson,
					Message: dtoApi.InvalidJsonMsg,
				},
			},
		}
		response.Send(w, http.StatusBadRequest, resp)
		return
	}

	fileField := "attachment"
	file, header, err := r.FormFile(fileField)
	if err != nil {
		if errors.Is(err, http.ErrMissingFile) {
			fileField = "file"
			file, header, err = r.FormFile(fileField)
		}
	}
	if err != nil {
		if !errors.Is(err, http.ErrMissingFile) {
			resp := dtoApi.ApiErrorResponse{
				Status: dtoApi.Error,
				Errors: []dtoApi.ApiError{
					{
						Code:    dtoApi.FileNotFound,
						Message: dtoApi.FileNotFoundMsg,
					},
				},
			}
			response.Send(w, http.StatusBadRequest, resp)
			return
		}
	} else {
		fileInput, err := media.FileInputFromMultipart(file, header)
		if err != nil {
			switch {
			case errors.Is(err, media.ErrEmptyFile):
				resp := dtoApi.ApiErrorResponse{
					Status: dtoApi.Error,
					Errors: []dtoApi.ApiError{
						{
							Code:    dtoApi.EmptyFile,
							Message: dtoApi.EmptyFileMsg,
						},
					},
				}
				response.Send(w, http.StatusBadRequest, resp)
				return
			case errors.Is(err, media.ErrFileTooLarge):
				resp := dtoApi.ApiErrorResponse{
					Status: dtoApi.Error,
					Errors: []dtoApi.ApiError{
						{
							Code:    dtoApi.FileTooLarge,
							Message: dtoApi.FileTooLargeMsg,
						},
					},
				}
				response.Send(w, http.StatusBadRequest, resp)
				return
			default:
				resp := dtoApi.ApiErrorResponse{
					Status: dtoApi.Error,
					Errors: []dtoApi.ApiError{
						{
							Code:    dtoApi.InternalError,
							Message: dtoApi.InternalErrorMsg,
						},
					},
				}
				log.Println(err.Error())
				response.Send(w, http.StatusInternalServerError, resp)
				return
			}
		}
		req.File = fileInput
	}

	complaint, err := h.ComplaintService.CreateAuthrozied(ctx, userId, req)
	if err != nil {
		resp := dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{
				{
					Code:    dtoApi.InternalError,
					Message: dtoApi.InternalErrorMsg,
				},
			},
		}
		log.Println(err.Error())
		response.Send(w, http.StatusInternalServerError, resp)
		return
	}

	resp := dtoApi.ApiSuccessResponse[dtoComplaint.ResponseCreateComplaint]{
		Status: dtoApi.Success,
		Body:   *complaint,
	}
	response.Send(w, http.StatusOK, resp)
}

func (h *ComplaintHandler) GetComplaint(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	_, ok := ctx.Value(middleware.UserID).(int64)
	if !ok {
		resp := dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{
				{
					Code:    dtoApi.Unauthorized,
					Message: dtoApi.UnauthorizedMsg,
				},
			},
		}
		response.Send(w, http.StatusUnauthorized, resp)
		return
	}

	complaintIDString := chi.URLParam(r, "id")
	complaintID, err := strconv.ParseInt(complaintIDString, 10, 64)
	if err != nil || complaintID < 1 {
		resp := dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{
				{
					Code:    dtoApi.InvalidID,
					Message: dtoApi.InvalidIDMsg,
				},
			},
		}
		response.Send(w, http.StatusBadRequest, resp)
		return
	}

	complaint, err := h.ComplaintService.GetComplaint(ctx, dtoComplaint.RequestGetComplaint{ComplaintId: complaintID})
	if err != nil {
		resp := dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{
				{
					Code:    dtoApi.InternalError,
					Message: dtoApi.InternalErrorMsg,
				},
			},
		}
		log.Println(err.Error())
		response.Send(w, http.StatusInternalServerError, resp)
		return
	}

	resp := dtoApi.ApiSuccessResponse[dtoComplaint.ResponseGetComplaint]{
		Status: dtoApi.Success,
		Body:   complaint,
	}
	response.Send(w, http.StatusOK, resp)
}

func (h *ComplaintHandler) GetMyComplaints(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := ctx.Value(middleware.UserID).(int64)
	if !ok {
		resp := dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{
				{
					Code:    dtoApi.Unauthorized,
					Message: dtoApi.UnauthorizedMsg,
				},
			},
		}
		response.Send(w, http.StatusUnauthorized, resp)
		return
	}

	complaints, err := h.ComplaintService.GetComplaintsByUser(ctx, userID)
	if err != nil {
		resp := dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{
				{
					Code:    dtoApi.InternalError,
					Message: dtoApi.InternalErrorMsg,
				},
			},
		}
		log.Println(err.Error())
		response.Send(w, http.StatusInternalServerError, resp)
		return
	}

	resp := dtoApi.ApiSuccessResponse[dtoComplaint.ResponseGetComplaints]{
		Status: dtoApi.Success,
		Body:   complaints,
	}
	response.Send(w, http.StatusOK, resp)
}

func (h *ComplaintHandler) UpdateComplaintStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	_, ok := ctx.Value(middleware.UserID).(int64)
	if !ok {
		resp := dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{
				{
					Code:    dtoApi.Unauthorized,
					Message: dtoApi.UnauthorizedMsg,
				},
			},
		}
		response.Send(w, http.StatusUnauthorized, resp)
		return
	}

	var req dtoComplaint.RequestUpdateComplaintStatus
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&req); err != nil {
		resp := dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{
				{
					Code:    dtoApi.InvalidJson,
					Message: dtoApi.InvalidJsonMsg,
				},
			},
		}
		response.Send(w, http.StatusBadRequest, resp)
		return
	}

	complaint, err := h.ComplaintService.UpdateComplaintStatus(ctx, req)
	if err != nil {
		resp := dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{
				{
					Code:    dtoApi.InternalError,
					Message: dtoApi.InternalErrorMsg,
				},
			},
		}
		log.Println(err.Error())
		response.Send(w, http.StatusInternalServerError, resp)
		return
	}

	resp := dtoApi.ApiSuccessResponse[dtoComplaint.ResponseGetComplaint]{
		Status: dtoApi.Success,
		Body:   complaint,
	}
	response.Send(w, http.StatusOK, resp)
}

func (h *ComplaintHandler) GetAllComplaints(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	complaints, err := h.ComplaintService.GetAllComplaints(ctx)
	if err != nil {
		resp := dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{
				{
					Code:    dtoApi.InternalError,
					Message: dtoApi.InternalErrorMsg,
				},
			},
		}
		log.Println(err.Error())
		response.Send(w, http.StatusInternalServerError, resp)
		return
	}

	resp := dtoApi.ApiSuccessResponse[dtoComplaint.ResponseGetComplaints]{
		Status: dtoApi.Success,
		Body:   complaints,
	}
	response.Send(w, http.StatusOK, resp)
}
