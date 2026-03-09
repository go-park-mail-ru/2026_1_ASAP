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

type ApiSucessResponse[T any] struct {
	Status ResponseStatus `json:"status"`
	Body   T              `json:"body,omitempty"`
}

type ApiErrorResponse struct {
	Status ResponseStatus `json:"status"`
	Errors []ApiError     `json:"errors"`
}
