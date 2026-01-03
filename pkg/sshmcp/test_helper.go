package sshmcp

import (
	"testing"

	"github.com/rs/zerolog"
)

// setupTestLogger creates a test logger that outputs to t.Log
func setupTestLogger(t *testing.T) *zerolog.Logger {
	// Create a console writer that outputs to test logger
	output := zerolog.NewConsoleWriter()
	output.Out = &testWriter{t: t}
	output.NoColor = true

	logger := zerolog.New(output).With().Timestamp().Logger()
	return &logger
}

// testWriter implements io.Writer for testing
type testWriter struct {
	t *testing.T
}

func (w *testWriter) Write(p []byte) (n int, err error) {
	w.t.Log(string(p))
	return len(p), nil
}
