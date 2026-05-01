package api

type ErrorCode string
type ErrorMessage string

const (
	InvalidJson              = "INVALID_JSON"
	InvalidCredentials       = "INVALID_CREDENTIALS"
	Unauthorized             = "UNAUTHORIZED"
	InternalError            = "INTERNAL_ERROR"
	EmailAlreadyRegistered   = "EMAIL_ALREADY_REGISTERED"
	LoginAlreadyRegistered   = "LOGIN_ALREADY_REGISTERED"
	DialogAlreadyExists      = "DIALOG_ALREADY_EXISTS"
	ContactAlreadyExists     = "CONTACT_ALREADY_EXISTS"
	ContactWithYourself      = "CANT_CREATE_CONTACT_WITH_YOURSELF"
	ContactNotFound          = "CONTACT_NOT_FOUND"
	CreateFailed             = "CREATE_FAILED"
	AccessDenied             = "ACCESS_DENIED"
	InvalidID                = "INVALID_ID"
	FileNotFound             = "FILE_NOT_FOUND"
	EmptyFile                = "EMPTY_FILE"
	InvalidFileFormat        = "INVALID_FILE_FORMAT"
	FileTooLarge             = "FILE_TOO_LARGE"
	EmptyBIO                 = "EMPTY_BIO"
	NotFound                 = "NOT_FOUND"
	InvalidDateFormat        = "INVALID_DATE_FORMAT"
	InvalidDate              = "INVALID_DATE"
	CantDeleteChat           = "CANT_DELETE_CHAT"
	CantChangeAvatar         = "CANT_CHANGE_AVATAR"
	CantFindChat             = "CANT_FIND_CHAT"
	CantChangeTitle          = "CANT_CHANGE_TITLE"
	CantAddMemberToDialog    = "CANT_ADD_MEMBER_TO_DIALOG"
	OnlyOwnerCanAddMembers   = "ONLY_OWNER_OF_CHAT_CAN_ADD_MEMBERS"
	MemberAlreadyInChat      = "MEMBER_ALREADY_IN_CHAT"
	NotMemberOfChat          = "NOT_MEMBER_OF_CHAT"
	DeleteMemberFromDialog   = "CANT_DELETE_MEMBER_FROM_DIALOG"
	OnlyOwnerCanDeleteMember = "ONLY_OWNER_OF_CHAT_CAN_DELETE_MEMBER"
	DeleteOwnerOfChat        = "CANT_DELETE_OWNER_OF_CHAT"
	MemberNotFound           = "MEMBER_NOT_FOUND"
	UserNotMemberOfChat      = "NOT_MEMBER_OF_CHAT"
	CantQuitDialog           = "CANT_QUIT_DIALOG"
	OwnerCantQuitChat        = "OWNER_CANT_QUIT_CHAT"
	CSRFTokenMissing         = "CSRF_TOKEN_MISSING"
	CSRFTokenMismatch        = "CSRF_TOKEN_MISMATCH"
	CSRFTokenExpired         = "CSRF_TOKEN_EXPIRED"
	CSRFTokenNotInSession    = "CSRF_TOKEN_NOT_IN_SESSION"
	EmptyFirstName           = "EMPTY_FIRST_NAME"
	UserNotFound             = "USER_NOT_FOUND"
	VKIDFailed               = "VKID_FAILED"
	NotImplemented           = "NOT_IMPLEMENTED"
	EmptyLogin               = "EMPTY_LOGIN"
	OnlyChannelCanBeJoined   = "ONLY_CHANNEL_CAN_BE_JOINED"
)

const (
	InvalidJsonMsg              = "Invalid request body"
	InvalidCredentialsMsg       = "Invalid credentials"
	UnauthorizedMsg             = "Unauthorized"
	InternalErrorMsg            = "Internal error"
	EmailAlreadyRegisteredMsg   = "Email already registered"
	LoginAlreadyRegisteredMsg   = "Login already registered"
	DialogAlreadyExistsMsg      = "Dialog already exists"
	ContactAlreadyExistsMsg     = "Contact already exists"
	ContactWithYourselfMsg      = "You cant create contact with yourself"
	ContactNotFoundMsg          = "Contact not found"
	CreateFailedMsg             = "Create failed"
	AccessDeniedMsg             = "Access denied"
	InvalidIDMsg                = "Invalid ID"
	FileNotFoundMsg             = "File not found"
	EmptyFileMsg                = "empty file"
	InvalidFileFormatMsg        = "Invalid file format"
	FileTooLargeMsg             = "File too large"
	EmptyBIOMsg                 = "Empty BIO"
	NotFoundMsg                 = "Not found"
	InvalidDateFormatMsg        = "Invalid date format"
	InvalidDateMsg              = "Invalid date"
	CantDeleteChatMsg           = "You cant delete this chat"
	NotMemberOfChatMsg          = "You are not member of this chat"
	CantChangeAvatarMsg         = "You cant change avatar in dialog"
	CantFindChatMsg             = "Cant find chat"
	CantChangeTitleMsg          = "You cant change title in dialog"
	CantAddMemberToDialogMsg    = "You cant add more members in your dialog"
	OnlyOwnerCanAddMembersMsg   = "Only owner of chat can add members"
	MemberAlreadyInChatMsg      = "Member you tried to add already in chat"
	DeleteMemberFromDialogMsg   = "You cant delete member from dialog chat"
	OnlyOwnerCanDeleteMemberMsg = "Only owner of chat can delete member"
	DeleteOwnerOfChatMsg        = "You cant delete owner of chat"
	MemberNotFoundMsg           = "Cant find member you try to delete"
	UserNotMemberOfChatMsg      = "User you try to delete is not member of this chat"
	CantQuitDialogMsg           = "You cant quit dialog"
	OwnerCantQuitChatMsg        = "As owner of chat you cant quit chat"
	CSRFTokenMissingMsg         = "CSRF token missing"
	CSRFTokenMismatchMsg        = "CSRF token mismatch"
	CSRFTokenExpiredMsg         = "CSRF token expired"
	CSRFTokenNotInSessionMsg    = "CSRF token not in session"
	EmptyFirstNameMsg           = "Empty first name"
	UserNotFoundMsg             = "Cant find profile you try to add"
	VKIDFailedMsg               = "VKID failed to login"
	NotImplementedMsg           = "This endpoint is not available yet"
	EmptyLoginMsg               = "Login query parameter is required"
	OnlyChannelCanBeJoinedMsg   = "you can join only channel"
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
