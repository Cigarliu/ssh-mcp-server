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
	}, "test-large")
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to create SSH session")
	}

	fmt.Printf("✅ Connected to server\n")

	// 使用新的默认尺寸（40x160）
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

	fmt.Printf("✅ Created shell with size: 40 rows x 160 cols\n\n")

	time.Sleep(500 * time.Millisecond)

	// Start htop
	fmt.Println("=== Starting htop ===")
	shell.WriteInput("htop\n")
	time.Sleep(2 * time.Second)

	// Get snapshot
	snapshot := shell.GetTerminalSnapshot()

	// Analyze display
	lines := strings.Split(snapshot, "\n")

	fmt.Printf("\n📊 Display Analysis (40x160):\n")
	fmt.Printf("  Total lines: %d\n", len(lines))
	fmt.Printf("  Non-empty lines: %d\n", countNonEmptyLines(lines))

	// Show first 10 lines
	fmt.Printf("\n📺 First 10 lines of htop output:\n")
	fmt.Println("================================================================================")
	lineCount := 0
	for i, line := range lines {
		if lineCount >= 10 {
			break
		}
		trimmed := strings.TrimSpace(line)
		if len(trimmed) > 0 {
			fmt.Printf("  [%02d] %s\n", i+1, trimmed)
			lineCount++
		}
	}
	fmt.Println("================================================================================")

	// Check for artifacts
	bCount := strings.Count(snapshot, "B")
	patternCount := 0
	patterns := []string{"0B[", "1B[", "2B[", "]B", "3B[", "4B[", "5B[", "6B[", "7B[", "8B[", "9B["}
	for _, pattern := range patterns {
		patternCount += strings.Count(snapshot, pattern)
	}

	fmt.Printf("\n🔍 Artifact Analysis:\n")
	fmt.Printf("  'B' character count: %d\n", bCount)
	fmt.Printf("  ANSI artifact patterns: %d\n", patternCount)
	fmt.Printf("  Artifacts per 1000 chars: %.1f\n", float64(patternCount)*1000/float64(len(snapshot)))

	// Quality assessment
	fmt.Printf("\n✅ Quality Assessment:\n")
	if patternCount > 50 {
		fmt.Printf("  ⚠️  POOR: Too many ANSI artifacts (%d detected)\n", patternCount)
		fmt.Printf("  → Need to implement cleanup function\n")
	} else if patternCount > 10 {
		fmt.Printf("  ⚠️  MARGINAL: Some ANSI artifacts (%d detected)\n", patternCount)
		fmt.Printf("  → Consider implementing cleanup function\n")
	} else {
		fmt.Printf("  ✅ GOOD: Minimal artifacts (%d detected)\n", patternCount)
	}

	// Show a clean section if possible
	fmt.Printf("\n📋 Sample clean output (if any):\n")
	if len(lines) > 15 {
		// Try to find a clean section in the middle
		for i := 10; i < min(20, len(lines)); i++ {
			line := strings.TrimSpace(lines[i])
			if len(line) > 20 && !strings.Contains(line, "B[") && !strings.Contains(line, "]B") {
				fmt.Printf("  Clean line example: %s\n", line)
				break
			}
		}
	}

	// Quit htop
	fmt.Println("\n=== Quitting htop ===")
	shell.WriteSpecialChars("ctrl+c")
	time.Sleep(500 * time.Millisecond)

	fmt.Println("✅ Test completed!")
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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
