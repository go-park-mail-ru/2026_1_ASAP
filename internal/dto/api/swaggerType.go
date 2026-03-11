package api

import (
	dtoAuth "github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/auth"
	dtoChat "github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/chat"
)

// типы ответов для swagger
type ResponseLoginSuccessForSwagger = ApiSucessResponse[dtoAuth.ResponseLoginSuccess]
type ResponseRegisterSuccessForSwagger = ApiSucessResponse[dtoAuth.ResponseRegisterSuccess]
type ResponseLogoutSuccessForSwagger = ApiSucessResponse[dtoAuth.ResponseLogoutSuccess]
type ResponseGetChatsSuccessForSwagger = ApiSucessResponse[[]dtoChat.ChatInformationDTO]
type ResponseCreateChatSuccessForSwagger = ApiSucessResponse[*dtoChat.ChatInformationDTO]
type ResponseGetChatByIDSuccessForSwagger = ApiSucessResponse[*dtoChat.ChatInformationDTO]
