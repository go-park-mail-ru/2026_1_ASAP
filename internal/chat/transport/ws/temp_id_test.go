package ws

import (
	"testing"

	"github.com/stretchr/testify/require"

	dto "github.com/go-park-mail-ru/2026_1_ASAP/internal/chat/dto/message"
)

func TestStripTempIDForViewer(t *testing.T) {
	t.Run("sender keeps temp_id", func(t *testing.T) {
		presented := &dto.ResponseSendMessage{TempID: "client-abc-123", SenderID: 10}
		stripTempIDForViewer(presented, 10, 10)
		require.Equal(t, "client-abc-123", presented.TempID)
	})

	t.Run("non-sender clears temp_id", func(t *testing.T) {
		presented := &dto.ResponseSendMessage{TempID: "client-abc-123", SenderID: 10}
		stripTempIDForViewer(presented, 20, 10)
		require.Empty(t, presented.TempID)
	})

	t.Run("nil presented is no-op", func(t *testing.T) {
		require.NotPanics(t, func() { stripTempIDForViewer(nil, 10, 10) })
	})
}
