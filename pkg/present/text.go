package present

import (
	"github.com/go-park-mail-ru/2026_1_ASAP/pkg/filter"
	"github.com/go-park-mail-ru/2026_1_ASAP/pkg/sanitize"
)

// TextForViewer sanitizes message-like HTML and optionally masks profanity.
func TextForViewer(raw string, subscriptionActive bool) string {
	s := sanitize.HTML(raw)
	if subscriptionActive {
		s = filter.MaskProfanity(s)
	}
	return s
}

// TextPtrForViewer is TextForViewer for optional strings.
func TextPtrForViewer(raw *string, subscriptionActive bool) *string {
	if raw == nil {
		return nil
	}
	s := TextForViewer(*raw, subscriptionActive)
	return &s
}

// PlainTextForViewer strips all HTML and optionally masks profanity (titles, names).
func PlainTextForViewer(raw string, subscriptionActive bool) string {
	s := sanitize.Text(raw)
	if subscriptionActive {
		s = filter.MaskProfanity(s)
	}
	return s
}

// PlainTextPtrForViewer is PlainTextForViewer for optional strings.
func PlainTextPtrForViewer(raw *string, subscriptionActive bool) *string {
	if raw == nil {
		return nil
	}
	s := PlainTextForViewer(*raw, subscriptionActive)
	return &s
}
