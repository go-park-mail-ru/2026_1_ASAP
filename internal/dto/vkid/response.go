package vkid

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
)

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

type publicInfoEnvelope struct {
	User publicInfoUserFields `json:"user"`
}

type publicInfoUserFields struct {
	UserIDRaw json.RawMessage `json:"user_id"`
	Email     string          `json:"email"`
	FirstName string          `json:"first_name"`
	LastName  string          `json:"last_name"`
	Avatar    string          `json:"avatar"`
}

func (u publicInfoUserFields) hasAny() bool {
	return len(u.UserIDRaw) > 0 || u.Email != "" || u.FirstName != "" ||
		u.LastName != "" || u.Avatar != ""
}

func (e publicInfoEnvelope) toRequestAuth() (*RequestAuth, error) {
	u := e.User
	vkID, err := parseFlexibleInt64(u.UserIDRaw)
	if err != nil {
		return nil, fmt.Errorf("user_id: %w", err)
	}
	return &RequestAuth{
		VKUserID:  vkID,
		Email:     u.Email,
		FirstName: u.FirstName,
		LastName:  u.LastName,
		AvatarURL: u.Avatar,
	}, nil
}

// RequestAuthFromPublicInfoJSON заполняет RequestAuth из JSON ответа public_info (вложенный user или плоский JSON).
func RequestAuthFromPublicInfoJSON(body []byte) (*RequestAuth, error) {
	var env publicInfoEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, err
	}
	if env.User.hasAny() {
		return env.toRequestAuth()
	}

	var flat RequestAuth
	if err := json.Unmarshal(body, &flat); err != nil {
		return nil, err
	}
	if flat.VKUserID == 0 && flat.Email == "" && flat.FirstName == "" {
		return nil, errors.New("public_info: empty or unrecognized user payload")
	}
	return &flat, nil
}

func parseFlexibleInt64(raw json.RawMessage) (int64, error) {
	if len(raw) == 0 {
		return 0, errors.New("missing user_id")
	}
	var n json.Number
	if err := json.Unmarshal(raw, &n); err == nil {
		return n.Int64()
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return 0, err
	}
	return strconv.ParseInt(s, 10, 64)
}
