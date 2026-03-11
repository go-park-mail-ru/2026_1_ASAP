package mapper

import (
	"testing"

	dtoApi "github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/api"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/utils/validation"
)

func TestMapValidationErrorsToApiErrors(t *testing.T) {
	errs := []validation.ValidationError{
		{Field: "email", Message: "invalid", Code: "EMAIL_INVALID"},
		{Field: "password", Message: "too short", Code: "PASSWORD_TOO_SHORT"},
	}

	apiErrors := MapValidationErrorsToApiErrors(errs)

	if len(apiErrors) != len(errs) {
		t.Fatalf("expected %d api errors, got %d", len(errs), len(apiErrors))
	}

	expected := []dtoApi.ApiError{
		{Code: "EMAIL_INVALID", Message: "invalid"},
		{Code: "PASSWORD_TOO_SHORT", Message: "too short"},
	}

	for i := range expected {
		if apiErrors[i] != expected[i] {
			t.Fatalf("at index %d expected %#v, got %#v", i, expected[i], apiErrors[i])
		}
	}
}
