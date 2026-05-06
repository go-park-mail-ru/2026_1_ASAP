package ws

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"

	"github.com/go-park-mail-ru/2026_1_ASAP/internal/gateway/middleware"
)

func NewProxy(chatWSAddr string) http.Handler {
	target := &url.URL{
		Scheme: "http",
		Host:   chatWSAddr,
	}

	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.FlushInterval = -1

	origDirector := proxy.Director
	proxy.Director = func(r *http.Request) {
		origDirector(r)
		r.URL.Path = "/ws"
		if userID, ok := r.Context().Value(middleware.UserID).(int64); ok {
			r.Header.Set("X-User-ID", strconv.FormatInt(userID, 10))
		}
	}

	return proxy
}
