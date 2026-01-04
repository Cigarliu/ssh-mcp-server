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
	Type         AuthType
	Password     string
	PrivateKey   string // 私钥内容或路径
	Passphrase   string // 私钥密码
	SudoPassword string // sudo密码（可选，用于自动注入sudo密码）
}

// Session represents an SSH session
type Session struct {
	// 基本信息
	ID       string       `json:"session_id"`
	Alias    string       `json:"alias,omitempty"` // 会话别名，简短易记的标识符
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
	Config     *SessionConfig `json:"-"`

	// 命令历史
	CommandHistory []CommandHistoryEntry `json:"command_history"`
	MaxHistorySize int                    `json:"-"` // 最大历史记录数，默认 100

	// 认证配置
	AuthConfig *AuthConfig `json:"-"` // 认证配置（包含sudo密码）

	// 并发控制
	mu sync.RWMutex `json:"-"`
}

// CommandHistoryEntry represents a single command execution history entry
type CommandHistoryEntry struct {
	Command       string        `json:"command"`        // 执行的命令
	ExitCode      int           `json:"exit_code"`      // 退出码
	ExecutionTime time.Duration `json:"execution_time"`  // 执行时长
	Timestamp     time.Time     `json:"timestamp"`      // 执行时间戳
	Success       bool          `json:"success"`        // 是否成功（exit code == 0）
	Source        string        `json:"source"`         // 命令来源: "exec" 或 "shell"
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
	Session        *ssh.Session
	Stdin          io.WriteCloser
	Stdout         io.Reader
	Stderr         io.Reader
	PTY            bool
	TerminalInfo   TerminalInfo
	Config         *ShellConfig // Shell configuration
	mu             sync.Mutex
	// Status tracking
	LastReadTime   time.Time
	LastWriteTime  time.Time
	currentDir     string
	hasUnreadData  bool
}

// TerminalInfo represents terminal information
type TerminalInfo struct {
	Term string // "xterm", "vt100", etc.
	Rows uint16
	Cols uint16
}

// TerminalMode represents the terminal mode
type TerminalMode int

const (
	// TerminalModeCooked (canonical mode) - line buffering, processes special characters
	TerminalModeCooked TerminalMode = iota
	// TerminalModeRaw - pass through input/output without processing
	TerminalModeRaw
)

func (m TerminalMode) String() string {
	switch m {
	case TerminalModeCooked:
		return "cooked"
	case TerminalModeRaw:
		return "raw"
	default:
		return "unknown"
	}
}

// ANSIMode determines how ANSI escape sequences are handled
type ANSIMode int

const (
	// ANSIRaw - pass through ANSI sequences unchanged
	ANSIRaw ANSIMode = iota
	// ANSIStrip - remove ANSI sequences
	ANSIStrip
	// ANSIParse - parse and provide structured data (future)
	ANSIParse
)

func (m ANSIMode) String() string {
	switch m {
	case ANSIRaw:
		return "raw"
	case ANSIStrip:
		return "strip"
	case ANSIParse:
		return "parse"
	default:
		return "unknown"
	}
}

// ShellConfig configures the shell session behavior
type ShellConfig struct {
	// Terminal mode (raw or cooked)
	Mode TerminalMode
	// ANSI escape sequence handling
	ANSIMode ANSIMode
	// Read timeout for non-blocking reads
	ReadTimeout time.Duration
	// Write timeout
	WriteTimeout time.Duration
	// Whether to auto-detect interactive programs
	AutoDetectInteractive bool
}

// ShellStatus represents the current status of a shell session
type ShellStatus struct {
	IsActive      bool      `json:"is_active"`       // Shell 是否活动
	CurrentDir    string    `json:"current_dir"`     // 当前工作目录
	HasUnreadOutput bool    `json:"has_unread_output"` // 是否有未读取的输出
	LastReadTime  time.Time `json:"last_read_time"`  // 最后读取时间
	LastWriteTime time.Time `json:"last_write_time"` // 最后写入时间
	TerminalType  string    `json:"terminal_type"`   // 终端类型
	Rows          uint16    `json:"rows"`            // 终端行数
	Cols          uint16    `json:"cols"`            // 终端列数
	Mode          string    `json:"mode"`            // 终端模式 (cooked/raw)
	ANSIMode      string    `json:"ansi_mode"`       // ANSI 处理模式
}

// DefaultShellConfig returns default configuration
func DefaultShellConfig() *ShellConfig {
	return &ShellConfig{
		Mode:                  TerminalModeCooked,
		ANSIMode:              ANSIStrip, // 默认使用 strip 模式，AI 友好
		ReadTimeout:           100 * time.Millisecond,
		WriteTimeout:          5 * time.Second,
		AutoDetectInteractive: true,
	}
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
	// 新增：进度和统计信息
	FileSize      int64   `json:"file_size,omitempty"`      // 文件总大小（字节）
	Progress      float64 `json:"progress,omitempty"`       // 进度百分比 (0-100)
	Speed         string  `json:"speed,omitempty"`          // 传输速度（如 "1.5 MB/s"）
	FilePath      string  `json:"file_path,omitempty"`      // 文件路径
	Operation     string  `json:"operation,omitempty"`      // 操作类型 ("upload" 或 "download")
}

// FileInfo represents file information for SFTP
type FileInfo struct {
	Name     string    `json:"name"`
	Type     string    `json:"type"`
	Size     int64     `json:"size"`
	Mode     string    `json:"mode"`
	Modified time.Time `json:"modified"`
}
