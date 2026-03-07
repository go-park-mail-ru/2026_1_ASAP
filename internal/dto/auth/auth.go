package auth

type RequestRegistrate struct {
	Login    string `json:"login"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type RequestLogin struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type RequestLogout struct {
	SessionID string `json:"session_id"`
}

type ResponseRegisterSuccess struct {
	Login string `json:"login"`
	Email string `json:"email"`
}

type ResponseLoginSuccess struct {
	Login string `json:"login"`
}

type ResponseLogoutSuccess struct {
}
