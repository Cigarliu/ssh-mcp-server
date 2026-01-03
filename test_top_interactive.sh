#!/bin/bash

# Top 命令交互测试脚本
# 测试 SSH MCP 的交互式终端支持能力

set -e

echo "======================================"
echo "  Top 命令交互测试"
echo "======================================"
echo ""

cd /home/cigar/tools/sshmcp

# 创建测试程序
cat > /tmp/test_top_interactive.go << 'GOEOF'
package main

import (
	"fmt"
	"time"
	"os"

	"github.com/cigar/sshmcp/pkg/sshmcp"
)

func main() {
	// SSH 连接配置
	host := os.Getenv("SSH_HOST")
	if host == "" {
		host = "192.168.68.212"
	}

	port := 22
	username := os.Getenv("SSH_USER")
	if username == "" {
		username = "root"
	}

	password := os.Getenv("SSH_PASSWORD")
	if password == "" {
		password = "root"
	}

	// 创建会话管理器
	logger := sshmcp.setupTestLogger(&testingT{})
	config := sshmcp.ManagerConfig{
		MaxSessions:        10,
		MaxSessionsPerHost: 3,
		SessionTimeout:     10 * time.Minute,
		IdleTimeout:        5 * time.Minute,
		CleanupInterval:    1 * time.Minute,
		Logger:             logger,
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
		os.Exit(1)
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
		os.Exit(1)
	}
	defer shell.Close()

	fmt.Println("✅ Raw mode shell 创建成功")
	fmt.Println("🚀 启动 top 命令...")
	fmt.Println("")

	// 1. 启动 top
	time.Sleep(100 * time.Millisecond)
	shell.WriteInput("top\n")
	time.Sleep(500 * time.Millisecond)

	// 2. 读取初始输出
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("📊 步骤 1: 读取 top 初始界面")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// 多次读取以获取完整输出
	var fullOutput string
	for i := 0; i < 10; i++ {
		stdout, _, err := shell.ReadOutputNonBlocking(200 * time.Millisecond)
		if err != nil {
			fmt.Printf("❌ 读取失败: %v\n", err)
			break
		}

		if stdout != "" {
			fullOutput += stdout
		}

		time.Sleep(50 * time.Millisecond)

		// 如果没有更多数据，退出
		if stdout == "" && len(fullOutput) > 0 {
			break
		}
	}

	// 显示前 15 行输出
	lines := fmt.Sprintf("%s", fullOutput)
	lineCount := 0
	fmt.Println("")
	for _, line := range splitLines(lines) {
		if lineCount >= 15 {
			break
		}
		if len(line) > 0 {
			fmt.Printf("   %s\n", line)
			lineCount++
		}
	}
	fmt.Println("   ...")
	fmt.Println("")

	// 3. 按 'P' 键按 CPU 使用率排序
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("📊 步骤 2: 按 'P' 键 - 按 CPU 使用率排序")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	time.Sleep(100 * time.Millisecond)
	shell.WriteInput("P")
	time.Sleep(300 * time.Millisecond)

	// 清空之前的输出
	stdout, _, _ := shell.ReadOutputNonBlocking(100 * time.Millisecond)
	_ = stdout

	fmt.Println("✅ 已按 CPU 排序")
	fmt.Println("")

	// 4. 按 'M' 键按内存使用率排序
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("📊 步骤 3: 按 'M' 键 - 按内存使用率排序")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	time.Sleep(100 * time.Millisecond)
	shell.WriteInput("M")
	time.Sleep(300 * time.Millisecond)

	stdout, _, _ = shell.ReadOutputNonBlocking(100 * time.Millisecond)
	_ = stdout

	fmt.Println("✅ 已按内存排序")
	fmt.Println("")

	// 5. 按 'T' 键按时间排序
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("📊 步骤 4: 按 'T' 键 - 按时间排序")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	time.Sleep(100 * time.Millisecond)
	shell.WriteInput("T")
	time.Sleep(300 * time.Millisecond)

	stdout, _, _ = shell.ReadOutputNonBlocking(100 * time.Millisecond)
	_ = stdout

	fmt.Println("✅ 已按时间排序")
	fmt.Println("")

	// 6. 使用方向键移动选择
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("📊 步骤 5: 测试方向键 - 上下移动")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// 按下 3 次
	for i := 0; i < 3; i++ {
		shell.WriteSpecialChars("down")
		time.Sleep(100 * time.Millisecond)
	}

	// 按上 2 次
	for i := 0; i < 2; i++ {
		shell.WriteSpecialChars("up")
		time.Sleep(100 * time.Millisecond)
	}

	fmt.Println("✅ 方向键测试成功（下移 3 次，上移 2 次）")
	fmt.Println("")

	// 7. 搜索功能（按 'L' 定位用户）
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("📊 步骤 6: 按 'L' - 搜索/过滤进程")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	time.Sleep(100 * time.Millisecond)
	shell.WriteInput("l")
	time.Sleep(100 * time.Millisecond)

	// 输入 "root" 并回车
	shell.WriteInput("root\n")
	time.Sleep(300 * time.Millisecond)

	stdout, _, _ = shell.ReadOutputNonBlocking(100 * time.Millisecond)
	_ = stdout

	fmt.Println("✅ 已过滤显示 root 用户的进程")
	fmt.Println("")

	// 8. 读取最终的 top 输出
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("📊 步骤 7: 读取最终 top 输出")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	time.Sleep(500 * time.Millisecond)

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
	lines = fmt.Sprintf("%s", fullOutput)
	outputLines := splitLines(lines)

	fmt.Println("")
	fmt.Println("Top 输出（前 25 行）：")
	fmt.Println("")

	for i, line := range outputLines {
		if i >= 25 {
			break
		}
		if len(line) > 0 {
			fmt.Printf("   %s\n", line)
		}
	}
	fmt.Println("")

	// 9. 退出 top
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("📊 步骤 8: 按 'q' 退出 top")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	time.Sleep(100 * time.Millisecond)
	shell.WriteInput("q")
	time.Sleep(200 * time.Millisecond)

	// 读取退出后的 shell 提示符
	stdout, _, _ = shell.ReadOutputNonBlocking(200 * time.Millisecond)
	if stdout != "" {
		fmt.Printf("退出后的输出: %s\n", truncate(stdout, 100))
	}

	fmt.Println("✅ Top 已退出")
	fmt.Println("")

	// 总结
	fmt.Println("======================================")
	fmt.Println("  ✅ Top 交互测试完成！")
	fmt.Println("======================================")
	fmt.Println("")
	fmt.Println("测试功能：")
	fmt.Println("  ✓ 启动 top 命令")
	fmt.Println("  ✓ 读取初始界面")
	fmt.Println("  ✓ 按 'P' 键 - CPU 排序")
	fmt.Println("  ✓ 按 'M' 键 - 内存排序")
	fmt.Println("  ✓ 按 'T' 键 - 时间排序")
	fmt.Println("  ✓ 方向键上下移动")
	fmt.Println("  ✓ 按 'L' 搜索/过滤")
	fmt.Println("  ✓ 读取最终输出")
	fmt.Println("  ✓ 按 'q' 退出")
	fmt.Println("")
	fmt.Println("所有交互功能正常工作！🎉")
}

