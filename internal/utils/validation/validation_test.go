package validation

import (
	"testing"

	"github.com/google/uuid"

	dtoAuth "github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/auth"
	dtoChat "github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/chat"
)

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		name  string
		email string
		valid bool
	}{
		{"empty", "", false},
		{"invalid", "invalid-email", false},
		{"valid", "user@example.com", true},
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
	if errs := ValidateLogin(""); len(errs) == 0 {
		t.Fatalf("expected error for empty login")
	}

	if errs := ValidateLogin("ab"); len(errs) == 0 {
		t.Fatalf("expected error for short login")
	}

	if errs := ValidateLogin("abc"); len(errs) != 0 {
		t.Fatalf("expected no errors for valid login, got %v", errs)
	}
}

func TestValidatePassword(t *testing.T) {
	if errs := ValidatePassword(""); len(errs) == 0 {
		t.Fatalf("expected error for empty password")
	}

	if errs := ValidatePassword("short"); len(errs) == 0 {
		t.Fatalf("expected error for too short password")
	}

	validPassword := "Passw0rd&"
	if errs := ValidatePassword(validPassword); len(errs) != 0 {
		t.Fatalf("expected no errors for valid password, got %v", errs)
	}
}

func TestValidationRequestRegistrate(t *testing.T) {
	req := &dtoAuth.RequestRegistrate{
		Login:    "user",
		Email:    "user@example.com",
		Password: "Passw0rd&",
	}

	if errs := ValidationRequestRegistrate(req); len(errs) != 0 {
		t.Fatalf("expected no errors, got %v", errs)
	}
}

func TestValidationRequestLogin(t *testing.T) {
	req := &dtoAuth.RequestLogin{
		Login:    "user",
		Password: "Passw0rd&",
	}

	if errs := ValidationRequestLogin(req); len(errs) != 0 {
		t.Fatalf("expected no errors, got %v", errs)
	}
}

func TestValidationChatCreate(t *testing.T) {
	user1 := uuid.New()
	user2 := uuid.New()

	req := &dtoChat.ChatCreate{
		Title:     "Test chat",
		Type:      dtoChat.ChatTypeDialog,
		MembersID: []uuid.UUID{user1, user2},
	}

	if errs := ValidationChatCreate(req); len(errs) != 0 {
		t.Fatalf("expected no errors, got %v", errs)
	}
}
