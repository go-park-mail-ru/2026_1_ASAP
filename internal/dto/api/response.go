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
	FileNotFound           = "FILE_NOT_FOUND"
	EmptyFile              = "EMPTY_FILE"
	InvalidFileFormat      = "INVALID_FILE_FORMAT"
	FileTooLarge           = "FILE_TOO_LARGE"
	EmptyBIO               = "EMPTY_BIO"
	NotFound               = "NOT_FOUND"
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
	FileNotFoundMsg           = "File not found"
	EmptyFileMsg              = "empty file"
	InvalidFileFormatMsg      = "Invalid file format"
	FileTooLargeMsg           = "File too large"
	EmptyBIOMsg               = "Empty BIO"
	NotFoundMsg               = "Not found"
)

type ApiError struct {
	Code    ErrorCode    `json:"code" example:"ERROR_CODE"`
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
