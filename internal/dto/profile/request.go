package profile

import "github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/media"

type RequestGetProfile struct {
	UserID int64
}

type RequestUpdateBio struct {
	UserID int64
	Bio    *string `json:"bio,omitempty"`
}

type RequestUpdateAvatar struct {
	UserID int64
	File   *media.FileInput
}
