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
}

type ResponseRegisterSuccess struct {
	Login string `json:"login"`
	Email string `json:"email"`
}