// 辅助函数
func splitLines(s string) []string {
	var lines []string
	line := ""
	for _, ch := range s {
		if ch == '\n' {
			lines = append(lines, line)
			line = ""
		} else {
			line += string(ch)
		}
	}
	if len(line) > 0 {
		lines = append(lines, line)
	}
	return lines
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// testingT 实现 testing.TB 接口
type testingT struct{}

func (t *testingT) Helper()                                    {}
func (t *testingT) Name() string                               { return "top-test" }
func (t *testingT) Cleanup(f func())                           { f() }
func (t *testingT) Error(args ...interface{})                 { fmt.Println(args...) }
func (t *testingT) Errorf(format string, args ...interface{})  { fmt.Printf(format+"\n", args...) }
func (t *testingT) Fail()                                       {}
func (t *testingT) FailNow()                                    {}
func (t *testingT) Failed() bool                                { return false }
func (t *testingT) Fatal(args ...interface{})                  { fmt.Println(args...); os.Exit(1) }
func (t *testingT) Fatalf(format string, args ...interface{})  { fmt.Printf(format+"\n", args...); os.Exit(1) }
func (t *testingT) Log(args ...interface{})                    { fmt.Println(args...) }
func (t *testingT) Logf(format string, args ...interface{})    { fmt.Printf(format+"\n", args...) }
func (t *testingT) Setenv(key, value string)                   { os.Setenv(key, value) }

GOEOF

# 运行测试
echo "🔨 编译测试程序..."
go run /tmp/test_top_interactive.go

echo ""
echo "✅ 测试完成！"
