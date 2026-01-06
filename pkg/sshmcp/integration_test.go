package sshmcp

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRealSSH_Connection 测试真实的 SSH 连接
func TestRealSSH_Connection(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试 (使用 -short 标志)")
	}

	// 从环境变量读取连接信息
	host := getEnvOrDefault("SSH_HOST", "192.168.3.7")
	port := 22
	username := getEnvOrDefault("SSH_USER", "cigar")
	password := getEnvOrDefault("SSH_PASSWORD", "liuxuejia.123")

	// 创建会话管理器
	logger := setupTestLogger(t)
	config := ManagerConfig{
		MaxSessions:        10,
		MaxSessionsPerHost: 30,
		SessionTimeout:     5 * time.Minute,
		IdleTimeout:        2 * time.Minute,
		CleanupInterval:    10 * time.Second,
		Logger:             logger,
	}

	sm := NewSessionManager(config)
	defer sm.Close()

	// 创建密码认证配置
	authConfig := &AuthConfig{
		Type:     AuthTypePassword,
		Password: password,
	}

	t.Logf("连接到 %s@%s:%d", username, host, port)

	// 创建 SSH 连接
	session, err := sm.CreateSession(host, port, username, authConfig, "")
	if err != nil {
		t.Fatalf("创建 SSH 连接失败: %v", err)
	}
	require.NotNil(t, session)

	t.Logf("✅ 成功连接到 SSH 服务器")
	t.Logf("   Session ID: %s", session.ID)
	t.Logf("   Host: %s:%d", session.Host, session.Port)
	t.Logf("   Username: %s", session.Username)

	// 验证会话状态
	assert.Equal(t, SessionStateActive, session.State)
	assert.Equal(t, host, session.Host)
	assert.Equal(t, port, session.Port)
	assert.Equal(t, username, session.Username)
	assert.NotEmpty(t, session.ID)

	// 测试断开连接
	err = sm.RemoveSession(session.ID)
	assert.NoError(t, err)
	t.Log("✅ 成功断开 SSH 连接")
}

// TestRealSSH_ExecuteCommand 测试执行命令
func TestRealSSH_ExecuteCommand(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	host := getEnvOrDefault("SSH_HOST", "192.168.3.7")
	username := getEnvOrDefault("SSH_USER", "cigar")
	password := getEnvOrDefault("SSH_PASSWORD", "liuxuejia.123")

	logger := setupTestLogger(t)
	config := ManagerConfig{
		MaxSessions:        10,
		MaxSessionsPerHost: 30,
		SessionTimeout:     5 * time.Minute,
		IdleTimeout:        2 * time.Minute,
		CleanupInterval:    10 * time.Second,
		Logger:             logger,
	}

	sm := NewSessionManager(config)
	defer sm.Close()

	authConfig := &AuthConfig{
		Type:     AuthTypePassword,
		Password: password,
	}

	// 创建连接
	session, err := sm.CreateSession(host, 22, username, authConfig, "")
	require.NoError(t, err)
	defer sm.RemoveSession(session.ID)

	t.Log("✅ 测试执行简单命令: echo hello")

	// 执行简单命令
	result, err := session.ExecuteCommand("echo 'hello from sshmcp'", 10*time.Second)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 0, result.ExitCode, "命令应该成功执行")
	assert.Contains(t, result.Stdout, "hello from sshmcp")

	t.Logf("   命令输出: %s", result.Stdout)
	t.Log("✅ 命令执行成功")

	// 测试多个命令
	t.Log("✅ 测试执行多个命令")
	commands := []string{
		"pwd",
		"whoami",
		"uname -a",
	}

	results, summary, err := session.ExecuteBatchCommands(commands, false, 10*time.Second)
	assert.NoError(t, err)
	assert.Len(t, results, 3)
	assert.Equal(t, 3, summary.Total)
	assert.Equal(t, 3, summary.Success)
	assert.Equal(t, 0, summary.Failed)

	for i, result := range results {
		t.Logf("   命令 %d: %s", i+1, commands[i])
		t.Logf("     退出码: %d", result.ExitCode)
		if result.Stdout != "" {
			t.Logf("     输出: %s", result.Stdout)
		}
	}

	t.Log("✅ 批量命令执行成功")
}

