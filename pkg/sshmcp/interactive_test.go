package sshmcp

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestInteractiveShell_NonBlockingRead tests non-blocking read with real SSH connection
func TestInteractiveShell_NonBlockingRead(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// 创建 SSH 会话
	session := createTestSession(t)
	defer cleanupTestSession(t, session)

	// 创建 Shell with raw mode for better interactivity
	config := &ShellConfig{
		Mode:         TerminalModeCooked,
		ANSIMode:     ANSIRaw,
		ReadTimeout:  100 * time.Millisecond,
		WriteTimeout: 5 * time.Second,
	}

	shell, err := session.CreateShellWithConfig("xterm-256color", 24, 80, config)
	require.NoError(t, err)
	require.NotNil(t, shell)
	defer shell.Close()

	t.Log("✅ Created shell with custom config")

	// 测试非阻塞读取
	t.Log("✅ Testing non-blocking read...")

	// 发送命令
	err = shell.WriteInput("echo 'Hello, World!'\n")
	require.NoError(t, err)

	// 多次读取以获取所有输出（非阻塞，所以不会卡住）
	var fullOutput string
	for i := 0; i < 10; i++ {
		stdout, stderr, err := shell.ReadOutputNonBlocking(100 * time.Millisecond)
		require.NoError(t, err)

		if stdout != "" {
			fullOutput += stdout
			t.Logf("Read iteration %d: got %d bytes", i+1, len(stdout))
		}

		if stderr != "" {
			t.Logf("stderr: %s", stderr)
		}

		// 如果没有更多数据，退出
		if stdout == "" && stderr == "" {
			break
		}

		time.Sleep(50 * time.Millisecond)
	}

	t.Logf("Full output:\n%s", fullOutput)
	assert.Contains(t, fullOutput, "Hello, World!")
	t.Log("✅ Non-blocking read successful")
}

// TestInteractiveShell_ANSIStrip tests ANSI escape sequence stripping
func TestInteractiveShell_ANSIStrip(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	session := createTestSession(t)
	defer cleanupTestSession(t, session)

	// 创建 Shell with ANSI strip mode
	config := &ShellConfig{
		Mode:         TerminalModeCooked,
		ANSIMode:     ANSIStrip, // Strip ANSI sequences
		ReadTimeout:  100 * time.Millisecond,
		WriteTimeout: 5 * time.Second,
	}

	shell, err := session.CreateShellWithConfig("xterm-256color", 24, 80, config)
	require.NoError(t, err)
	require.NotNil(t, shell)
	defer shell.Close()

	t.Log("✅ Created shell with ANSI strip mode")

	// 发送一个带颜色的命令
	err = shell.WriteInput("echo -e '\\033[31mRed Text\\033[0m'\n")
	require.NoError(t, err)

	// 读取输出
	time.Sleep(200 * time.Millisecond)
	stdout, _, err := shell.ReadOutputNonBlocking(100 * time.Millisecond)
	require.NoError(t, err)

	t.Logf("Output (ANSI stripped): %s", stdout)

	// 验证 ANSI 序列已被去除
	assert.NotContains(t, stdout, "\x1b[")
	assert.NotContains(t, stdout, "\033[")
	assert.Contains(t, stdout, "Red Text")

	t.Log("✅ ANSI stripping successful")
}

// TestInteractiveShell_SpecialChars tests special character input
func TestInteractiveShell_SpecialChars(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	session := createTestSession(t)
	defer cleanupTestSession(t, session)

	config := DefaultShellConfig()
	shell, err := session.CreateShellWithConfig("xterm-256color", 24, 80, config)
	require.NoError(t, err)
	require.NotNil(t, shell)
	defer shell.Close()

	t.Log("✅ Created shell for special character tests")

	// 测试 Ctrl+L (clear screen)
	err = shell.WriteSpecialChars("ctrl+l")
	require.NoError(t, err)
	t.Log("✅ Sent Ctrl+L (clear screen)")

	// 测试 Ctrl+C (interrupt)
	err = shell.WriteSpecialChars("ctrl+c")
	require.NoError(t, err)
	t.Log("✅ Sent Ctrl+C (interrupt)")

	// 测试方向键
	testCases := []string{"up", "down", "left", "right"}
	for _, tc := range testCases {
		err = shell.WriteSpecialChars(tc)
		require.NoError(t, err)
		t.Logf("✅ Sent %s arrow key", tc)
	}

	t.Log("✅ All special characters sent successfully")
}

