package speechkit

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestClient_recognizeOgg_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "Api-Key test-key", r.Header.Get("Authorization"))
		require.Contains(t, r.URL.RawQuery, "lang=ru-RU")
		require.Contains(t, r.URL.RawQuery, "format=oggopus")
		_, _ = w.Write([]byte("привет"))
	}))
	t.Cleanup(srv.Close)

	c := NewClient(Config{APIKey: "test-key", Lang: "ru-RU"})
	c.http = srv.Client()
	c.recognizeURL = srv.URL + "?lang=ru-RU&format=oggopus"

	text, err := c.recognizeOgg(context.Background(), []byte{1, 2, 3})
	require.NoError(t, err)
	require.Equal(t, "привет", text)
}

func TestClient_Transcribe_NotConfigured(t *testing.T) {
	c := NewClient(Config{})
	_, err := c.Transcribe(context.Background(), []byte{1}, "audio/ogg")
	require.ErrorIs(t, err, ErrNotConfigured)
}

func TestClient_recognizeOgg_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("bad request"))
	}))
	t.Cleanup(srv.Close)

	c := NewClient(Config{APIKey: "test-key"})
	c.http = srv.Client()
	c.recognizeURL = srv.URL

	_, err := c.recognizeOgg(context.Background(), []byte{1, 2, 3})
	require.ErrorIs(t, err, ErrTranscriptionFailed)
}