// TestRealSSH_SFTP 测试 SFTP 文件传输
func TestRealSSH_SFTP(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	host := getEnvOrDefault("SSH_HOST", "192.168.3.7")
	username := getEnvOrDefault("SSH_USER", "cigar")
	password := getEnvOrDefault("SSH_PASSWORD", "liuxuejia.123")

	logger := setupTestLogger(t)
	config := ManagerConfig{
		MaxSessions:        10,
		MaxSessionsPerHost: 30,
		SessionTimeout:     5 * time.Minute,
		IdleTimeout:        2 * time.Minute,
		CleanupInterval:    10 * time.Second,
		Logger:             logger,
	}

	sm := NewSessionManager(config)
	defer sm.Close()

	authConfig := &AuthConfig{
		Type:     AuthTypePassword,
		Password: password,
	}

	// 创建连接
	session, err := sm.CreateSession(host, 22, username, authConfig, "")
	require.NoError(t, err)
	defer sm.RemoveSession(session.ID)

	// 创建测试文件（使用跨平台路径）
	localTestFile := filepath.Join(os.TempDir(), "sshmcp_test.txt")
	remoteTestFile := "/tmp/sshmcp_test_remote.txt" // 远程服务器使用 /tmp
	testContent := "Hello from SSH MCP Server at " + time.Now().Format(time.RFC3339)

	// 写入本地测试文件
	err = os.WriteFile(localTestFile, []byte(testContent), 0644)
	require.NoError(t, err)
	defer os.Remove(localTestFile)

	t.Log("✅ 测试 SFTP 上传文件")
	result, err := session.UploadFile(localTestFile, remoteTestFile, false, true)
	assert.NoError(t, err)
	assert.Equal(t, "success", result.Status)
	assert.Greater(t, result.BytesTransferred, int64(0))
	t.Logf("   上传 %d 字节, 耗时 %s", result.BytesTransferred, result.Duration)

	// 列出远程文件
	t.Log("✅ 测试 SFTP 列出目录")
	files, err := session.ListDirectory("/tmp", false)
	assert.NoError(t, err)
	assert.NotNil(t, files)
	remoteFileFound := false
	for _, file := range files {
		if file.Name == "sshmcp_test_remote.txt" {
			remoteFileFound = true
			t.Logf("   找到远程文件: %s (%d 字节)", file.Name, file.Size)
			break
		}
	}
	assert.True(t, remoteFileFound, "应该找到上传的文件")

	// 下载文件
	t.Log("✅ 测试 SFTP 下载文件")
	localDownloadFile := filepath.Join(os.TempDir(), "sshmcp_test_download.txt")
	defer os.Remove(localDownloadFile)

	result2, err := session.DownloadFile(remoteTestFile, localDownloadFile, false, true)
	assert.NoError(t, err)
	assert.Equal(t, "success", result2.Status)
	assert.Greater(t, result2.BytesTransferred, int64(0))
	t.Logf("   下载 %d 字节, 耗时 %s", result2.BytesTransferred, result2.Duration)

	// 验证下载的内容
	downloadedContent, err := os.ReadFile(localDownloadFile)
	assert.NoError(t, err)
	assert.Equal(t, testContent, string(downloadedContent), "下载的内容应该与上传的一致")

	t.Log("✅ SFTP 文件传输测试完成")

	// 清理远程文件
	t.Log("清理远程测试文件")
	_ = session.RemoveFile(remoteTestFile, false)
}

