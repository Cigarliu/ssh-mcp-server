package main

import (
	"fmt"
	"log"
	"time"

	"github.com/cigar/sshmcp/pkg/sshmcp"
)

func main() {
	fmt.Println("=== 调试 OutputBuffer 状态 ===\n")

	// 创建 SSH 客户端
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

	// 创建会话对象
	session := &sshmcp.Session{
		ID:         "debug-buffer",
		Host:       "192.168.3.7",
		Port:       22,
		Username:   "cigar",
		SSHClient:  client,
		AuthConfig: authConfig,
	}

	// 测试: ANSIRaw 模式 (NonBlockingRead 使用的模式)
	config := &sshmcp.ShellConfig{
		Mode:         sshmcp.TerminalModeCooked,
		ANSIMode:     sshmcp.ANSIRaw,
		ReadTimeout:  100 * time.Millisecond,
		WriteTimeout: 5 * time.Second,
	}

	shell, err := session.CreateShellWithConfig("xterm-256color", 24, 80, config)
	if err != nil {
		log.Fatalf("创建 Shell 失败: %v", err)
	}
	defer shell.Close()

	fmt.Println("✅ 创建了 Shell (ANSIRaw 模式)")

	// 发送命令
	err = shell.WriteInput("echo 'Test OutputBuffer monitoring'\n")
	if err != nil {
		log.Printf("发送命令失败: %v", err)
	} else {
		fmt.Println("✅ 发送了命令")
	}

	// 等待命令执行
	time.Sleep(500 * time.Millisecond)

	// 检查 OutputBuffer 的状态
	fmt.Println("\n=== 检查 OutputBuffer ===")
	lines := shell.OutputBuffer.ReadLatestLines(100)
	fmt.Printf("OutputBuffer 中有 %d 行\n", len(lines))
	if len(lines) > 0 {
		fmt.Println("Buffer 内容:")
		for i, line := range lines {
			fmt.Printf("  [%d] %s\n", i+1, line)
		}
	}

	// 现在使用 ReadOutputNonBlocking 读取
	fmt.Println("\n=== 使用 ReadOutputNonBlocking 读取 ===")
	stdout, stderr, err := shell.ReadOutputNonBlocking(200 * time.Millisecond)
	if err != nil {
		log.Printf("读取失败: %v", err)
	} else {
		fmt.Printf("stdout (%d bytes): %s\n", len(stdout), stdout)
		fmt.Printf("stderr (%d bytes): %s\n", len(stderr), stderr)
	}

	// 再次检查 OutputBuffer
	fmt.Println("\n=== ReadOutputNonBlocking 后再次检查 OutputBuffer ===")
	lines2 := shell.OutputBuffer.ReadLatestLines(100)
	fmt.Printf("OutputBuffer 中有 %d 行\n", len(lines2))

	fmt.Println("\n=== 调试完成 ===")
}