// TestInteractiveShell_RawMode tests raw mode terminal
func TestInteractiveShell_RawMode(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	session := createTestSession(t)
	defer cleanupTestSession(t, session)

	// 创建 Raw mode shell
	config := &ShellConfig{
		Mode:         TerminalModeRaw, // Raw mode
		ANSIMode:     ANSIRaw,
		ReadTimeout:  50 * time.Millisecond,
		WriteTimeout: 5 * time.Second,
	}

	shell, err := session.CreateShellWithConfig("xterm-256color", 24, 80, config)
	require.NoError(t, err)
	require.NotNil(t, shell)
	defer shell.Close()

	t.Log("✅ Created shell in raw mode")

	// 在 raw mode 下测试命令输入
	err = shell.WriteInput("pwd\n")
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)
	stdout, _, err := shell.ReadOutputNonBlocking(100 * time.Millisecond)
	require.NoError(t, err)

	t.Logf("Raw mode output: %s", stdout)
	assert.Contains(t, stdout, "/root")

	t.Log("✅ Raw mode test successful")
}

// TestInteractiveShell_ProgramDetection tests interactive program detection
func TestInteractiveShell_ProgramDetection(t *testing.T) {
	tests := []struct {
		name     string
		cmd      string
		expected bool
	}{
		{"vim", "vim file.txt", true},
		{"top", "top", true},
		{"python", "python3", true},
		{"ls", "ls -la", false},
		{"cat", "cat file.txt", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsInteractiveProgram(tt.cmd)
			assert.Equal(t, tt.expected, result)
		})
	}

	t.Log("✅ Interactive program detection test successful")
}

// TestInteractiveShell_Configuration tests different shell configurations
func TestInteractiveShell_Configuration(t *testing.T) {
	t.Run("Default configuration", func(t *testing.T) {
		config := DefaultShellConfig()
		assert.Equal(t, TerminalModeCooked, config.Mode)
		assert.Equal(t, ANSIRaw, config.ANSIMode)
		assert.Equal(t, 100*time.Millisecond, config.ReadTimeout)
		assert.True(t, config.AutoDetectInteractive)
	})

	t.Run("Custom configuration", func(t *testing.T) {
		config := &ShellConfig{
			Mode:         TerminalModeRaw,
			ANSIMode:     ANSIStrip,
			ReadTimeout:  200 * time.Millisecond,
			WriteTimeout: 10 * time.Second,
		}

		assert.Equal(t, TerminalModeRaw, config.Mode)
		assert.Equal(t, ANSIStrip, config.ANSIMode)
		assert.Equal(t, 200*time.Millisecond, config.ReadTimeout)
	})

	t.Log("✅ Configuration tests successful")
}

// TestInteractiveShell_Performance tests non-blocking read performance
func TestInteractiveShell_Performance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	session := createTestSession(t)
	defer cleanupTestSession(t, session)

	config := DefaultShellConfig()
	shell, err := session.CreateShellWithConfig("xterm-256color", 24, 80, config)
	require.NoError(t, err)
	require.NotNil(t, shell)
	defer shell.Close()

	t.Log("✅ Starting performance test...")

	// 测试多次快速读取不会阻塞
	start := time.Now()
	readCount := 0

	for i := 0; i < 50; i++ {
		_, _, err := shell.ReadOutputNonBlocking(10 * time.Millisecond)
		if err == nil {
			readCount++
		}
	}

	elapsed := time.Since(start)

	t.Logf("✅ Completed %d reads in %v", readCount, elapsed)
	t.Logf("Average time per read: %v", elapsed/time.Duration(readCount))

	// 非阻塞读取应该非常快
	assert.Less(t, elapsed, 5*time.Second, "Non-blocking reads should be fast")

	t.Log("✅ Performance test successful")
}

