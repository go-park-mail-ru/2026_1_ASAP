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

func TestValidateMessageAttachment_VoiceMIME(t *testing.T) {
	t.Parallel()
	in := &FileInput{
		Body:        bytes.NewReader([]byte("voice")),
		ContentType: "audio/webm",
		Size:        5,
	}
	require.NoError(t, in.ValidateMessageAttachment(MessageAttachmentKindVoice))
	require.ErrorIs(t, in.ValidateMessageAttachment(MessageAttachmentKindPhoto), ErrInvalidFileType)
}

func TestValidateMessageAttachment_VoiceMIME_VideoWebM(t *testing.T) {
	t.Parallel()
	in := &FileInput{
		Body:        bytes.NewReader([]byte("voice")),
		ContentType: "video/webm",
		Size:        5,
	}
	require.NoError(t, in.ValidateMessageAttachment(MessageAttachmentKindVoice))
	require.Equal(t, "audio/webm", in.ContentType)
}

func TestValidateMessageAttachment_VoiceMIME_CodecsParam(t *testing.T) {
	t.Parallel()
	in := &FileInput{
		Body:        bytes.NewReader([]byte("voice")),
		ContentType: "audio/webm;codecs=opus",
		Size:        5,
	}
	require.NoError(t, in.ValidateMessageAttachment(MessageAttachmentKindVoice))
	require.Equal(t, "audio/webm", in.ContentType)
}

func TestNormalizeVoiceContentType(t *testing.T) {
	t.Parallel()
	require.Equal(t, "audio/webm", NormalizeVoiceContentType("video/webm"))
	require.Equal(t, "audio/webm", NormalizeVoiceContentType("audio/webm;codecs=opus"))
	require.Equal(t, "audio/ogg", NormalizeVoiceContentType("audio/ogg;codecs=opus"))
}
