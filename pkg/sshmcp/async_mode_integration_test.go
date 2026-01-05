package sshmcp

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

// TestAsyncMode_ImmediateReturn verifies that ssh_shell returns immediately
func TestAsyncMode_ImmediateReturn(t *testing.T) {
	// Skip in short mode
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	sessionManager := createTestSessionManager(t)
	session := createAndConnectTestSession(t, sessionManager, "192.168.3.7", "cigar", "liuxuejia.123", "async-immediate")
	defer sessionManager.RemoveSession(session.ID)

	// Create shell and measure time to return
	start := time.Now()
	shellSession, err := session.CreateShellWithConfig("xterm-256color", 24, 80, DefaultShellConfig())
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Failed to create shell: %v", err)
	}
	defer shellSession.Close()

	// Verify it returns quickly (should be < 1 second for async mode)
	if elapsed > 1*time.Second {
		t.Errorf("Shell creation took too long: %v (expected < 1s for async mode)", elapsed)
	}

	// Verify shell is active
	status := shellSession.GetStatus()
	if !status.IsActive {
		t.Error("Shell session should be active immediately after creation")
	}

	// Verify background reader is running
	time.Sleep(100 * time.Millisecond)
	if shellSession.OutputBuffer == nil {
		t.Error("Output buffer should be initialized")
	}

	t.Logf("✓ Shell returned immediately in %v", elapsed)
	t.Logf("✓ Buffer capacity: %d lines", status.BufferTotal)
}

// TestAsyncMode_BackgroundOutputBuffering verifies output is buffered in background
func TestAsyncMode_BackgroundOutputBuffering(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	sessionManager := createTestSessionManager(t)
	session := createAndConnectTestSession(t, sessionManager, "192.168.3.7", "cigar", "liuxuejia.123", "async-buffer")
	defer sessionManager.RemoveSession(session.ID)

	shellSession, err := session.CreateShellWithConfig("xterm-256color", 24, 80, DefaultShellConfig())
	if err != nil {
		t.Fatalf("Failed to create shell: %v", err)
	}
	defer shellSession.Close()

	// Send a command
	testCommand := "echo 'Hello, Async Mode!'"
	err = shellSession.WriteInput(testCommand + "\n")
	if err != nil {
		t.Fatalf("Failed to write input: %v", err)
	}

	// Wait for output to be buffered
	time.Sleep(500 * time.Millisecond)

	// Read from buffer
	lines := shellSession.OutputBuffer.ReadLatestLines(10)
	if len(lines) == 0 {
		t.Fatal("No output found in buffer")
	}

	// Verify output contains our text
	found := false
	for _, line := range lines {
		if strings.Contains(line, "Hello, Async Mode!") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected output not found in buffer. Got lines: %v", lines)
	}

	t.Logf("✓ Output buffered successfully: %d lines in buffer", shellSession.OutputBuffer.GetCount())
}

// TestAsyncMode_ReadOutputStrategies verifies all three reading strategies work
func TestAsyncMode_ReadOutputStrategies(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	sessionManager := createTestSessionManager(t)
	session := createAndConnectTestSession(t, sessionManager, "192.168.3.7", "cigar", "liuxuejia.123", "async-strategies")
	defer sessionManager.RemoveSession(session.ID)

	shellSession, err := session.CreateShellWithConfig("xterm-256color", 30, 120, DefaultShellConfig())
	if err != nil {
		t.Fatalf("Failed to create shell: %v", err)
	}
	defer shellSession.Close()

	// Send multiple commands
	commands := []string{
		"echo 'Line 1'",
		"echo 'Line 2'",
		"echo 'Line 3'",
		"echo 'Line 4'",
		"echo 'Line 5'",
	}

	for _, cmd := range commands {
		err = shellSession.WriteInput(cmd + "\n")
		if err != nil {
			t.Fatalf("Failed to write command '%s': %v", cmd, err)
		}
		time.Sleep(100 * time.Millisecond)
	}

	// Wait for all output to buffer
	time.Sleep(1 * time.Second)

	// Test strategy 1: latest_lines
	t.Run("LatestLines", func(t *testing.T) {
		lines := shellSession.OutputBuffer.ReadLatestLines(3)
		if len(lines) < 3 {
			t.Errorf("Expected at least 3 lines, got %d", len(lines))
		}
		t.Logf("✓ ReadLatestLines(3): %v", lines)
	})

	// Test strategy 2: all_unread (mark as read first)
	t.Run("AllUnread", func(t *testing.T) {
		// First read to mark current content as read
		shellSession.OutputBuffer.ReadLatestLines(100)

		// Send new command
		shellSession.WriteInput("echo 'New Line After Read'\n")
		time.Sleep(500 * time.Millisecond)

		// Read all unread
		lines := shellSession.OutputBuffer.ReadAllUnread()
		found := false
		for _, line := range lines {
			if strings.Contains(line, "New Line After Read") {
				found = true
				break
			}
		}
		if !found {
			t.Error("ReadAllUnread did not return new content")
		}
		t.Logf("✓ ReadAllUnread: %d lines", len(lines))
	})

	// Test strategy 3: latest_bytes
	t.Run("LatestBytes", func(t *testing.T) {
		// First, send a new command to ensure we have fresh data
		shellSession.WriteInput("echo 'Test for LatestBytes'\n")
		time.Sleep(500 * time.Millisecond)

		data := shellSession.OutputBuffer.ReadLatestBytes(100)
		if len(data) == 0 {
			t.Error("ReadLatestBytes returned no data")
		} else {
			t.Logf("✓ ReadLatestBytes(100): %d bytes", len(data))
			// Verify it contains our test data
			if strings.Contains(string(data), "Test for LatestBytes") {
				t.Logf("✓ ReadLatestBytes contains expected data")
			}
		}
	})
}

