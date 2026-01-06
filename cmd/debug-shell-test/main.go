package main

import (
	"fmt"
	"log"
	"time"

	"github.com/cigar/sshmcp/pkg/sshmcp"
)

func main() {
	fmt.Println("=== 调试 Shell 输出问题 ===\n")

	// 创建 SSH 会话
	sm := sshmcp.NewSessionManager(sshmcp.ManagerConfig{
		MaxSessions:        10,
		MaxSessionsPerHost: 5,
		CleanupInterval:    30 * time.Second,
	})
	defer sm.Close()

	authConfig := &sshmcp.AuthConfig{
		Type:     sshmcp.AuthTypePassword,
		Password: "liuxuejia.123",
	}

	// 连接到服务器
	session, err := sm.CreateSession("192.168.3.7", 22, "cigar", authConfig, "debug-test")
	if err != nil {
		log.Fatalf("连接失败: %v", err)
	}
	// SessionManager 会自动清理

	fmt.Println("✅ 已连接到服务器")

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
	err = shell1.WriteInput("echo -e '\\033[31mRed Text\\033[0m'\n")
	if err != nil {
		log.Printf("发送命令失败: %v", err)
	} else {
		fmt.Println("✅ 发送了命令: echo -e '\\033[31mRed Text\\033[0m'")
	}

	// 等待并多次读取
	time.Sleep(500 * time.Millisecond)
	for i := 0; i < 10; i++ {
		stdout, stderr, err := shell1.ReadOutputNonBlocking(200 * time.Millisecond)
		if err != nil {
			log.Printf("读取输出失败 (iteration %d): %v", i+1, err)
		}

		if stdout != "" {
			fmt.Printf("📥 stdout [%d]:\n%s\n", i+1, stdout)
		}
		if stderr != "" {
			fmt.Printf("📥 stderr [%d]:\n%s\n", i+1, stderr)
		}

		if stdout == "" && stderr == "" {
			fmt.Printf("⏸️  无更多数据 (iteration %d)\n", i+1)
			break
		}

		time.Sleep(100 * time.Millisecond)
	}

	// 测试 2: Raw 模式
	fmt.Println("\n=== 测试 2: Raw 模式 ===")
	time.Sleep(1 * time.Second) // 等待前一个 shell 完全关闭

	config2 := &sshmcp.ShellConfig{
		Mode:         sshmcp.TerminalModeRaw,
		ANSIMode:     sshmcp.ANSIRaw,
		ReadTimeout:  50 * time.Millisecond,
		WriteTimeout: 5 * time.Second,
	}

	shell2, err := session.CreateShellWithConfig("xterm-256color", 24, 80, config2)
	if err != nil {
		log.Fatalf("创建 Shell 失败: %v", err)
	}
	defer shell2.Close()

	fmt.Println("✅ 创建了 Raw 模式的 Shell")

	// 发送命令
	err = shell2.WriteInput("pwd\n")
	if err != nil {
		log.Printf("发送命令失败: %v", err)
	} else {
		fmt.Println("✅ 发送了命令: pwd")
	}

	// 等待并多次读取
	time.Sleep(500 * time.Millisecond)
	for i := 0; i < 10; i++ {
		stdout, stderr, err := shell2.ReadOutputNonBlocking(200 * time.Millisecond)
		if err != nil {
			log.Printf("读取输出失败 (iteration %d): %v", i+1, err)
		}

		if stdout != "" {
			fmt.Printf("📥 stdout [%d]:\n%s\n", i+1, stdout)
		}
		if stderr != "" {
			fmt.Printf("📥 stderr [%d]:\n%s\n", i+1, stderr)
		}

		if stdout == "" && stderr == "" {
			fmt.Printf("⏸️  无更多数据 (iteration %d)\n", i+1)
			break
		}

		time.Sleep(100 * time.Millisecond)
	}

	// 测试 3: 正常模式 (作为对照)
	fmt.Println("\n=== 测试 3: 正常 Cooked 模式 (对照) ===")
	time.Sleep(1 * time.Second)

	config3 := &sshmcp.ShellConfig{
		Mode:         sshmcp.TerminalModeCooked,
		ANSIMode:     sshmcp.ANSIRaw,
		ReadTimeout:  100 * time.Millisecond,
		WriteTimeout: 5 * time.Second,
	}

	shell3, err := session.CreateShellWithConfig("xterm-256color", 24, 80, config3)
	if err != nil {
		log.Fatalf("创建 Shell 失败: %v", err)
	}
	defer shell3.Close()

	fmt.Println("✅ 创建了正常 Cooked 模式的 Shell")

	// 发送命令
	err = shell3.WriteInput("echo 'Hello from normal mode'\n")
	if err != nil {
		log.Printf("发送命令失败: %v", err)
	} else {
		fmt.Println("✅ 发送了命令: echo 'Hello from normal mode'")
	}

	// 等待并多次读取
	time.Sleep(500 * time.Millisecond)
	for i := 0; i < 10; i++ {
		stdout, stderr, err := shell3.ReadOutputNonBlocking(200 * time.Millisecond)
		if err != nil {
			log.Printf("读取输出失败 (iteration %d): %v", i+1, err)
		}

		if stdout != "" {
			fmt.Printf("📥 stdout [%d]:\n%s\n", i+1, stdout)
		}
		if stderr != "" {
			fmt.Printf("📥 stderr [%d]:\n%s\n", i+1, stderr)
		}

		if stdout == "" && stderr == "" {
			fmt.Printf("⏸️  无更多数据 (iteration %d)\n", i+1)
			break
		}

		time.Sleep(100 * time.Millisecond)
	}

	fmt.Println("\n=== 调试完成 ===")
}
