package speechkit

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/go-park-mail-ru/2026_1_ASAP/internal/media/audio"
)

var (
	ErrTranscriptionFailed = errors.New("speech recognition failed")
	ErrNotConfigured       = errors.New("speechkit is not configured")
)

type Config struct {
	APIKey string
	Lang   string
}

type Client struct {
	apiKey       string
	lang         string
	recognizeURL string
	http         *http.Client
}

func NewClient(cfg Config) *Client {
	lang := cfg.Lang
	if lang == "" {
		lang = "ru-RU"
	}
	return &Client{
		apiKey: strings.TrimSpace(cfg.APIKey),
		lang:   lang,
		http: &http.Client{
			Timeout: 90 * time.Second,
		},
	}
}

func (c *Client) Transcribe(ctx context.Context, data []byte, contentType string) (string, error) {
	if c == nil || c.apiKey == "" {
		return "", ErrNotConfigured
	}
	if len(data) == 0 {
		return "", audio.ErrInvalidVoiceAudio
	}

	ogg, err := convertToOggOpus(data, contentType)
	if err != nil {
		return "", fmt.Errorf("convert audio: %w", err)
	}
	return c.recognizeOgg(ctx, ogg)
}

func (c *Client) recognizeOgg(ctx context.Context, ogg []byte) (string, error) {
	endpoint := c.recognizeURL
	if endpoint == "" {
		endpoint = fmt.Sprintf(
			"https://stt.api.cloud.yandex.net/speech/v1/stt:recognize?lang=%s&format=oggopus",
			url.QueryEscape(c.lang),
		)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(ogg))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Api-Key "+c.apiKey)
	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%w: status %d: %s", ErrTranscriptionFailed, resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var parsed RecognizeResponse
	if err := parsed.UnmarshalJSON(body); err != nil {
		return "", fmt.Errorf("%w: invalid response: %w", ErrTranscriptionFailed, err)
	}

	text := strings.TrimSpace(parsed.Result)
	if text == "" {
		return "", ErrTranscriptionFailed
	}
	return text, nil
}

func convertToOggOpus(data []byte, contentType string) ([]byte, error) {
	if !audio.FFmpegAvailable() {
		return nil, fmt.Errorf("ffmpeg not available")
	}

	tmpIn, err := os.CreateTemp("", "voice-in-*"+extensionForContentType(contentType))
	if err != nil {
		return nil, err
	}
	inPath := tmpIn.Name()
	defer func() { _ = os.Remove(inPath) }()

	if _, err = tmpIn.Write(data); err != nil {
		_ = tmpIn.Close()
		return nil, err
	}
	if err = tmpIn.Close(); err != nil {
		return nil, err
	}

	tmpOut, err := os.CreateTemp("", "voice-out-*.ogg")
	if err != nil {
		return nil, err
	}
	outPath := tmpOut.Name()
	_ = tmpOut.Close()
	defer func() { _ = os.Remove(outPath) }()

	cmd := exec.Command("ffmpeg", "-y", "-i", inPath, "-c:a", "libopus", "-b:a", "48k", "-vn", outPath)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err = cmd.Run(); err != nil {
		return nil, fmt.Errorf("ffmpeg: %w: %s", err, strings.TrimSpace(stderr.String()))
	}

	return os.ReadFile(outPath)
}

func extensionForContentType(contentType string) string {
	switch strings.ToLower(contentType) {
	case "audio/webm", "video/webm":
		return ".webm"
	case "audio/ogg":
		return ".ogg"
	case "audio/mp4", "audio/x-m4a":
		return ".m4a"
	case "audio/mpeg":
		return ".mp3"
	default:
		return ".bin"
	}
}