// TestRealSSH_InteractiveShell 测试交互式 Shell
func TestRealSSH_InteractiveShell(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	host := getEnvOrDefault("SSH_HOST", "192.168.3.7")
	username := getEnvOrDefault("SSH_USER", "cigar")
	password := getEnvOrDefault("SSH_PASSWORD", "liuxuejia.123")

	logger := setupTestLogger(t)
	config := ManagerConfig{
		MaxSessions:        10,
		MaxSessionsPerHost: 30,
		SessionTimeout:     5 * time.Minute,
		IdleTimeout:        2 * time.Minute,
		CleanupInterval:    10 * time.Second,
		Logger:             logger,
	}

	sm := NewSessionManager(config)
	defer sm.Close()

	authConfig := &AuthConfig{
		Type:     AuthTypePassword,
		Password: password,
	}

	// 创建连接
	session, err := sm.CreateSession(host, 22, username, authConfig, "")
	require.NoError(t, err)
	defer sm.RemoveSession(session.ID)

	t.Log("✅ 测试创建交互式 Shell")
	shellSession, err := session.CreateShell("xterm-256color", 24, 80)
	assert.NoError(t, err)
	assert.NotNil(t, shellSession)
	assert.True(t, shellSession.PTY)
	assert.Equal(t, "xterm-256color", shellSession.TerminalInfo.Term)
	assert.Equal(t, uint16(24), shellSession.TerminalInfo.Rows)
	assert.Equal(t, uint16(80), shellSession.TerminalInfo.Cols)

	t.Log("   Shell 创建成功")

	// 测试调整终端大小
	t.Log("✅ 测试调整终端窗口大小")
	err = shellSession.Resize(40, 120)
	assert.NoError(t, err)
	assert.Equal(t, uint16(40), shellSession.TerminalInfo.Rows)
	assert.Equal(t, uint16(120), shellSession.TerminalInfo.Cols)
	t.Logf("   终端调整为 %dx%d", 40, 120)

	// 测试写入输入
	t.Log("✅ 测试向 Shell 写入输入")
	err = shellSession.WriteInput("echo test\n")
	assert.NoError(t, err)
	t.Log("   输入已写入")

	// 读取输出
	t.Log("✅ 测试从 Shell 读取输出")
	stdout, stderr, err := shellSession.ReadOutput(2 * time.Second)
	assert.NoError(t, err)
	if stdout != "" {
		t.Logf("   收到输出: %s", stdout)
	}
	if stderr != "" {
		t.Logf("   错误输出: %s", stderr)
	}

	t.Log("✅ 交互式 Shell 测试完成")
}

// TestRealSSH_SessionList 测试会话列表
func TestRealSSH_SessionList(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	host := getEnvOrDefault("SSH_HOST", "192.168.3.7")
	username := getEnvOrDefault("SSH_USER", "cigar")
	password := getEnvOrDefault("SSH_PASSWORD", "liuxuejia.123")

	logger := setupTestLogger(t)
	config := ManagerConfig{
		MaxSessions:        10,
		MaxSessionsPerHost: 30,
		SessionTimeout:     5 * time.Minute,
		IdleTimeout:        2 * time.Minute,
		CleanupInterval:    10 * time.Second,
		Logger:             logger,
	}

	sm := NewSessionManager(config)
	defer sm.Close()

	authConfig := &AuthConfig{
		Type:     AuthTypePassword,
		Password: password,
	}

	// 创建多个连接
	t.Log("✅ 测试多会话管理")
	var sessionIDs []string

	for i := 0; i < 3; i++ {
		session, err := sm.CreateSession(host, 22, username, authConfig, "")
		require.NoError(t, err)
		sessionIDs = append(sessionIDs, session.ID)
		t.Logf("   创建会话 %d: %s", i+1, session.ID)
	}

	// 列出所有会话
	sessions := sm.ListSessions()
	assert.Len(t, sessions, 3, "应该有 3 个活跃会话")
	t.Logf("   总会话数: %d", len(sessions))

	for i, sess := range sessions {
		t.Logf("   会话 %d: %s@%s:%d (%s)", i+1, sess.Username, sess.Host, sess.Port, sess.State)
	}

	// 计算特定主机的会话数
	count := sm.CountSessionsForHost(host)
	assert.Equal(t, 3, count, "应该有 3 个到该主机的会话")
	t.Logf("   主机 %s 的会话数: %d", host, count)

	// 清理所有会话
	for _, sessionID := range sessionIDs {
		err := sm.RemoveSession(sessionID)
		assert.NoError(t, err)
	}

	// 验证所有会话已关闭
	sessions = sm.ListSessions()
	assert.Len(t, sessions, 0, "所有会话应该已关闭")
	t.Log("✅ 多会话测试完成")
}

// getEnvOrDefault 获取环境变量或返回默认值
func getEnvOrDefault(key, defaultValue string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultValue
}

