package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/cigar/sshmcp/internal/logger"
	"github.com/rs/zerolog"
	"github.com/spf13/viper"
)

const defaultConfig = `# SSH MCP Server Configuration
# Generated automatically on first run

server:
  name: "ssh-mcp-server"
  version: "1.0.0"

ssh:
  default_port: 22
  timeout: 30s
  keepalive_interval: 30s

session:
  max_sessions: 100
  max_sessions_per_host: 10
  idle_timeout: 10m
  session_timeout: 30m
  cleanup_interval: 1m

sftp:
  max_file_size: 1073741824  # 1GB in bytes
  chunk_size: 4194304        # 4MB in bytes
  transfer_timeout: 5m

# files exposes the ten primary remote-operation tools, including compact SFTP tools. Use core for SSH and terminal-only access or advanced for all tools.
tools:
  profile: files

# Serial devices are opened on demand with connection_open(transport="serial").
# On Linux, run the server as an account permitted to open the device (commonly
# the dialout group), or launch the service with the necessary device access.

# Shared SQLite state for persistent connection IDs and execution history.
state:
  database_path: ""

# Compatibility import source. Each key becomes a persistent connection_id when
# absent from SQLite. New connections should be registered via connection_open
# with connection_id and description, then reopened by ID only.
hosts:
  # Example:
  # prod:
  #   host: "192.168.1.100"
  #   port: 22
  #   username: "root"
  #   password: "your-password"
  #   description: "Production server"

logging:
  level: info  # debug, info, warn, error
  format: console  # json, console
  output: stderr  # stdout is reserved for MCP protocol messages
`

// Config represents the application configuration
type Config struct {
	Server  ServerConfig  `mapstructure:"server"`
	SSH     SSHConfig     `mapstructure:"ssh"`
	Session SessionConfig `mapstructure:"session"`
	SFTP    SFTPConfig    `mapstructure:"sftp"`
	Tools   ToolConfig    `mapstructure:"tools"`
	Hosts   HostsConfig   `mapstructure:"hosts"`
	State   StateConfig   `mapstructure:"state"`
	Logging logger.Config `mapstructure:"logging"`
}

// ServerConfig represents the server configuration
type ServerConfig struct {
	Name    string `mapstructure:"name"`
	Version string `mapstructure:"version"`
}

// SSHConfig represents the SSH configuration
type SSHConfig struct {
	DefaultPort       int           `mapstructure:"default_port"`
	Timeout           time.Duration `mapstructure:"timeout"`
	KeepAliveInterval time.Duration `mapstructure:"keepalive_interval"`
}

// SessionConfig represents the session configuration
type SessionConfig struct {
	MaxSessions        int           `mapstructure:"max_sessions"`
	MaxSessionsPerHost int           `mapstructure:"max_sessions_per_host"`
	IdleTimeout        time.Duration `mapstructure:"idle_timeout"`
	SessionTimeout     time.Duration `mapstructure:"session_timeout"`
	CleanupInterval    time.Duration `mapstructure:"cleanup_interval"`
}

// SFTPConfig represents the SFTP configuration
type SFTPConfig struct {
	MaxFileSize     int64         `mapstructure:"max_file_size"`
	ChunkSize       int64         `mapstructure:"chunk_size"`
	TransferTimeout time.Duration `mapstructure:"transfer_timeout"`
}

// ToolConfig controls which MCP tools are exposed to the client.
type ToolConfig struct {
	Profile string `mapstructure:"profile"`
}

// StateConfig controls the local SQLite state shared by MCP instances.
type StateConfig struct {
	DatabasePath string `mapstructure:"database_path"`
}

// HostConfig represents a predefined SSH host configuration
type HostConfig struct {
	Host           string `mapstructure:"host"`
	Port           int    `mapstructure:"port"`
	Username       string `mapstructure:"username"`
	Password       string `mapstructure:"password,omitempty"`
	PrivateKeyPath string `mapstructure:"private_key_path,omitempty"`
	Description    string `mapstructure:"description,omitempty"`
}

// HostsConfig represents the predefined hosts configuration
type HostsConfig map[string]HostConfig

// LoadConfig loads the configuration from file and environment variables.
func LoadConfig(configPath string) (*Config, error) {
	cfg, _, err := LoadConfigWithPath(configPath)
	return cfg, err
}

