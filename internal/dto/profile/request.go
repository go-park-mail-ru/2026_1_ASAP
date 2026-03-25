package profile

import "github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/media"

type RequestUpdateBio struct {
	Bio *string `json:"bio,omitempty"`
}

type RequestUpdateAvatar struct {
	File *media.FileInput
}
