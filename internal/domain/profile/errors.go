package profile

import "errors"

var (
	ErrNotFound          = errors.New("profile not found")
	ErrEmptyBio          = errors.New("profile bio cannot be empty")
	ErrEmptyAvatar       = errors.New("profile avatar cannot be empty")
	ErrAvatarTooLarge    = errors.New("profile avatar is too large")
	ErrInvalidAvatarType = errors.New("invalid avatar type")
)
