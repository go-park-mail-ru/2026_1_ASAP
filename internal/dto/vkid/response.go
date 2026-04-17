package vkid

type CallbackResponseFromVKID struct {
	RefreshToken string `json:"refresh_token"`
	AccessToken  string `json:"access_token"`
	IDToken      string `json:"id_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	UserID       int64  `json:"user_id"`
	State        string `json:"state"`
	Scope        string `json:"scope"`
}
