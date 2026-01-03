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

// TestSessionManager_AliasExists tests checking if an alias exists
func TestSessionManager_AliasExists(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过需要真实 SSH 连接的测试")
	}

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

	// 初始状态别名不存在
	assert.False(t, sm.AliasExists("prod"), "Expected alias 'prod' to not exist initially")

	// 创建带别名的会话
	authConfig := &AuthConfig{
		Type:     AuthTypePassword,
		Password: getEnvOrDefault("SSH_PASSWORD", "root"),
	}

	host := getEnvOrDefault("SSH_HOST", "192.168.68.212")
	username := getEnvOrDefault("SSH_USER", "root")

	session, err := sm.CreateSession(host, 22, username, authConfig, "prod")
	if err != nil {
		t.Skipf("Skipping test: SSH connection failed: %v", err)
		return
	}
	defer sm.RemoveSession(session.ID)

	// 现在别名应该存在
	assert.True(t, sm.AliasExists("prod"), "Expected alias 'prod' to exist")
	assert.False(t, sm.AliasExists("staging"), "Expected alias 'staging' to not exist")
}

// TestSessionManager_GetSessionByAlias tests retrieving a session by alias
func TestSessionManager_GetSessionByAlias(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过需要真实 SSH 连接的测试")
	}

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

	authConfig := &AuthConfig{
		Type:     AuthTypePassword,
		Password: getEnvOrDefault("SSH_PASSWORD", "root"),
	}

	host := getEnvOrDefault("SSH_HOST", "192.168.68.212")
	username := getEnvOrDefault("SSH_USER", "root")

	session, err := sm.CreateSession(host, 22, username, authConfig, "test-alias")
	if err != nil {
		t.Skipf("Skipping test: SSH connection failed: %v", err)
		return
	}
	defer sm.RemoveSession(session.ID)

	// 通过别名获取会话
	retrieved, err := sm.GetSessionByAlias("test-alias")
	assert.NoError(t, err)
	assert.NotNil(t, retrieved)
	assert.Equal(t, session.ID, retrieved.ID)
	assert.Equal(t, "test-alias", retrieved.Alias)

	// 尝试获取不存在的别名
	_, err = sm.GetSessionByAlias("non-existent")
	assert.Error(t, err)
}

// TestSessionManager_GetSessionByIDOrAlias tests retrieving by ID or alias
func TestSessionManager_GetSessionByIDOrAlias(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过需要真实 SSH 连接的测试")
	}

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

	authConfig := &AuthConfig{
		Type:     AuthTypePassword,
		Password: getEnvOrDefault("SSH_PASSWORD", "root"),
	}

	host := getEnvOrDefault("SSH_HOST", "192.168.68.212")
	username := getEnvOrDefault("SSH_USER", "root")

	session, err := sm.CreateSession(host, 22, username, authConfig, "multi")
	if err != nil {
		t.Skipf("Skipping test: SSH connection failed: %v", err)
		return
	}
	defer sm.RemoveSession(session.ID)

	// 通过 ID 获取
	byID, err := sm.GetSessionByIDOrAlias(session.ID)
	assert.NoError(t, err)
	assert.Equal(t, session.ID, byID.ID)

	// 通过别名获取
	byAlias, err := sm.GetSessionByIDOrAlias("multi")
	assert.NoError(t, err)
	assert.Equal(t, session.ID, byAlias.ID)

	// 不存在的会话
	_, err = sm.GetSessionByIDOrAlias("non-existent")
	assert.Error(t, err)
}

// TestSessionManager_GenerateUniqueAlias tests generating unique aliases
func TestSessionManager_GenerateUniqueAlias(t *testing.T) {
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

	// 空的 manager 应该能生成 "s1"（空 base 时默认为 "s" 并从 1 开始）
	alias := sm.GenerateUniqueAlias("")
	assert.Equal(t, "s1", alias)

	// 模拟添加别名为 "s1" 的会话
	sm.sessions.Store("fake-id", &Session{
		ID:     "fake-id",
		Alias:  "s1",
		State:  SessionStateActive,
	})

	// 现在应该生成 "s2"
	alias = sm.GenerateUniqueAlias("")
	assert.Equal(t, "s2", alias)

	// 再添加一个
	sm.sessions.Store("fake-id-2", &Session{
		ID:     "fake-id-2",
		Alias:  "s2",
		State:  SessionStateActive,
	})

	// 应该生成 "s3"
	alias = sm.GenerateUniqueAlias("")
	assert.Equal(t, "s3", alias)

	// 测试自定义 base
	// 模拟添加别名为 "prod1" 的会话
	sm.sessions.Store("fake-prod-1", &Session{
		ID:     "fake-prod-1",
		Alias:  "prod1",
		State:  SessionStateActive,
	})

	// 使用 "prod" 作为 base，应该生成 "prod2"（因为 prod1 已存在）
	alias = sm.GenerateUniqueAlias("prod")
	assert.Equal(t, "prod2", alias)
}

// TestSessionManager_AliasConflict tests alias conflict detection
func TestSessionManager_AliasConflict(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过需要真实 SSH 连接的测试")
	}

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

	authConfig := &AuthConfig{
		Type:     AuthTypePassword,
		Password: getEnvOrDefault("SSH_PASSWORD", "root"),
	}

	host := getEnvOrDefault("SSH_HOST", "192.168.68.212")
	username := getEnvOrDefault("SSH_USER", "root")

	// 创建第一个会话，别名为 "conflict"
	session1, err := sm.CreateSession(host, 22, username, authConfig, "conflict")
	if err != nil {
		t.Skipf("Skipping test: SSH connection failed: %v", err)
		return
	}
	defer sm.RemoveSession(session1.ID)

	// 尝试创建同名别名的会话，应该失败
	_, err = sm.CreateSession(host, 22, username, authConfig, "conflict")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

// TestSessionManager_AutoGenerateAlias tests automatic alias generation
func TestSessionManager_AutoGenerateAlias(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过需要真实 SSH 连接的测试")
	}

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

	authConfig := &AuthConfig{
		Type:     AuthTypePassword,
		Password: getEnvOrDefault("SSH_PASSWORD", "root"),
	}

	host := getEnvOrDefault("SSH_HOST", "192.168.68.212")
	username := getEnvOrDefault("SSH_USER", "root")

	// 创建会话时不指定别名，应该自动生成
	session1, err := sm.CreateSession(host, 22, username, authConfig, "")
	if err != nil {
		t.Skipf("Skipping test: SSH connection failed: %v", err)
		return
	}
	defer sm.RemoveSession(session1.ID)

	// 应该自动生成了别名
	assert.NotEmpty(t, session1.Alias, "Expected auto-generated alias")
	assert.Equal(t, "s1", session1.Alias, "First session should have alias 's1'")

	// 再创建一个，应该生成 s2
	session2, err := sm.CreateSession(host, 22, username, authConfig, "")
	if err != nil {
		t.Skipf("Skipping test: SSH connection failed: %v", err)
		return
	}
	defer sm.RemoveSession(session2.ID)

	assert.Equal(t, "s2", session2.Alias, "Second session should have alias 's2'")
}

// TestSession_AliasField tests that Session struct has Alias field
func TestSession_AliasField(t *testing.T) {
	session := &Session{
		ID:    "test-id",
		Alias: "test-alias",
		Host:  "localhost",
		Port:  22,
		State: SessionStateActive,
	}

	assert.Equal(t, "test-alias", session.Alias)
}

