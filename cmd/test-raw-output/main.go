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
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()

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

	session, err := sm.CreateSession("cctv.mba", 9022, "cigar", &sshmcp.AuthConfig{
		Type:     sshmcp.AuthTypePassword,
		Password: "liuxuejia.123",
	}, "test-raw")
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to create SSH session")
	}

	fmt.Println("✅ Connected to server")

	shellConfig := &sshmcp.ShellConfig{
		Mode:         sshmcp.TerminalModeRaw,
		ANSIMode:     sshmcp.ANSIRaw,
		ReadTimeout:  100 * time.Millisecond,
		WriteTimeout: 5 * time.Second,
	}

	shell, err := session.CreateShellWithConfig("xterm-256color", 40, 160, shellConfig)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to create shell")
	}
	defer shell.Close()

	time.Sleep(500 * time.Millisecond)

	// Start htop
	fmt.Println("\n=== Starting htop ===")
	shell.WriteInput("htop\n")
	time.Sleep(2 * time.Second)

	// Get the raw VT100 content (directly from emulator)
	fmt.Println("\n=== Analyzing VT100 Content ===")
	emulatorContent := shell.TerminalCapturer.Emulator.GetScreenContent()

	fmt.Printf("Terminal size: %d rows x %d cols\n", len(emulatorContent), len(emulatorContent[0]))

	// Find first non-empty line
	var firstLine []rune
	for y, row := range emulatorContent {
		lineContent := strings.TrimSpace(string(row))
		if len(lineContent) > 10 {
			firstLine = row
			fmt.Printf("\nFirst meaningful line (row %d):\n", y)
			break
		}
	}

	if firstLine != nil {
		// Print first 100 chars
		fmt.Printf("First 100 chars: ")
		for i, r := range firstLine {
			if i >= 100 {
				break
			}
			if r == 0 {
				continue
			}
			fmt.Printf("%c", r)
		}
		fmt.Println()

		// Check for artifacts
		lineStr := string(firstLine)
		hasB := strings.Contains(lineStr, "B")
		hasBracket := strings.Contains(lineStr, "[")
		hasDigit := strings.ContainsAny(lineStr, "0123456789")

		fmt.Printf("\nArtifacts check:\n")
		fmt.Printf("  - Contains 'B': %v\n", hasB)
		fmt.Printf("  - Contains '[': %v\n", hasBracket)
		fmt.Printf("  - Contains digits: %v\n", hasDigit)

		// Look for patterns like "0B[", "1B[", etc.
		patterns := []string{"0B[", "1B[", "2B[", "B[", "]B"}
		fmt.Printf("\nPattern check:\n")
		for _, pattern := range patterns {
			if strings.Contains(lineStr, pattern) {
				fmt.Printf("  - Found '%s': YES\n", pattern)
			}
		}
	}

	// Get snapshot and analyze
	fmt.Println("\n=== Comparing Snapshots ===")

	// Plain text snapshot
	plainSnapshot := shell.GetTerminalSnapshot()
	fmt.Printf("\nPlain snapshot (%d chars):\n", len(plainSnapshot))
	lines := strings.Split(plainSnapshot, "\n")
	for i, line := range lines {
		if i >= 3 {
			break
		}
		if len(strings.TrimSpace(line)) > 10 {
			fmt.Printf("  Line %d: %s\n", i+1, line[:min(100, len(line))])
		}
	}

	// Check what characters are appearing
	fmt.Println("\n=== Character Analysis ===")
	charCount := make(map[rune]int)
	for _, r := range plainSnapshot {
		if r >= 32 && r <= 126 { // printable ASCII
			charCount[r]++
		}
	}

	// Print top characters
	fmt.Println("Top 10 most frequent printable characters:")
	count := 0
	for r, c := range charCount {
		if count >= 10 {
			break
		}
		fmt.Printf("  '%c' (%d): %d times\n", r, r, c)
		count++
	}

	// Check specifically for 'B' character
	bCount := charCount['B']
	fmt.Printf("\nCharacter 'B' appears: %d times\n", bCount)
	if bCount > 100 {
		fmt.Printf("⚠️  WARNING: 'B' appears too frequently, likely an artifact!\n")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