// TestRealSSH_AliasFeature 完整测试会话别名功能
func TestRealSSH_AliasFeature(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	host := getEnvOrDefault("SSH_HOST", "192.168.3.7")
	username := getEnvOrDefault("SSH_USER", "cigar")
	password := getEnvOrDefault("SSH_PASSWORD", "liuxuejia.123")

	logger := setupTestLogger(t)
	config := ManagerConfig{
		MaxSessions:        10,
		MaxSessionsPerHost: 30,
		SessionTimeout:     5 * time.Minute,
		IdleTimeout:        2 * time.Minute,
		CleanupInterval:    10 * time.Second,
		Logger:             logger,
	}

	sm := NewSessionManager(config)
	defer sm.Close()

	authConfig := &AuthConfig{
		Type:     AuthTypePassword,
		Password: password,
	}

	t.Log("=== 会话别名功能测试 ===")

	// 1. 创建带别名的会话
	t.Log("1. 创建带别名的会话 'prod'")
	prodSession, err := sm.CreateSession(host, 22, username, authConfig, "prod")
	require.NoError(t, err)
	defer sm.RemoveSession(prodSession.ID)

	assert.Equal(t, "prod", prodSession.Alias, "会话别名应该是 'prod'")
	t.Logf("   ✅ 创建成功: Session ID=%s, Alias=%s", prodSession.ID, prodSession.Alias)

	// 2. 创建另一个带别名的会话
	t.Log("2. 创建带别名的会话 'staging'")
	stagingSession, err := sm.CreateSession(host, 22, username, authConfig, "staging")
	require.NoError(t, err)
	defer sm.RemoveSession(stagingSession.ID)

	assert.Equal(t, "staging", stagingSession.Alias, "会话别名应该是 'staging'")
	t.Logf("   ✅ 创建成功: Session ID=%s, Alias=%s", stagingSession.ID, stagingSession.Alias)

	// 3. 测试别名冲突检测
	t.Log("3. 测试别名冲突检测")
	_, err = sm.CreateSession(host, 22, username, authConfig, "prod")
	assert.Error(t, err, "创建同名别名应该失败")
	assert.Contains(t, err.Error(), "already in use", "错误信息应该包含 'already in use'")
	t.Logf("   ✅ 冲突检测正常: %v", err)

	// 4. 测试通过别名获取会话
	t.Log("4. 测试通过别名获取会话")
	retrieved, err := sm.GetSessionByAlias("prod")
	assert.NoError(t, err)
	assert.Equal(t, prodSession.ID, retrieved.ID)
	assert.Equal(t, "prod", retrieved.Alias)
	t.Logf("   ✅ 通过别名 'prod' 成功获取会话")

	// 5. 测试通过 ID 或别名获取会话
	t.Log("5. 测试通过 ID 或别名获取会话")
	byID, err := sm.GetSessionByIDOrAlias(prodSession.ID)
	assert.NoError(t, err)
	assert.Equal(t, prodSession.ID, byID.ID)

	byAlias, err := sm.GetSessionByIDOrAlias("staging")
	assert.NoError(t, err)
	assert.Equal(t, stagingSession.ID, byAlias.ID)
	t.Logf("   ✅ GetSessionByIDOrAlias 工作正常")

	// 6. 测试使用别名执行命令
	t.Log("6. 测试使用别名执行命令")
	result, err := retrieved.ExecuteCommand("echo 'hello from prod'", 10*time.Second)
	assert.NoError(t, err)
	assert.Equal(t, 0, result.ExitCode)
	assert.Contains(t, result.Stdout, "hello from prod")
	t.Logf("   ✅ 使用别名执行命令成功: %s", result.Stdout)

	// 7. 测试不存在的别名
	t.Log("7. 测试不存在的别名")
	_, err = sm.GetSessionByAlias("non-existent")
	assert.Error(t, err)
	t.Logf("   ✅ 不存在的别名正确返回错误")

	// 8. 测试自动生成别名
	t.Log("8. 测试自动生成别名")
	autoSession, err := sm.CreateSession(host, 22, username, authConfig, "")
	require.NoError(t, err)
	defer sm.RemoveSession(autoSession.ID)

	assert.NotEmpty(t, autoSession.Alias, "应该自动生成别名")
	assert.Equal(t, "s1", autoSession.Alias, "第一个自动生成的别名应该是 's1'")
	t.Logf("   ✅ 自动生成别名: %s", autoSession.Alias)

	// 再创建一个，验证自动递增
	autoSession2, err := sm.CreateSession(host, 22, username, authConfig, "")
	require.NoError(t, err)
	defer sm.RemoveSession(autoSession2.ID)

	assert.Equal(t, "s2", autoSession2.Alias, "第二个自动生成的别名应该是 's2'")
	t.Logf("   ✅ 自动递增别名: %s", autoSession2.Alias)

	// 9. 测试列出会话时显示别名
	t.Log("9. 测试列出会话时显示别名")
	sessions := sm.ListSessions()
	assert.GreaterOrEqual(t, len(sessions), 2, "至少应该有 2 个会话")

	for _, s := range sessions {
		if s.Alias != "" {
			t.Logf("   会话 %s: 别名='%s', 主机=%s:%d", s.ID, s.Alias, s.Host, s.Port)
		}
	}

	// 验证 prod 和 staging 会话都在列表中
	foundProd := false
	foundStaging := false
	for _, s := range sessions {
		if s.Alias == "prod" {
			foundProd = true
		}
		if s.Alias == "staging" {
			foundStaging = true
		}
	}
	assert.True(t, foundProd, "应该找到别名为 'prod' 的会话")
	assert.True(t, foundStaging, "应该找到别名为 'staging' 的会话")
	t.Logf("   ✅ 会话列表正确显示别名")

	t.Log("=== 会话别名功能测试全部通过 ===")
}

