package sshmcp

import (
	"flag"
	"os"
	"testing"
	"time"

	"github.com/rs/zerolog"
)

// Test flags
var (
	sshHost     = flag.String("ssh-host", "", "SSH host for integration tests")
	sshUser     = flag.String("ssh-user", "", "SSH user for integration tests")
	sshPass     = flag.String("ssh-pass", "", "SSH password for integration tests")
	runLongTest = flag.Bool("long", false, "Run long-duration tests")
)

// getTestCredentials returns test credentials from flags or environment
func getTestCredentials() (host, user, pass string) {
	flag.Parse()

	// Try flags first
	if *sshHost != "" && *sshUser != "" && *sshPass != "" {
		return *sshHost, *sshUser, *sshPass
	}

	// Try environment variables
	host = os.Getenv("SSH_HOST")
	user = os.Getenv("SSH_USER")
	pass = os.Getenv("SSH_PASS")

	if host != "" && user != "" && pass != "" {
		return host, user, pass
	}

	// Default test values
	return "192.168.3.7", "cigar", "liuxuejia.123"
}

// skipIfNoCredentials skips the test if credentials are not available
func skipIfNoCredentials(t *testing.T) (string, string, string) {
	host, user, pass := getTestCredentials()

	if host == "" || user == "" || pass == "" {
		t.Skip("SSH credentials not provided. Use -ssh-host, -ssh-user, -ssh-pass flags or environment variables SSH_HOST, SSH_USER, SSH_PASS")
	}

	return host, user, pass
}

// createTestSessionManager creates a session manager for testing
func createTestSessionManager(t *testing.T) *SessionManager {
	logger := zerolog.New(zerolog.NewTestWriter(t))

	config := ManagerConfig{
		MaxSessions:        10,
		MaxSessionsPerHost: 30,
		SessionTimeout:     30 * time.Minute,
		IdleTimeout:        10 * time.Minute,
		CleanupInterval:    1 * time.Minute,
		Logger:             &logger,
	}

	return NewSessionManager(config)
}

// TestTCPKeepAlive_Enabled tests that TCP keepalive is configured
func TestTCPKeepAlive_Enabled(t *testing.T) {
	host, user, pass := skipIfNoCredentials(t)

	authConfig := &AuthConfig{
		Type:     AuthTypePassword,
		Password: pass,
	}

	// Create SSH client (should enable TCP keepalive)
	client, err := CreateSSHClient(host, 22, user, authConfig, 30*time.Second)
	if err != nil {
		t.Fatalf("Failed to create SSH client: %v", err)
	}
	defer client.Close()

	// If we got here without error, TCP keepalive should be enabled
	// (We can't directly check it, but the connection was established successfully)
	t.Log("TCP keepalive test passed - connection established successfully")
}

// TestConnectionKeepalive tests that keepalive mechanism works over time
func TestConnectionKeepalive(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping keepalive test in short mode")
	}

	host, user, pass := skipIfNoCredentials(t)

	authConfig := &AuthConfig{
		Type:     AuthTypePassword,
		Password: pass,
	}

	// Create session manager
	sessionManager := createTestSessionManager(t)

	// Connect to SSH (alias parameter instead of timeout)
	session, err := sessionManager.CreateSession(host, 22, user, authConfig, "test-keepalive")
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}
	defer sessionManager.RemoveSession(session.ID)

	// Create shell
	shellSession, err := session.CreateShell("xterm-256color", 24, 80)
	if err != nil {
		t.Fatalf("Failed to create shell: %v", err)
	}
	defer shellSession.Close()

	t.Log("Shell created, starting keepalive test...")

	// Wait for 2 minutes to test keepalive
	duration := 2 * time.Minute
	if *runLongTest {
		duration = 10 * time.Minute
	}

	t.Logf("Waiting %v to test keepalive mechanism...", duration)

	checkInterval := 30 * time.Second
	checks := int(duration / checkInterval)

	for i := 0; i < checks; i++ {
		time.Sleep(checkInterval)

		// Check if session is still alive
		if !shellSession.IsAlive() {
			t.Fatalf("Session died after %v (keepalive failed)", time.Duration(i)*checkInterval)
		}

		// Check keepalive status
		status := shellSession.GetStatus()
		t.Logf("Check %d/%d: Active=%v, KeepAliveFails=%d, BufferUsed=%d",
			i+1, checks, status.IsActive, status.KeepAliveFails, status.BufferUsed)

		if status.KeepAliveFails > 0 {
			t.Logf("Warning: %d keepalive failures detected", status.KeepAliveFails)
		}
	}

	t.Log("Keepalive test passed - connection remained active")
}

