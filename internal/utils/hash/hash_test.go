package hash

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPositiveHash_HashPassword(t *testing.T) {
	type fields struct{}

	type args struct {
		password string
	}

	tests := []struct {
		name    string
		prepare func(*fields)
		args    args
	}{
		{
			name:    "Produces verifiable Argon2id string",
			prepare: nil,
			args:    args{password: "SecurePass99!"},
		},
		{
			name:    "Unicode password round-trips",
			prepare: nil,
			args:    args{password: "пароль-🔑-99"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var f fields
			if tt.prepare != nil {
				tt.prepare(&f)
			}

			got, err := HashPassword(tt.args.password)
			require.NoError(t, err)
			require.NotEmpty(t, got)
			require.True(t, strings.HasPrefix(got, argonStoredPrefix))
			_, decErr := base64.RawStdEncoding.DecodeString(got[len(argonStoredPrefix):])
			require.NoError(t, decErr)
			require.True(t, CheckPassword(got, tt.args.password), "CheckPassword must accept freshly hashed password")
		})
	}
}

func TestPositiveHash_CheckPassword(t *testing.T) {
	type fields struct{}

	type args struct {
		password    string
		checkSecret string
	}

	tests := []struct {
		name    string
		prepare func(*fields)
		args    args
	}{
		{
			name:    "Correct password matches hash",
			prepare: nil,
			args: args{
				password:    "CorrectHorse99!",
				checkSecret: "CorrectHorse99!",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var f fields
			if tt.prepare != nil {
				tt.prepare(&f)
			}

			stored, err := HashPassword(tt.args.password)
			require.NoError(t, err)
			require.True(t, CheckPassword(stored, tt.args.checkSecret))
		})
	}
}

func TestNegativeHash_CheckPassword(t *testing.T) {
	type fields struct{}

	type args struct {
		stored string
		plain  string
	}

	validStored, err := HashPassword("OnlyForNegativeTable99!")
	require.NoError(t, err)

	tooShortDecoded := make([]byte, 10)
	tooShortEncoded := argonStoredPrefix + base64.RawStdEncoding.EncodeToString(tooShortDecoded)

	tests := []struct {
		name    string
		prepare func(*fields)
		args    args
	}{
		{
			name:    "Wrong password for valid hash",
			prepare: nil,
			args:    args{stored: validStored, plain: "wrong-password"},
		},
		{
			name:    "Missing Argon prefix",
			prepare: nil,
			args:    args{stored: "plain:" + base64.RawStdEncoding.EncodeToString([]byte("not-a-hash")), plain: "x"},
		},
		{
			name:    "Invalid base64 payload",
			prepare: nil,
			args:    args{stored: argonStoredPrefix + "not!!!valid-base64", plain: "x"},
		},
		{
			name:    "Decoded blob too short",
			prepare: nil,
			args:    args{stored: tooShortEncoded, plain: "x"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var f fields
			if tt.prepare != nil {
				tt.prepare(&f)
			}

			require.False(t, CheckPassword(tt.args.stored, tt.args.plain))
		})
	}
}

func TestPositiveHash_HashPasswordDistinctSalts(t *testing.T) {
	type fields struct{}

	type args struct {
		password string
	}

	tests := []struct {
		name    string
		prepare func(*fields)
		args    args
	}{
		{
			name:    "Same password yields different encodings",
			prepare: nil,
			args:    args{password: "SamePassword123!"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var f fields
			if tt.prepare != nil {
				tt.prepare(&f)
			}

			a, errA := HashPassword(tt.args.password)
			require.NoError(t, errA)
			b, errB := HashPassword(tt.args.password)
			require.NoError(t, errB)
			require.NotEqual(t, a, b)
			require.True(t, CheckPassword(a, tt.args.password))
			require.True(t, CheckPassword(b, tt.args.password))
		})
	}
}
