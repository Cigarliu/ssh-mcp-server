package sshmcp

import (
	"io"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
	"github.com/pkg/sftp"
)

// SessionState represents the state of a session
type SessionState int

const (
	SessionStateActive SessionState = iota
	SessionStateIdle
	SessionStateClosed
)

func (s SessionState) String() string {
	switch s {
	case SessionStateActive:
		return "active"
	case SessionStateIdle:
		return "idle"
	case SessionStateClosed:
		return "closed"
	default:
		return "unknown"
	}
}

// AuthType represents the authentication type
type AuthType string

const (
	AuthTypePassword   AuthType = "password"
	AuthTypePrivateKey AuthType = "private_key"
	AuthTypeSSHAgent   AuthType = "ssh_agent"
	AuthTypeKeyboard   AuthType = "keyboard"
)

// AuthConfig represents authentication configuration
type AuthConfig struct {
	Type       AuthType
	Password   string
	PrivateKey string // 私钥内容或路径
	Passphrase string // 私钥密码
}

// Session represents an SSH session
type Session struct {
	// 基本信息
	ID       string       `json:"session_id"`
	Host     string       `json:"host"`
	Port     int          `json:"port"`
	Username string       `json:"username"`
	State    SessionState `json:"state"`

	// 客户端连接
	SSHClient  *ssh.Client  `json:"-"`
	SFTPClient *sftp.Client `json:"-"`

	// 交互式 Shell
	ShellSession *SSHShellSession `json:"-"`

	// 时间戳
	CreatedAt  time.Time `json:"created_at"`
	LastUsedAt time.Time `json:"last_used_at"`
	ExpiresAt  time.Time `json:"expires_at"`

	// 配置
	Config *SessionConfig `json:"-"`

	// 并发控制
	mu sync.RWMutex `json:"-"`
}

// GetShellSession returns the shell session (used by mcp package)
func (s *Session) GetShellSession() *SSHShellSession {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.ShellSession
}

// RLock acquires a read lock on the session (used by mcp package)
func (s *Session) RLock() {
	s.mu.RLock()
}

// RUnlock releases a read lock on the session (used by mcp package)
func (s *Session) RUnlock() {
	s.mu.RUnlock()
}

// SessionConfig represents session configuration
type SessionConfig struct {
	// 连接配置
	Timeout          time.Duration
	KeepAliveInterval time.Duration

	// 执行配置
	CommandTimeout time.Duration
	MaxRetries     int

	// 安全配置
	MaxIdleTime   time.Duration
	AutoReconnect bool
}

// SSHShellSession represents an interactive shell session
type SSHShellSession struct {
	Session      *ssh.Session
	Stdin        io.WriteCloser
	Stdout       io.Reader
	Stderr       io.Reader
	PTY          bool
	TerminalInfo TerminalInfo
	mu           sync.Mutex
}

// TerminalInfo represents terminal information
type TerminalInfo struct {
	Term string // "xterm", "vt100", etc.
	Rows uint16
	Cols uint16
}

// CommandResult represents the result of a command execution
type CommandResult struct {
	ExitCode     int    `json:"exit_code"`
	Stdout       string `json:"stdout"`
	Stderr       string `json:"stderr"`
	ExecutionTime string `json:"execution_time"`
	Error        error  `json:"error,omitempty"`
}

// FileTransferResult represents the result of a file transfer
type FileTransferResult struct {
	Status           string  `json:"status"`
	BytesTransferred int64   `json:"bytes_transferred"`
	Duration         string  `json:"duration"`
	Error            error   `json:"error,omitempty"`
}

// FileInfo represents file information for SFTP
type FileInfo struct {
	Name     string    `json:"name"`
	Type     string    `json:"type"`
	Size     int64     `json:"size"`
	Mode     string    `json:"mode"`
	Modified time.Time `json:"modified"`
}
