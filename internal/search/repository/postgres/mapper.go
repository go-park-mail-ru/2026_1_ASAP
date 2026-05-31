package postgres

import (
	"strings"

	searchdomain "github.com/go-park-mail-ru/2026_1_ASAP/internal/search/domain/search"
	"github.com/go-park-mail-ru/2026_1_ASAP/pkg/null"
)

const maxPreviewRunes = 240

func likePattern(term string) string {
	escaped := strings.ReplaceAll(term, `\`, `\\`)
	escaped = strings.ReplaceAll(escaped, `%`, `\%`)
	escaped = strings.ReplaceAll(escaped, `_`, `\_`)
	return "%" + escaped + "%"
}

func truncatePreview(s string) string {
	runes := []rune(s)
	if len(runes) <= maxPreviewRunes {
		return s
	}
	return string(runes[:maxPreviewRunes]) + "…"
}

func rowToGlobalChannelHit(row *globalChannelSearchRow) searchdomain.GlobalChannelHit {
	h := searchdomain.GlobalChannelHit{
		ChatID:   row.ID,
		Title:    row.Title,
		IsMember: row.IsMember,
	}
	if row.AvatarURL.Valid {
		v := row.AvatarURL.String
		h.AvatarURL = &v
	}
	if row.LastMessagePreview.Valid {
		prev := truncatePreview(row.LastMessagePreview.String)
		h.LastMessagePreview = &prev
	}
	if row.LastMessageAt.Valid {
		t := row.LastMessageAt.Time
		h.LastMessageAt = &t
	}
	return h
}

func rowToChatHit(row *chatSearchRow) searchdomain.ChatHit {
	h := searchdomain.ChatHit{
		ChatID:      row.ID,
		Type:        searchdomain.ChatType(row.Type),
		Title:       row.Title,
		UnreadCount: 0,
	}
	if row.AvatarURL.Valid {
		v := row.AvatarURL.String
		h.AvatarURL = &v
	}
	if row.LastMessagePreview.Valid {
		prev := truncatePreview(row.LastMessagePreview.String)
		h.LastMessagePreview = &prev
	}
	if row.LastMessageAt.Valid {
		t := row.LastMessageAt.Time
		h.LastMessageAt = &t
	}
	return h
}

func rowToContactHit(row *contactSearchRow) searchdomain.ContactHit {
	fn := strings.TrimSpace(row.FName)
	var ln string
	if row.LastName.Valid {
		ln = strings.TrimSpace(row.LastName.String)
	}
	display := strings.TrimSpace(strings.TrimSpace(fn + " " + ln))

	h := searchdomain.ContactHit{
		UserID:      row.UserID,
		DisplayName: display,
	}
	h.Login = null.NullStringToPtrString(row.Login)
	h.AvatarURL = null.NullStringToPtrString(row.Avatar)
	h.LastSeenAt = null.NullTimeToPtrTime(row.LastSeen)
	return h
}

func rowToMessageHit(row *messageSearchRow) searchdomain.MessageHit {
	text := ""
	if row.Content.Valid {
		text = truncatePreview(row.Content.String)
	}
	return searchdomain.MessageHit{
		MessageID:   row.ID,
		ChatID:      row.ChatID,
		SenderID:    row.SenderID,
		TextPreview: text,
		CreatedAt:   row.CreatedAt,
		Highlights:  nil,
	}
}
