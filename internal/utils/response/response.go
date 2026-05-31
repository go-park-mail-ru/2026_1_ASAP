package response

import (
	"encoding/json"
	"net/http"

	"github.com/go-park-mail-ru/2026_1_ASAP/internal/gateway/dto/api"
)

func Send(w http.ResponseWriter, status int, response interface{}) {
	body, err := json.Marshal(response)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(api.ApiErrorResponse{
			Status: api.Error,
			Errors: []api.ApiError{{Code: api.InternalError, Message: api.InternalErrorMsg}},
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}
