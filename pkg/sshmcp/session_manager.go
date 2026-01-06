package sshmcp

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/sftp"
	"github.com/rs/zerolog"
)

// ManagerConfig represents the session manager configuration
type ManagerConfig struct {
	// 会话限制
	MaxSessions        int
	MaxSessionsPerHost int

	// 超时配置
	SessionTimeout  time.Duration
	IdleTimeout     time.Duration

	// 清理配置
	CleanupInterval time.Duration

	// 日志
	Logger *zerolog.Logger
}

// DefaultManagerConfig returns the default manager configuration
func DefaultManagerConfig(logger *zerolog.Logger) ManagerConfig {
	return ManagerConfig{
		MaxSessions:        100,
		MaxSessionsPerHost: 30,
		SessionTimeout:     30 * time.Minute,
		IdleTimeout:        10 * time.Minute,
		CleanupInterval:    1 * time.Minute,
		Logger:             logger,
	}
}

// SessionManager manages SSH sessions
type SessionManager struct {
	// 会话存储（并发安全）
	sessions sync.Map // map[string]*Session

	// 配置
	config ManagerConfig

	// 清理
	done chan struct{}
	wg   sync.WaitGroup
}

// NewSessionManager creates a new session manager
func NewSessionManager(config ManagerConfig) *SessionManager {
	sm := &SessionManager{
		sessions: sync.Map{},
		config:   config,
		done:     make(chan struct{}),
	}

	// 启动定期清理 goroutine
	sm.wg.Add(1)
	go sm.cleanupLoop()

	return sm
}

// CreateSession creates a new SSH session
func (sm *SessionManager) CreateSession(host string, port int, username string, authConfig *AuthConfig, alias string) (*Session, error) {
	// 检查是否超过最大会话数
	if count := sm.CountSessions(); count >= sm.config.MaxSessions {
		return nil, fmt.Errorf("maximum sessions limit reached: %d", sm.config.MaxSessions)
	}

	// 检查每个主机的最大会话数
	if count := sm.CountSessionsForHost(host); count >= sm.config.MaxSessionsPerHost {
		return nil, fmt.Errorf("maximum sessions per host limit reached: %d for host %s", sm.config.MaxSessionsPerHost, host)
	}

	// 检查别名是否已存在，并进行健康检查
	if alias != "" && sm.AliasExists(alias) {
		existingSession, isHealthy, err := sm.GetSessionByAliasWithHealthCheck(alias)
		if err == nil {
			if isHealthy {
				// 会话仍然活跃，返回友好提示
				return nil, fmt.Errorf("alias '%s' is already in use by an active session (ID: %s, Host: %s:%d). Please use a different alias or disconnect the existing session first",
					alias, existingSession.ID, existingSession.Host, existingSession.Port)
			} else {
				// 会话已断开，自动清理
				sm.config.Logger.Info().
					Str("session_id", existingSession.ID).
					Str("alias", alias).
					Str("host", existingSession.Host).
					Msg("Detected unhealthy session with same alias, cleaning up for reconnection")

				// 清理旧会话
				if err := sm.RemoveSession(existingSession.ID); err != nil {
					sm.config.Logger.Warn().
						Str("session_id", existingSession.ID).
						Err(err).
						Msg("Failed to remove unhealthy session, continuing anyway")
				}
			}
		}
	}

	// 如果没有指定别名，自动生成一个
	if alias == "" {
		alias = sm.GenerateUniqueAlias("")
	}

	sessionID := uuid.New().String()

	// 创建 SSH 客户端
	client, err := CreateSSHClient(host, port, username, authConfig, 30*time.Second)
	if err != nil {
		return nil, fmt.Errorf("create SSH client: %w", err)
	}

	// 创建 SFTP 客户端
	sftpClient, err := sftp.NewClient(client)
	if err != nil {
		client.Close()
		return nil, fmt.Errorf("create SFTP client: %w", err)
	}

	// 创建会话配置
	config := &SessionConfig{
		Timeout:          30 * time.Second,
		KeepAliveInterval: 30 * time.Second,
		CommandTimeout:   30 * time.Second,
		MaxRetries:       3,
		MaxIdleTime:      sm.config.IdleTimeout,
		AutoReconnect:    false,
	}

	session := &Session{
		ID:          sessionID,
		Alias:       alias,
		Host:        host,
		Port:        port,
		Username:    username,
		State:       SessionStateActive,
		SSHClient:   client,
		SFTPClient:  sftpClient,
		CreatedAt:   time.Now(),
		LastUsedAt:  time.Now(),
		ExpiresAt:   time.Now().Add(sm.config.SessionTimeout),
		Config:      config,
		AuthConfig:  authConfig, // 保存认证配置（包含sudo密码）
	}

	// 存储会话
	sm.sessions.Store(sessionID, session)

	sm.config.Logger.Info().
		Str("session_id", sessionID).
		Str("alias", alias).
		Str("host", host).
		Int("port", port).
		Str("username", username).
		Msg("Created new SSH session")

	return session, nil
}

