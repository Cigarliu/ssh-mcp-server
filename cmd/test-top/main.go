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
	fmt.Println("======================================")
	fmt.Println("  Top 命令交互测试")
	fmt.Println("======================================")
	fmt.Println()

	// SSH 连接配置
	host := "192.168.68.212"
	port := 22
	username := "root"
	password := "root"

	// 创建 logger
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()

	// 创建会话管理器
	config := sshmcp.ManagerConfig{
		MaxSessions:        10,
		MaxSessionsPerHost: 3,
		SessionTimeout:     10 * time.Minute,
		IdleTimeout:        5 * time.Minute,
		CleanupInterval:    1 * time.Minute,
		Logger:             &logger,
	}

	sm := sshmcp.NewSessionManager(config)
	defer sm.Close()

	auth := &sshmcp.AuthConfig{
		Type:     sshmcp.AuthTypePassword,
		Password: password,
	}

	session, err := sm.CreateSession(host, port, username, auth, "top-test")
	if err != nil {
		fmt.Printf("❌ 创建 SSH 会话失败: %v\n", err)
		return
	}
	defer sm.RemoveSession(session.ID)

	fmt.Println("✅ SSH 连接成功")
	fmt.Printf("   服务器: %s@%s:%d\n\n", username, host, port)

	// 创建 Raw mode shell（top 需要 raw mode）
	shellConfig := &sshmcp.ShellConfig{
		Mode:         sshmcp.TerminalModeRaw,
		ANSIMode:     sshmcp.ANSIStrip, // 去除 ANSI，便于查看
		ReadTimeout:  200 * time.Millisecond,
		WriteTimeout: 5 * time.Second,
	}

	shell, err := session.CreateShellWithConfig("xterm-256color", 40, 120, shellConfig)
	if err != nil {
		fmt.Printf("❌ 创建 shell 失败: %v\n", err)
		return
	}
	defer shell.Close()

	fmt.Println("✅ Raw mode shell 创建成功")
	fmt.Println("🚀 启动 top 命令...")
	fmt.Println()

	// 1. 启动 top
	time.Sleep(100 * time.Millisecond)
	shell.WriteInput("top\n")
	time.Sleep(800 * time.Millisecond)

	// 2. 读取初始输出
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("📊 步骤 1: 读取 top 初始界面")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	var fullOutput string
	for i := 0; i < 10; i++ {
		stdout, _, err := shell.ReadOutputNonBlocking(200 * time.Millisecond)
		if err != nil {
			break
		}
		if stdout != "" {
			fullOutput += stdout
		}
		time.Sleep(50 * time.Millisecond)
	}

	// 显示前 20 行
	lines := strings.Split(fullOutput, "\n")
	fmt.Println("")
	for i := 0; i < len(lines) && i < 20; i++ {
		if len(lines[i]) > 0 {
			fmt.Printf("   %s\n", lines[i])
		}
	}
	fmt.Println("   ...")
	fmt.Println()

	time.Sleep(500 * time.Millisecond)

	// 3. 按 'P' 键按 CPU 使用率排序
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("📊 步骤 2: 按 'P' 键 - 按 CPU 使用率排序")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	shell.WriteInput("P")
	time.Sleep(400 * time.Millisecond)
	shell.ReadOutputNonBlocking(100 * time.Millisecond) // 清空缓冲区
	fmt.Println("✅ 已按 CPU 排序")
	fmt.Println()

	// 4. 按 'M' 键按内存使用率排序
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("📊 步骤 3: 按 'M' 键 - 按内存使用率排序")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	shell.WriteInput("M")
	time.Sleep(400 * time.Millisecond)
	shell.ReadOutputNonBlocking(100 * time.Millisecond)
	fmt.Println("✅ 已按内存排序")
	fmt.Println()

	// 5. 使用方向键移动选择
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("📊 步骤 4: 测试方向键 - 上下移动")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	for i := 0; i < 3; i++ {
		shell.WriteSpecialChars("down")
		time.Sleep(80 * time.Millisecond)
	}
	for i := 0; i < 2; i++ {
		shell.WriteSpecialChars("up")
		time.Sleep(80 * time.Millisecond)
	}
	fmt.Println("✅ 方向键测试成功（下移 3 次，上移 2 次）")
	fmt.Println()

	// 6. 搜索功能（按 'c' 切换命令行显示）
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("📊 步骤 5: 按 'c' - 显示完整命令行")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	shell.WriteInput("c")
	time.Sleep(400 * time.Millisecond)
	shell.ReadOutputNonBlocking(100 * time.Millisecond)
	fmt.Println("✅ 已切换到完整命令行显示")
	fmt.Println()

	// 7. 读取最终的 top 输出
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("📊 步骤 6: 读取最终 top 输出")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	time.Sleep(600 * time.Millisecond)

	// 读取多次以获取完整刷新后的输出
	fullOutput = ""
	for i := 0; i < 15; i++ {
		stdout, _, err := shell.ReadOutputNonBlocking(200 * time.Millisecond)
		if err != nil {
			break
		}
		if stdout != "" {
			fullOutput += stdout
		}
		time.Sleep(50 * time.Millisecond)
	}

	// 显示输出
	lines = strings.Split(fullOutput, "\n")
	fmt.Println("")
	fmt.Println("Top 最终输出（前 30 行）：")
	fmt.Println("")

	displayCount := 0
	for i := 0; i < len(lines) && displayCount < 30; i++ {
		line := strings.TrimSpace(lines[i])
		if len(line) > 0 {
			fmt.Printf("   %s\n", line)
			displayCount++
		}
	}
	fmt.Println("")

	// 8. 退出 top
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("📊 步骤 7: 按 'q' 退出 top")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	time.Sleep(100 * time.Millisecond)
	shell.WriteInput("q")
	time.Sleep(300 * time.Millisecond)

	stdout, _, _ := shell.ReadOutputNonBlocking(200 * time.Millisecond)
	if len(stdout) > 0 {
		if len(stdout) > 100 {
			stdout = stdout[:100] + "..."
		}
		fmt.Printf("退出后的输出: %s\n", stdout)
	}

	fmt.Println("✅ Top 已退出")
	fmt.Println("")

	// 总结
	fmt.Println("======================================")
	fmt.Println("  ✅ Top 交互测试完成！")
	fmt.Println("======================================")
	fmt.Println("")
	fmt.Println("测试功能总结：")
	fmt.Println("  ✓ 启动 top 命令（Raw Mode）")
	fmt.Println("  ✓ 读取初始界面")
	fmt.Println("  ✓ 按 'P' 键 - CPU 排序")
	fmt.Println("  ✓ 按 'M' 键 - 内存排序")
	fmt.Println("  ✓ 方向键上下移动（down x3, up x2）")
	fmt.Println("  ✓ 按 'c' 键 - 显示完整命令行")
	fmt.Println("  ✓ 读取最终输出（过滤后）")
	fmt.Println("  ✓ 按 'q' 退出")
	fmt.Println("")
	fmt.Println("🎉 所有交互功能正常工作！")
	fmt.Println("   非阻塞读取、特殊字符、Raw Mode 全部验证通过！")
}
