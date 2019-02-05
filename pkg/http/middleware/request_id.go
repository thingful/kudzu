package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

const (
	requestIDHeader = "X-Correlation-ID"

	requestIDKey = contextKey("requestID")
)

// RequestIDMiddleware adds a simple middleware that generates a unique ID for
// incoming requests, and this ID is stashed in the context.
func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rid := r.Header.Get(requestIDHeader)
		if rid == "" {
			rid = uuid.New().String()
		}

		w.Header().Set(requestIDHeader, rid)
		ctx := context.WithValue(r.Context(), requestIDKey, rid)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequestIDFromContext is a helper function that returns the request ID we have
// previously stashed into the context.
func RequestIDFromContext(ctx context.Context) string {
	if requestID, ok := ctx.Value(requestIDKey).(string); ok {
		return requestID
	}

	return ""
}
