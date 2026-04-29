package profile

import (
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/profile/dto/media"
)

type RequestCreateProfile struct {
	ProfileID int64  `json:"profile_id"`
	FirstName string `json:"first_name"`
}

type RequestSearchIdByLogin struct {
	Login string `json:"login"`
}
type RequestUpdateBio struct {
	Bio *string `json:"bio,omitempty"`
}

type RequestUpdateAvatar struct {
	File *media.FileInput
}

type RequestUpdateAvatarURL struct {
	AvatarURL string `json:"avatar_url"`
}

type RequestUpdateBirthDate struct {
	BirthDate *string `json:"birth_date,omitempty"`
}

type RequestUpdateName struct {
	LastName  *string `json:"last_name,omitempty"`
	FirstName string  `json:"first_name"`
}

type RequestUpdateEmail struct {
	Email string `json:"email"`
}

type RequestUpdateLogin struct {
	Login string `json:"login"`
}
