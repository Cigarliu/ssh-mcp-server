package sshmcp

import (
	"os"
	"strconv"
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

// getTestHost returns the opt-in SSH integration-test host.
func getTestHost() string {
	return os.Getenv("SSHMCP_TEST_SSH_HOST")
}

// getTestPort returns the opt-in SSH integration-test port.
func getTestPort() int {
	port, err := strconv.Atoi(os.Getenv("SSHMCP_TEST_SSH_PORT"))
	if err != nil || port < 1 || port > 65535 {
		return 22
	}
	return port
}

// getTestUser returns the opt-in SSH integration-test username.
func getTestUser() string {
	return os.Getenv("SSHMCP_TEST_SSH_USER")
}

// getTestPassword returns the opt-in SSH integration-test password.
func getTestPassword() string {
	return os.Getenv("SSHMCP_TEST_SSH_PASSWORD")
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
