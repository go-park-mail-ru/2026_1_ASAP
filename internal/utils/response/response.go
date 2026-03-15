package response

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/api"
)

func Send(w http.ResponseWriter, status int, response interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		resp := fmt.Sprintf(`{"status":%s, "errors":[{"Code":"%s", "Message":"%s"}]}`, api.Error, api.InternalError, api.InternalErrorMsg)
		w.Write([]byte(resp))
	}
}
