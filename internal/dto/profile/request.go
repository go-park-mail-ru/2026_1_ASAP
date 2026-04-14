package profile

import "github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/media"

type RequestSearchIdByLogin struct {
	Login string `json:"login"`
}
type RequestUpdateBio struct {
	Bio *string `json:"bio,omitempty"`
}

type RequestUpdateAvatar struct {
	File *media.FileInput
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
