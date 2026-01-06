package main

import (
	"fmt"
	"log"
	"time"

	"github.com/cigar/sshmcp/pkg/sshmcp"
)

func main() {
	fmt.Println("=== 测试 Buffer 清空 ===\n")

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

	// 创建会话
	session := &sshmcp.Session{
		ID:         "debug-flush",
		Host:       "192.168.3.7",
		Port:       22,
		Username:   "cigar",
		SSHClient:  client,
		AuthConfig: authConfig,
	}

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

	// 等待初始化
	time.Sleep(300 * time.Millisecond)

	fmt.Println("=== 第一次读取（初始化消息） ===")
	stdout1, _, _ := shell.ReadOutputNonBlocking(200 * time.Millisecond)
	fmt.Printf("读到 %d 字节\n", len(stdout1))

	// 检查buffer
	lines1 := shell.OutputBuffer.ReadLatestLines(10)
	fmt.Printf("OutputBuffer 还有 %d 行\n", len(lines1))

	// 发送命令
	fmt.Println("\n=== 发送 pwd 命令 ===")
	shell.WriteInput("pwd\n")
	time.Sleep(500 * time.Millisecond)

	fmt.Println("\n=== 第二次读取（应该有pwd输出） ===")
	stdout2, _, _ := shell.ReadOutputNonBlocking(200 * time.Millisecond)
	fmt.Printf("读到 %d 字节\n", len(stdout2))
	if stdout2 != "" {
		fmt.Printf("内容: %s\n", stdout2)
	}

	// 检查buffer
	lines2 := shell.OutputBuffer.ReadLatestLines(10)
	fmt.Printf("OutputBuffer 还有 %d 行\n", len(lines2))

	fmt.Println("\n=== 测试完成 ===")
}