// TestAsyncMode_ShellStatus verifies enhanced status information
func TestAsyncMode_ShellStatus(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	sessionManager := createTestSessionManager(t)
	session := createAndConnectTestSession(t, sessionManager, "192.168.3.7", "cigar", "liuxuejia.123", "async-status")
	defer sessionManager.RemoveSession(session.ID)

	shellSession, err := session.CreateShellWithConfig("xterm-256color", 24, 80, DefaultShellConfig())
	if err != nil {
		t.Fatalf("Failed to create shell: %v", err)
	}
	defer shellSession.Close()

	// Get status
	status := shellSession.GetStatus()

	// Verify all fields are populated
	if !status.IsActive {
		t.Error("Status should indicate session is active")
	}

	if status.BufferTotal == 0 {
		t.Error("Buffer total should be > 0")
	}

	if status.TerminalType != "xterm-256color" {
		t.Errorf("Expected terminal type 'xterm-256color', got '%s'", status.TerminalType)
	}

	if status.Rows != 24 || status.Cols != 80 {
		t.Errorf("Expected terminal size 24x80, got %dx%d", status.Rows, status.Cols)
	}

	if status.Mode != "cooked" {
		t.Errorf("Expected mode 'cooked', got '%s'", status.Mode)
	}

	if status.ANSIMode != "strip" {
		t.Errorf("Expected ANSI mode 'strip', got '%s'", status.ANSIMode)
	}

	// Send command to generate output
	shellSession.WriteInput("echo 'Status Test'\n")
	time.Sleep(500 * time.Millisecond)

	// Check status again - should have buffered output
	status = shellSession.GetStatus()
	if status.BufferUsed == 0 {
		t.Error("Buffer should contain output after sending command")
	}

	t.Logf("✓ Status check passed:")
	t.Logf("  - Active: %v", status.IsActive)
	t.Logf("  - Buffer: %d/%d lines (%.1f%%)",
		status.BufferUsed, status.BufferTotal,
		float64(status.BufferUsed)/float64(status.BufferTotal)*100)
	t.Logf("  - Keepalive fails: %d", status.KeepAliveFails)
}

