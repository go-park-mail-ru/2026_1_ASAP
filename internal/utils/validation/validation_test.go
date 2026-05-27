package validation

import (
	"strings"
	"testing"

	dtoAuth "github.com/go-park-mail-ru/2026_1_ASAP/internal/auth/dto/auth"
	dtoContact "github.com/go-park-mail-ru/2026_1_ASAP/internal/profile/dto/contact"
)

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		name  string
		email string
		valid bool
	}{
		{"empty", "", false},
		{"invalid", "invalid-email", false},
		{"valid", "profile@example.com", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := ValidateEmail(tt.email)
			if tt.valid && len(errs) != 0 {
				t.Fatalf("expected no errors, got %v", errs)
			}
			if !tt.valid && len(errs) == 0 {
				t.Fatalf("expected errors, got none")
			}
		})
	}
}

func TestValidateLogin(t *testing.T) {
	tests := []struct {
		name    string
		login   string
		wantErr bool
	}{
		{name: "empty", login: "", wantErr: true},
		{name: "too short ascii", login: "ab", wantErr: true},
		{name: "valid ascii", login: "abc", wantErr: false},
		{name: "too short unicode runes", login: "юя", wantErr: true},
		{name: "valid unicode runes", login: "юзер", wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := ValidateLogin(tt.login)
			if tt.wantErr && len(errs) == 0 {
				t.Fatalf("expected errors, got none")
			}
			if !tt.wantErr && len(errs) != 0 {
				t.Fatalf("expected no errors, got %v", errs)
			}
		})
	}
}

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{name: "empty", password: "", wantErr: true},
		{name: "too short", password: "short", wantErr: true},
		{name: "valid", password: "Passw0rd&", wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := ValidatePassword(tt.password)
			if tt.wantErr && len(errs) == 0 {
				t.Fatalf("expected errors, got none")
			}
			if !tt.wantErr && len(errs) != 0 {
				t.Fatalf("expected no errors, got %v", errs)
			}
		})
	}
}

func TestValidationRequestRegistrate(t *testing.T) {
	tests := []struct {
		name string
		req  *dtoAuth.RequestRegistrate
	}{
		{
			name: "valid",
			req: &dtoAuth.RequestRegistrate{
				Login:    "profile",
				Email:    "profile@example.com",
				Password: "Passw0rd&",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if errs := ValidationRequestRegistrate(tt.req); len(errs) != 0 {
				t.Fatalf("expected no errors, got %v", errs)
			}
		})
	}
}

func TestValidationRequestLogin(t *testing.T) {
	tests := []struct {
		name string
		req  *dtoAuth.RequestLogin
	}{
		{
			name: "valid",
			req: &dtoAuth.RequestLogin{
				Login:    "profile",
				Password: "Passw0rd&",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if errs := ValidationRequestLogin(tt.req); len(errs) != 0 {
				t.Fatalf("expected no errors, got %v", errs)
			}
		})
	}
}

func TestValidationContactCreate(t *testing.T) {
	longName := strings.Repeat("a", 101)
	validLastName := "Smith"

	tests := []struct {
		req       *dtoContact.AddContactRequest
		name      string
		wantCodes []string
	}{
		{
			name: "valid",
			req: &dtoContact.AddContactRequest{
				FirstName:     "Ann",
				LastName:      &validLastName,
				ContactUserID: 10,
			},
		},
		{
			name: "long first name",
			req: &dtoContact.AddContactRequest{
				FirstName:     longName,
				ContactUserID: 10,
			},
			wantCodes: []string{"CONTACT_FIRST_NAME_MUST_LESS_100_CHARACTERS"},
		},
		{
			name: "long last name",
			req: &dtoContact.AddContactRequest{
				LastName:      &longName,
				ContactUserID: 10,
			},
			wantCodes: []string{"CONTACT_LAST_NAME_MUST_LESS_100_CHARACTERS"},
		},
		{
			name: "missing contact user id",
			req: &dtoContact.AddContactRequest{
				ContactUserID: 0,
			},
			wantCodes: []string{"CONTACT_USER_ID_REQUIRED"},
		},
		{
			name: "negative contact user id",
			req: &dtoContact.AddContactRequest{
				ContactUserID: -1,
			},
			wantCodes: []string{"CONTACT_USER_ID_INVALID"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := ValidationContactCreate(tt.req)
			if len(errs) != len(tt.wantCodes) {
				t.Fatalf("expected %d errors, got %v", len(tt.wantCodes), errs)
			}
			for i, code := range tt.wantCodes {
				if errs[i].Code != code {
					t.Fatalf("error code[%d] = %q, want %q", i, errs[i].Code, code)
				}
			}
		})
	}
}
