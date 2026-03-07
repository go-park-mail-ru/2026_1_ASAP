package mapper

import (
	dtoApi "github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/api"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/utils/validation"
)

func MapValidationErrorsToApiErrors(errs []validation.ValidationError) []dtoApi.ApiError {
	apiErrors := make([]dtoApi.ApiError, len(errs))
	for i, e := range errs {
		apiErrors[i] = dtoApi.ApiError{
			Code:    e.Code,
			Message: e.Message,
		}
	}
	return apiErrors
}
