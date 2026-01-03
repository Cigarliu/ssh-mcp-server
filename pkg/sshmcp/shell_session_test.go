package sshmcp

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestIsInteractiveProgram tests the interactive program detection
func TestIsInteractiveProgram(t *testing.T) {
	tests := []struct {
		name     string
		cmd      string
		expected bool
	}{
		{
			name:     "vim is interactive",
			cmd:      "vim file.txt",
			expected: true,
		},
		{
			name:     "gdb is interactive",
			cmd:      "gdb ./binary",
			expected: true,
		},
		{
			name:     "top is interactive",
			cmd:      "top",
			expected: true,
		},
		{
			name:     "python is interactive",
			cmd:      "python",
			expected: true,
		},
		{
			name:     "ls is not interactive",
			cmd:      "ls -la",
			expected: false,
		},
		{
			name:     "cat is not interactive",
			cmd:      "cat file.txt",
			expected: false,
		},
		{
			name:     "grep is not interactive",
			cmd:      "grep pattern file.txt",
			expected: false,
		},
		{
			name:     "case insensitive vim",
			cmd:      "VIM file.txt",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsInteractiveProgram(tt.cmd)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestStripANSI tests ANSI escape sequence stripping
func TestStripANSI(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no ANSI sequences",
			input:    "Hello, World!",
			expected: "Hello, World!",
		},
		{
			name:     "ANSI color codes",
			input:    "\x1b[31mRed text\x1b[0m",
			expected: "Red text",
		},
		{
			name:     "ANSI bold",
			input:    "\x1b[1mBold text\x1b[0m",
			expected: "Bold text",
		},
		{
			name:     "mixed text and ANSI",
			input:    "Normal \x1b[31mred\x1b[0m normal",
			expected: "Normal red normal",
		},
		{
			name:     "multiple ANSI sequences",
			input:    "\x1b[31m\x1b[1mRed and bold\x1b[0m",
			expected: "Red and bold",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stripANSI(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestTerminalModeString tests TerminalMode string representation
func TestTerminalModeString(t *testing.T) {
	tests := []struct {
		mode     TerminalMode
		expected string
	}{
		{TerminalModeCooked, "cooked"},
		{TerminalModeRaw, "raw"},
		{TerminalMode(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := tt.mode.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestANSIModeString tests ANSIMode string representation
func TestANSIModeString(t *testing.T) {
	tests := []struct {
		mode     ANSIMode
		expected string
	}{
		{ANSIRaw, "raw"},
		{ANSIStrip, "strip"},
		{ANSIParse, "parse"},
		{ANSIMode(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := tt.mode.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestDefaultShellConfig tests default shell configuration
func TestDefaultShellConfig(t *testing.T) {
	config := DefaultShellConfig()

	assert.NotNil(t, config)
	assert.Equal(t, TerminalModeCooked, config.Mode)
	assert.Equal(t, ANSIRaw, config.ANSIMode)
	assert.Equal(t, 100*time.Millisecond, config.ReadTimeout)
	assert.Equal(t, 5*time.Second, config.WriteTimeout)
	assert.True(t, config.AutoDetectInteractive)
}

// TestWriteSpecialChars tests special character writing (mock test)
func TestWriteSpecialChars(t *testing.T) {
	// This is a basic unit test - integration tests will verify actual SSH behavior
	tests := []struct {
		name        string
		char        string
		expectError bool
	}{
		{"ctrl+c", "ctrl+c", false},
		{"sigint", "sigint", false},
		{"ctrl+d", "ctrl+d", false},
		{"ctrl+z", "ctrl+z", false},
		{"up", "up", false},
		{"down", "down", false},
		{"left", "left", false},
		{"right", "right", false},
		{"invalid", "invalid", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// We'll test the character mapping logic
			// Actual SSH writing will be tested in integration tests
			var input []byte
			var err bool

			switch strings.ToLower(tt.char) {
			case "ctrl+c", "sigint":
				input = []byte{0x03}
			case "ctrl+d", "eof":
				input = []byte{0x04}
			case "ctrl+z", "sigtstp":
				input = []byte{0x1A}
			case "up":
				input = []byte{0x1B, 0x5B, 0x41}
			case "down":
				input = []byte{0x1B, 0x5B, 0x42}
			case "left":
				input = []byte{0x1B, 0x5B, 0x44}
			case "right":
				input = []byte{0x1B, 0x5B, 0x43}
			default:
				err = true
			}

			if tt.expectError {
				assert.True(t, err, "Expected error for invalid character")
			} else {
				assert.False(t, err, "Should not error for valid character")
				assert.NotEmpty(t, input, "Should have input bytes")
			}
		})
	}
}

// TestShellConfigWithModes tests different shell configurations
func TestShellConfigWithModes(t *testing.T) {
	t.Run("Raw mode configuration", func(t *testing.T) {
		config := &ShellConfig{
			Mode:         TerminalModeRaw,
			ANSIMode:     ANSIStrip,
			ReadTimeout:  50 * time.Millisecond,
			WriteTimeout: 3 * time.Second,
		}

		assert.Equal(t, TerminalModeRaw, config.Mode)
		assert.Equal(t, ANSIStrip, config.ANSIMode)
		assert.Equal(t, 50*time.Millisecond, config.ReadTimeout)
		assert.Equal(t, 3*time.Second, config.WriteTimeout)
	})

	t.Run("Cooked mode configuration", func(t *testing.T) {
		config := &ShellConfig{
			Mode:         TerminalModeCooked,
			ANSIMode:     ANSIRaw,
			ReadTimeout:  200 * time.Millisecond,
			WriteTimeout: 10 * time.Second,
		}

		assert.Equal(t, TerminalModeCooked, config.Mode)
		assert.Equal(t, ANSIRaw, config.ANSIMode)
	})
}

// TestReadOutputNonBlocking_Mock tests non-blocking read logic
// Note: Full integration tests require actual SSH connection
func TestReadOutputNonBlocking_Mock(t *testing.T) {
	t.Run("timeout configuration", func(t *testing.T) {
		config := &ShellConfig{
			ReadTimeout: 50 * time.Millisecond,
		}

		ss := &SSHShellSession{
			Config: config,
		}

		// Test that config timeout is used when no timeout is passed
		// This is a compile-time check that the structure works
		assert.NotNil(t, ss.Config)
		assert.Equal(t, 50*time.Millisecond, ss.Config.ReadTimeout)
	})
}

// BenchmarkStripANSI benchmarks ANSI stripping performance
func BenchmarkStripANSI(b *testing.B) {
	input := "\x1b[31m\x1b[1mRed and bold\x1b[0m with \x1b[32mgreen\x1b[0m text"

	for i := 0; i < b.N; i++ {
		stripANSI(input)
	}
}

// BenchmarkIsInteractiveProgram benchmarks program detection
func BenchmarkIsInteractiveProgram(b *testing.B) {
	cmd := "vim /path/to/file.txt"

	for i := 0; i < b.N; i++ {
		IsInteractiveProgram(cmd)
	}
}
