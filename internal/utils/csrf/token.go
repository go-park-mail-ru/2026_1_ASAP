package csrf

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
)

func GenerateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return "", fmt.Errorf("csrf: read random: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
