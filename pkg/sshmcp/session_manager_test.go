package sshmcp

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestSessionManager_CreateSession tests creating a new session
func TestSessionManager_CreateSession(t *testing.T) {
	// 创建测试用的 logger
	logger := setupTestLogger(t)

	config := ManagerConfig{
		MaxSessions:        10,
		MaxSessionsPerHost: 5,
		SessionTimeout:     5 * time.Minute,
		IdleTimeout:        2 * time.Minute,
		CleanupInterval:    10 * time.Second,
		Logger:             logger,
	}

	sm := NewSessionManager(config)
	defer sm.Close()

	// 测试创建会话
	authConfig := &AuthConfig{
		Type:     AuthTypePrivateKey,
		PrivateKey: "test-key", // 这个会失败，但我们可以测试错误处理
	}

	// 使用无效的私钥，应该会失败
	session, err := sm.CreateSession("localhost", 22, "testuser", authConfig, "")
	assert.Error(t, err, "Expected error with invalid private key")
	assert.Nil(t, session)
}

// TestSessionManager_ListSessions tests listing sessions
func TestSessionManager_ListSessions(t *testing.T) {
	logger := setupTestLogger(t)

	config := ManagerConfig{
		MaxSessions:        10,
		MaxSessionsPerHost: 5,
		SessionTimeout:     5 * time.Minute,
		IdleTimeout:        2 * time.Minute,
		CleanupInterval:    10 * time.Second,
		Logger:             logger,
	}

	sm := NewSessionManager(config)
	defer sm.Close()

	// 初始状态应该没有会话
	sessions := sm.ListSessions()
	assert.Equal(t, 0, len(sessions), "Expected no sessions initially")
}

// TestSessionManager_CountSessions tests counting sessions
func TestSessionManager_CountSessions(t *testing.T) {
	logger := setupTestLogger(t)

	config := ManagerConfig{
		MaxSessions:        10,
		MaxSessionsPerHost: 5,
		SessionTimeout:     5 * time.Minute,
		IdleTimeout:        2 * time.Minute,
		CleanupInterval:    10 * time.Second,
		Logger:             logger,
	}

	sm := NewSessionManager(config)
	defer sm.Close()

	// 初始计数应该是 0
	count := sm.CountSessions()
	assert.Equal(t, 0, count, "Expected zero sessions initially")
}

// TestSessionManager_CountSessionsPerHost tests counting sessions per host
func TestSessionManager_CountSessionsPerHost(t *testing.T) {
	logger := setupTestLogger(t)

	config := ManagerConfig{
		MaxSessions:        10,
		MaxSessionsPerHost: 5,
		SessionTimeout:     5 * time.Minute,
		IdleTimeout:        2 * time.Minute,
		CleanupInterval:    10 * time.Second,
		Logger:             logger,
	}

	sm := NewSessionManager(config)
	defer sm.Close()

	// 初始计数应该是 0
	count := sm.CountSessionsForHost("localhost")
	assert.Equal(t, 0, count, "Expected zero sessions for localhost initially")
}

// TestSessionManager_RemoveSession tests removing a non-existent session
func TestSessionManager_RemoveSession(t *testing.T) {
	logger := setupTestLogger(t)

	config := ManagerConfig{
		MaxSessions:        10,
		MaxSessionsPerHost: 5,
		SessionTimeout:     5 * time.Minute,
		IdleTimeout:        2 * time.Minute,
		CleanupInterval:    10 * time.Second,
		Logger:             logger,
	}

	sm := NewSessionManager(config)
	defer sm.Close()

	// 尝试删除不存在的会话
	err := sm.RemoveSession("non-existent-id")
	assert.Error(t, err, "Expected error when removing non-existent session")
}

// TestSessionState_String tests SessionState String method
func TestSessionState_String(t *testing.T) {
	tests := []struct {
		state    SessionState
		expected string
	}{
		{SessionStateActive, "active"},
		{SessionStateIdle, "idle"},
		{SessionStateClosed, "closed"},
		{SessionState(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := tt.state.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestAuthConfig_AuthMethod tests AuthConfig.AuthMethod method
func TestAuthConfig_AuthMethod(t *testing.T) {
	tests := []struct {
		name        string
		authConfig  *AuthConfig
		expectError bool
	}{
		{
			name: "password auth",
			authConfig: &AuthConfig{
				Type:     AuthTypePassword,
				Password: "testpass",
			},
			expectError: false,
		},
		{
			name: "private key auth",
			authConfig: &AuthConfig{
				Type:       AuthTypePrivateKey,
				PrivateKey: "invalid-key",
			},
			expectError: true, // 无效的私钥应该返回错误
		},
		{
			name: "ssh agent auth",
			authConfig: &AuthConfig{
				Type: AuthTypeSSHAgent,
			},
			expectError: true, // SSH agent 未实现
		},
		{
			name: "invalid auth type",
			authConfig: &AuthConfig{
				Type: AuthType("invalid"),
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			methods, err := tt.authConfig.AuthMethod()
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, methods)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, methods)
			}
		})
	}
}

// TestSession_GetShellSession tests GetShellSession method
func TestSession_GetShellSession(t *testing.T) {
	session := &Session{
		ID:     "test-id",
		ShellSession: nil,
	}

	// 没有shell会话时应该返回nil
	shell := session.GetShellSession()
	assert.Nil(t, shell, "Expected nil when no shell session")
}
