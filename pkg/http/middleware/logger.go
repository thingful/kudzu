package middleware

import (
	"context"
	"net/http"
	"time"

	kitlog "github.com/go-kit/kit/log"
	"github.com/thingful/kuzu/pkg/logger"
)

const (
	loggerKey = contextKey("logger")
)

// loggingResponseWriter is a struct that allows us to capture the status code
// after a request has finished
type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader is our overridden function that captures the status code.
func (lrw *loggingResponseWriter) WriteHeader(statusCode int) {
	lrw.statusCode = statusCode
	lrw.ResponseWriter.WriteHeader(statusCode)
}

// newLoggingResponseWriter creates a new capturing response writer.
func newLoggingResponseWriter(w http.ResponseWriter) *loggingResponseWriter {
	return &loggingResponseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}
}

// LoggingMiddleware is our middleware that implements logging functionality.
type LoggingMiddleware struct {
	logger  kitlog.Logger
	verbose bool
}

// Handler is the middleware handler function.
func (l *LoggingMiddleware) Handler(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		lrw := newLoggingResponseWriter(w)

		requestID := RequestIDFromContext(r.Context())
		logger := kitlog.With(l.logger, "requestID", requestID)

		if l.verbose {
			defer func(begin time.Time) {
				logger.Log(
					"method", r.Method,
					"path", r.URL.Path,
					"remoteAddr", r.RemoteAddr,
					"status", lrw.statusCode,
					"duration", time.Since(begin),
				)
			}(time.Now())
		}

		ctx := context.WithValue(r.Context(), loggerKey, logger)
		next.ServeHTTP(lrw, r.WithContext(ctx))
	}

	return http.HandlerFunc(fn)
}

// NewLoggingMiddleware returns a new instance of our logging middleware.
func NewLoggingMiddleware(logger kitlog.Logger, verbose bool) *LoggingMiddleware {
	return &LoggingMiddleware{
		logger:  logger,
		verbose: verbose,
	}
}

// LoggerFromContext returns a logger from the context. We attempt to get a
// properly initialized logger from the context, but if not we return a valid
// but unitialized logger.
func LoggerFromContext(ctx context.Context) kitlog.Logger {
	if logger, ok := ctx.Value(loggerKey).(kitlog.Logger); ok {
		return logger
	}

	return logger.NewLogger()
}