// LoadConfigWithPath loads configuration and returns the file path that was used.
// If configPath is empty and no config file is found, it creates
// ~/.sshmcp/config.yaml. If configPath points to a missing file, it creates that
// file instead. Existing files are never overwritten.
func LoadConfigWithPath(configPath string) (*Config, string, error) {
	v := viper.New()
	setDefaults(v)

	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
		v.AddConfigPath("/etc/sshmcp/")

		homeDir, err := os.UserHomeDir()
		if err == nil {
			v.AddConfigPath(filepath.Join(homeDir, ".sshmcp"))
			if homeEnv := os.Getenv("HOME"); homeEnv != "" {
				v.AddConfigPath(filepath.Join(homeEnv, ".sshmcp"))
			}
		}
	}

	v.SetEnvPrefix("SSHMCP")
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		if !isConfigNotFound(err) {
			return nil, "", fmt.Errorf("read config: %w", err)
		}

		fmt.Fprintln(os.Stderr, "No configuration file found, generating default config...")
		defaultConfigPath, err := generateDefaultConfig(configPath)
		if err != nil {
			return nil, "", fmt.Errorf("generate default config: %w", err)
		}

		fmt.Fprintf(os.Stderr, "Generated default configuration at: %s\n", defaultConfigPath)
		fmt.Fprintln(os.Stderr, "You can edit this file to customize the settings")

		v.SetConfigFile(defaultConfigPath)
		if readErr := v.ReadInConfig(); readErr != nil {
			return nil, "", fmt.Errorf("read generated config: %w", readErr)
		}
	}

	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, "", fmt.Errorf("unmarshal config: %w", err)
	}
	if config.State.DatabasePath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, "", fmt.Errorf("get home directory for state database: %w", err)
		}
		config.State.DatabasePath = filepath.Join(homeDir, ".sshmcp", "state.db")
	}

	return &config, v.ConfigFileUsed(), nil
}

// DefaultConfigPath returns the default auto-generated configuration path.
func DefaultConfigPath() (string, error) {
	return defaultConfigPath()
}

func generateDefaultConfig(configPath string) (string, error) {
	configFile := configPath
	if configFile == "" {
		defaultPath, err := defaultConfigPath()
		if err != nil {
			return "", err
		}
		configFile = defaultPath
	}

	configFile, err := filepath.Abs(configFile)
	if err != nil {
		return "", fmt.Errorf("resolve config path: %w", err)
	}

	if _, err := os.Stat(configFile); err == nil {
		return configFile, nil
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("stat config file: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(configFile), 0755); err != nil {
		return "", fmt.Errorf("create config directory: %w", err)
	}

	if err := os.WriteFile(configFile, []byte(defaultConfig), 0644); err != nil {
		return "", fmt.Errorf("write config file: %w", err)
	}

	return configFile, nil
}

func defaultConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home directory: %w", err)
	}
	return filepath.Join(homeDir, ".sshmcp", "config.yaml"), nil
}

func isConfigNotFound(err error) bool {
	if _, ok := err.(viper.ConfigFileNotFoundError); ok {
		return true
	}
	return os.IsNotExist(err)
}

// setDefaults sets the default configuration values.
func setDefaults(v *viper.Viper) {
	v.SetDefault("server.name", "ssh-mcp-server")
	v.SetDefault("server.version", "1.0.0")
	v.SetDefault("ssh.default_port", 22)
	v.SetDefault("ssh.timeout", "30s")
	v.SetDefault("ssh.keepalive_interval", "30s")
	v.SetDefault("session.max_sessions", 100)
	v.SetDefault("session.max_sessions_per_host", 10)
	v.SetDefault("session.idle_timeout", "10m")
	v.SetDefault("session.session_timeout", "30m")
	v.SetDefault("session.cleanup_interval", "1m")
	v.SetDefault("sftp.max_file_size", int64(1073741824))
	v.SetDefault("sftp.chunk_size", int64(4194304))
	v.SetDefault("sftp.transfer_timeout", "5m")
	v.SetDefault("tools.profile", "files")
	v.SetDefault("state.database_path", "")
	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.format", "console")
	v.SetDefault("logging.output", "stderr")
}

// GetLogger creates a logger from the logging configuration.
func (c *Config) GetLogger() (*zerolog.Logger, error) {
	return logger.NewLogger(c.Logging)
}

// GetMCPLogger creates a logger that preserves stdout for the MCP protocol.
func (c *Config) GetMCPLogger() (*zerolog.Logger, error) {
	return logger.NewMCPLogger(c.Logging)
}
