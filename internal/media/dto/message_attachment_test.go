package dto

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateMessageAttachment_Photo(t *testing.T) {
	t.Parallel()
	in := &FileInput{
		Body:        bytes.NewReader([]byte("x")),
		ContentType: "image/jpeg",
		Size:        1,
	}
	require.NoError(t, in.ValidateMessageAttachment(MessageAttachmentKindPhoto))
	require.ErrorIs(t, in.ValidateMessageAttachment(MessageAttachmentKindVideo), ErrInvalidFileType)
}

func TestValidateMessageAttachment_VideoSize(t *testing.T) {
	t.Parallel()
	in := &FileInput{
		Body:        bytes.NewReader(make([]byte, MaxMessageVideoBytes+1)),
		ContentType: "video/mp4",
		Size:        MaxMessageVideoBytes + 1,
	}
	require.ErrorIs(t, in.ValidateMessageAttachment(MessageAttachmentKindVideo), ErrFileTooLarge)
}

func TestValidateMessageAttachment_FileMIME(t *testing.T) {
	t.Parallel()
	in := &FileInput{
		Body:        bytes.NewReader([]byte("%PDF")),
		ContentType: "application/pdf",
		Size:        4,
	}
	require.NoError(t, in.ValidateMessageAttachment(MessageAttachmentKindFile))
}
