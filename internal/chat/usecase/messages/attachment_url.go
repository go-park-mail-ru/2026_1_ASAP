package messages

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

const AttachmentProxyPathPrefix = "/api/v1/messages/attachments/"

func BuildAttachmentProxyURL(publicBaseURL, objectKey string) string {
	base := strings.TrimRight(publicBaseURL, "/")
	key := strings.TrimPrefix(objectKey, "/")
	segments := strings.Split(key, "/")
	for i, seg := range segments {
		segments[i] = url.PathEscape(seg)
	}
	return base + AttachmentProxyPathPrefix + strings.Join(segments, "/")
}

func ObjectKeyFromAttachmentURL(raw string) (objectKey string, ownerUserID int64, ok bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", 0, false
	}

	if idx := strings.Index(raw, AttachmentProxyPathPrefix); idx >= 0 {
		pathPart := raw[idx+len(AttachmentProxyPathPrefix):]
		if q := strings.IndexByte(pathPart, '?'); q >= 0 {
			pathPart = pathPart[:q]
		}
		key, err := url.PathUnescape(pathPart)
		if err != nil {
			key = pathPart
		}
		return parseMessageObjectKey(key)
	}

	if u, err := url.Parse(raw); err == nil && u.Path != "" {
		path := strings.TrimPrefix(u.Path, "/")
		if idx := strings.Index(path, "message/"); idx >= 0 {
			return parseMessageObjectKey(path[idx:])
		}
	}

	if strings.HasPrefix(raw, "message/") {
		return parseMessageObjectKey(raw)
	}

	return "", 0, false
}

func parseMessageObjectKey(key string) (objectKey string, ownerUserID int64, ok bool) {
	key = strings.TrimPrefix(strings.TrimSpace(key), "/")
	if !strings.HasPrefix(key, "message/") {
		return "", 0, false
	}
	parts := strings.SplitN(key, "/", 3)
	if len(parts) < 3 {
		return "", 0, false
	}
	uid, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil || uid <= 0 {
		return "", 0, false
	}
	return key, uid, true
}

func IsAttachmentOwnedByUser(attachmentURL string, userID int64) bool {
	_, owner, ok := ObjectKeyFromAttachmentURL(attachmentURL)
	return ok && owner == userID
}

func ValidateMessageObjectKey(objectKey string) error {
	key := strings.TrimPrefix(strings.TrimSpace(objectKey), "/")
	if !strings.HasPrefix(key, "message/") {
		return fmt.Errorf("invalid object key prefix")
	}
	parts := strings.Split(key, "/")
	if len(parts) != 3 || parts[2] == "" {
		return fmt.Errorf("invalid object key format")
	}
	if _, err := strconv.ParseInt(parts[1], 10, 64); err != nil {
		return fmt.Errorf("invalid owner in object key")
	}
	return nil
}
