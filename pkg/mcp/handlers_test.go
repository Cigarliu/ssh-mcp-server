package mcp

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/cigar/sshmcp/pkg/sshmcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestServer creates a test server with session manager
func setupTestServer(t *testing.T) (*Server, *sshmcp.SessionManager) {
	logger := setupTestLogger()
	sessionManager := sshmcp.NewSessionManager(sshmcp.ManagerConfig{
		MaxSessions:        10,
		MaxSessionsPerHost: 5,
		SessionTimeout:     5 * time.Minute,
		IdleTimeout:        2 * time.Minute,
		CleanupInterval:    10 * time.Second,
		Logger:             logger,
	})

	server, err := NewServer(sessionManager, logger)
	require.NoError(t, err)
	require.NotNil(t, server)

	return server, sessionManager
}

// createTestSession creates a real SSH session for testing
func createTestSession(t *testing.T, sm *sshmcp.SessionManager) *sshmcp.Session {
	authConfig := &sshmcp.AuthConfig{
		Type:     sshmcp.AuthTypePassword,
		Password: getEnvOrDefault("SSH_PASSWORD", "root"),
	}

	host := getEnvOrDefault("SSH_HOST", "192.168.68.212")
	username := getEnvOrDefault("SSH_USER", "root")

	session, err := sm.CreateSession(host, 22, username, authConfig, "")
	if err != nil {
		t.Skip("Skipping test: SSH connection not available")
		return nil
	}
	return session
}

// TestHandleSSHExec tests ssh_exec handler
func TestHandleSSHExec(t *testing.T) {
	server, sm := setupTestServer(t)
	defer sm.Close()

	session := createTestSession(t, sm)
	if session == nil {
		return
	}
	defer sm.RemoveSession(session.ID)

	// Test executing a command
	args := map[string]any{
		"session_id": session.ID,
		"command":    "echo 'test output'",
	}

	result, output, err := server.handleSSHExec(context.Background(), nil, args)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Nil(t, output)
	assert.NotNil(t, result.Content)
	assert.Greater(t, len(result.Content), 0)
}

// TestHandleSSHExecBatch tests ssh_exec_batch handler
func TestHandleSSHExecBatch(t *testing.T) {
	server, sm := setupTestServer(t)
	defer sm.Close()

	session := createTestSession(t, sm)
	if session == nil {
		return
	}
	defer sm.RemoveSession(session.ID)

	// Test executing batch commands
	commands := []any{"pwd", "whoami"}
	args := map[string]any{
		"session_id":    session.ID,
		"commands":      commands,
		"stop_on_error": false,
	}

	result, output, err := server.handleSSHExecBatch(context.Background(), nil, args)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Nil(t, output)
	assert.NotNil(t, result.Content)
}

// TestHandleSSHShell tests ssh_shell handler
func TestHandleSSHShell(t *testing.T) {
	server, sm := setupTestServer(t)
	defer sm.Close()

	session := createTestSession(t, sm)
	if session == nil {
		return
	}
	defer sm.RemoveSession(session.ID)

	// Test creating shell
	args := map[string]any{
		"session_id": session.ID,
		"term":       "xterm-256color",
		"rows":       float64(24),
		"cols":       float64(80),
	}

	result, output, err := server.handleSSHShell(context.Background(), nil, args)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Nil(t, output)
	assert.NotNil(t, result.Content)

	// Verify shell was created
	session.RLock()
	shell := session.GetShellSession()
	session.RUnlock()
	assert.NotNil(t, shell, "Shell should be created")
}

// TestHandleSFTPUpload tests sftp_upload handler
func TestHandleSFTPUpload(t *testing.T) {
	server, sm := setupTestServer(t)
	defer sm.Close()

	session := createTestSession(t, sm)
	if session == nil {
		return
	}
	defer sm.RemoveSession(session.ID)

	// Create a test file
	testFile := "/tmp/sshmcp_upload_test.txt"
	err := os.WriteFile(testFile, []byte("test content"), 0644)
	require.NoError(t, err)
	defer os.Remove(testFile)

	// Test file upload
	args := map[string]any{
		"session_id":  session.ID,
		"local_path":  testFile,
		"remote_path": "/tmp/sshmcp_remote_upload.txt",
	}

	result, output, err := server.handleSFTPUpload(context.Background(), nil, args)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Nil(t, output)
	assert.NotNil(t, result.Content)
}

// TestHandleSFTPDownload tests sftp_download handler
func TestHandleSFTPDownload(t *testing.T) {
	server, sm := setupTestServer(t)
	defer sm.Close()

	session := createTestSession(t, sm)
	if session == nil {
		return
	}
	defer sm.RemoveSession(session.ID)

	// First create a remote file
	_, err := session.ExecuteCommand("echo 'test' > /tmp/sshmcp_download_test.txt", 5*time.Second)
	require.NoError(t, err)

	localPath := "/tmp/sshmcp_local_download.txt"
	defer os.Remove(localPath)

	// Test file download
	args := map[string]any{
		"session_id":  session.ID,
		"remote_path": "/tmp/sshmcp_download_test.txt",
		"local_path":  localPath,
	}

	result, output, err := server.handleSFTPDownload(context.Background(), nil, args)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Nil(t, output)
	assert.NotNil(t, result.Content)
}

