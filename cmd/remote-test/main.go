package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/cigar/sshmcp/pkg/sshmcp"
	"github.com/rs/zerolog"
)

func main() {
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

	fmt.Println("🔧 连接到服务器 cigar@192.168.3.7...")
	session, err := sm.CreateSession("192.168.3.7", 22, "cigar", &sshmcp.AuthConfig{
		Type:     sshmcp.AuthTypePassword,
		Password: "liuxuejia.123",
	}, "remote-test")
	if err != nil {
		logger.Fatal().Err(err).Msg("❌ 连接失败")
	}
	defer sm.DisconnectSession(session.ID)

	fmt.Println("✅ 已连接")

	// 上传测试程序
	fmt.Println("📤 上传测试程序到服务器...")
	localBinary := "bin/test-bubbleterm-linux"
	remotePath := "/tmp/test-bubbleterm-linux"

	// 读取本地文件
	localFile, err := os.Open(localBinary)
	if err != nil {
		logger.Fatal().Err(err).Msg("❌ 无法打开本地文件")
	}
	defer localFile.Close()

	fileInfo, _ := localFile.Stat()
	fileSize := fileInfo.Size()

	fmt.Printf("  文件大小: %.2f MB\n", float64(fileSize)/(1024*1024))

	// 使用 SFTP 上传
	err = session.SFTPUpload(localBinary, remotePath, true)
	if err != nil {
		logger.Fatal().Err(err).Msg("❌ 上传失败")
	}

	fmt.Println("✅ 上传成功")

	// 设置执行权限
	fmt.Println("🔧 设置执行权限...")
	_, err = session.Execute(fmt.Sprintf("chmod +x %s", remotePath))
	if err != nil {
		logger.Warn().Err(err).Msg("⚠️  设置权限失败，继续尝试运行")
	}

	// 设置环境变量并运行
	fmt.Println("\n🚀 在 Linux 服务器上运行 Bubbleterm 测试...")
	fmt.Println("==========================================\n")

	cmd := fmt.Sprintf("SSH_MCP_TERMINAL_EMULATOR=bubbleterm %s", remotePath)
	output, err := session.Execute(cmd)
	if err != nil {
		logger.Error().Err(err).Msg("⚠️  执行失败，但可能有部分输出")
	}

	if output != "" {
		fmt.Println(output)
	}

	fmt.Println("\n==========================================")
	fmt.Println("✅ 远程测试完成")

	// 清理
	fmt.Println("\n🧹 清理临时文件...")
	_, _ = session.Execute(fmt.Sprintf("rm -f %s", remotePath))
	fmt.Println("✅ 清理完成")
}

// Helper function to copy file (if needed)
func copyFile(src, dst string) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()

	_, err = io.Copy(destination, source)
	return err
}

func init() {
	// Change to project directory
	projectDir := filepath.Join("c:", "Users", "cigar", "Desktop", "temp", "code", "ssh-mcp-server")
	err := os.Chdir(projectDir)
	if err != nil {
		fmt.Printf("Warning: Could not change to project directory: %v\n", err)
	}
}
