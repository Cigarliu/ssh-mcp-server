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
		MaxSessionsPerHost: 10,
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
func (sm *SessionManager) CreateSession(host string, port int, username string, authConfig *AuthConfig) (*Session, error) {
	// 检查是否超过最大会话数
	if count := sm.CountSessions(); count >= sm.config.MaxSessions {
		return nil, fmt.Errorf("maximum sessions limit reached: %d", sm.config.MaxSessions)
	}

	// 检查每个主机的最大会话数
	if count := sm.CountSessionsForHost(host); count >= sm.config.MaxSessionsPerHost {
		return nil, fmt.Errorf("maximum sessions per host limit reached: %d for host %s", sm.config.MaxSessionsPerHost, host)
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
	}

	// 存储会话
	sm.sessions.Store(sessionID, session)

	sm.config.Logger.Info().
		Str("session_id", sessionID).
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
