package response

import (
	"encoding/json"
	"net/http"

	dtoApi "github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/api"
)

func Send(w http.ResponseWriter, status int, response dtoApi.ApiResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	json.NewEncoder(w).Encode(response)
}
