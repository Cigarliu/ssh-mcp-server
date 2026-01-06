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
	// 强制使用 vt100 模拟器
	os.Setenv("SSH_MCP_TERMINAL_EMULATOR", "vt100")

	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()

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

	fmt.Println("🔧 连接到服务器...")
	session, err := sm.CreateSession("cctv.mba", 9022, "cigar", &sshmcp.AuthConfig{
		Type:     sshmcp.AuthTypePassword,
		Password: "liuxuejia.123",
	}, "test-vt100")
	if err != nil {
		logger.Fatal().Err(err).Msg("❌ 连接失败")
	}

	fmt.Println("✅ 已连接")

	shellConfig := &sshmcp.ShellConfig{
		Mode:         sshmcp.TerminalModeRaw,
		ANSIMode:     sshmcp.ANSIRaw,
		ReadTimeout:  100 * time.Millisecond,
		WriteTimeout: 5 * time.Second,
	}

	fmt.Println("🖥️  创建交互式 shell (vt100 模拟器)...")
	shell, err := session.CreateShellWithConfig("xterm-256color", 40, 160, shellConfig)
	if err != nil {
		logger.Fatal().Err(err).Msg("❌ 创建 shell 失败")
	}
	defer shell.Close()

	time.Sleep(500 * time.Millisecond)
	fmt.Println("✅ Shell 创建成功")
	fmt.Println()

	// 启动 htop
	fmt.Println("📊 启动 htop...")
	shell.WriteInput("htop\n")
	time.Sleep(2 * time.Second)

	// 获取快照
	fmt.Println("📸 捕获屏幕快照...")
	snapshot := shell.GetTerminalSnapshot()

	// 分析输出
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("📊 VT100 模拟器 - HTOP 输出分析")
	fmt.Println(strings.Repeat("=", 80))

	// 统计字符
	charCount := make(map[rune]int)
	for _, r := range snapshot {
		if r >= 32 && r <= 126 { // 可打印 ASCII
			charCount[r]++
		}
	}

	// 检查 'B' 字符（伪影指标）
	bCount := charCount['B']
	fmt.Printf("\n📈 字符统计：\n")
	fmt.Printf("  - 总字符数: %d\n", len(snapshot))
	fmt.Printf("  - 'B' 字符数量: %d\n", bCount)

	// 查找第一个有意义的行
	lines := strings.Split(snapshot, "\n")
	var firstMeaningfulLine string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if len(trimmed) > 20 {
			firstMeaningfulLine = line
			break
		}
	}

	if firstMeaningfulLine != "" {
		fmt.Printf("\n📄 第一行内容示例（前100字符）:\n")
		if len(firstMeaningfulLine) > 100 {
			fmt.Printf("  %s\n", firstMeaningfulLine[:100])
		} else {
			fmt.Printf("  %s\n", firstMeaningfulLine)
		}

		// 检查伪影模式
		hasB := strings.Contains(firstMeaningfulLine, "B")
		hasBracket := strings.Contains(firstMeaningfulLine, "[")
		hasDigit := strings.ContainsAny(firstMeaningfulLine, "0123456789")

		fmt.Printf("\n🔍 伪影检测：\n")
		fmt.Printf("  - 包含 'B': %v\n", hasB)
		fmt.Printf("  - 包含 '[': %v\n", hasBracket)
		fmt.Printf("  - 包含数字: %v\n", hasDigit)

		// 检查特定模式
		patterns := []string{"0B[", "1B[", "2B[", "B[", "]B"}
		foundPatterns := []string{}
		for _, pattern := range patterns {
			if strings.Contains(firstMeaningfulLine, pattern) {
				foundPatterns = append(foundPatterns, pattern)
			}
		}

		if len(foundPatterns) > 0 {
			fmt.Printf("  - ⚠️  发现伪影模式: %v\n", foundPatterns)
		}
	}

	// 显示前 5 行内容
	fmt.Printf("\n📺 屏幕内容（前5行）:\n")
	fmt.Println(strings.Repeat("-", 80))
	lineCount := 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if len(trimmed) > 10 && lineCount < 5 {
			fmt.Printf("  %s\n", trimmed)
			lineCount++
		}
	}
	fmt.Println(strings.Repeat("-", 80))

	// 结论
	fmt.Printf("\n🎯 VT100 模拟器测试结论：\n")
	if bCount > 100 {
		fmt.Printf("  ❌ 发现大量 'B' 字符伪影 (%d 个)\n", bCount)
		fmt.Printf("  ⚠️  VT100 模拟器存在 ANSI 解析问题\n")
	} else {
		fmt.Printf("  ✅ 输出正常，无明显伪影\n")
	}

	// 退出 htop
	fmt.Println("\n🛑 退出 htop...")
	shell.WriteInput("q")
	time.Sleep(500 * time.Millisecond)

	fmt.Println("\n✅ VT100 模拟器测试完成")
}
