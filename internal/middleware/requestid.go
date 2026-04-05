package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

const requestID ctxKey = "request_id"

func RequestIDMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			newRequestID := uuid.New().String()
			ctx := context.WithValue(r.Context(), requestID, newRequestID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
