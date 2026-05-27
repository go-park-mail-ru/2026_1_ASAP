package profile

import "github.com/go-park-mail-ru/2026_1_ASAP/internal/profile/dto/media"

type RequestUpdateAvatar struct {
	File *media.FileInput
}
