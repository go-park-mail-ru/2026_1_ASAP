package api

type ErrorCode string
type ErrorMessage string

const (
	InvalidJson        = "INVALID_JSON"
	InvalidCredentials = "INVALID_CREDENTIALS"
	Unauthorized       = "UNAUTHORIZED"
	FailLogout         = "FAIL_LOGOUT"
	InternalError      = "INTERNAL_ERROR"

	CreateFailed = "CREATE_FAILED"
	AccessDenied = "ACCESS_DENIED"
	InvalidID    = "INVALID_ID"
)

const (
	InvalidJsonMsg        = "Invalid request body"
	InvalidCredentialsMsg = "Invalid credentials"
	UnauthorizedMsg       = "Unauthorized"
	FailLogoutMsg         = "Fail logout"
	InternalErrorMsg      = "Internal error"

	CreateFailedMsg = "Create failed"
	AccessDeniedMsg = "Access denied"
	InvalidIDMsg    = "Invalid ID"
)

type ApiError struct {
	Code    ErrorCode    `json:"code" enums:"INVALID_JSON,INVALID_CREDENTIALS,UNAUTHORIZED,FAIL_LOGOUT"`
	Message ErrorMessage `json:"message" example:"Error message"`
}

type ResponseStatus string

const (
	Success ResponseStatus = "success"
	Error   ResponseStatus = "error"
)

type ApiSucessResponse[T any] struct {
	Status ResponseStatus `json:"status" example:"success"`
	Body   T              `json:"body"`
}

type ApiErrorResponse struct {
	Status ResponseStatus `json:"status" example:"error"`
	Errors []ApiError     `json:"errors"`
}