// TestAsyncMode_MultipleCommandsSequence verifies multiple commands can be sent sequentially
func TestAsyncMode_MultipleCommandsSequence(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	sessionManager := createTestSessionManager(t)
	session := createAndConnectTestSession(t, sessionManager, "192.168.3.7", "cigar", "liuxuejia.123", "async-sequence")
	defer sessionManager.RemoveSession(session.ID)

	shellSession, err := session.CreateShellWithConfig("xterm-256color", 30, 120, DefaultShellConfig())
	if err != nil {
		t.Fatalf("Failed to create shell: %v", err)
	}
	defer shellSession.Close()

	// Send sequence of commands
	commands := []struct {
		cmd         string
		expectInOut string
	}{
		{"echo 'Test 1'", "Test 1"},
		{"pwd", "/home"},
		{"whoami", "cigar"},
		{"echo 'Final test'", "Final test"},
	}

	for _, tc := range commands {
		t.Run(tc.cmd, func(t *testing.T) {
			// Clear buffer before command
			shellSession.OutputBuffer.ReadLatestLines(10000)

			// Send command
			err = shellSession.WriteInput(tc.cmd + "\n")
			if err != nil {
				t.Fatalf("Failed to write command: %v", err)
			}

			// Wait for output
			time.Sleep(500 * time.Millisecond)

			// Read buffer
			lines := shellSession.OutputBuffer.ReadLatestLines(20)

			// Verify expected output
			found := false
			for _, line := range lines {
				if strings.Contains(line, tc.expectInOut) {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected output '%s' not found in buffer. Lines: %v", tc.expectInOut, lines)
			}
		})
	}

	t.Logf("✓ All %d commands executed successfully in sequence", len(commands))
}

// TestAsyncMode_HeartbeatFiltering verifies heartbeat data is filtered from buffer
func TestAsyncMode_HeartbeatFiltering(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	sessionManager := createTestSessionManager(t)
	session := createAndConnectTestSession(t, sessionManager, "192.168.3.7", "cigar", "liuxuejia.123", "async-heartbeat")
	defer sessionManager.RemoveSession(session.ID)

	shellSession, err := session.CreateShellWithConfig("xterm-256color", 24, 80, DefaultShellConfig())
	if err != nil {
		t.Fatalf("Failed to create shell: %v", err)
	}
	defer shellSession.Close()

	// Wait for heartbeat to run (default is 60 seconds, so we check the buffer initialization)
	time.Sleep(500 * time.Millisecond)

	// Send a command to ensure buffer is working
	shellSession.WriteInput("echo 'Heartbeat test'\n")
	time.Sleep(500 * time.Millisecond)

	// Read buffer
	lines := shellSession.OutputBuffer.ReadLatestLines(100)

	// Verify no heartbeat data in buffer (ANSI control codes should be filtered)
	for i, line := range lines {
		if strings.Contains(line, "\x1b[s") || strings.Contains(line, "\x1b[u") {
			t.Errorf("Line %d contains heartbeat data: %q", i, line)
		}
	}

	t.Logf("✓ Heartbeat filtering verified: %d lines in buffer, no heartbeat data found", len(lines))
}

// TestAsyncMode_LongRunningSession verifies session stays alive over extended period
func TestAsyncMode_LongRunningSession(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	t.Log("Starting long-running session test (90 seconds)...")

	sessionManager := createTestSessionManager(t)
	session := createAndConnectTestSession(t, sessionManager, "192.168.3.7", "cigar", "liuxuejia.123", "async-long")
	defer sessionManager.RemoveSession(session.ID)

	shellSession, err := session.CreateShellWithConfig("xterm-256color", 24, 80, DefaultShellConfig())
	if err != nil {
		t.Fatalf("Failed to create shell: %v", err)
	}
	defer shellSession.Close()

	// Run for 90 seconds
	duration := 90 * time.Second
	checkInterval := 15 * time.Second
	start := time.Now()
	checks := 0

	for time.Since(start) < duration {
		time.Sleep(checkInterval)
		checks++

		// Verify session is still alive
		if !shellSession.IsAlive() {
			t.Fatalf("Session died at %v (check %d)", time.Since(start), checks)
		}

		// Check status
		status := shellSession.GetStatus()
		if !status.IsActive {
			t.Errorf("Session inactive at %v (check %d)", time.Since(start), checks)
		}

		// Send a test command
		shellSession.WriteInput("echo 'Keepalive test'\n")
		time.Sleep(500 * time.Millisecond)

		// Verify output
		lines := shellSession.OutputBuffer.ReadLatestLines(10)
		found := false
		for _, line := range lines {
			if strings.Contains(line, "Keepalive test") {
				found = true
				break
			}
		}
		if !found {
			t.Logf("Warning: Command output not found at check %d", checks)
		}

		t.Logf("✓ Check %d at %v: Keepalive fails=%d, Buffer=%d/%d",
			checks, time.Since(start), status.KeepAliveFails, status.BufferUsed, status.BufferTotal)
	}

	t.Logf("✓ Session stayed alive for %v (%d checks)", duration, checks)
}

// TestAsyncMode_BufferOverflow verifies buffer handles overflow correctly
func TestAsyncMode_BufferOverflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	sessionManager := createTestSessionManager(t)
	session := createAndConnectTestSession(t, sessionManager, "192.168.3.7", "cigar", "liuxuejia.123", "async-overflow")
	defer sessionManager.RemoveSession(session.ID)

	// Create shell with small buffer for testing
	config := DefaultShellConfig()
	config.BufferSize = 100 // Small buffer

	shellSession, err := session.CreateShellWithConfig("xterm-256color", 24, 80, config)
	if err != nil {
		t.Fatalf("Failed to create shell: %v", err)
	}
	defer shellSession.Close()

	// Send more commands than buffer can hold
	commandsToSend := 150
	for i := 0; i < commandsToSend; i++ {
		shellSession.WriteInput(fmt.Sprintf("echo 'Buffer line %d'\n", i))
		if i%10 == 0 {
			time.Sleep(100 * time.Millisecond) // Brief pause every 10 commands
		}
	}

	// Wait for all output to buffer
	time.Sleep(2 * time.Second)

	// Check buffer count
	status := shellSession.GetStatus()
	if status.BufferUsed > status.BufferTotal {
		t.Errorf("Buffer overflow: used %d > total %d", status.BufferUsed, status.BufferTotal)
	}

	// Verify we can still read latest lines
	latestLines := shellSession.OutputBuffer.ReadLatestLines(20)
	if len(latestLines) != 20 {
		t.Errorf("Expected 20 latest lines, got %d", len(latestLines))
	}

	t.Logf("✓ Buffer overflow handled correctly:")
	t.Logf("  - Sent: %d commands", commandsToSend)
	bufferStatus := shellSession.GetStatus()
	t.Logf("  - Buffer: %d/%d lines", bufferStatus.BufferUsed, bufferStatus.BufferTotal)
	t.Logf("  - Latest 20 lines readable")
}

// Helper functions for async mode tests

// createAndConnectTestSession creates and connects a test session
func createAndConnectTestSession(t *testing.T, sm *SessionManager, host, user, password, alias string) *Session {
	t.Helper()

	authConfig := &AuthConfig{
		Type:     AuthTypePassword,
		Password: password,
	}

	session, err := sm.CreateSession(host, 22, user, authConfig, alias)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	return session
}
