package logger

import (
	"io"
	"os"
	"path/filepath"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Config represents the logger configuration
type Config struct {
 Level  string // debug, info, warn, error
 Format string // json, console
 Output string // stdout, stderr, or file path
}

// NewLogger creates a new logger
func NewLogger(config Config) (*zerolog.Logger, error) {
	// 设置日志级别
	level, err := zerolog.ParseLevel(config.Level)
	if err != nil {
		level = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(level)

	// 设置时间格式
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnixMs

	// 设置输出
	var output io.Writer
	switch config.Output {
	case "stdout":
		output = os.Stdout
	case "stderr":
		output = os.Stderr
	default:
		// 输出到文件
		if err := os.MkdirAll(filepath.Dir(config.Output), 0755); err != nil {
			return nil, err
		}
		file, err := os.OpenFile(config.Output, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return nil, err
		}
		output = file
	}

	// 设置格式
	if config.Format == "console" {
		output = zerolog.ConsoleWriter{Out: output}
	}

	logger := zerolog.New(output).With().Timestamp().Logger()

	return &logger, nil
}

// NewDefaultLogger creates a logger with default configuration
func NewDefaultLogger() *zerolog.Logger {
	return &log.Logger
}
