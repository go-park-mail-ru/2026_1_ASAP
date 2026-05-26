package message

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"testing"

	"github.com/stretchr/testify/require"

	chatv1 "github.com/go-park-mail-ru/2026_1_ASAP/gen/go/chat/v1"
	mediav1 "github.com/go-park-mail-ru/2026_1_ASAP/gen/go/media/v1"
	dtoApi "github.com/go-park-mail-ru/2026_1_ASAP/internal/gateway/dto/api"
	profilemedia "github.com/go-park-mail-ru/2026_1_ASAP/internal/profile/dto/media"
	"github.com/go-park-mail-ru/2026_1_ASAP/pkg/grpcerr"
	"google.golang.org/grpc/codes"
)

func TestParseAttachmentKind(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		raw     string
		want    chatv1.MessageAttachmentKind
		wantErr bool
	}{
		{name: "photo", raw: " photo ", want: chatv1.MessageAttachmentKind_MESSAGE_ATTACHMENT_KIND_PHOTO},
		{name: "video", raw: "VIDEO", want: chatv1.MessageAttachmentKind_MESSAGE_ATTACHMENT_KIND_VIDEO},
		{name: "file", raw: "file", want: chatv1.MessageAttachmentKind_MESSAGE_ATTACHMENT_KIND_FILE},
		{name: "voice", raw: "voice", want: chatv1.MessageAttachmentKind_MESSAGE_ATTACHMENT_KIND_VOICE},
		{name: "invalid", raw: "archive", want: chatv1.MessageAttachmentKind_MESSAGE_ATTACHMENT_KIND_UNSPECIFIED, wantErr: true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := parseAttachmentKind(tt.raw)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, tt.want, got)
		})
	}
}

func TestMaxBytesForKind(t *testing.T) {
	t.Parallel()

	tests := []struct {
		kind chatv1.MessageAttachmentKind
		want int64
	}{
		{kind: chatv1.MessageAttachmentKind_MESSAGE_ATTACHMENT_KIND_VIDEO, want: 50 * 1024 * 1024},
		{kind: chatv1.MessageAttachmentKind_MESSAGE_ATTACHMENT_KIND_FILE, want: 20 * 1024 * 1024},
		{kind: chatv1.MessageAttachmentKind_MESSAGE_ATTACHMENT_KIND_VOICE, want: 5 * 1024 * 1024},
		{kind: chatv1.MessageAttachmentKind_MESSAGE_ATTACHMENT_KIND_PHOTO, want: 10 * 1024 * 1024},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.kind.String(), func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tt.want, maxBytesForKind(tt.kind))
		})
	}
}

func TestBuildProxyURL(t *testing.T) {
	t.Parallel()

	h := NewGatewayMessageHandler(nil, nil, "https://cdn.example.com/")
	require.Equal(t, "https://cdn.example.com/api/v1/messages/attachments/chat/file.png", h.buildProxyURL("/chat/file.png"))
}

func TestReadMultipartFile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		file        multipart.File
		header      *multipart.FileHeader
		maxBytes    int64
		wantContent []byte
		wantType    string
		wantErr     error
	}{
		{
			name:        "from header",
			file:        newMessageMultipartFile([]byte("hello")),
			header:      &multipart.FileHeader{Header: textproto.MIMEHeader{"Content-Type": []string{"text/plain"}}},
			maxBytes:    10,
			wantContent: []byte("hello"),
			wantType:    "text/plain",
		},
		{
			name:        "detects type",
			file:        newMessageMultipartFile([]byte{0x89, 'P', 'N', 'G'}),
			header:      &multipart.FileHeader{},
			maxBytes:    10,
			wantContent: []byte{0x89, 'P', 'N', 'G'},
			wantType:    "text/plain; charset=utf-8",
		},
		{name: "nil file", maxBytes: 10, wantErr: profilemedia.ErrEmptyFile},
		{name: "empty", file: newMessageMultipartFile(nil), maxBytes: 10, wantErr: profilemedia.ErrEmptyFile},
		{name: "too large", file: newMessageMultipartFile([]byte("hello")), maxBytes: 4, wantErr: profilemedia.ErrFileTooLarge},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			content, contentType, err := readMultipartFile(tt.file, tt.header, tt.maxBytes)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.wantContent, content)
			require.Equal(t, tt.wantType, contentType)
		})
	}
}

