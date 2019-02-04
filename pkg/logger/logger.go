package logger

import (
	"os"

	kitlog "github.com/go-kit/kit/log"
	"github.com/thingful/kuzu/pkg/version"
)

// NewLogger returns a new kitlog.Logger instance ready for use.
func NewLogger() kitlog.Logger {
	logger := kitlog.NewLogfmtLogger(kitlog.NewSyncWriter(os.Stdout))
	return kitlog.With(logger, "service", version.BinaryName, "ts", kitlog.DefaultTimestampUTC)
}
