package hash

import "testing"

func TestHashAndCheckPassword_Success(t *testing.T) {
	password := "Passw0rd&"

	hashed, err := HashPassword(password)
	if err != nil {
		t.Fatalf("expected no error hashing password, got %v", err)
	}

	if hashed == "" {
		t.Fatalf("expected non-empty hashed password")
	}

	if !CheckPassword(hashed, password) {
		t.Fatalf("expected password to match hash")
	}
}

func TestCheckPassword_Invalid(t *testing.T) {
	password := "Passw0rd&"
	hashed, err := HashPassword(password)
	if err != nil {
		t.Fatalf("expected no error hashing password, got %v", err)
	}

	if CheckPassword(hashed, "wrongPassword1!") {
		t.Fatalf("expected password not to match hash")
	}
}
