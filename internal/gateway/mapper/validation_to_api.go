package mapper

import (
	dtoApi "github.com/go-park-mail-ru/2026_1_ASAP/internal/gateway/dto/api"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/utils/validation"
)

func MapValidationErrorsToApiErrors(errs []validation.ValidationError) []dtoApi.ApiError {
	apiErrors := make([]dtoApi.ApiError, len(errs))
	for i, e := range errs {
		apiErrors[i] = dtoApi.ApiError{
			Code:    dtoApi.ErrorCode(e.Code),
			Message: dtoApi.ErrorMessage(e.Message),
		}
	}
	return apiErrors
}
