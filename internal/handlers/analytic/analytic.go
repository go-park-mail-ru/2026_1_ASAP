package analytic

import (
	"context"
	"log"
	"net/http"

	dtoAnalytic "github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/analytic"
	dtoApi "github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/api"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/middleware"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/utils/response"
)

type AnalyticServiceInterface interface {
	GetUserComplaintAnalytic(ctx context.Context, request dtoAnalytic.RequestComplaintAnalytic) (dtoAnalytic.ResponseComplaintAnalytic, error)
}

type AnalyticHandler struct {
	AnalyticService AnalyticServiceInterface
}

func NewAnalyticHandler(analyticService AnalyticServiceInterface) *AnalyticHandler {
	return &AnalyticHandler{
		AnalyticService: analyticService,
	}
}

func (h *AnalyticHandler) GetUserComplaintAnalytic(w http.ResponseWriter, r *http.Request) {
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

	var request dtoAnalytic.RequestComplaintAnalytic
	request.UserID = userId

	res, err := h.AnalyticService.GetUserComplaintAnalytic(ctx, request)
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

	resp := dtoApi.ApiSuccessResponse[dtoAnalytic.ResponseComplaintAnalytic]{
		Status: dtoApi.Success,
		Body:   res,
	}
	response.Send(w, http.StatusOK, resp)
}
