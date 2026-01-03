package config

import (
	"fmt"
	"time"

	"github.com/cigar/sshmcp/internal/logger"
	"github.com/rs/zerolog"
	"github.com/spf13/viper"
)

// Config represents the application configuration
type Config struct {
	Server  ServerConfig  `mapstructure:"server"`
	SSH     SSHConfig     `mapstructure:"ssh"`
	Session SessionConfig `mapstructure:"session"`
	SFTP    SFTPConfig    `mapstructure:"sftp"`
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

// LoadConfig loads the configuration from file and environment variables
func LoadConfig(configPath string) (*Config, error) {
	// 设置默认值
	setDefaults()

	// 读取配置文件
	if configPath != "" {
		viper.SetConfigFile(configPath)
	} else {
		// 查找配置文件
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
		viper.AddConfigPath(".")
		viper.AddConfigPath("/etc/sshmcp/")
		viper.AddConfigPath("$HOME/.sshmcp/")
	}

	// 环境变量
	viper.SetEnvPrefix("SSHMCP")
	viper.AutomaticEnv()

	// 读取配置
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// 配置文件未找到，使用默认值
			fmt.Println("Config file not found, using defaults")
		} else {
			return nil, fmt.Errorf("read config: %w", err)
		}
	}

	// 解析配置
	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	return &config, nil
}

// setDefaults sets the default configuration values
func setDefaults() {
	// Server
	viper.SetDefault("server.name", "ssh-mcp-server")
	viper.SetDefault("server.version", "1.0.0")

	// SSH
	viper.SetDefault("ssh.default_port", 22)
	viper.SetDefault("ssh.timeout", "30s")
	viper.SetDefault("ssh.keepalive_interval", "30s")

	// Session
	viper.SetDefault("session.max_sessions", 100)
	viper.SetDefault("session.max_sessions_per_host", 10)
	viper.SetDefault("session.idle_timeout", "10m")
	viper.SetDefault("session.session_timeout", "30m")
	viper.SetDefault("session.cleanup_interval", "1m")

	// SFTP
	viper.SetDefault("sftp.max_file_size", "1GB")
	viper.SetDefault("sftp.chunk_size", "4MB")
	viper.SetDefault("sftp.transfer_timeout", "5m")

	// Logging
	viper.SetDefault("logging.level", "info")
	viper.SetDefault("logging.format", "console")
	viper.SetDefault("logging.output", "stdout")
}

// GetLogger creates a logger from the logging configuration
func (c *Config) GetLogger() (*zerolog.Logger, error) {
	return logger.NewLogger(c.Logging)
}
