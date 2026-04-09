package validation

import (
	"regexp"
	"unicode"

	dtoAuth "github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/auth"
	dtoChat "github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/chat"
	dtoContact "github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/contacts"
)

type ValidationError struct {
	Field   string
	Message string
	Code    string
}

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

func ValidateEmail(email string) []ValidationError {
	var errs []ValidationError

	if email == "" {
		errs = append(errs, ValidationError{
			Field:   "email",
			Message: "Email is required",
			Code:    "EMAIL_REQUIRED",
		})
		return errs
	}

	if !emailRegex.MatchString(email) {
		errs = append(errs, ValidationError{
			Field:   "email",
			Message: "Invalid email format",
			Code:    "EMAIL_INVALID",
		})
	}

	return errs
}

func ValidateLogin(login string) []ValidationError {
	var errs []ValidationError

	if login == "" {
		errs = append(errs, ValidationError{
			Field:   "login",
			Message: "Login is required",
			Code:    "LOGIN_REQUIRED",
		})
		return errs
	}

	if len(login) < 3 {
		errs = append(errs, ValidationError{
			Field:   "login",
			Message: "Login must be at least 3 characters",
			Code:    "LOGIN_TOO_SHORT",
		})
	}

	return errs
}

func ValidatePassword(password string) []ValidationError {
	var errs []ValidationError

	if password == "" {
		errs = append(errs, ValidationError{
			Field:   "password",
			Message: "Password is required",
			Code:    "PASSWORD_REQUIRED",
		})
		return errs
	}

	if len(password) < 6 {
		errs = append(errs, ValidationError{
			Field:   "password",
			Message: "Password must be at least 6 characters",
			Code:    "PASSWORD_TOO_SHORT",
		})
	}

	if len(password) > 64 {
		errs = append(errs, ValidationError{
			Field:   "password",
			Message: "Password must be less than 64 characters",
			Code:    "PASSWORD_TOO_LONG",
		})
	}

	var hasUpper, hasLower, hasDigit, hasSpecial bool

	for _, c := range password {
		switch {
		case unicode.IsUpper(c):
			hasUpper = true
		case unicode.IsLower(c):
			hasLower = true
		case unicode.IsDigit(c):
			hasDigit = true
		case unicode.IsSpace(c):
			errs = append(errs, ValidationError{
				Field:   "password",
				Message: "Password must not contain spaces",
				Code:    "PASSWORD_HAS_SPACE",
			})
		default:
			hasSpecial = true
		}
	}

	if !hasUpper {
		errs = append(errs, ValidationError{
			Field:   "password",
			Message: "Password must contain at least one uppercase letter",
			Code:    "PASSWORD_NO_UPPERCASE",
		})
	}

	if !hasLower {
		errs = append(errs, ValidationError{
			Field:   "password",
			Message: "Password must contain at least one lowercase letter",
			Code:    "PASSWORD_NO_LOWERCASE",
		})
	}

	if !hasDigit {
		errs = append(errs, ValidationError{
			Field:   "password",
			Message: "Password must contain at least one digit",
			Code:    "PASSWORD_NO_DIGIT",
		})
	}

	if !hasSpecial {
		errs = append(errs, ValidationError{
			Field:   "password",
			Message: "Password must contain at least one special character",
			Code:    "PASSWORD_NO_SPECIAL",
		})
	}

	return errs
}

func ValidationRequestRegistrate(request *dtoAuth.RequestRegistrate) []ValidationError {
	var errs []ValidationError
	errs = ValidateEmail(request.Email)
	errs = append(errs, ValidateLogin(request.Login)...)
	errs = append(errs, ValidatePassword(request.Password)...)

	return errs
}

func ValidationRequestLogin(request *dtoAuth.RequestLogin) []ValidationError {
	var errs []ValidationError
	errs = append(errs, ValidateLogin(request.Login)...)
	errs = append(errs, ValidatePassword(request.Password)...)

	return errs
}

func ValidationChatCreate(req *dtoChat.ChatCreate) []ValidationError {
	var errs []ValidationError

	if req.Title == "" {
		errs = append(errs, ValidationError{
			Field:   "title",
			Message: "Title is required",
			Code:    "TITLE_REQUIRED",
		})
	} else if len(req.Title) > 100 {
		errs = append(errs, ValidationError{
			Field:   "title",
			Message: "Len of chat title must be less than 100 characters",
			Code:    "TITLE_TOO_LONG",
		})
	}

	if req.Type == "" {
		errs = append(errs, ValidationError{
			Field:   "type",
			Message: "Type is required",
			Code:    "TYPE_REQUIRED",
		})
	} else {
		validTypes := map[dtoChat.ChatType]bool{
			dtoChat.ChatTypeDialog:  true,
			dtoChat.ChatTypeGroup:   true,
			dtoChat.ChatTypeChannel: true,
		}

		if !validTypes[req.Type] {
			errs = append(errs, ValidationError{
				Field:   "type",
				Message: "Invalid type",
				Code:    "INVALID_TYPE",
			})
		}
	}

	if len(req.MembersID) == 0 {
		errs = append(errs, ValidationError{
			Field:   "members_id",
			Message: "At least one member is required",
			Code:    "MEMBERS_REQUIRED",
		})
	}

	if req.Type == dtoChat.ChatTypeDialog && len(req.MembersID) > 2 {
		errs = append(errs, ValidationError{
			Field:   "members_id",
			Message: "Dialog must have only 2 members",
			Code:    "MUST_HAVE_2_MEMBERS",
		})
	}

	if len(req.MembersID) > 1 {
		memb := make(map[int64]bool)
		for _, id := range req.MembersID {
			if memb[id] {
				errs = append(errs, ValidationError{
					Field:   "members_id",
					Message: "Duplicate users",
					Code:    "USER_DUPLICATE",
				})
				break
			}
			memb[id] = true
		}
	}
	return errs
}

func ValidationContactCreate(req *dtoContact.AddContactRequest) []ValidationError {
	var errs []ValidationError

	if req.FirstName != "" && len(req.FirstName) > 100{
		errs = append(errs, ValidationError{
			Field: "first_name",
			Message: "contact firstname must be less than 100 caracters",
			Code: "CONTACT_FIRST_NAME_MUST_LESS_100_CHARACTERS",
		})
	}

	

	if len(*req.LastName) > 100{
		errs = append(errs, ValidationError{
			Field: "last_name",
			Message: "contact lastname must be less than 100 caracters",
			Code: "CONTACT_LAST_NAME_MUST_LESS_100_CHARACTERS",
		})
	}

	return errs
}