package dto

import "strings"

func NormalizeVoiceContentType(contentType string) string {
	ct := strings.ToLower(strings.TrimSpace(contentType))
	if i := strings.Index(ct, ";"); i >= 0 {
		ct = strings.TrimSpace(ct[:i])
	}
	switch ct {
	case "video/webm":
		return "audio/webm"
	default:
		return ct
	}
}

func normalizeVoiceFileInput(f *FileInput) {
	if f == nil {
		return
	}
	f.ContentType = NormalizeVoiceContentType(f.ContentType)
}