// TestRealSSH_AliasEndToEnd 端到端测试：模拟实际使用场景
func TestRealSSH_AliasEndToEnd(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	host := getEnvOrDefault("SSH_HOST", "192.168.3.7")
	username := getEnvOrDefault("SSH_USER", "cigar")
	password := getEnvOrDefault("SSH_PASSWORD", "liuxuejia.123")

	logger := setupTestLogger(t)
	config := ManagerConfig{
		MaxSessions:        10,
		MaxSessionsPerHost: 30,
		SessionTimeout:     5 * time.Minute,
		IdleTimeout:        2 * time.Minute,
		CleanupInterval:    10 * time.Second,
		Logger:             logger,
	}

	sm := NewSessionManager(config)
	defer sm.Close()

	authConfig := &AuthConfig{
		Type:     AuthTypePassword,
		Password: password,
	}

	t.Log("=== 端到端场景测试 ===")

	// 场景：用户需要管理三台服务器
	t.Log("场景：用户连接到生产、测试和开发服务器")

	// 1. 连接到生产服务器
	t.Log("1. 连接到生产服务器 (192.168.68.212)，别名 'prod'")
	prod, err := sm.CreateSession(host, 22, username, authConfig, "prod")
	require.NoError(t, err)
	defer sm.RemoveSession(prod.ID)
	t.Logf("   ✅ 已连接: prod@%s", host)

	// 2. 在生产服务器上执行命令
	t.Log("2. 在生产服务器上执行命令检查磁盘空间")
	prodSession, _ := sm.GetSessionByIDOrAlias("prod")
	result, err := prodSession.ExecuteCommand("df -h | head -2", 10*time.Second)
	assert.NoError(t, err)
	assert.Equal(t, 0, result.ExitCode)
	t.Logf("   ✅ 磁盘使用情况:\n%s", result.Stdout)

	// 3. 连接到测试服务器（实际上还是同一台，但用不同别名模拟）
	t.Log("3. 连接到测试服务器，别名 'testing'")
	testing, err := sm.CreateSession(host, 22, username, authConfig, "testing")
	require.NoError(t, err)
	defer sm.RemoveSession(testing.ID)
	t.Logf("   ✅ 已连接: testing@%s", host)

	// 4. 在测试服务器上执行命令（使用别名）
	t.Log("4. 在测试服务器上执行命令（使用别名）")
	testingSession, _ := sm.GetSessionByIDOrAlias("testing")
	result, err = testingSession.ExecuteCommand("uname -a", 10*time.Second)
	assert.NoError(t, err)
	t.Logf("   ✅ 系统信息: %s", result.Stdout)

	// 5. 列出所有活跃连接
	t.Log("5. 列出所有活跃连接")
	sessions := sm.ListSessions()
	t.Logf("   当前活跃连接数: %d", len(sessions))
	for _, s := range sessions {
		t.Logf("   - %s: %s (%s@%s:%d)", s.Alias, s.ID[:8], s.Username, s.Host, s.Port)
	}

	// 6. 使用别名断开特定连接
	t.Log("6. 断开测试服务器连接（使用别名）")
	err = sm.RemoveSession(testing.ID)
	assert.NoError(t, err)
	t.Logf("   ✅ 已断开 'testing' 连接")

	// 7. 验证 prod 连接仍然活跃
	t.Log("7. 验证 prod 连接仍然活跃")
	prodStillActive, err := sm.GetSessionByAlias("prod")
	assert.NoError(t, err)
	assert.Equal(t, prod.ID, prodStillActive.ID)
	t.Logf("   ✅ 'prod' 连接仍然活跃")

	t.Log("=== 端到端场景测试通过 ===")
}
