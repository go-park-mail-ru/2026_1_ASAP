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

type publicInfoUserFields struct {
	UserIDRaw json.RawMessage `json:"user_id"`
	Email     string          `json:"email"`
	FirstName string          `json:"first_name"`
	LastName  string          `json:"last_name"`
	Avatar    string          `json:"avatar"`
}

func RequestAuthFromPublicInfoJSON(body []byte, oauthUserID int64) (*RequestAuth, error) {
	var root map[string]json.RawMessage
	if err := json.Unmarshal(body, &root); err != nil {
		return nil, fmt.Errorf("public_info json: %w", err)
	}

	if _, hasErr := root["error"]; hasErr {
		var apiErr struct {
			Error            string `json:"error"`
			ErrorDescription string `json:"error_description"`
		}
		_ = json.Unmarshal(body, &apiErr)
		if apiErr.ErrorDescription != "" {
			return nil, fmt.Errorf("public_info api: %s: %s", apiErr.Error, apiErr.ErrorDescription)
		}
		return nil, fmt.Errorf("public_info api: %s", apiErr.Error)
	}

	if userRaw, ok := root["profile"]; ok && len(userRaw) > 0 && string(userRaw) != "null" {
		return decodePublicInfoUser(userRaw, oauthUserID)
	}

	var flat RequestAuth
	if err := json.Unmarshal(body, &flat); err != nil {
		return nil, err
	}
	if flat.VKUserID != 0 || flat.Email != "" || flat.FirstName != "" ||
		flat.LastName != "" || flat.AvatarURL != "" {
		if flat.VKUserID == 0 && oauthUserID != 0 {
			flat.VKUserID = oauthUserID
		}
		return &flat, nil
	}
	if oauthUserID != 0 {
		return &RequestAuth{VKUserID: oauthUserID}, nil
	}
	return nil, errors.New("public_info: empty payload")
}

func decodePublicInfoUser(userRaw json.RawMessage, oauthUserID int64) (*RequestAuth, error) {
	var u publicInfoUserFields
	if err := json.Unmarshal(userRaw, &u); err != nil {
		return nil, err
	}

	if len(u.UserIDRaw) == 0 {
		var loose map[string]json.RawMessage
		if err := json.Unmarshal(userRaw, &loose); err == nil {
			for _, key := range []string{"user_id", "id"} {
				if v, ok := loose[key]; ok && len(v) > 0 && string(v) != "null" {
					u.UserIDRaw = v
					break
				}
			}
		}
	}

	vkID, err := parseFlexibleInt64(u.UserIDRaw)
	if err != nil || vkID == 0 {
		if oauthUserID != 0 {
			vkID = oauthUserID
			err = nil
		}
	}
	if err != nil || vkID == 0 {
		return nil, fmt.Errorf("public_info profile without user_id (oauth fallback empty): %s", truncateRunes(string(userRaw), 220))
	}

	return &RequestAuth{
		VKUserID:  vkID,
		Email:     u.Email,
		FirstName: u.FirstName,
		LastName:  u.LastName,
		AvatarURL: u.Avatar,
	}, nil
}

func truncateRunes(s string, maxRunes int) string {
	r := []rune(s)
	if len(r) <= maxRunes {
		return s
	}
	return string(r[:maxRunes]) + "…"
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