func TestSendUploadFileError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   dtoApi.ErrorCode
	}{
		{name: "empty", err: profilemedia.ErrEmptyFile, wantStatus: http.StatusBadRequest, wantCode: dtoApi.EmptyFile},
		{name: "too large", err: profilemedia.ErrFileTooLarge, wantStatus: http.StatusBadRequest, wantCode: dtoApi.FileTooLarge},
		{name: "internal", err: io.ErrUnexpectedEOF, wantStatus: http.StatusInternalServerError, wantCode: dtoApi.InternalError},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rr := httptest.NewRecorder()
			sendUploadFileError(rr, tt.err)

			require.Equal(t, tt.wantStatus, rr.Code)
			requireErrorCode(t, rr.Body.Bytes(), tt.wantCode)
		})
	}
}

func TestSendUploadGRPCError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   dtoApi.ErrorCode
	}{
		{
			name:       "too large",
			err:        grpcerr.New(codes.InvalidArgument, int32(mediav1.MediaErrorCode_MEDIA_ERROR_FILE_TOO_LARGE), "too large"),
			wantStatus: http.StatusBadRequest,
			wantCode:   dtoApi.FileTooLarge,
		},
		{
			name:       "invalid",
			err:        grpcerr.New(codes.InvalidArgument, int32(mediav1.MediaErrorCode_MEDIA_ERROR_FILE_INVALID_TYPE), "invalid"),
			wantStatus: http.StatusBadRequest,
			wantCode:   dtoApi.InvalidFileFormat,
		},
		{
			name:       "empty",
			err:        grpcerr.New(codes.InvalidArgument, int32(mediav1.MediaErrorCode_MEDIA_ERROR_FILE_EMPTY), "empty"),
			wantStatus: http.StatusBadRequest,
			wantCode:   dtoApi.EmptyFile,
		},
		{
			name:       "voice too long",
			err:        grpcerr.New(codes.InvalidArgument, int32(mediav1.MediaErrorCode_MEDIA_ERROR_VOICE_TOO_LONG), "long"),
			wantStatus: http.StatusBadRequest,
			wantCode:   dtoApi.VoiceTooLong,
		},
		{
			name:       "internal",
			err:        io.ErrUnexpectedEOF,
			wantStatus: http.StatusInternalServerError,
			wantCode:   dtoApi.InternalError,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rr := httptest.NewRecorder()
			sendUploadGRPCError(rr, tt.err)

			require.Equal(t, tt.wantStatus, rr.Code)
			requireErrorCode(t, rr.Body.Bytes(), tt.wantCode)
		})
	}
}

func TestSendDownloadErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		send       func(http.ResponseWriter, error)
		err        error
		wantStatus int
		wantCode   dtoApi.ErrorCode
	}{
		{
			name:       "chat forbidden",
			send:       sendDownloadGRPCError,
			err:        grpcerr.New(codes.PermissionDenied, int32(chatv1.ChatErrorCode_CHAT_ERROR_NOT_MEMBER), "not member"),
			wantStatus: http.StatusForbidden,
			wantCode:   dtoApi.Unauthorized,
		},
		{
			name:       "chat not found",
			send:       sendDownloadGRPCError,
			err:        io.ErrUnexpectedEOF,
			wantStatus: http.StatusNotFound,
			wantCode:   dtoApi.NotFound,
		},
		{
			name:       "media too large",
			send:       sendDownloadMediaError,
			err:        grpcerr.New(codes.InvalidArgument, int32(mediav1.MediaErrorCode_MEDIA_ERROR_FILE_TOO_LARGE), "too large"),
			wantStatus: http.StatusBadRequest,
			wantCode:   dtoApi.FileTooLarge,
		},
		{
			name:       "media not found",
			send:       sendDownloadMediaError,
			err:        io.ErrUnexpectedEOF,
			wantStatus: http.StatusNotFound,
			wantCode:   dtoApi.NotFound,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rr := httptest.NewRecorder()
			tt.send(rr, tt.err)

			require.Equal(t, tt.wantStatus, rr.Code)
			requireErrorCode(t, rr.Body.Bytes(), tt.wantCode)
		})
	}
}

type messageMultipartFile struct {
	*bytes.Reader
}

func newMessageMultipartFile(data []byte) multipart.File {
	return &messageMultipartFile{Reader: bytes.NewReader(data)}
}

func (f *messageMultipartFile) Close() error {
	return nil
}

func requireErrorCode(t *testing.T, body []byte, want dtoApi.ErrorCode) {
	t.Helper()

	var got dtoApi.ApiErrorResponse
	require.NoError(t, json.Unmarshal(body, &got))
	require.NotEmpty(t, got.Errors)
	require.Equal(t, want, got.Errors[0].Code)
}
