package api

type ApiError struct {
	Code    string `json:"code" enums:"INVALID_JSON,INVALID_CREDENTIALS,UNAUTHORIZED,FAIL_LOGOUT"`
	Message string `json:"message" example:"Error message"`
}

type ResponseStatus string

const (
	SUCCESS ResponseStatus = "success"
	ERROR   ResponseStatus = "error"
)

type ApiSucessResponse[T any] struct {
	Status ResponseStatus `json:"status" example:"success"`
	Body   T              `json:"body,omitempty"`
}

type ApiErrorResponse struct {
	Status ResponseStatus `json:"status" example:"error"`
	Errors []ApiError     `json:"errors"`
}
