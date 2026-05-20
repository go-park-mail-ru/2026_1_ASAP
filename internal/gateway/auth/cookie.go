package auth

import (
	"net/http"
	"strings"

	"github.com/go-park-mail-ru/2026_1_ASAP/config"
)

func gatewaySameSite(s string) http.SameSite {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "strict":
		return http.SameSiteStrictMode
	case "none":
		return http.SameSiteNoneMode
	default:
		return http.SameSiteLaxMode
	}
}

func gatewaySessionCookie(cfg config.GatewaySessionCookieConfig, value string, maxAge int) *http.Cookie {
	return &http.Cookie{
		Name:     "session_id",
		Value:    value,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: cfg.HTTPOnly,
		Secure:   cfg.Secure,
		SameSite: gatewaySameSite(cfg.SameSite),
	}
}
