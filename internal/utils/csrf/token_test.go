package csrf

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

type errReader struct{}

func (errReader) Read(_ []byte) (int, error) {
	return 0, errors.New("entropy unavailable")
}

func TestPositiveCSRF_GenerateToken(t *testing.T) {
	type fields struct{}

	type args struct{}

	tests := []struct {
		args    args
		prepare func(*fields)
		name    string
	}{
		{
			name:    "Generates URL-safe opaque token",
			prepare: nil,
			args:    args{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var f fields
			if tt.prepare != nil {
				tt.prepare(&f)
			}

			tok, err := GenerateToken()
			require.NoError(t, err)
			require.NotEmpty(t, tok)
			require.Len(t, tok, base64.RawURLEncoding.EncodedLen(32))
			raw, derr := base64.RawURLEncoding.DecodeString(tok)
			require.NoError(t, derr)
			require.Len(t, raw, 32)

			other, err2 := GenerateToken()
			require.NoError(t, err2)
			require.NotEqual(t, tok, other)
		})
	}
}

func TestNegativeCSRF_GenerateToken(t *testing.T) {
	type fields struct {
		restoreRand func()
	}

	type args struct{}

	tests := []struct {
		name       string
		prepare    func(*fields)
		args       args
		wantSubstr string
		wantAnyErr bool
	}{
		{
			name: "Random reader returns error",
			prepare: func(f *fields) {
				orig := rand.Reader
				rand.Reader = errReader{}
				f.restoreRand = func() { rand.Reader = orig }
			},
			args:       args{},
			wantAnyErr: true,
			wantSubstr: "csrf: read random",
		},
		{
			name: "Insufficient random bytes",
			prepare: func(f *fields) {
				orig := rand.Reader
				rand.Reader = bytes.NewReader([]byte{1, 2, 3, 4, 5})
				f.restoreRand = func() { rand.Reader = orig }
			},
			args:       args{},
			wantAnyErr: true,
			wantSubstr: "csrf: read random",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := fields{}
			if tt.prepare != nil {
				tt.prepare(&f)
			}
			if f.restoreRand != nil {
				defer f.restoreRand()
			}

			_, err := GenerateToken()
			if tt.wantAnyErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.wantSubstr)
			}
		})
	}
}
