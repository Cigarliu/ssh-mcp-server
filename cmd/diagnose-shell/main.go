package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/cigar/sshmcp/pkg/sshmcp"
	"github.com/rs/zerolog"
)

func main() {
	// 创建 logger
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()

	// 创建会话管理器
	config := sshmcp.ManagerConfig{
		MaxSessions:       10,
		SessionTimeout:    10 * time.Minute,
		IdleTimeout:       5 * time.Minute,
		CleanupInterval:   1 * time.Minute,
		Logger:            &logger,
	}
	sm := sshmcp.NewSessionManager(config)
	defer sm.Close()

	// 连接到服务器
	fmt.Println("=== 连接到服务器 ===")
	session, err := sm.CreateSession("192.168.3.7", 22, "cigar", &sshmcp.AuthConfig{
		Type:     sshmcp.AuthTypePassword,
		Password: "liuxuejia.123",
	}, "diagnose")
	if err != nil {
		logger.Fatal().Err(err).Msg("连接失败")
	}
	defer sm.RemoveSession(session.ID)

	fmt.Printf("✅ 连接成功: %s\n\n", session.ID)

	// 创建 Shell
	fmt.Println("=== 创建 Shell ===")
	shellConfig := sshmcp.DefaultShellConfig()
	shell, err := session.CreateShellWithConfig("xterm-256color", 24, 80, shellConfig)
	if err != nil {
		logger.Fatal().Err(err).Msg("创建 Shell 失败")
	}
	defer shell.Close()

	fmt.Println("✅ Shell 已创建")
	fmt.Printf("  缓冲区容量: %d 行\n", shell.BufferSize)
	fmt.Printf("  终端: %dx%d\n\n", shell.TerminalInfo.Cols, shell.TerminalInfo.Rows)

	// 等待 Shell 初始化
	time.Sleep(500 * time.Millisecond)

	// 检查初始状态
	fmt.Println("=== 初始状态 ===")
	status := shell.GetStatus()
	fmt.Printf("  活动: %v\n", status.IsActive)
	fmt.Printf("  缓冲区使用: %d/%d 行\n\n", status.BufferUsed, status.BufferTotal)

	// 发送命令
	fmt.Println("=== 发送命令: echo 'Hello' ===")
	err = shell.WriteInput("echo 'Hello from Shell'\n")
	if err != nil {
		logger.Fatal().Err(err).Msg("写入输入失败")
	}
	fmt.Println("✅ 命令已发送\n")

	// 等待输出
	time.Sleep(1 * time.Second)

	// 读取输出
	fmt.Println("=== 读取输出（非阻塞）===")
	stdout, stderr, err := shell.ReadOutputNonBlocking(200 * time.Millisecond)
	if err != nil {
		logger.Error().Err(err).Msg("读取输出失败")
	}
	fmt.Printf("  stdout 字节数: %d\n", len(stdout))
	fmt.Printf("  stderr 字节数: %d\n", len(stderr))
	if stdout != "" {
		fmt.Printf("  内容:\n%s\n", stdout)
	} else {
		fmt.Println("  ⚠️ 无输出")
	}
	fmt.Println()

	// 再次检查状态
	fmt.Println("=== 命令后状态 ===")
	status = shell.GetStatus()
	fmt.Printf("  活动: %v\n", status.IsActive)
	fmt.Printf("  缓冲区使用: %d/%d 行\n", status.BufferUsed, status.BufferTotal)
	fmt.Printf("  最后读取: %v\n\n", status.LastReadTime)

	// 直接从 OutputBuffer 读取
	fmt.Println("=== 直接从 OutputBuffer 读取 ===")
	lines := shell.OutputBuffer.ReadLatestLines(50)
	fmt.Printf("  读取到 %d 行\n", len(lines))
	for i, line := range lines {
		fmt.Printf("  [%d] %s\n", i, line)
	}
	fmt.Println()

	// 检查 Stdout 状态
	fmt.Println("=== Stdout 状态 ===")
	fmt.Printf("  Stdout 是否为 nil: %v\n", shell.Stdout == nil)
	fmt.Printf("  Stdin 是否为 nil: %v\n", shell.Stdin == nil)
	fmt.Println()

	// 尝试读取 TerminalCapturer
	fmt.Println("=== Terminal Capturer 快照 ===")
	if shell.TerminalCapturer != nil {
		snapshot := shell.TerminalCapturer.GetScreenSnapshot()
		lines := strings.Split(snapshot, "\n")
		fmt.Printf("  快照行数: %d\n", len(lines))
		if len(lines) > 0 && len(snapshot) > 0 {
			fmt.Println("  快照内容:")
			for i, line := range lines {
				if len(line) > 0 {
					fmt.Printf("    [%d] %s\n", i, line)
				}
			}
		}
	}
	fmt.Println()

	// 发送第二个命令
	fmt.Println("=== 发送第二个命令: pwd ===")
	err = shell.WriteInput("pwd\n")
	if err != nil {
		logger.Error().Err(err).Msg("写入输入失败")
	}
	time.Sleep(1 * time.Second)

	// 再次读取
	stdout, _, _ = shell.ReadOutputNonBlocking(200 * time.Millisecond)
	fmt.Printf("✅ pwd 输出 (%d 字节):\n%s\n\n", len(stdout), stdout)

	// 最终状态
	fmt.Println("=== 最终状态 ===")
	status = shell.GetStatus()
	fmt.Printf("  活动: %v\n", status.IsActive)
	fmt.Printf("  缓冲区使用: %d/%d 行\n", status.BufferUsed, status.BufferTotal)

	fmt.Println("\n=== 诊断完成 ===")
}