// TestHandleSFTPListDir tests sftp_list_dir handler
func TestHandleSFTPListDir(t *testing.T) {
	server, sm := setupTestServer(t)
	defer sm.Close()

	session := createTestSession(t, sm)
	if session == nil {
		return
	}
	defer sm.RemoveSession(session.ID)

	// Test listing directory
	args := map[string]any{
		"session_id":  session.ID,
		"remote_path": "/tmp",
	}

	result, output, err := server.handleSFTPListDir(context.Background(), nil, args)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Nil(t, output)
	assert.NotNil(t, result.Content)
}

// TestHandleSFTPMkdir tests sftp_mkdir handler
func TestHandleSFTPMkdir(t *testing.T) {
	server, sm := setupTestServer(t)
	defer sm.Close()

	session := createTestSession(t, sm)
	if session == nil {
		return
	}
	defer sm.RemoveSession(session.ID)

	// Test creating directory
	testDir := "/tmp/sshmcp_mkdir_test_12345"
	args := map[string]any{
		"session_id":  session.ID,
		"remote_path": testDir,
	}

	result, output, err := server.handleSFTPMkdir(context.Background(), nil, args)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Nil(t, output)
	assert.NotNil(t, result.Content)

	// Clean up
	session.SFTPClient.Remove(testDir)
}

// TestHandleSFTPDelete tests sftp_delete handler
func TestHandleSFTPDelete(t *testing.T) {
	server, sm := setupTestServer(t)
	defer sm.Close()

	session := createTestSession(t, sm)
	if session == nil {
		return
	}
	defer sm.RemoveSession(session.ID)

	// First create a file using SFTP
	testFile := "/tmp/sshmcp_delete_test.txt"
	f, err := session.SFTPClient.Create(testFile)
	require.NoError(t, err)
	f.Write([]byte("test"))
	f.Close()

	// Test deleting file
	args := map[string]any{
		"session_id":  session.ID,
		"remote_path": testFile,
	}

	result, output, err := server.handleSFTPDelete(context.Background(), nil, args)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Nil(t, output)
	assert.NotNil(t, result.Content)
}

// TestHandleSSHWriteInput tests ssh_write_input handler
func TestHandleSSHWriteInput(t *testing.T) {
	server, sm := setupTestServer(t)
	defer sm.Close()

	session := createTestSession(t, sm)
	if session == nil {
		return
	}
	defer sm.RemoveSession(session.ID)

	// Create shell first
	_, err := session.CreateShell("xterm-256color", 24, 80)
	require.NoError(t, err)

	// Test writing input
	args := map[string]any{
		"session_id": session.ID,
		"input":      "echo test\n",
	}

	result, output, err := server.handleSSHWriteInput(context.Background(), nil, args)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Nil(t, output)
	assert.NotNil(t, result.Content)
}

// TestHandleSSHReadOutput tests ssh_read_output handler
func TestHandleSSHReadOutput(t *testing.T) {
	server, sm := setupTestServer(t)
	defer sm.Close()

	session := createTestSession(t, sm)
	if session == nil {
		return
	}
	defer sm.RemoveSession(session.ID)

	// Create shell first
	_, err := session.CreateShell("xterm-256color", 24, 80)
	require.NoError(t, err)

	// Write something and then read
	session.RLock()
	shell := session.GetShellSession()
	session.RUnlock()
	shell.WriteInput("echo test\n")
	time.Sleep(100 * time.Millisecond) // Give it time to process

	// Test reading output
	args := map[string]any{
		"session_id": session.ID,
		"timeout":    float64(1),
	}

	result, output, err := server.handleSSHReadOutput(context.Background(), nil, args)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Nil(t, output)
	assert.NotNil(t, result.Content)
}

// TestHandleSSHResizePty tests ssh_resize_pty handler
func TestHandleSSHResizePty(t *testing.T) {
	server, sm := setupTestServer(t)
	defer sm.Close()

	session := createTestSession(t, sm)
	if session == nil {
		return
	}
	defer sm.RemoveSession(session.ID)

	// Create shell first
	_, err := session.CreateShell("xterm-256color", 24, 80)
	require.NoError(t, err)

	// Test resizing PTY
	args := map[string]any{
		"session_id": session.ID,
		"rows":       float64(40),
		"cols":       float64(120),
	}

	result, output, err := server.handleSSHResizePty(context.Background(), nil, args)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Nil(t, output)
	assert.NotNil(t, result.Content)
}

// Helper function to get environment variable with default
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
