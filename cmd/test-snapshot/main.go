package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/cigar/sshmcp/pkg/sshmcp"
	"github.com/rs/zerolog"
)

func main() {
	// Setup logger
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()

	// Create session manager
	config := sshmcp.ManagerConfig{
		MaxSessions:        10,
		MaxSessionsPerHost: 3,
		SessionTimeout:     10 * time.Minute,
		IdleTimeout:        5 * time.Minute,
		CleanupInterval:    1 * time.Minute,
		Logger:             &logger,
	}

	sm := sshmcp.NewSessionManager(config)
	defer sm.Close()

	// Connect to server
	host := "cctv.mba"
	port := 9022
	username := "cigar"
	password := "liuxuejia.123"

	auth := &sshmcp.AuthConfig{
		Type:     sshmcp.AuthTypePassword,
		Password: password,
	}

	alias := fmt.Sprintf("test-%d", time.Now().UnixNano())

	session, err := sm.CreateSession(host, port, username, auth, alias)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to create SSH session")
	}

	fmt.Printf("✅ Connected to %s@%s:%d (alias: %s)\n", username, host, port, alias)

	// Create shell with raw mode for htop
	shellConfig := &sshmcp.ShellConfig{
		Mode:         sshmcp.TerminalModeRaw,
		ANSIMode:     sshmcp.ANSIRaw,
		ReadTimeout:  100 * time.Millisecond,
		WriteTimeout: 5 * time.Second,
	}

	shell, err := session.CreateShellWithConfig("xterm-256color", 30, 120, shellConfig)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to create shell")
	}
	defer shell.Close()

	fmt.Println("✅ Created shell in raw mode")

	// Wait for shell to initialize
	time.Sleep(500 * time.Millisecond)

	// Test 1: Simple command (pwd)
	fmt.Println("\n=== Test 1: pwd command ===")
	err = shell.WriteInput("pwd\n")
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to write input")
	}
	time.Sleep(200 * time.Millisecond)

	snapshot1 := shell.GetTerminalSnapshot()
	fmt.Printf("Snapshot (pwd):\n%s\n", snapshot1)

	// Test 2: Start htop
	fmt.Println("\n=== Test 2: Starting htop ===")
	err = shell.WriteInput("htop\n")
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to start htop")
	}
	time.Sleep(1 * time.Second) // Wait for htop to start

	snapshot2 := shell.GetTerminalSnapshot()
	fmt.Printf("Snapshot (htop started, first 500 chars):\n%s\n", truncate(snapshot2, 500))

	// Test 3: Colored snapshot
	fmt.Println("\n=== Test 3: Colored snapshot ===")
	snapshot3 := shell.GetTerminalSnapshotWithColor()
	fmt.Printf("Colored snapshot length: %d bytes\n", len(snapshot3))
	if len(snapshot3) > 0 {
		// Check for ANSI color codes
		if strings.Contains(snapshot3, "\x1b[") {
			fmt.Println("✅ Colored snapshot contains ANSI codes")
		}
	}

	// Test 4: Cursor position and terminal size
	fmt.Println("\n=== Test 4: Terminal info ===")
	x, y := shell.GetCursorPosition()
	w, h := shell.GetTerminalSize()
	fmt.Printf("Cursor position: (%d, %d)\n", x, y)
	fmt.Printf("Terminal size: %dx%d\n", w, h)

	// Test 5: Quit htop
	fmt.Println("\n=== Test 5: Quitting htop ===")
	err = shell.WriteSpecialChars("ctrl+c")
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to send Ctrl+C")
	}
	time.Sleep(200 * time.Millisecond)

	snapshot4 := shell.GetTerminalSnapshot()
	fmt.Printf("Snapshot (after quit, first 200 chars):\n%s\n", truncate(snapshot4, 200))

	fmt.Println("\n✅ All tests passed!")
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "... (truncated)"
}

