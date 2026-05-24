package message

import (
	"testing"

	chatv1 "github.com/go-park-mail-ru/2026_1_ASAP/gen/go/chat/v1"
	"github.com/stretchr/testify/require"
)

func TestParseAttachmentKind_Voice(t *testing.T) {
	t.Parallel()

	kind, err := parseAttachmentKind("voice")
	require.NoError(t, err)
	require.Equal(t, chatv1.MessageAttachmentKind_MESSAGE_ATTACHMENT_KIND_VOICE, kind)
}

func TestMaxBytesForKind_Voice(t *testing.T) {
	t.Parallel()

	require.Equal(t, int64(5*1024*1024), maxBytesForKind(chatv1.MessageAttachmentKind_MESSAGE_ATTACHMENT_KIND_VOICE))
}
