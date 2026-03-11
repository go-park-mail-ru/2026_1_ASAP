package auth

type RequestRegistrate struct {
	Login    string `json:"login" example:"user1"`
	Email    string `json:"email" example:"testemail@dev.mail.ru"`
	Password string `json:"password" example:"passWo1d&"`
}

type RequestLogin struct {
	Login    string `json:"login" example:"user1"`
	Password string `json:"password" example:"passWo1d&"`
}

type RequestLogout struct {
	SessionID string `json:"session_id"`
}

type ResponseRegisterSuccess struct {
	Login string `json:"login" example:"user1"`
	Email string `json:"email" example:"testemail@dev.mail.ru"`
}

type ResponseLoginSuccess struct {
	Login string `json:"login" example:"user1"`
}

type ResponseLogoutSuccess struct {
}
