package auth

type ResponseRegisterSuccess struct {
	Login string `json:"login" example:"user1"`
	Email string `json:"email" example:"testemail@dev.mail.ru"`
}

type ResponseLoginSuccess struct {
	Login string `json:"login" example:"user1"`
}

type ResponseLogoutSuccess struct {
}