// TestInteractiveShell_RealWorldScenario simulates a real-world interactive session
func TestInteractiveShell_RealWorldScenario(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	session := createTestSession(t)
	defer cleanupTestSession(t, session)

	config := DefaultShellConfig()
	shell, err := session.CreateShellWithConfig("xterm-256color", 24, 80, config)
	require.NoError(t, err)
	require.NotNil(t, shell)
	defer shell.Close()

	t.Log("=== Real-world scenario test ===")

	// 场景 1: 执行命令并读取输出
	t.Log("Scenario 1: Execute command and read output")
	err = shell.WriteInput("uname -a\n")
	require.NoError(t, err)

	time.Sleep(200 * time.Millisecond)
	stdout, _, _ := shell.ReadOutputNonBlocking(200 * time.Millisecond)
	assert.Contains(t, stdout, "Linux")
	t.Logf("Command output: %s", stdout[:min(100, len(stdout))])

	// 场景 2: 使用特殊字符清屏
	t.Log("Scenario 2: Clear screen with Ctrl+L")
	err = shell.WriteSpecialChars("ctrl+l")
	require.NoError(t, err)

	// 场景 3: 快速连续读取（模拟 AI 代理轮询）
	t.Log("Scenario 3: AI agent polling simulation")
	for i := 0; i < 5; i++ {
		stdout, _, err := shell.ReadOutputNonBlocking(50 * time.Millisecond)
		require.NoError(t, err)
		_ = stdout // 非阻塞，即使没有输出也不会卡住
	}

	t.Log("✅ Real-world scenario test completed successfully")
}

// Helper function to avoid slice overflow
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// createTestSession creates a test SSH session
func createTestSession(t *testing.T) *Session {
	host := getTestHost()
	port := getTestPort()
	username := getTestUser()
	password := getTestPassword()

	// Create session manager with minimal config and logger
	logger := setupTestLogger(t)
	config := ManagerConfig{
		MaxSessions:        10,
		MaxSessionsPerHost: 3,
		SessionTimeout:     10 * time.Minute,
		IdleTimeout:        5 * time.Minute,
		CleanupInterval:    1 * time.Minute,
		Logger:             logger,
	}

	sm := NewSessionManager(config)
	t.Cleanup(func() {
		sm.Close()
	})

	auth := &AuthConfig{
		Type:     AuthTypePassword,
		Password: password,
	}

	// Generate a unique alias for this test
	alias := fmt.Sprintf("test-%d", time.Now().UnixNano())

	session, err := sm.CreateSession(host, port, username, auth, alias)
	require.NoError(t, err, "Failed to create SSH session")
	require.NotNil(t, session)

	t.Logf("Created test session: %s@%s:%d (alias: %s)", username, host, port, alias)
	return session
}

// cleanupTestSession cleans up a test session
func cleanupTestSession(t *testing.T, session *Session) {
	// Session cleanup is handled by SessionManager
	if session != nil {
		t.Logf("Session %s will be cleaned up by manager", session.ID)
	}
}

// Test shell config getter
func TestInteractiveShell_GetConfig(t *testing.T) {
	config := &ShellConfig{
		Mode:         TerminalModeRaw,
		ANSIMode:     ANSIStrip,
		ReadTimeout:  200 * time.Millisecond,
		WriteTimeout: 10 * time.Second,
	}

	shell := &SSHShellSession{
		Config: config,
	}

	retrievedConfig := shell.GetConfig()
	assert.Equal(t, config.Mode, retrievedConfig.Mode)
	assert.Equal(t, config.ANSIMode, retrievedConfig.ANSIMode)
	assert.Equal(t, config.ReadTimeout, retrievedConfig.ReadTimeout)

	t.Log("✅ GetConfig test successful")
}
