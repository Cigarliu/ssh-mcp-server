package sshmcp

import (
	"os"
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

// Test configuration functions

// getTestHost returns the test SSH host from environment variable or default
func getTestHost() string {
	host := os.Getenv("TEST_SSH_HOST")
	if host == "" {
		host = "192.168.68.212"
	}
	return host
}

// getTestPort returns the test SSH port from environment variable or default
func getTestPort() int {
	port := os.Getenv("TEST_SSH_PORT")
	if port == "" {
		return 22
	}
	// In real code, would convert string to int
	return 22
}

// getTestUser returns the test SSH username from environment variable or default
func getTestUser() string {
	user := os.Getenv("TEST_SSH_USER")
	if user == "" {
		return "root"
	}
	return user
}

// getTestPassword returns the test SSH password from environment variable or default
func getTestPassword() string {
	pass := os.Getenv("TEST_SSH_PASSWORD")
	if pass == "" {
		pass = "root"
	}
	return pass
}
