package main

import (
	"fmt"
	"log"
	"time"

	"github.com/cigar/sshmcp/pkg/sshmcp"
)

func main() {
	fmt.Println("=== 简化调试: Shell 输出问题 ===\n")

	// 直接创建 SSH 客户端
	authConfig := &sshmcp.AuthConfig{
		Type:     sshmcp.AuthTypePassword,
		Password: "liuxuejia.123",
	}

	client, err := sshmcp.CreateSSHClient("192.168.3.7", 22, "cigar", authConfig, 10*time.Second)
	if err != nil {
		log.Fatalf("创建 SSH 客户端失败: %v", err)
	}
	defer client.Close()

	fmt.Println("✅ 已连接到服务器")

	// 创建会话对象（不使用 SessionManager）
	session := &sshmcp.Session{
		ID:         "debug-session",
		Host:       "192.168.3.7",
		Port:       22,
		Username:   "cigar",
		SSHClient:  client,
		AuthConfig: authConfig,
	}

	// 测试 1: ANSIStrip 模式
	fmt.Println("\n=== 测试 1: ANSIStrip 模式 ===")
	config1 := &sshmcp.ShellConfig{
		Mode:         sshmcp.TerminalModeCooked,
		ANSIMode:     sshmcp.ANSIStrip,
		ReadTimeout:  100 * time.Millisecond,
		WriteTimeout: 5 * time.Second,
	}

	shell1, err := session.CreateShellWithConfig("xterm-256color", 24, 80, config1)
	if err != nil {
		log.Fatalf("创建 Shell 失败: %v", err)
	}
	defer shell1.Close()

	fmt.Println("✅ 创建了 ANSIStrip 模式的 Shell")

	// 发送命令
	err = shell1.WriteInput("echo 'Test ANSIStrip mode'\n")
	if err != nil {
		log.Printf("发送命令失败: %v", err)
	} else {
		fmt.Println("✅ 发送了命令")
	}

	// 等待并多次读取
	time.Sleep(500 * time.Millisecond)
	fmt.Println("\n开始读取输出...")
	for i := 0; i < 20; i++ {
		stdout, stderr, err := shell1.ReadOutputNonBlocking(200 * time.Millisecond)
		if err != nil {
			log.Printf("读取失败 (iteration %d): %v", i+1, err)
			break
		}

		if stdout != "" {
			fmt.Printf("📥 stdout [%d] (%d bytes):\n%s\n", i+1, len(stdout), stdout)
		}
		if stderr != "" {
			fmt.Printf("📥 stderr [%d] (%d bytes):\n%s\n", i+1, len(stderr), stderr)
		}

		if stdout == "" && stderr == "" {
			fmt.Printf("⏸️  无更多数据 (iteration %d)\n", i+1)
			break
		}

		time.Sleep(100 * time.Millisecond)
	}

	fmt.Println("\n=== 调试完成 ===")
}
