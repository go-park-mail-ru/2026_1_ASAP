package api

import (
	dtoAuth "github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/auth"
	dtoChat "github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/chat"
	dtoContact "github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/contacts"
)

// типы ответов для swagger
type ResponseLoginSuccessForSwagger = ApiSuccessResponse[dtoAuth.ResponseLoginSuccess]
type ResponseRegisterSuccessForSwagger = ApiSuccessResponse[dtoAuth.ResponseRegisterSuccess]
type ResponseLogoutSuccessForSwagger = ApiSuccessResponse[dtoAuth.ResponseLogoutSuccess]
type ResponseGetChatsSuccessForSwagger = ApiSuccessResponse[[]dtoChat.ChatInformationDTO]
type ResponseCreateChatSuccessForSwagger = ApiSuccessResponse[*dtoChat.ChatInformationDTO]
type ResponseGetChatByIDSuccessForSwagger = ApiSuccessResponse[*dtoChat.ChatInformationDTO]
type ResponseGetContactsSuccessForSwagger = ApiSuccessResponse[[]dtoContact.ContactResponse]
type ResponseCreateContactSuccessForSwagger = ApiSuccessResponse[*dtoContact.ContactResponse]
type ResponseDeleteContactSuccessForSwagger = ApiSuccessResponse[string]
