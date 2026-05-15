package ws

import (
	"net/http"
	"strconv"

	"github.com/google/uuid"
)

func AuthMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw := r.Header.Get("X-User-ID")
			if raw == "" {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			userID, err := strconv.ParseInt(raw, 10, 64)
			if err != nil || userID <= 0 {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			ctx := WithUserID(r.Context(), userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func RequestIDMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := WithRequestID(r.Context(), uuid.NewString())
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
