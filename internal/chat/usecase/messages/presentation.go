package messages

import (
	dto "github.com/go-park-mail-ru/2026_1_ASAP/internal/chat/dto/message"
	"github.com/go-park-mail-ru/2026_1_ASAP/pkg/present"
)

func formatTextForViewer(raw string, subscriptionActive bool) string {
	return present.TextForViewer(raw, subscriptionActive)
}

func formatPlainTextForViewer(raw string, subscriptionActive bool) string {
	return present.PlainTextForViewer(raw, subscriptionActive)
}

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

func PresentEditMessageForViewer(resp *dto.ResponseEditMessage, subscriptionActive bool) *dto.ResponseEditMessage {
	if resp == nil {
		return nil
	}
	out := *resp
	raw := resp.ContentRaw
	if raw == "" {
		raw = resp.Text
	}
	out.Text = formatTextForViewer(raw, subscriptionActive)
	if resp.LastMessage != nil {
		out.LastMessage = presentLastMessageForViewer(resp.LastMessage, subscriptionActive)
	}
	return &out
}

func PresentClearMessageForViewer(resp *dto.ResponseClearMessage, subscriptionActive bool) *dto.ResponseClearMessage {
	if resp == nil {
		return nil
	}
	out := *resp
	if resp.LastMessage != nil {
		out.LastMessage = presentLastMessageForViewer(resp.LastMessage, subscriptionActive)
	}
	return &out
}

func presentLastMessageForViewer(lm *dto.LastMessageDTO, subscriptionActive bool) *dto.LastMessageDTO {
	if lm == nil {
		return nil
	}
	out := *lm
	raw := lm.ContentRaw
	if raw == "" {
		raw = lm.Text
	}
	out.Text = formatTextForViewer(raw, subscriptionActive)
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
