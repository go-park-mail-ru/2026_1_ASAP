package present

import (
	"github.com/go-park-mail-ru/2026_1_ASAP/pkg/filter"
	"github.com/go-park-mail-ru/2026_1_ASAP/pkg/sanitize"
)

func TextForViewer(raw string, subscriptionActive bool) string {
	s := sanitize.Text(raw)
	if subscriptionActive {
		s = filter.MaskProfanity(s)
	}
	return s
}

func TextPtrForViewer(raw *string, subscriptionActive bool) *string {
	if raw == nil {
		return nil
	}
	s := TextForViewer(*raw, subscriptionActive)
	return &s
}
