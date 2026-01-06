package main

import (
	"fmt"
	"log"
	"time"

	"github.com/cigar/sshmcp/pkg/sshmcp"
)

func main() {
	fmt.Println("=== 测试多次 ReadOutputNonBlocking ===\n")

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
		ID:         "debug-dual-read",
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

	// 等待shell初始化
	time.Sleep(300 * time.Millisecond)

	// 发送命令
	shell.WriteInput("echo 'First Command'\n")
	time.Sleep(200 * time.Millisecond)

	// 第一次读取
	fmt.Println("=== 第一次 ReadOutputNonBlocking ===")
	stdout1, _, err1 := shell.ReadOutputNonBlocking(200 * time.Millisecond)
	if err1 != nil {
		log.Printf("错误: %v", err1)
	} else {
		fmt.Printf("读到 %d 字节: [%s]\n", len(stdout1), stdout1)
	}

	// 第二次读取（立即）
	fmt.Println("\n=== 第二次 ReadOutputNonBlocking (立即) ===")
	stdout2, _, err2 := shell.ReadOutputNonBlocking(200 * time.Millisecond)
	if err2 != nil {
		log.Printf("错误: %v", err2)
	} else {
		fmt.Printf("读到 %d 字节: [%s]\n", len(stdout2), stdout2)
	}

	// 第三次读取（等一下）
	time.Sleep(100 * time.Millisecond)
	fmt.Println("\n=== 第三次 ReadOutputNonBlocking (等100ms) ===")
	stdout3, _, err3 := shell.ReadOutputNonBlocking(200 * time.Millisecond)
	if err3 != nil {
		log.Printf("错误: %v", err3)
	} else {
		fmt.Printf("读到 %d 字节: [%s]\n", len(stdout3), stdout3)
	}

	// 检查OutputBuffer
	fmt.Println("\n=== 检查 OutputBuffer ===")
	lines := shell.OutputBuffer.ReadLatestLines(50)
	fmt.Printf("OutputBuffer 有 %d 行\n", len(lines))

	fmt.Println("\n=== 测试完成 ===")
}
