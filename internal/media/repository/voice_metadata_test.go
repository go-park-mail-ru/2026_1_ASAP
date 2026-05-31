package repository

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEncodeParseVoiceUserMetadata(t *testing.T) {
	t.Parallel()

	meta, err := encodeVoiceUserMetadata(12500, []uint8{1, 2, 3})
	require.NoError(t, err)

	parsed, err := parseVoiceUserMetadata(meta, "audio/webm", 1024)
	require.NoError(t, err)
	require.Equal(t, 12500, parsed.DurationMs)
	require.Equal(t, []uint8{1, 2, 3}, parsed.Waveform)
	require.Equal(t, "audio/webm", parsed.MimeType)
	require.Equal(t, int64(1024), parsed.FileSize)
}

func TestParseVoiceUserMetadata_NotVoice(t *testing.T) {
	t.Parallel()

	_, err := parseVoiceUserMetadata(map[string]string{"attachment-kind": "photo"}, "audio/webm", 1)
	require.Error(t, err)
}