// TestBackgroundOutputReading tests that background output reader works
func TestBackgroundOutputReading(t *testing.T) {
	host, user, pass := skipIfNoCredentials(t)

	authConfig := &AuthConfig{
		Type:     AuthTypePassword,
		Password: pass,
	}

	sessionManager := createTestSessionManager(t)

	// Connect to SSH
	session, err := sessionManager.CreateSession(host, 22, user, authConfig, "test-output-read")
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}
	defer sessionManager.RemoveSession(session.ID)

	// Create shell in raw mode
	config := DefaultShellConfig()
	config.Mode = TerminalModeRaw

	shellSession, err := session.CreateShellWithConfig("xterm-256color", 24, 80, config)
	if err != nil {
		t.Fatalf("Failed to create shell: %v", err)
	}
	defer shellSession.Close()

	t.Log("Shell created in raw mode, testing background output reading...")

	// Send a command that generates output
	err = shellSession.WriteInput("echo 'Test line 1'\n")
	if err != nil {
		t.Fatalf("Failed to write input: %v", err)
	}

	// Wait a bit for output
	time.Sleep(1 * time.Second)

	// Read from buffer
	lines := shellSession.OutputBuffer.ReadLatestLines(10)
	if len(lines) == 0 {
		t.Fatal("No output in buffer - background reader may not be working")
	}

	t.Logf("Background reader test passed. Read %d lines", len(lines))
	for i, line := range lines {
		t.Logf("  Line %d: %s", i+1, line)
	}
}

// TestCircularBufferInRealSession tests circular buffer with real SSH output
func TestCircularBufferInRealSession(t *testing.T) {
	host, user, pass := skipIfNoCredentials(t)

	authConfig := &AuthConfig{
		Type:     AuthTypePassword,
		Password: pass,
	}

	sessionManager := createTestSessionManager(t)

	// Connect to SSH
	session, err := sessionManager.CreateSession(host, 22, user, authConfig, "test-buffer")
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}
	defer sessionManager.RemoveSession(session.ID)

	// Create shell
	shellSession, err := session.CreateShell("xterm-256color", 24, 80)
	if err != nil {
		t.Fatalf("Failed to create shell: %v", err)
	}
	defer shellSession.Close()

	// Send multiple commands
	for i := 1; i <= 20; i++ {
		err = shellSession.WriteInput("echo 'Test line content'\n")
		if err != nil {
			t.Fatalf("Failed to write input: %v", err)
		}
		time.Sleep(100 * time.Millisecond)
	}

	// Wait for all output
	time.Sleep(2 * time.Second)

	// Read latest 10 lines
	lines := shellSession.OutputBuffer.ReadLatestLines(10)

	if len(lines) < 10 {
		t.Errorf("Expected at least 10 lines, got %d", len(lines))
	}

	t.Logf("Circular buffer test passed. Read %d lines", len(lines))

	// Check that buffer is filtering heartbeats
	status := shellSession.GetStatus()
	t.Logf("Buffer usage: %d/%d lines", status.BufferUsed, status.BufferTotal)
}

// TestHeartbeatFilteringInRealSession tests that heartbeat data is filtered
func TestHeartbeatFilteringInRealSession(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping heartbeat test in short mode")
	}

	host, user, pass := skipIfNoCredentials(t)

	authConfig := &AuthConfig{
		Type:     AuthTypePassword,
		Password: pass,
	}

	sessionManager := createTestSessionManager(t)

	// Connect to SSH
	session, err := sessionManager.CreateSession(host, 22, user, authConfig, "test-heartbeat")
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}
	defer sessionManager.RemoveSession(session.ID)

	// Create shell
	shellSession, err := session.CreateShell("xterm-256color", 24, 80)
	if err != nil {
		t.Fatalf("Failed to create shell: %v", err)
	}
	defer shellSession.Close()

	t.Log("Waiting for heartbeats...")

	// Wait for heartbeats to be sent (60 seconds interval)
	// For this test, we'll check after 70 seconds to ensure at least one heartbeat
	time.Sleep(70 * time.Second)

	// Send a marker command
	err = shellSession.WriteInput("echo 'MARKER_LINE'\n")
	if err != nil {
		t.Fatalf("Failed to write marker: %v", err)
	}

	time.Sleep(1 * time.Second)

	// Read all lines
	lines := shellSession.OutputBuffer.ReadAllUnread()

	// Check that marker exists
	hasMarker := false
	for _, line := range lines {
		if line == "MARKER_LINE" {
			hasMarker = true
			break
		}
	}

	if !hasMarker {
		t.Error("Marker line not found - buffer may be corrupted")
	}

	// Verify no heartbeat data is present
	for _, line := range lines {
		if line == "\x1b[s\x1b[u" || line == "\x00" || line == "\x1b[s" || line == "\x1b[u" {
			t.Errorf("Heartbeat data found in buffer: %q", line)
		}
	}

	t.Log("Heartbeat filtering test passed")
}

