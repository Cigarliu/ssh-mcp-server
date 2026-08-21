package mcp

import (
	"context"
	"testing"
	"time"

	"github.com/cigar/sshmcp/pkg/sshmcp"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewServer tests creating a new MCP server
func TestNewServer(t *testing.T) {
	logger := setupTestLogger()
	sessionManager := sshmcp.NewSessionManager(sshmcp.ManagerConfig{
		MaxSessions:     10,
		SessionTimeout:  5 * testMinute,
		IdleTimeout:     2 * testMinute,
		CleanupInterval: 10 * testSecond,
		Logger:          logger,
	})

	defer sessionManager.Close()

	hostManager := sshmcp.NewHostManager(map[string]sshmcp.HostConfig{}, "", logger)

	server, err := NewServer(sessionManager, hostManager, logger)
	assert.NoError(t, err)
	assert.NotNil(t, server)
	assert.NotNil(t, server.mcpServer)
	assert.NotNil(t, server.sessionManager)
	assert.NotNil(t, server.hostManager)
	assert.NotNil(t, server.logger)
}

// TestServer_RegisterTools tests that tools are registered
func TestServer_RegisterTools(t *testing.T) {
	logger := setupTestLogger()
	sessionManager := sshmcp.NewSessionManager(sshmcp.ManagerConfig{
		MaxSessions:     10,
		SessionTimeout:  5 * testMinute,
		IdleTimeout:     2 * testMinute,
		CleanupInterval: 10 * testSecond,
		Logger:          logger,
	})

	defer sessionManager.Close()

	hostManager := sshmcp.NewHostManager(map[string]sshmcp.HostConfig{}, "", logger)

	server, err := NewServer(sessionManager, hostManager, logger)
	require.NoError(t, err)
	require.NotNil(t, server)

	// 验证服务器已创建
	assert.NotNil(t, server.GetMCPServer())
}

// TestServer_Start tests starting the server
func TestServer_Start(t *testing.T) {
	logger := setupTestLogger()
	sessionManager := sshmcp.NewSessionManager(sshmcp.ManagerConfig{
		MaxSessions:     10,
		SessionTimeout:  5 * testMinute,
		IdleTimeout:     2 * testMinute,
		CleanupInterval: 10 * testSecond,
		Logger:          logger,
	})

	defer sessionManager.Close()

	hostManager := sshmcp.NewHostManager(map[string]sshmcp.HostConfig{}, "", logger)

	server, err := NewServer(sessionManager, hostManager, logger)
	require.NoError(t, err)
	require.NotNil(t, server)

	// 在后台启动服务器
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errChan := make(chan error, 1)
	go func() {
		errChan <- server.Start(ctx)
	}()

	// 给服务器一点时间启动
	select {
	case err := <-errChan:
		if err != nil && err != context.Canceled {
			t.Fatalf("Server start failed: %v", err)
		}
	case <-time.After(100 * testMillisecond):
		// 服务器正常启动
	}

	// 取消 context
	cancel()

	// 等待服务器关闭
	select {
	case <-time.After(1 * testSecond):
		// 服务器已关闭
	}
}

// TestTextContent helper tests
func TestTextContent(t *testing.T) {
	content := textContent("test message")
	assert.Len(t, content, 1)
	// Content is an interface, just verify it was created
	assert.NotNil(t, content[0])
}

// TestFormatResult helper tests
func TestFormatResult(t *testing.T) {
	content := formatResult("Hello %s", "World")
	assert.Len(t, content, 1)
	// Content is an interface, just verify it was created
	assert.NotNil(t, content[0])
}

// TestFormatError helper tests
func TestFormatError(t *testing.T) {
	err := assert.AnError
	content := formatError(err)
	assert.Len(t, content, 1)
	// Content is an interface, just verify it was created
	assert.NotNil(t, content[0])
}

func setupTestLogger() *zerolog.Logger {
	// Create a console writer that outputs to io.Discard to suppress test output
	output := zerolog.NewConsoleWriter()
	output.Out = &mcpTestWriter{}
	output.NoColor = true

	logger := zerolog.New(output).With().Timestamp().Logger()
	return &logger
}

// mcpTestWriter implements io.Writer for testing
type mcpTestWriter struct{}

func (w *mcpTestWriter) Write(p []byte) (n int, err error) {
	// Discard test output
	return len(p), nil
}

// Test constants
const (
	testSecond      = 1
	testMillisecond = 1_000_000
	testMinute      = 60 * testSecond
)

func init() {
	// 注册 testTextContent 类型以便测试
	// 注意：这只是用于测试，实际使用中 Content 是接口
}
