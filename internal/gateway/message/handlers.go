package message

import (
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	chatv1 "github.com/go-park-mail-ru/2026_1_ASAP/gen/go/chat/v1"
	mediav1 "github.com/go-park-mail-ru/2026_1_ASAP/gen/go/media/v1"
	chatmedia "github.com/go-park-mail-ru/2026_1_ASAP/internal/chat/dto/media"
	dtoApi "github.com/go-park-mail-ru/2026_1_ASAP/internal/gateway/dto/api"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/gateway/middleware"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/profile/dto/media"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/utils/response"
	"github.com/go-park-mail-ru/2026_1_ASAP/pkg/grpcerr"
)

type GatewayMessageHandler struct {
	ChatService   chatv1.ChatClient
	MediaService  mediav1.MediaClient
	PublicBaseURL string
}

func NewGatewayMessageHandler(chat chatv1.ChatClient, media mediav1.MediaClient, publicBaseURL string) *GatewayMessageHandler {
	return &GatewayMessageHandler{
		ChatService:   chat,
		MediaService:  media,
		PublicBaseURL: strings.TrimRight(publicBaseURL, "/"),
	}
}

type UploadAttachmentResponse struct {
	AttachmentURL string  `json:"attachment_url"`
	ObjectKey     string  `json:"object_key,omitempty"`
	MimeType      string  `json:"mime_type"`
	FileSize      int64   `json:"file_size"`
	FileName      *string `json:"file_name,omitempty"`
}

func (h *GatewayMessageHandler) UploadAttachment(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	uid, ok := r.Context().Value(middleware.UserID).(int64)
	if !ok {
		response.Send(w, http.StatusUnauthorized, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{Code: dtoApi.Unauthorized, Message: dtoApi.UnauthorizedMsg}},
		})
		return
	}

	kind, err := parseAttachmentKind(r.URL.Query().Get("type"))
	if err != nil {
		response.Send(w, http.StatusBadRequest, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{Code: dtoApi.InvalidFileFormat, Message: dtoApi.InvalidFileFormatMsg}},
		})
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		response.Send(w, http.StatusBadRequest, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{Code: dtoApi.FileNotFound, Message: dtoApi.FileNotFoundMsg}},
		})
		return
	}

	maxBytes := maxBytesForKind(kind)
	content, contentType, err := readMultipartFile(file, header, maxBytes)
	if err != nil {
		sendUploadFileError(w, err)
		return
	}

	req := &chatv1.RequestUploadMessageAttachment{
		UserId:  uid,
		Kind:    kind,
		Content: content,
		Type:    contentType,
	}
	if header != nil && header.Filename != "" {
		name := header.Filename
		req.FileName = &name
	}

	resp, err := h.ChatService.UploadMessageAttachment(ctx, req)
	if err != nil {
		sendUploadGRPCError(w, err)
		return
	}

	attachmentURL := resp.GetAttachmentUrl()
	if attachmentURL == "" && resp.GetObjectKey() != "" {
		attachmentURL = h.buildProxyURL(resp.GetObjectKey())
	}

	body := UploadAttachmentResponse{
		AttachmentURL: attachmentURL,
		ObjectKey:     resp.GetObjectKey(),
		MimeType:      resp.GetMimeType(),
		FileSize:      resp.GetFileSize(),
	}
	if name := resp.GetFileName(); name != "" {
		body.FileName = &name
	}

	response.Send(w, http.StatusOK, dtoApi.ApiSuccessResponse[UploadAttachmentResponse]{
		Status: dtoApi.Success,
		Body:   body,
	})
}

func (h *GatewayMessageHandler) DownloadAttachment(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	uid, ok := r.Context().Value(middleware.UserID).(int64)
	if !ok {
		response.Send(w, http.StatusUnauthorized, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{Code: dtoApi.Unauthorized, Message: dtoApi.UnauthorizedMsg}},
		})
		return
	}

	objectKey := strings.TrimPrefix(chi.URLParam(r, "*"), "/")
	if objectKey == "" || strings.Contains(objectKey, "..") {
		response.Send(w, http.StatusBadRequest, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{Code: dtoApi.InvalidID, Message: dtoApi.InvalidIDMsg}},
		})
		return
	}

	_, err := h.ChatService.AuthorizeMessageAttachment(ctx, &chatv1.RequestAuthorizeMessageAttachment{
		UserId:    uid,
		ObjectKey: objectKey,
	})
	if err != nil {
		sendDownloadGRPCError(w, err)
		return
	}

	mediaResp, err := h.MediaService.GetMessageAttachment(ctx, &mediav1.RequestGetMessageAttachment{
		ObjectKey: objectKey,
	})
	if err != nil {
		sendDownloadMediaError(w, err)
		return
	}

	if ct := mediaResp.GetContentType(); ct != "" {
		w.Header().Set("Content-Type", ct)
	}
	w.Header().Set("Content-Length", strconv.FormatInt(mediaResp.GetSize(), 10))
	w.Header().Set("Cache-Control", "private, max-age=3600")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(mediaResp.GetContent())
}

