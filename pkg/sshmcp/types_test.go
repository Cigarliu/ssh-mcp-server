package sshmcp

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestCommandResult tests CommandResult structure
func TestCommandResult(t *testing.T) {
	result := &CommandResult{
		ExitCode:     0,
		Stdout:       "test output",
		Stderr:       "",
		ExecutionTime: "1s",
		Error:        nil,
	}

	assert.Equal(t, 0, result.ExitCode)
	assert.Equal(t, "test output", result.Stdout)
	assert.Equal(t, "", result.Stderr)
	assert.Equal(t, "1s", result.ExecutionTime)
	assert.Nil(t, result.Error)
}

// TestFileTransferResult tests FileTransferResult structure
func TestFileTransferResult(t *testing.T) {
	result := &FileTransferResult{
		Status:           "success",
		BytesTransferred: 1024,
		Duration:         "2s",
		Error:            nil,
	}

	assert.Equal(t, "success", result.Status)
	assert.Equal(t, int64(1024), result.BytesTransferred)
	assert.Equal(t, "2s", result.Duration)
	assert.Nil(t, result.Error)
}

// TestFileInfo tests FileInfo structure
func TestFileInfo(t *testing.T) {
	fileInfo := &FileInfo{
		Name:     "test.txt",
		Type:     "file",
		Size:     2048,
		Mode:     "-rw-r--r--",
		Modified: time.Now(),
	}

	assert.Equal(t, "test.txt", fileInfo.Name)
	assert.Equal(t, "file", fileInfo.Type)
	assert.Equal(t, int64(2048), fileInfo.Size)
	assert.Equal(t, "-rw-r--r--", fileInfo.Mode)
}

// TestBatchResultSummary tests BatchResultSummary structure
func TestBatchResultSummary(t *testing.T) {
	summary := &BatchResultSummary{
		Total:   5,
		Success: 3,
		Failed:  2,
	}

	assert.Equal(t, 5, summary.Total)
	assert.Equal(t, 3, summary.Success)
	assert.Equal(t, 2, summary.Failed)
}

// TestSessionConfig tests SessionConfig with defaults
func TestSessionConfig(t *testing.T) {
	config := &SessionConfig{
		Timeout:          30 * testSecond,
		KeepAliveInterval: 30 * testSecond,
		CommandTimeout:   30 * testSecond,
		MaxRetries:       3,
		MaxIdleTime:      10 * testMinute,
		AutoReconnect:    false,
	}

	assert.NotNil(t, config)
	assert.Equal(t, 3, config.MaxRetries)
	assert.False(t, config.AutoReconnect)
}

// TestSSHShellSession tests SSHShellSession structure
func TestSSHShellSession(t *testing.T) {
	session := &SSHShellSession{
		PTY: true,
		TerminalInfo: TerminalInfo{
			Term: "xterm-256color",
			Rows: 24,
			Cols: 80,
		},
	}

	assert.True(t, session.PTY)
	assert.Equal(t, "xterm-256color", session.TerminalInfo.Term)
	assert.Equal(t, uint16(24), session.TerminalInfo.Rows)
	assert.Equal(t, uint16(80), session.TerminalInfo.Cols)
}

// TestAuthType constants
func TestAuthType(t *testing.T) {
	tests := []struct {
		authType AuthType
		expected string
	}{
		{AuthTypePassword, "password"},
		{AuthTypePrivateKey, "private_key"},
		{AuthTypeSSHAgent, "ssh_agent"},
		{AuthTypeKeyboard, "keyboard"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.authType))
		})
	}
}

// Helper functions for testing
const (
	testSecond = 1
	testMinute = 60 * testSecond
)