// GetSession retrieves a session by ID
func (sm *SessionManager) GetSession(sessionID string) (*Session, error) {
	val, ok := sm.sessions.Load(sessionID)
	if !ok {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	session := val.(*Session)
	session.mu.Lock()
	defer session.mu.Unlock()

	// 检查会话是否已关闭
	if session.State == SessionStateClosed {
		return nil, fmt.Errorf("session is closed: %s", sessionID)
	}

	// 更新最后使用时间
	session.LastUsedAt = time.Now()

	return session, nil
}

// RemoveSession removes and closes a session
func (sm *SessionManager) RemoveSession(sessionID string) error {
	val, ok := sm.sessions.Load(sessionID)
	if !ok {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	session := val.(*Session)
	session.mu.Lock()
	defer session.mu.Unlock()

	// 关闭 SFTP 客户端
	if session.SFTPClient != nil {
		if err := session.SFTPClient.Close(); err != nil {
			sm.config.Logger.Error().
				Str("session_id", sessionID).
				Err(err).
				Msg("Failed to close SFTP client")
		}
	}

	// 关闭 Shell 会话
	if session.ShellSession != nil {
		if err := session.ShellSession.Close(); err != nil {
			sm.config.Logger.Error().
				Str("session_id", sessionID).
				Err(err).
				Msg("Failed to close shell session")
		}
	}

	// 关闭 SSH 连接
	if session.SSHClient != nil {
		if err := session.SSHClient.Close(); err != nil {
			sm.config.Logger.Error().
				Str("session_id", sessionID).
				Err(err).
				Msg("Failed to close SSH client")
		}
	}

	session.State = SessionStateClosed
	sm.sessions.Delete(sessionID)

	sm.config.Logger.Info().
		Str("session_id", sessionID).
		Str("host", session.Host).
		Msg("Removed SSH session")

	return nil
}

// ListSessions returns all active sessions
func (sm *SessionManager) ListSessions() []*Session {
	var sessions []*Session

	sm.sessions.Range(func(key, value interface{}) bool {
		session := value.(*Session)
		session.mu.RLock()
		defer session.mu.RUnlock()

		if session.State != SessionStateClosed {
			sessions = append(sessions, session)
		}
		return true
	})

	return sessions
}

// CountSessions returns the total number of sessions
func (sm *SessionManager) CountSessions() int {
	count := 0
	sm.sessions.Range(func(key, value interface{}) bool {
		session := value.(*Session)
		session.mu.RLock()
		defer session.mu.RUnlock()

		if session.State != SessionStateClosed {
			count++
		}
		return true
	})
	return count
}

// CountSessionsForHost returns the number of sessions for a specific host
func (sm *SessionManager) CountSessionsForHost(host string) int {
	count := 0
	sm.sessions.Range(func(key, value interface{}) bool {
		session := value.(*Session)
		session.mu.RLock()
		defer session.mu.RUnlock()

		if session.State != SessionStateClosed && session.Host == host {
			count++
		}
		return true
	})
	return count
}

// cleanupLoop runs the periodic cleanup loop
func (sm *SessionManager) cleanupLoop() {
	defer sm.wg.Done()

	ticker := time.NewTicker(sm.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			sm.cleanupExpiredSessions()
		case <-sm.done:
			return
		}
	}
}

// cleanupExpiredSessions removes expired and idle sessions
func (sm *SessionManager) cleanupExpiredSessions() {
	now := time.Now()

	sm.sessions.Range(func(key, value interface{}) bool {
		sessionID := key.(string)
		session := value.(*Session)

		session.mu.RLock()
		idle := now.Sub(session.LastUsedAt) > sm.config.IdleTimeout
		expired := now.After(session.ExpiresAt)
		session.mu.RUnlock()

		if idle || expired {
			sm.config.Logger.Info().
				Str("session_id", sessionID).
				Str("host", session.Host).
				Bool("idle", idle).
				Bool("expired", expired).
				Msg("Cleaning up session")

			sm.RemoveSession(sessionID)
		}

		return true
	})
}

// Close closes the session manager and all sessions
func (sm *SessionManager) Close() {
	close(sm.done)
	sm.wg.Wait()

	// 关闭所有会话
	sm.sessions.Range(func(key, value interface{}) bool {
		sessionID := key.(string)
		sm.RemoveSession(sessionID)
		return true
	})

	sm.config.Logger.Info().Msg("Session manager closed")
}

// AliasExists checks if an alias already exists
func (sm *SessionManager) AliasExists(alias string) bool {
	if alias == "" {
		return false
	}

	exists := false
	sm.sessions.Range(func(key, value interface{}) bool {
		session := value.(*Session)
		session.mu.RLock()
		defer session.mu.RUnlock()

		if session.Alias == alias && session.State != SessionStateClosed {
			exists = true
			return false
		}
		return true
	})

	return exists
}

// GetSessionByAlias retrieves a session by alias
func (sm *SessionManager) GetSessionByAlias(alias string) (*Session, error) {
	if alias == "" {
		return nil, fmt.Errorf("alias cannot be empty")
	}

	var result *Session
	sm.sessions.Range(func(key, value interface{}) bool {
		session := value.(*Session)
		session.mu.RLock()
		defer session.mu.RUnlock()

		if session.Alias == alias && session.State != SessionStateClosed {
			result = session
			return false
		}
		return true
	})

	if result == nil {
		return nil, fmt.Errorf("session not found with alias: %s", alias)
	}

	// 更新最后使用时间
	result.mu.Lock()
	result.LastUsedAt = time.Now()
	result.mu.Unlock()

	return result, nil
}

// GetSessionByIDOrAlias retrieves a session by ID or alias
func (sm *SessionManager) GetSessionByIDOrAlias(idOrAlias string) (*Session, error) {
	// 先尝试按 ID 查找
	if session, err := sm.GetSession(idOrAlias); err == nil {
		return session, nil
	}

	// 再尝试按 alias 查找
	return sm.GetSessionByAlias(idOrAlias)
}

// GenerateUniqueAlias generates a unique short alias
func (sm *SessionManager) GenerateUniqueAlias(base string) string {
	// 如果没有提供基础别名，使用 "s1" 作为起始
	if base == "" {
		base = "s"
	}

	// 尝试 s1, s2, s3, ... 格式
	for i := 1; i <= 1000; i++ {
		candidate := fmt.Sprintf("%s%d", base, i)
		if !sm.AliasExists(candidate) {
			return candidate
		}
	}

	// 如果还是冲突（极端情况），使用 UUID 前缀
	return fmt.Sprintf("s-%s", uuid.New().String()[:8])
}

// IsSessionHealthy checks if a session is still healthy (connected and responsive)
func (sm *SessionManager) IsSessionHealthy(session *Session) bool {
	session.mu.RLock()
	defer session.mu.RUnlock()

	// 检查会话状态
	if session.State == SessionStateClosed {
		return false
	}

	// 检查 SSH 客户端是否存在
	if session.SSHClient == nil {
		return false
	}

	// 尝试创建一个新的 session 来测试连接
	// 这是一个轻量级的健康检查
	testSession, err := session.SSHClient.NewSession()
	if err != nil {
		sm.config.Logger.Debug().
			Str("session_id", session.ID).
			Str("alias", session.Alias).
			Err(err).
			Msg("Session health check failed: cannot create new session")
		return false
	}
	testSession.Close()

	return true
}

// GetSessionByAliasWithHealthCheck retrieves a session by alias and checks its health
func (sm *SessionManager) GetSessionByAliasWithHealthCheck(alias string) (*Session, bool, error) {
	if alias == "" {
		return nil, false, fmt.Errorf("alias cannot be empty")
	}

	var result *Session
	sm.sessions.Range(func(key, value interface{}) bool {
		session := value.(*Session)
		session.mu.RLock()
		defer session.mu.RUnlock()

		if session.Alias == alias && session.State != SessionStateClosed {
			result = session
			return false
		}
		return true
	})

	if result == nil {
		return nil, false, fmt.Errorf("session not found with alias: %s", alias)
	}

	// 检查会话健康状态
	isHealthy := sm.IsSessionHealthy(result)

	return result, isHealthy, nil
}
