package profile

import "errors"

var (
	ErrNotFound               = errors.New("profile not found")
	ErrEmptyBio               = errors.New("profile bio cannot be empty")
	ErrEmptyAvatar            = errors.New("profile avatar cannot be empty")
	ErrAvatarTooLarge         = errors.New("profile avatar is too large")
	ErrInvalidAvatarType      = errors.New("invalid avatar type")
	ErrInvalidBirthDate       = errors.New("invalid birth date")
	ErrInvalidBirthDateFormat = errors.New("invalid birth date format")
	ErrEmptyFirstName         = errors.New("profile first name cannot be empty")
)
