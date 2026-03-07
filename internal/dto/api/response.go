package api

type ApiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ResponseStatus string

const (
	SUCCESS ResponseStatus = "success"
	ERROR   ResponseStatus = "error"
)

type ApiResponse struct {
	Status ResponseStatus `json:"status"`
	Body   interface{}    `json:"body,omitempty"`
	Error  []ApiError     `json:"errors,omitempty"`
}
