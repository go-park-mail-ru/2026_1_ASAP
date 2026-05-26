package messages

import (
	dto "github.com/go-park-mail-ru/2026_1_ASAP/internal/chat/dto/message"
	"github.com/go-park-mail-ru/2026_1_ASAP/pkg/filter"
	"github.com/go-park-mail-ru/2026_1_ASAP/pkg/sanitize"
)

func formatTextForViewer(raw string, subscriptionActive bool) string {
	s := sanitize.Text(raw)
	if subscriptionActive {
		s = filter.MaskProfanity(s)
	}
	return s
}

// PresentSendMessageForViewer applies subscription-specific presentation to an outgoing message payload.
func PresentSendMessageForViewer(resp *dto.ResponseSendMessage, subscriptionActive bool) *dto.ResponseSendMessage {
	if resp == nil {
		return nil
	}
	raw := resp.ContentRaw
	if raw == "" {
		raw = resp.Text
	}
	out := *resp
	out.Text = formatTextForViewer(raw, subscriptionActive)
	if len(resp.Attachments) > 0 {
		out.Attachments = applyAttachmentBlurForViewer(resp.Attachments, subscriptionActive)
	}
	return &out
}

func applyAttachmentBlurForViewer(attachments []dto.MessageAttachmentDTO, subscriptionActive bool) []dto.MessageAttachmentDTO {
	if len(attachments) == 0 {
		return nil
	}
	out := make([]dto.MessageAttachmentDTO, len(attachments))
	for i, a := range attachments {
		out[i] = a
		if out[i].Type == "photo" {
			out[i].IsBlur = subscriptionActive && out[i].IsCapybara
		}
	}
	return out
}
