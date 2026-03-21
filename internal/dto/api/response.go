package api

type ErrorCode string
type ErrorMessage string

const (
	InvalidJson            = "INVALID_JSON"
	InvalidCredentials     = "INVALID_CREDENTIALS"
	Unauthorized           = "UNAUTHORIZED"
	InternalError          = "INTERNAL_ERROR"
	EmailAlreadyRegistered = "EMAIL_ALREADY_REGISTERED"
	LoginAlreadyRegistered = "LOGIN_ALREADY_REGISTERED"
	DialogAlreadyExists    = "DIALOG_ALREADY_EXISTS"
	CreateFailed           = "CREATE_FAILED"
	AccessDenied           = "ACCESS_DENIED"
	InvalidID              = "INVALID_ID"
)

const (
	InvalidJsonMsg            = "Invalid request body"
	InvalidCredentialsMsg     = "Invalid credentials"
	UnauthorizedMsg           = "Unauthorized"
	InternalErrorMsg          = "Internal error"
	EmailAlreadyRegisteredMsg = "Email already registered"
	LoginAlreadyRegisteredMsg = "Login already registered"
	DialogAlreadyExistsMsg    = "Dialog already exists"
	CreateFailedMsg           = "Create failed"
	AccessDeniedMsg           = "Access denied"
	InvalidIDMsg              = "Invalid ID"
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

type ApiSuccessResponse[T any] struct {
	Body   T              `json:"body"`
	Status ResponseStatus `json:"status" example:"success"`
}

type ApiErrorResponse struct {
	Status ResponseStatus `json:"status" example:"error"`
	Errors []ApiError     `json:"errors"`
}
