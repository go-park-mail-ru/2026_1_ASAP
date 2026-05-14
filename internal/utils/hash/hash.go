package hash

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	saltLen       = 16 // в примере 8 байт; 16 — типичный минимум для Argon2
	argonTime     = 1
	argonMemoryKB = 64 * 1024 // 64 MiB (параметр memory в KiB)
	argonThreads  = 4
	argonKeyLen   = 32
)

const argonStoredPrefix = "a2id1:"

func hashPass(salt []byte, plainPassword string) []byte {
	hashedPass := argon2.IDKey([]byte(plainPassword), salt, argonTime, argonMemoryKB, argonThreads, argonKeyLen)
	out := make([]byte, 0, len(salt)+len(hashedPass))
	out = append(out, salt...)
	out = append(out, hashedPass...)
	return out
}

func HashPassword(password string) (string, error) {
	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	blob := hashPass(salt, password)
	return argonStoredPrefix + base64.RawStdEncoding.EncodeToString(blob), nil
}

func CheckPassword(storedPassword, plainPassword string) bool {
	if !strings.HasPrefix(storedPassword, argonStoredPrefix) {
		return false
	}
	raw, err := base64.RawStdEncoding.DecodeString(storedPassword[len(argonStoredPrefix):])
	if err != nil || len(raw) < saltLen+argonKeyLen {
		return false
	}
	salt := raw[:saltLen]
	want := raw[saltLen:]
	got := argon2.IDKey([]byte(plainPassword), salt, argonTime, argonMemoryKB, argonThreads, uint32(len(want)))
	return bytes.Equal(got, want)
}
