package api

import (
	dtoAuth "github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/auth"
	dtoChat "github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/chat"
	dto "github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/profile"
)

// типы ответов для swagger
type ResponseLoginSuccessForSwagger = ApiSuccessResponse[dtoAuth.ResponseLoginSuccess]
type ResponseRegisterSuccessForSwagger = ApiSuccessResponse[dtoAuth.ResponseRegisterSuccess]
type ResponseLogoutSuccessForSwagger = ApiSuccessResponse[dtoAuth.ResponseLogoutSuccess]
type ResponseGetChatsSuccessForSwagger = ApiSuccessResponse[[]dtoChat.ChatInformationDTO]
type ResponseCreateChatSuccessForSwagger = ApiSuccessResponse[*dtoChat.ChatInformationDTO]
type ResponseGetChatByIDSuccessForSwagger = ApiSuccessResponse[*dtoChat.ChatInformationDTO]
type ResponseGetProfileSuccessForSwagger = ApiSuccessResponse[*dto.ResponseGetProfile]
type ResponseUpdateProfileForSwagger = ApiSuccessResponse[*dto.ResponseUpdateProfile]
