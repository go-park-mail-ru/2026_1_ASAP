package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

const requestID ctxKey = "request_id"

// RequestIDFromContext возвращает request_id, если его установил RequestIDMiddleware.
func RequestIDFromContext(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(requestID).(string)
	return id, ok
}

func RequestIDMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			newRequestID := uuid.New().String()
			ctx := context.WithValue(r.Context(), requestID, newRequestID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
