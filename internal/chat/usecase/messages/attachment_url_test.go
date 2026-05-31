package messages

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestObjectKeyFromAttachmentURL_Proxy(t *testing.T) {
	t.Parallel()
	key, owner, ok := ObjectKeyFromAttachmentURL("https://pulseapp.space/api/v1/messages/attachments/message/10/abc.jpg")
	require.True(t, ok)
	require.Equal(t, int64(10), owner)
	require.Equal(t, "message/10/abc.jpg", key)
}

func TestObjectKeyFromAttachmentURL_S3Legacy(t *testing.T) {
	t.Parallel()
	key, owner, ok := ObjectKeyFromAttachmentURL("https://pulseapp.space/media/message/10/abc.jpg")
	require.True(t, ok)
	require.Equal(t, int64(10), owner)
	require.Equal(t, "message/10/abc.jpg", key)
}

func TestBuildAttachmentProxyURL(t *testing.T) {
	t.Parallel()
	got := BuildAttachmentProxyURL("https://pulseapp.space", "message/10/abc.jpg")
	require.Equal(t, "https://pulseapp.space/api/v1/messages/attachments/message/10/abc.jpg", got)
}