func (h *GatewayMessageHandler) buildProxyURL(objectKey string) string {
	key := strings.TrimPrefix(objectKey, "/")
	return h.PublicBaseURL + "/api/v1/messages/attachments/" + key
}

func parseAttachmentKind(raw string) (chatv1.MessageAttachmentKind, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "photo":
		return chatv1.MessageAttachmentKind_MESSAGE_ATTACHMENT_KIND_PHOTO, nil
	case "video":
		return chatv1.MessageAttachmentKind_MESSAGE_ATTACHMENT_KIND_VIDEO, nil
	case "file":
		return chatv1.MessageAttachmentKind_MESSAGE_ATTACHMENT_KIND_FILE, nil
	default:
		return chatv1.MessageAttachmentKind_MESSAGE_ATTACHMENT_KIND_UNSPECIFIED, errors.New("invalid kind")
	}
}

func maxBytesForKind(kind chatv1.MessageAttachmentKind) int64 {
	switch kind {
	case chatv1.MessageAttachmentKind_MESSAGE_ATTACHMENT_KIND_VIDEO:
		return 50 * 1024 * 1024
	case chatv1.MessageAttachmentKind_MESSAGE_ATTACHMENT_KIND_FILE:
		return 20 * 1024 * 1024
	default:
		return 10 * 1024 * 1024
	}
}

func readMultipartFile(file multipart.File, header *multipart.FileHeader, maxBytes int64) ([]byte, string, error) {
	if file == nil {
		return nil, "", media.ErrEmptyFile
	}
	defer func() { _ = file.Close() }()

	data, err := io.ReadAll(io.LimitReader(file, maxBytes+1))
	if err != nil {
		return nil, "", err
	}
	if len(data) == 0 {
		return nil, "", media.ErrEmptyFile
	}
	if int64(len(data)) > maxBytes {
		return nil, "", media.ErrFileTooLarge
	}
	ct := ""
	if header != nil {
		ct = header.Header.Get("Content-Type")
	}
	return data, chatmedia.NewFileInput(data, ct).ContentType, nil
}

func sendUploadFileError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, media.ErrEmptyFile):
		response.Send(w, http.StatusBadRequest, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{Code: dtoApi.EmptyFile, Message: dtoApi.EmptyFileMsg}},
		})
	case errors.Is(err, media.ErrFileTooLarge):
		response.Send(w, http.StatusBadRequest, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{Code: dtoApi.FileTooLarge, Message: dtoApi.FileTooLargeMsg}},
		})
	default:
		response.Send(w, http.StatusInternalServerError, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{Code: dtoApi.InternalError, Message: dtoApi.InternalErrorMsg}},
		})
	}
}

func sendUploadGRPCError(w http.ResponseWriter, err error) {
	_, appCode, _ := grpcerr.Error(err)
	switch mediav1.MediaErrorCode(appCode) {
	case mediav1.MediaErrorCode_MEDIA_ERROR_FILE_TOO_LARGE:
		response.Send(w, http.StatusBadRequest, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{Code: dtoApi.FileTooLarge, Message: dtoApi.FileTooLargeMsg}},
		})
	case mediav1.MediaErrorCode_MEDIA_ERROR_FILE_INVALID_TYPE:
		response.Send(w, http.StatusBadRequest, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{Code: dtoApi.InvalidFileFormat, Message: dtoApi.InvalidFileFormatMsg}},
		})
	case mediav1.MediaErrorCode_MEDIA_ERROR_FILE_EMPTY:
		response.Send(w, http.StatusBadRequest, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{Code: dtoApi.EmptyFile, Message: dtoApi.EmptyFileMsg}},
		})
	default:
		response.Send(w, http.StatusInternalServerError, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{Code: dtoApi.InternalError, Message: dtoApi.InternalErrorMsg}},
		})
	}
}

func sendDownloadGRPCError(w http.ResponseWriter, err error) {
	_, appCode, _ := grpcerr.Error(err)
	switch chatv1.ChatErrorCode(appCode) {
	case chatv1.ChatErrorCode_CHAT_ERROR_NOT_MEMBER, chatv1.ChatErrorCode_CHAT_ERROR_INVALID_ATTACHMENT:
		response.Send(w, http.StatusForbidden, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{Code: dtoApi.Unauthorized, Message: dtoApi.UnauthorizedMsg}},
		})
	default:
		response.Send(w, http.StatusNotFound, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{Code: dtoApi.NotFound, Message: dtoApi.NotFoundMsg}},
		})
	}
}

func sendDownloadMediaError(w http.ResponseWriter, err error) {
	_, appCode, _ := grpcerr.Error(err)
	switch mediav1.MediaErrorCode(appCode) {
	case mediav1.MediaErrorCode_MEDIA_ERROR_FILE_TOO_LARGE:
		response.Send(w, http.StatusBadRequest, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{Code: dtoApi.FileTooLarge, Message: dtoApi.FileTooLargeMsg}},
		})
	default:
		response.Send(w, http.StatusNotFound, dtoApi.ApiErrorResponse{
			Status: dtoApi.Error,
			Errors: []dtoApi.ApiError{{Code: dtoApi.NotFound, Message: dtoApi.NotFoundMsg}},
		})
	}
}
