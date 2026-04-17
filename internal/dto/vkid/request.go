package vkid

type RequestVKID struct {
	Code         string `json:"code"`
	State        string `json:"state"`
	CodeVerifier string `json:"code_verifier"`
	DeviceID     string `json:"device_id"`
}

type RequestAuth struct {
	VKUserID  int64  `json:"user_id"`
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	AvatarURL string `json:"avatar"`
}