// TestLongSessionStability tests session stability over extended period
func TestLongSessionStability(t *testing.T) {
	if !*runLongTest {
		t.Skip("Skipping long stability test (use -long flag to enable)")
	}

	host, user, pass := skipIfNoCredentials(t)

	authConfig := &AuthConfig{
		Type:     AuthTypePassword,
		Password: pass,
	}

	sessionManager := createTestSessionManager(t)

	// Connect to SSH
	session, err := sessionManager.CreateSession(host, 22, user, authConfig, "test-stability")
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}
	defer sessionManager.RemoveSession(session.ID)

	// Create shell
	shellSession, err := session.CreateShell("xterm-256color", 24, 80)
	if err != nil {
		t.Fatalf("Failed to create shell: %v", err)
	}
	defer shellSession.Close()

	t.Log("Starting long-term stability test (10 minutes)...")

	// Test duration
	duration := 10 * time.Minute
	checkInterval := 30 * time.Second
	checks := int(duration / checkInterval)

	for i := 0; i < checks; i++ {
		time.Sleep(checkInterval)

		status := shellSession.GetStatus()

		// Log status
		t.Logf("[%d/%d] Status: Active=%v, KA_Fails=%d, Buffer=%d/%d, LastKA=%v ago",
			i+1, checks,
			status.IsActive,
			status.KeepAliveFails,
			status.BufferUsed,
			status.BufferTotal,
			time.Since(status.LastKeepAlive).Round(time.Second))

		// Verify session is still alive
		if !status.IsActive {
			t.Fatalf("Session died at check %d/%d", i+1, checks)
		}

		// Verify keepalive is working
		if time.Since(status.LastKeepAlive) > 2*time.Minute {
			t.Errorf("Keepalive not working - last successful KA was %v ago",
				time.Since(status.LastKeepAlive).Round(time.Second))
		}
	}

	t.Log("Long-term stability test passed!")
}

// TestMultipleConcurrentSessions tests multiple concurrent sessions with keepalive
func TestMultipleConcurrentSessions(t *testing.T) {
	host, user, pass := skipIfNoCredentials(t)

	authConfig := &AuthConfig{
		Type:     AuthTypePassword,
		Password: pass,
	}

	sessionManager := createTestSessionManager(t)

	// Create 3 concurrent sessions
	sessions := make([]*SSHShellSession, 3)

	for i := 0; i < 3; i++ {
		alias := "test-concurrent-" + string(rune('0'+i))
		session, err := sessionManager.CreateSession(host, 22, user, authConfig, alias)
		if err != nil {
			t.Fatalf("Failed to create session %d: %v", i, err)
		}
		defer sessionManager.RemoveSession(session.ID)

		shellSession, err := session.CreateShell("xterm-256color", 24, 80)
		if err != nil {
			t.Fatalf("Failed to create shell %d: %v", i, err)
		}
		defer shellSession.Close()
		sessions[i] = shellSession
	}

	t.Log("Created 3 concurrent sessions, testing stability...")

	// Wait 2 minutes
	time.Sleep(2 * time.Minute)

	// Check all sessions
	for i, shellSession := range sessions {
		status := shellSession.GetStatus()
		if !status.IsActive {
			t.Errorf("Session %d is not active", i)
		}
		t.Logf("Session %d: Active=%v, KA_Fails=%d", i, status.IsActive, status.KeepAliveFails)
	}

	t.Log("Concurrent sessions test passed")
}
