package repository

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	mediadto "github.com/go-park-mail-ru/2026_1_ASAP/internal/media/dto"
)

type fakeCapybaraDetector struct {
	result bool
	err    error
}

func (d fakeCapybaraDetector) DetectBytes(ctx context.Context, data []byte) (bool, error) {
	return d.result, d.err
}

func TestReadInputBody(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   *mediadto.FileInput
		max     int
		want    []byte
		wantErr error
	}{
		{name: "nil input", max: 10, wantErr: mediadto.ErrEmptyFile},
		{name: "nil body", input: &mediadto.FileInput{}, max: 10, wantErr: mediadto.ErrEmptyFile},
		{name: "empty body", input: &mediadto.FileInput{Body: bytes.NewReader(nil)}, max: 10, wantErr: mediadto.ErrEmptyFile},
		{name: "too large", input: &mediadto.FileInput{Body: bytes.NewReader([]byte("hello"))}, max: 4, wantErr: mediadto.ErrFileTooLarge},
		{name: "ok", input: &mediadto.FileInput{Body: bytes.NewReader([]byte("hello"))}, max: 5, want: []byte("hello")},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := readInputBody(tt.input, tt.max)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				require.Nil(t, got)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestValidateMessageObjectKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		key     string
		wantErr bool
	}{
		{key: "message/1/file.png"},
		{key: "/message/1/file.png"},
		{key: " avatar/1.png", wantErr: true},
		{key: "message/1", wantErr: true},
		{key: "message/1/", wantErr: true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.key, func(t *testing.T) {
			t.Parallel()

			err := validateMessageObjectKey(tt.key)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestMaxBytesForKind(t *testing.T) {
	t.Parallel()

	tests := []struct {
		kind mediadto.MessageAttachmentKind
		want int
	}{
		{kind: mediadto.MessageAttachmentKindVideo, want: mediadto.MaxMessageVideoBytes},
		{kind: mediadto.MessageAttachmentKindFile, want: mediadto.MaxMessageFileBytes},
		{kind: mediadto.MessageAttachmentKindVoice, want: mediadto.MaxMessageVoiceBytes},
		{kind: mediadto.MessageAttachmentKindPhoto, want: mediadto.MaxMessagePhotoBytes},
	}

	for _, tt := range tests {
		tt := tt
		t.Run("", func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tt.want, maxBytesForKind(tt.kind))
		})
	}
}

func TestGetExtensionFromContentType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		contentType string
		want        string
	}{
		{contentType: "image/jpeg", want: ".jpg"},
		{contentType: "image/png", want: ".png"},
		{contentType: "image/webp", want: ".webp"},
		{contentType: "image/gif", want: ".gif"},
		{contentType: "video/mp4", want: ".mp4"},
		{contentType: "video/webm", want: ".webm"},
		{contentType: "video/quicktime", want: ".mov"},
		{contentType: "application/pdf", want: ".pdf"},
		{contentType: "application/zip", want: ".zip"},
		{contentType: "application/msword", want: ".doc"},
		{contentType: "application/vnd.openxmlformats-officedocument.wordprocessingml.document", want: ".docx"},
		{contentType: "application/vnd.ms-excel", want: ".xls"},
		{contentType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", want: ".xlsx"},
		{contentType: "text/plain", want: ".txt"},
		{contentType: "audio/ogg", want: ".ogg"},
		{contentType: "audio/mp4", want: ".m4a"},
		{contentType: "audio/mpeg", want: ".mp3"},
		{contentType: "application/octet-stream", want: ".bin"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.contentType, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tt.want, getExtensionFromContentType(tt.contentType))
		})
	}
}

func TestMediaRepository_SetCapybaraDetectorAndClose(t *testing.T) {
	t.Parallel()

	repo := &MediaRepository{}
	detector := fakeCapybaraDetector{result: true, err: errors.New("ignored")}

	repo.SetCapybaraDetector(detector)
	require.NotNil(t, repo.capybaraDetector)
	require.NotPanics(t, repo.Close)
}
