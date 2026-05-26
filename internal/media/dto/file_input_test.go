package dto

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/textproto"
	"testing"

	"github.com/stretchr/testify/require"
)

type testMultipartFile struct {
	*bytes.Reader
	closeErr error
}

func (f *testMultipartFile) Close() error {
	return f.closeErr
}

func newTestMultipartFile(data []byte) multipart.File {
	return &testMultipartFile{Reader: bytes.NewReader(data)}
}

func TestFileInputFromMultipart(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		file        multipart.File
		header      *multipart.FileHeader
		wantType    string
		wantSize    int64
		wantErr     error
		wantContent []byte
	}{
		{
			name: "content type from header",
			file: newTestMultipartFile([]byte{0x89, 'P', 'N', 'G'}),
			header: &multipart.FileHeader{Header: textproto.MIMEHeader{
				"Content-Type": []string{"image/png"},
			}},
			wantType:    "image/png",
			wantSize:    4,
			wantContent: []byte{0x89, 'P', 'N', 'G'},
		},
		{
			name:        "detects content type",
			file:        newTestMultipartFile([]byte("plain text")),
			header:      &multipart.FileHeader{Header: textproto.MIMEHeader{}},
			wantType:    "text/plain; charset=utf-8",
			wantSize:    10,
			wantContent: []byte("plain text"),
		},
		{
			name:    "nil file",
			header:  &multipart.FileHeader{},
			wantErr: ErrEmptyFile,
		},
		{
			name:    "nil header",
			file:    newTestMultipartFile([]byte("x")),
			wantErr: ErrEmptyFile,
		},
		{
			name:    "empty file",
			file:    newTestMultipartFile(nil),
			header:  &multipart.FileHeader{},
			wantErr: ErrEmptyFile,
		},
		{
			name:    "too large",
			file:    newTestMultipartFile(bytes.Repeat([]byte("x"), MaxAvatarBytes+1)),
			header:  &multipart.FileHeader{},
			wantErr: ErrFileTooLarge,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := FileInputFromMultipart(tt.file, tt.header)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				require.Nil(t, got)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.wantType, got.ContentType)
			require.Equal(t, tt.wantSize, got.Size)
			body, err := io.ReadAll(got.Body)
			require.NoError(t, err)
			require.Equal(t, tt.wantContent, body)
		})
	}
}

func TestFileInputValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   *FileInput
		wantErr error
	}{
		{
			name:  "valid",
			input: &FileInput{Body: bytes.NewReader([]byte("x")), ContentType: "image/webp", Size: 1},
		},
		{
			name:    "nil input",
			wantErr: ErrEmptyFile,
		},
		{
			name:    "nil body",
			input:   &FileInput{ContentType: "image/png", Size: 1},
			wantErr: ErrEmptyFile,
		},
		{
			name:    "invalid type",
			input:   &FileInput{Body: bytes.NewReader([]byte("x")), ContentType: "text/plain", Size: 1},
			wantErr: ErrInvalidFileType,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.input.Validate()
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestValidateAvatarFile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		contentType string
		size        int
		wantErr     error
	}{
		{name: "jpeg", contentType: "image/jpeg", size: 1},
		{name: "empty", contentType: "image/jpeg", size: 0, wantErr: ErrEmptyFile},
		{name: "too large", contentType: "image/png", size: MaxAvatarBytes + 1, wantErr: ErrFileTooLarge},
		{name: "bad type", contentType: "application/pdf", size: 1, wantErr: ErrInvalidFileType},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := ValidateAvatarFile(tt.contentType, tt.size)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}
}
