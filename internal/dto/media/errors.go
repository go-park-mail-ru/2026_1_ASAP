package media

import "errors"

var (
	ErrEmptyFile       = errors.New("file is empty")
	ErrFileTooLarge    = errors.New("file is too large")
	ErrInvalidFileType = errors.New("file is invalid")
)
