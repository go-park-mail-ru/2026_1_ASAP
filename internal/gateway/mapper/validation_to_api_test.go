package mapper

import (
	"testing"

	dtoApi "github.com/go-park-mail-ru/2026_1_ASAP/internal/gateway/dto/api"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/utils/validation"
)

func TestMapValidationErrorsToApiErrors(t *testing.T) {
	tests := []struct {
		name string
		errs []validation.ValidationError
		want []dtoApi.ApiError
	}{
		{
			name: "maps fields and codes",
			errs: []validation.ValidationError{
				{Field: "email", Message: "invalid", Code: "EMAIL_INVALID"},
				{Field: "password", Message: "too short", Code: "PASSWORD_TOO_SHORT"},
			},
			want: []dtoApi.ApiError{
				{Code: "EMAIL_INVALID", Message: "invalid"},
				{Code: "PASSWORD_TOO_SHORT", Message: "too short"},
			},
		},
		{
			name: "empty input",
			errs: nil,
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apiErrors := MapValidationErrorsToApiErrors(tt.errs)
			if len(apiErrors) != len(tt.want) {
				t.Fatalf("expected %d api errors, got %d", len(tt.want), len(apiErrors))
			}
			for i := range tt.want {
				if apiErrors[i] != tt.want[i] {
					t.Fatalf("at index %d expected %#v, got %#v", i, tt.want[i], apiErrors[i])
				}
			}
		})
	}
}
