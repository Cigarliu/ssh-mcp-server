package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/cigar/sshmcp/pkg/mcp"
	"github.com/cigar/sshmcp/pkg/sshmcp"
	"github.com/rs/zerolog/log"
)

func main() {
	// 加载配置（自动发现配置文件）
	cfg, configPath, err := loadConfigWithPath()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// 创建 logger
	logger, err := cfg.GetLogger()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create logger: %v\n", err)
		os.Exit(1)
	}

	log.Logger = *logger

	log.Info().
		Str("version", cfg.Server.Version).
		Msg("Starting SSH MCP Server")

	// 创建会话管理器
	managerConfig := sshmcp.ManagerConfig{
		MaxSessions:        cfg.Session.MaxSessions,
		MaxSessionsPerHost: cfg.Session.MaxSessionsPerHost,
		SessionTimeout:     cfg.Session.SessionTimeout,
		IdleTimeout:        cfg.Session.IdleTimeout,
		CleanupInterval:    cfg.Session.CleanupInterval,
		Logger:             logger,
	}

	sessionManager := sshmcp.NewSessionManager(managerConfig)
	defer sessionManager.Close()

	// 转换配置中的 hosts 为 sshmcp.HostConfig
	hostsConfig := make(map[string]sshmcp.HostConfig)
	for name, hostCfg := range cfg.Hosts {
		hostsConfig[name] = sshmcp.HostConfig{
			Host:           hostCfg.Host,
			Port:           hostCfg.Port,
			Username:       hostCfg.Username,
			Password:       hostCfg.Password,
			PrivateKeyPath: hostCfg.PrivateKeyPath,
			Description:    hostCfg.Description,
		}
	}

	// 创建主机管理器
	hostManager := sshmcp.NewHostManager(hostsConfig, configPath, logger)

	// 创建 MCP 服务器
	mcpServer, err := mcp.NewServer(sessionManager, hostManager, logger)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create MCP server")
	}

	// 设置信号处理
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Info().Msg("Received shutdown signal")
		cancel()
	}()

	// 启动服务器
	log.Info().Msg("MCP server is running")
	if err := mcpServer.Start(ctx); err != nil {
		log.Fatal().Err(err).Msg("MCP server error")
	}

	log.Info().Msg("Server shutdown complete")
}
