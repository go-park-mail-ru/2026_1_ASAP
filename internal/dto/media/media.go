package media

import "io"

type FileInput struct {
	Body        io.Reader
	ContentType string
	Size        int64
}
