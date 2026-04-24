package api

import (
	dtoAuth "github.com/go-park-mail-ru/2026_1_ASAP/internal/auth/dto/auth"
	dtoContact "github.com/go-park-mail-ru/2026_1_ASAP/internal/profile/dto/contact"
	dto "github.com/go-park-mail-ru/2026_1_ASAP/internal/profile/dto/profile"
)

// типы ответов для swagger
type ResponseLoginSuccessForSwagger = ApiSuccessResponse[dtoAuth.ResponseLoginSuccess]
type ResponseRegisterSuccessForSwagger = ApiSuccessResponse[dtoAuth.ResponseRegisterSuccess]
type ResponseLogoutSuccessForSwagger = ApiSuccessResponse[dtoAuth.ResponseLogoutSuccess]
type ResponseGetContactsSuccessForSwagger = ApiSuccessResponse[[]dtoContact.ContactResponse]
type ResponseCreateContactSuccessForSwagger = ApiSuccessResponse[*dtoContact.ContactResponse]
type ResponseDeleteContactSuccessForSwagger = ApiSuccessResponse[string]
type ResponseGetProfileSuccessForSwagger = ApiSuccessResponse[*dto.ResponseGetProfile]
type ResponseUpdateProfileForSwagger = ApiSuccessResponse[*dto.ResponseUpdateProfile]
