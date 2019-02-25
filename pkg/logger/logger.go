package logger

import (
	"context"
	"os"

	kitlog "github.com/go-kit/kit/log"
	"github.com/thingful/kudzu/pkg/version"
)

type contextKey string

const (
	loggerKey = contextKey("logger")
)

// NewLogger returns a new kitlog.Logger instance ready for use.
func NewLogger() kitlog.Logger {
	logger := kitlog.NewLogfmtLogger(kitlog.NewSyncWriter(os.Stdout))
	return kitlog.With(logger, "service", version.BinaryName, "ts", kitlog.DefaultTimestampUTC)
}

// FromContext returns a logger instance from the given context. If the logger
// is not found we return a new unscoped but usable logger.
func FromContext(ctx context.Context) kitlog.Logger {
	if logger, ok := ctx.Value(loggerKey).(kitlog.Logger); ok {
		return logger
	}

	logger := NewLogger()
	return kitlog.With(logger, "module", "logger")
}

// ToContext sets the given logger into a child context which it now returns.
func ToContext(ctx context.Context, logger kitlog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}
