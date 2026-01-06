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

	// Create shell with raw mode - 使用更大的终端尺寸
	shellConfig := &sshmcp.ShellConfig{
		Mode:         sshmcp.TerminalModeRaw,
		ANSIMode:     sshmcp.ANSIRaw,
		ReadTimeout:  100 * time.Millisecond,
		WriteTimeout: 5 * time.Second,
	}

	// 尝试不同的终端尺寸
	testSizes := []struct {
		rows, cols int
		name       string
	}{
		{40, 160, "Large (40x160)"},
		{30, 120, "Medium (30x120)"},
		{24, 80, "Small (24x80)"},
	}

	for _, size := range testSizes {
		fmt.Printf("\n\n========== Testing with %s ===========\n", size.name)

		shell, err := session.CreateShellWithConfig("xterm-256color", uint16(size.rows), uint16(size.cols), shellConfig)
		if err != nil {
			logger.Fatal().Err(err).Msg("Failed to create shell")
		}

		// Wait for shell to initialize
		time.Sleep(300 * time.Millisecond)

		// Start htop
		err = shell.WriteInput("htop\n")
		if err != nil {
			logger.Fatal().Err(err).Msg("Failed to start htop")
		}
		time.Sleep(1 * time.Second)

		// Get plain text snapshot
		snapshot := shell.GetTerminalSnapshot()

		// Analyze the snapshot
		lines := strings.Split(snapshot, "\n")
		fmt.Printf("📊 Snapshot Analysis (%s):\n", size.name)
		fmt.Printf("  - Total lines: %d\n", len(lines))
		fmt.Printf("  - Actual content lines: %d\n", countNonEmptyLines(lines))
		fmt.Printf("  - First 3 lines:\n")
		for i, line := range lines {
			if i >= 3 || i >= len(lines) {
				break
			}
			if len(strings.TrimSpace(line)) > 0 {
				fmt.Printf("    [%d] %s\n", i+1, line)
			}
		}

		// Check for ANSI artifacts
		hasArtifacts := strings.Contains(snapshot, "[B") || strings.Contains(snapshot, "0B") || strings.Contains(snapshot, "1B")
		fmt.Printf("  - Has ANSI artifacts: %v\n", hasArtifacts)

		// Check display quality
		if hasArtifacts {
			fmt.Printf("  ⚠️  Display quality: POOR (ANSI artifacts detected)\n")
		} else if countNonEmptyLines(lines) >= 10 {
			fmt.Printf("  ✅ Display quality: GOOD (clear display)\n")
		} else {
			fmt.Printf("  ⚠️  Display quality: MARGINAL (few lines)\n")
		}

		// Close this shell
		shell.WriteSpecialChars("ctrl+c")
		time.Sleep(200 * time.Millisecond)
		shell.Close()
		time.Sleep(200 * time.Millisecond)
	}

	fmt.Println("\n✅ Test completed!")
}

func countNonEmptyLines(lines []string) int {
	count := 0
	for _, line := range lines {
		if len(strings.TrimSpace(line)) > 0 {
			count++
		}
	}
	return count
}
