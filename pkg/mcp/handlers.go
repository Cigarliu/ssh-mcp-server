package mcp

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/cigar/sshmcp/pkg/sshmcp"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// formatBytes converts bytes to human-readable format
func formatBytes(bytes float64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%.1f B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", bytes/float64(div), "KMGTPE"[exp])
}

// Tool handler parameter structures (使用 map[string]any 作为输入类型)

// handleSSHConnect handles the ssh_connect tool
func (s *Server) handleSSHConnect(ctx context.Context, req *mcp.CallToolRequest, args map[string]any) (*mcp.CallToolResult, any, error) {
	hostname, _ := args["hostname"].(string)
	host, _ := args["host"].(string)
	username, _ := args["username"].(string)
	authType, _ := args["auth_type"].(string)
	password, _ := args["password"].(string)
	privateKey, _ := args["private_key"].(string)
	passphrase, _ := args["passphrase"].(string)
	sudoPassword, _ := args["sudo_password"].(string)
	portVal, _ := args["port"].(float64)
	alias, _ := args["alias"].(string)

	// If hostname is provided, load from predefined hosts
	if hostname != "" {
		if s.hostManager == nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: "Host manager is not available"}},
				IsError: true,
			}, nil, nil
		}

		hostConfig, err := s.hostManager.GetHost(hostname)
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Host '%s' not found: %v\nUse ssh_list_hosts to see available hosts", hostname, err)}},
				IsError: true,
			}, nil, nil
		}

		// Use values from host config if not explicitly provided
		if host == "" {
			host = hostConfig.Host
		}
		if username == "" {
			username = hostConfig.Username
		}
		if portVal == 0 && hostConfig.Port > 0 {
			portVal = float64(hostConfig.Port)
		}
		if password == "" && hostConfig.Password != "" {
			password = hostConfig.Password
			authType = "password"
		}
		if privateKey == "" && hostConfig.PrivateKeyPath != "" {
			privateKey = hostConfig.PrivateKeyPath
			authType = "private_key"
		}
	}

	// Validate required parameters
	if host == "" {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: "Host address is required (provide either 'host' or 'hostname')"}},
			IsError: true,
		}, nil, nil
	}

	if username == "" {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: "Username is required"}},
			IsError: true,
		}, nil, nil
	}

	port := int(portVal)
	if port == 0 {
		port = 22
	}

	// 默认认证类型为 password
	if authType == "" {
		authType = "password"
	}

	authConfig := &sshmcp.AuthConfig{
		Type:         sshmcp.AuthType(authType),
		SudoPassword: sudoPassword, // 设置 sudo 密码
	}

	switch authConfig.Type {
	case sshmcp.AuthTypePassword:
		authConfig.Password = password
	case sshmcp.AuthTypePrivateKey:
		authConfig.PrivateKey = privateKey
		authConfig.Passphrase = passphrase
	case sshmcp.AuthTypeSSHAgent:
		// SSH Agent 暂不支持
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: "SSH agent authentication is not yet implemented"}},
			IsError: true,
		}, nil, nil
	default:
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Unsupported auth type: %s", authType)}},
			IsError: true,
		}, nil, nil
	}

	session, err := s.sessionManager.CreateSession(host, port, username, authConfig, alias)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Failed to create session: %v", err)}},
			IsError: true,
		}, nil, nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Successfully connected to %s@%s:%d\nSession ID: %s\nAlias: %s",
			username, host, port, session.ID, session.Alias)}},
	}, nil, nil
}

// handleSSHDisconnect handles the ssh_disconnect tool
func (s *Server) handleSSHDisconnect(ctx context.Context, req *mcp.CallToolRequest, args map[string]any) (*mcp.CallToolResult, any, error) {
	sessionID, _ := args["session_id"].(string)

	err := s.sessionManager.RemoveSession(sessionID)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Failed to disconnect: %v", err)}},
			IsError: true,
		}, nil, nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Session %s closed successfully", sessionID)}},
	}, nil, nil
}

// handleSSHListSessions handles the ssh_list_sessions tool
func (s *Server) handleSSHListSessions(ctx context.Context, req *mcp.CallToolRequest, args map[string]any) (*mcp.CallToolResult, any, error) {
	sessions := s.sessionManager.ListSessions()

	output := fmt.Sprintf("Total sessions: %d\n\n", len(sessions))
	for _, session := range sessions {
		session.RLock()
		output += fmt.Sprintf("- Session ID: %s\n", session.ID)
		if session.Alias != "" {
			output += fmt.Sprintf("  Alias: %s\n", session.Alias)
		}
		output += fmt.Sprintf("  Host: %s:%d\n", session.Host, session.Port)
		output += fmt.Sprintf("  Username: %s\n", session.Username)
		output += fmt.Sprintf("  State: %s\n", session.State)
		output += fmt.Sprintf("  Created: %s\n", session.CreatedAt.Format(time.RFC3339))
		output += fmt.Sprintf("  Last Used: %s\n\n", session.LastUsedAt.Format(time.RFC3339))
		session.RUnlock()
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: output}},
	}, nil, nil
}

// handleSSHExec handles the ssh_exec tool
func (s *Server) handleSSHExec(ctx context.Context, req *mcp.CallToolRequest, args map[string]any) (*mcp.CallToolResult, any, error) {
	sessionID, _ := args["session_id"].(string)
	command, _ := args["command"].(string)
	timeoutVal, _ := args["timeout"].(float64)
	workingDir, _ := args["working_dir"].(string)

	session, err := s.sessionManager.GetSessionByIDOrAlias(sessionID)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Session not found: %v\nHint: Use ssh_list_sessions() to see all active sessions", err)}},
			IsError: true,
		}, nil, nil
	}

	timeout := 30 * time.Second
	if timeoutVal > 0 {
		timeout = time.Duration(timeoutVal) * time.Second
	}

	var result *sshmcp.CommandResult
	if workingDir != "" {
		result, err = session.ExecuteCommandWithWorkingDir(command, workingDir, timeout)
	} else {
		result, err = session.ExecuteCommand(command, timeout)
	}

	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Command execution failed: %v", err)}},
			IsError: true,
		}, nil, nil
	}

	output := fmt.Sprintf("Exit Code: %d\n\n", result.ExitCode)
	if result.Stdout != "" {
		output += fmt.Sprintf("STDOUT:\n%s\n\n", result.Stdout)
	}
	if result.Stderr != "" {
		output += fmt.Sprintf("STDERR:\n%s\n\n", result.Stderr)
	}
	output += fmt.Sprintf("Execution Time: %s", result.ExecutionTime)

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: output}},
	}, nil, nil
}

// handleSSHExecBatch handles the ssh_exec_batch tool
func (s *Server) handleSSHExecBatch(ctx context.Context, req *mcp.CallToolRequest, args map[string]any) (*mcp.CallToolResult, any, error) {
	sessionID, _ := args["session_id"].(string)
	commandsInterface, _ := args["commands"].([]any)
	stopOnErrorVal, _ := args["stop_on_error"].(bool)
	timeoutVal, _ := args["timeout"].(float64)
	compactVal, _ := args["compact"].(bool)

	commands := make([]string, len(commandsInterface))
	for i, cmd := range commandsInterface {
		commands[i], _ = cmd.(string)
	}

	session, err := s.sessionManager.GetSessionByIDOrAlias(sessionID)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Session not found: %v\nHint: Use ssh_list_sessions() to see all active sessions", err)}},
			IsError: true,
		}, nil, nil
	}

	timeout := 30 * time.Second
	if timeoutVal > 0 {
		timeout = time.Duration(timeoutVal) * time.Second
	}

	results, summary, err := session.ExecuteBatchCommands(commands, stopOnErrorVal, timeout)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Batch execution failed: %v", err)}},
			IsError: true,
		}, nil, nil
	}

	// compact 模式：简洁输出
	if compactVal {
		output := "✓ Batch execution completed\n"
		output += fmt.Sprintf("  Total: %d | Success: %d | Failed: %d\n", summary.Total, summary.Success, summary.Failed)
		if summary.Failed > 0 {
			output += "\nFailed commands:\n"
			for i, result := range results {
				if result.ExitCode != 0 {
					output += fmt.Sprintf("  %d. %s (exit: %d)\n", i+1, commands[i], result.ExitCode)
				}
			}
		}
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: output}},
		}, nil, nil
	}

	// 默认：详细输出
	output := fmt.Sprintf("Batch Execution Summary:\n")
	output += fmt.Sprintf("Total: %d, Success: %d, Failed: %d\n\n", summary.Total, summary.Success, summary.Failed)

	for i, result := range results {
		output += fmt.Sprintf("Command %d: %s\n", i+1, commands[i])
		output += fmt.Sprintf("Exit Code: %d\n", result.ExitCode)
		if result.Stdout != "" {
			output += fmt.Sprintf("STDOUT: %s\n", result.Stdout)
		}
		if result.Stderr != "" {
			output += fmt.Sprintf("STDERR: %s\n", result.Stderr)
		}
		output += "\n"
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: output}},
	}, nil, nil
}

// handleSSHShell handles the ssh_shell tool
func (s *Server) handleSSHShell(ctx context.Context, req *mcp.CallToolRequest, args map[string]any) (*mcp.CallToolResult, any, error) {
	sessionID, _ := args["session_id"].(string)
	rowsVal, _ := args["rows"].(float64)
	colsVal, _ := args["cols"].(float64)
	workingDir, _ := args["working_dir"].(string)

	session, err := s.sessionManager.GetSessionByIDOrAlias(sessionID)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Session not found: %v\nHint: Use ssh_list_sessions() to see all active sessions", err)}},
			IsError: true,
		}, nil, nil
	}

	rows := uint16(rowsVal)
	cols := uint16(colsVal)
	if rows == 0 {
		rows = 40  // 默认 40 行，适合 htop
	}
	if cols == 0 {
		cols = 160  // 默认 160 列，适合查看表格
	}

	// 创建 Shell 配置（自动设置为 raw 模式）
	config := sshmcp.DefaultShellConfig()
	config.Mode = sshmcp.TerminalModeRaw  // 强制使用 raw 模式（交互式程序专用）
	config.ANSIMode = sshmcp.ANSIRaw      // 保留 ANSI 序列（支持颜色和光标）
	// read_timeout 使用默认值 100ms

	// 使用固定的终端类型
	term := "xterm-256color"

	// 使用配置创建 Shell
	shellSession, err := session.CreateShellWithConfig(term, rows, cols, config)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Failed to create shell: %v", err)}},
			IsError: true,
		}, nil, nil
	}

	// 如果指定了工作目录，切换到该目录
	var workingDirMsg string
	if workingDir != "" {
		shellSession.WriteInput(fmt.Sprintf("cd %s\n", workingDir))
		workingDirMsg = fmt.Sprintf("- 初始目录: %s\n", workingDir)
	}

	// 获取会话状态
	status := shellSession.GetStatus()

	// ⚡ 立即返回，不等待输出（异步模式）
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{
			Text: fmt.Sprintf(`✅ Shell 会话已启动（后台运行模式）

📋 会话信息：
- 会话 ID: %s
- 模式: %s
- 终端: %dx%d
- ANSI 模式: %s
%s
💾 后台缓冲区：
- 容量: %d 行 (~%d MB)
- 状态: 输出持续读取中

❤️ 保活机制（已启用）：
- TCP Keepalive: 30秒间隔
- SSH Keepalive: 30秒间隔
- 应用层心跳: 60秒间隔

🔧 后续操作指引：

1️⃣ 发送命令（启动交互式程序）：
   ssh_write_input(session_id="%s", input="htop")

2️⃣ 查看界面：

   a) 查看交互式程序界面（推荐）：
      ssh_terminal_snapshot(session_id="%s")

   b) 查看大量文本输出（日志等）：
      ssh_read_output(session_id="%s", strategy="latest_lines", limit=50)

3️⃣ 查看会话状态：
   ssh_shell_status(session_id="%s")

4️⃣ 退出交互式程序：
   ssh_write_input(session_id="%s", special_char="ctrl+c")  # 中断程序

💡 提示：
- 本会话专门用于交互式程序（htop/vim/gdb/tmux）
- 简单命令建议使用 ssh_exec，更高效
- 使用 ssh_terminal_snapshot 查看完整的交互式界面
- 会话在后台持续运行，随时可查看
`,
				func() string {
					if session.Alias != "" {
						return session.Alias
					}
					return sessionID
				}(),
				"raw",  // 固定为 raw 模式
				cols, rows,
				"raw",  // 固定为 raw ANSI 模式
				workingDirMsg,
				status.BufferTotal,
				status.BufferTotal / 1024,  // 估算 KB
				sessionID, sessionID, sessionID, sessionID),
		}},
	}, nil, nil
}

// handleSFTPUpload handles the sftp_upload tool
func (s *Server) handleSFTPUpload(ctx context.Context, req *mcp.CallToolRequest, args map[string]any) (*mcp.CallToolResult, any, error) {
	sessionID, _ := args["session_id"].(string)
	localPath, _ := args["local_path"].(string)
	remotePath, _ := args["remote_path"].(string)
	createDirsVal, _ := args["create_dirs"].(bool)
	overwriteVal, _ := args["overwrite"].(bool)

	session, err := s.sessionManager.GetSessionByIDOrAlias(sessionID)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Session not found: %v\nHint: Use ssh_list_sessions() to see all active sessions", err)}},
			IsError: true,
		}, nil, nil
	}

	result, err := session.UploadFile(localPath, remotePath, createDirsVal, overwriteVal)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Upload failed: %v", err)}},
			IsError: true,
		}, nil, nil
	}

	// 构建详细的输出消息
	output := fmt.Sprintf("Upload successful:\n")
	output += fmt.Sprintf("  Status: %s\n", result.Status)
	output += fmt.Sprintf("  Local: %s\n", localPath)
	output += fmt.Sprintf("  Remote: %s\n", remotePath)
	output += fmt.Sprintf("  Size: %s\n", formatBytes(float64(result.FileSize)))
	output += fmt.Sprintf("  Transferred: %s\n", formatBytes(float64(result.BytesTransferred)))
	output += fmt.Sprintf("  Progress: %.1f%%\n", result.Progress)
	if result.Speed != "" {
		output += fmt.Sprintf("  Speed: %s\n", result.Speed)
	}
	output += fmt.Sprintf("  Duration: %s\n", result.Duration)

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: output}},
	}, nil, nil
}

// handleSFTPDownload handles the sftp_download tool
func (s *Server) handleSFTPDownload(ctx context.Context, req *mcp.CallToolRequest, args map[string]any) (*mcp.CallToolResult, any, error) {
	sessionID, _ := args["session_id"].(string)
	remotePath, _ := args["remote_path"].(string)
	localPath, _ := args["local_path"].(string)
	createDirsVal, _ := args["create_dirs"].(bool)
	overwriteVal, _ := args["overwrite"].(bool)

	session, err := s.sessionManager.GetSessionByIDOrAlias(sessionID)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Session not found: %v\nHint: Use ssh_list_sessions() to see all active sessions", err)}},
			IsError: true,
		}, nil, nil
	}

	result, err := session.DownloadFile(remotePath, localPath, createDirsVal, overwriteVal)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Download failed: %v", err)}},
			IsError: true,
		}, nil, nil
	}

	// 构建详细的输出消息
	output := fmt.Sprintf("Download successful:\n")
	output += fmt.Sprintf("  Status: %s\n", result.Status)
	output += fmt.Sprintf("  Remote: %s\n", remotePath)
	output += fmt.Sprintf("  Local: %s\n", localPath)
	output += fmt.Sprintf("  Size: %s\n", formatBytes(float64(result.FileSize)))
	output += fmt.Sprintf("  Transferred: %s\n", formatBytes(float64(result.BytesTransferred)))
	output += fmt.Sprintf("  Progress: %.1f%%\n", result.Progress)
	if result.Speed != "" {
		output += fmt.Sprintf("  Speed: %s\n", result.Speed)
	}
	output += fmt.Sprintf("  Duration: %s\n", result.Duration)

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: output}},
	}, nil, nil
}

// handleSFTPListDir handles the sftp_list_dir tool
func (s *Server) handleSFTPListDir(ctx context.Context, req *mcp.CallToolRequest, args map[string]any) (*mcp.CallToolResult, any, error) {
	sessionID, _ := args["session_id"].(string)
	remotePath, _ := args["remote_path"].(string)
	recursiveVal, _ := args["recursive"].(bool)

	session, err := s.sessionManager.GetSessionByIDOrAlias(sessionID)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Session not found: %v\nHint: Use ssh_list_sessions() to see all active sessions", err)}},
			IsError: true,
		}, nil, nil
	}

	files, err := session.ListDirectory(remotePath, recursiveVal)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("List directory failed: %v", err)}},
			IsError: true,
		}, nil, nil
	}

	output := fmt.Sprintf("Directory listing for: %s\n", remotePath)
	output += fmt.Sprintf("Total entries: %d\n\n", len(files))

	for _, file := range files {
		output += fmt.Sprintf("- %s (%s, %d bytes)\n", file.Name, file.Type, file.Size)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: output}},
	}, nil, nil
}

// handleSFTPMkdir handles the sftp_mkdir tool
func (s *Server) handleSFTPMkdir(ctx context.Context, req *mcp.CallToolRequest, args map[string]any) (*mcp.CallToolResult, any, error) {
	sessionID, _ := args["session_id"].(string)
	remotePath, _ := args["remote_path"].(string)
	recursiveVal, _ := args["recursive"].(bool)
	modeVal, _ := args["mode"].(float64)

	session, err := s.sessionManager.GetSessionByIDOrAlias(sessionID)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Session not found: %v\nHint: Use ssh_list_sessions() to see all active sessions", err)}},
			IsError: true,
		}, nil, nil
	}

	mode := os.FileMode(modeVal)
	if mode == 0 {
		mode = 0755
	}

	err = session.MakeDirectory(remotePath, recursiveVal, mode)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Mkdir failed: %v", err)}},
			IsError: true,
		}, nil, nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Directory created: %s", remotePath)}},
	}, nil, nil
}

// handleSFTPDelete handles the sftp_delete tool
func (s *Server) handleSFTPDelete(ctx context.Context, req *mcp.CallToolRequest, args map[string]any) (*mcp.CallToolResult, any, error) {
	sessionID, _ := args["session_id"].(string)
	remotePath, _ := args["remote_path"].(string)
	recursiveVal, _ := args["recursive"].(bool)

	session, err := s.sessionManager.GetSessionByIDOrAlias(sessionID)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Session not found: %v\nHint: Use ssh_list_sessions() to see all active sessions", err)}},
			IsError: true,
		}, nil, nil
	}

	err = session.RemoveFile(remotePath, recursiveVal)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Delete failed: %v", err)}},
			IsError: true,
		}, nil, nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Deleted: %s", remotePath)}},
	}, nil, nil
}

// handleSSHWriteInput handles the ssh_write_input tool
func (s *Server) handleSSHWriteInput(ctx context.Context, req *mcp.CallToolRequest, args map[string]any) (*mcp.CallToolResult, any, error) {
	sessionID, _ := args["session_id"].(string)
	input, _ := args["input"].(string)
	specialChar, _ := args["special_char"].(string)

	session, err := s.sessionManager.GetSessionByIDOrAlias(sessionID)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Session not found: %v\nHint: Use ssh_list_sessions() to see all active sessions", err)}},
			IsError: true,
		}, nil, nil
	}

	session.RLock()
	hasShell := session.ShellSession != nil
	session.RUnlock()

	if !hasShell {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("No active shell session for session_id: %s\nHint: Use ssh_shell() to start an interactive shell first", sessionID)}},
			IsError: true,
		}, nil, nil
	}

	// Use special character if provided
	if specialChar != "" {
		err = session.ShellSession.WriteSpecialChars(specialChar)
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Write special character failed: %v", err)}},
				IsError: true,
			}, nil, nil
		}
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Special character '%s' sent to shell session %s", specialChar, sessionID)}},
		}, nil, nil
	}

	// Check if input contains newline - if so, automatically send Enter after writing
	containsNewline := strings.Contains(input, "\n")
	if containsNewline {
		// Split by newline and write each part
		lines := strings.Split(input, "\n")
		for i, line := range lines {
			if len(line) > 0 {
				err = session.ShellSession.WriteInput(line)
				if err != nil {
					return &mcp.CallToolResult{
						Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Write input failed: %v", err)}},
						IsError: true,
					}, nil, nil
				}
			}
			// Send Enter after each line except the last empty one
			if i < len(lines)-1 || (len(lines) > 0 && lines[len(lines)-1] == "") {
				err = session.ShellSession.WriteSpecialChars("enter")
				if err != nil {
					return &mcp.CallToolResult{
						Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Send Enter failed: %v", err)}},
						IsError: true,
					}, nil, nil
				}
			}
		}
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Input written to shell session %s (auto-sent Enter due to newline)", sessionID)}},
		}, nil, nil
	}

	// Otherwise write regular input
	err = session.ShellSession.WriteInput(input)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Write input failed: %v", err)}},
			IsError: true,
		}, nil, nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Input written to shell session %s", sessionID)}},
	}, nil, nil
}

// handleSSHReadOutput handles the ssh_read_output tool (异步模式)
func (s *Server) handleSSHReadOutput(ctx context.Context, req *mcp.CallToolRequest, args map[string]any) (*mcp.CallToolResult, any, error) {
	sessionID, _ := args["session_id"].(string)
	strategy, _ := args["strategy"].(string)
	limitVal, _ := args["limit"].(float64)

	session, err := s.sessionManager.GetSessionByIDOrAlias(sessionID)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Session not found: %v\nHint: Use ssh_list_sessions() to see all active sessions", err)}},
			IsError: true,
		}, nil, nil
	}

	session.RLock()
	hasShell := session.ShellSession != nil
	session.RUnlock()

	if !hasShell {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("No active shell session for session_id: %s\nHint: Use ssh_shell() to start an interactive shell first", sessionID)}},
			IsError: true,
		}, nil, nil
	}

	shellSession := session.ShellSession

	// 获取当前状态
	status := shellSession.GetStatus()

	// 设置默认值
	if strategy == "" {
		strategy = "latest_lines"
	}

	limit := 20
	if limitVal > 0 {
		limit = int(limitVal)
	}

	// 根据 strategy 读取数据
	var output string
	var lineCount int
	var byteCount int

	switch strategy {
	case "latest_lines":
		lines := shellSession.OutputBuffer.ReadLatestLines(limit)
		output = strings.Join(lines, "\n")
		lineCount = len(lines)
		if output != "" {
			byteCount = len(output)
		}

	case "all_unread":
		lines := shellSession.OutputBuffer.ReadAllUnread()
		output = strings.Join(lines, "\n")
		lineCount = len(lines)
		if output != "" {
			byteCount = len(output)
		}

	case "latest_bytes":
		output = shellSession.OutputBuffer.ReadLatestBytes(limit)
		if output != "" {
			byteCount = len(output)
			lineCount = len(strings.Split(output, "\n"))
		}

	default:
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Invalid strategy: %s\nValid strategies: latest_lines, all_unread, latest_bytes", strategy)}},
			IsError: true,
		}, nil, nil
	}

	// 重新获取状态（可能已更新）
	status = shellSession.GetStatus()

	// 计算缓冲区使用率
	bufferPercent := float64(status.BufferUsed) / float64(status.BufferTotal) * 100

	// 构建返回消息
	var result string
	if output != "" {
		result = fmt.Sprintf(`📄 输出读取结果

读取策略: %s
读取行数: %d
读取字节数: %d
剩余未读: %d 行

💾 缓冲区状态：
- 已用: %d/%d 行 (%.1f%%)

--- 输出内容 ---
%s
--- 输出结束 ---

💡 提示：
- 如需查看更多输出，增加 limit 参数
- 如需查看所有未读输出，使用 strategy="all_unread"
- 查看详细状态：ssh_shell_status(session_id="%s")`,
			strategy,
			lineCount,
			byteCount,
			status.BufferUsed,
			status.BufferUsed,
			status.BufferTotal,
			bufferPercent,
			output,
			sessionID)
	} else {
		result = fmt.Sprintf(`📄 输出读取结果

读取策略: %s
结果: 无新输出

💾 缓冲区状态：
- 已用: %d/%d 行 (%.1f%%)
- 未读数据: 否

💡 提示：
- 暂无新输出，可能需要：
  1. 等待程序产生输出
  2. 发送命令或输入
  3. 检查会话状态：ssh_shell_status(session_id="%s")`,
			strategy,
			status.BufferUsed,
			status.BufferTotal,
			bufferPercent,
			sessionID)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: result}},
	}, nil, nil
}

// handleSSHResizePty handles the ssh_resize_pty tool
func (s *Server) handleSSHResizePty(ctx context.Context, req *mcp.CallToolRequest, args map[string]any) (*mcp.CallToolResult, any, error) {
	sessionID, _ := args["session_id"].(string)
	rowsVal, _ := args["rows"].(float64)
	colsVal, _ := args["cols"].(float64)

	session, err := s.sessionManager.GetSessionByIDOrAlias(sessionID)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Session not found: %v\nHint: Use ssh_list_sessions() to see all active sessions", err)}},
			IsError: true,
		}, nil, nil
	}

	session.RLock()
	hasShell := session.ShellSession != nil
	session.RUnlock()

	if !hasShell {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("No active shell session for session_id: %s\nHint: Use ssh_shell() to start an interactive shell first", sessionID)}},
			IsError: true,
		}, nil, nil
	}

	rows := uint16(rowsVal)
	cols := uint16(colsVal)

	err = session.ShellSession.Resize(rows, cols)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Resize PTY failed: %v", err)}},
			IsError: true,
		}, nil, nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Terminal resized to %dx%d for session %s", rows, cols, sessionID)}},
	}, nil, nil
}

// handleSSHTerminalSnapshot handles the ssh_terminal_snapshot tool
func (s *Server) handleSSHTerminalSnapshot(ctx context.Context, req *mcp.CallToolRequest, args map[string]any) (*mcp.CallToolResult, any, error) {
	sessionID, _ := args["session_id"].(string)
	withColor, _ := args["with_color"].(bool)
	includeCursorInfo, _ := args["include_cursor_info"].(bool)

	session, err := s.sessionManager.GetSessionByIDOrAlias(sessionID)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Session not found: %v\nHint: Use ssh_list_sessions() to see all active sessions", err)}},
			IsError: true,
		}, nil, nil
	}

	session.RLock()
	hasShell := session.ShellSession != nil
	session.RUnlock()

	if !hasShell {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("No active shell session for session_id: %s\nHint: Use ssh_shell() to start an interactive shell first", sessionID)}},
			IsError: true,
		}, nil, nil
	}

	// Get the terminal snapshot
	var snapshot string
	if withColor {
		snapshot = session.ShellSession.GetTerminalSnapshotWithColor()
	} else {
		snapshot = session.ShellSession.GetTerminalSnapshot()
	}

	// Build result
	result := fmt.Sprintf("📸 Terminal Snapshot for session %s\n\n", sessionID)

	if includeCursorInfo {
		x, y := session.ShellSession.GetCursorPosition()
		w, h := session.ShellSession.GetTerminalSize()
		result += fmt.Sprintf("Cursor Position: (%d, %d)\n", x, y)
		result += fmt.Sprintf("Terminal Size: %dx%d\n\n", w, h)
	}

	result += "```\n"
	result += snapshot
	result += "\n```"

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: result}},
	}, nil, nil
}

// handleSSHListHosts handles the ssh_list_hosts tool
func (s *Server) handleSSHListHosts(ctx context.Context, req *mcp.CallToolRequest, args map[string]any) (*mcp.CallToolResult, any, error) {
	if s.hostManager == nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: "Host manager is not available"}},
			IsError: true,
		}, nil, nil
	}

	hosts := s.hostManager.ListHosts()

	if len(hosts) == 0 {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: "No predefined hosts configured.\nYou can save hosts using ssh_save_host."}},
		}, nil, nil
	}

	output := fmt.Sprintf("Predefined hosts (%d):\n\n", len(hosts))
	for name, host := range hosts {
		output += fmt.Sprintf("- %s:\n", name)
		output += fmt.Sprintf("  Host: %s:%d\n", host.Host, host.Port)
		output += fmt.Sprintf("  Username: %s\n", host.Username)
		if host.Description != "" {
			output += fmt.Sprintf("  Description: %s\n", host.Description)
		}
		if host.Password != "" {
			output += "  Auth: password\n"
		} else if host.PrivateKeyPath != "" {
			output += fmt.Sprintf("  Auth: private_key (%s)\n", host.PrivateKeyPath)
		}
		output += "\n"
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: output}},
	}, nil, nil
}

// handleSSHSaveHost handles the ssh_save_host tool
func (s *Server) handleSSHSaveHost(ctx context.Context, req *mcp.CallToolRequest, args map[string]any) (*mcp.CallToolResult, any, error) {
	if s.hostManager == nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: "Host manager is not available"}},
			IsError: true,
		}, nil, nil
	}

	name, _ := args["name"].(string)
	host, _ := args["host"].(string)
	username, _ := args["username"].(string)
	portVal, _ := args["port"].(float64)
	password, _ := args["password"].(string)
	privateKeyPath, _ := args["private_key_path"].(string)
	description, _ := args["description"].(string)

	if name == "" {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: "Host name is required"}},
			IsError: true,
		}, nil, nil
	}

	if host == "" {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: "Host address is required"}},
			IsError: true,
		}, nil, nil
	}

	if username == "" {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: "Username is required"}},
			IsError: true,
		}, nil, nil
	}

	// Check if host already exists
	if s.hostManager.HostExists(name) {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Host '%s' already exists. Please use a different name or remove the existing host first.", name)}},
			IsError: true,
		}, nil, nil
	}

	port := int(portVal)
	if port == 0 {
		port = 22
	}

	hostConfig := sshmcp.HostConfig{
		Host:            host,
		Port:            port,
		Username:        username,
		Password:        password,
		PrivateKeyPath:  privateKeyPath,
		Description:     description,
	}

	if err := s.hostManager.SaveHost(name, hostConfig); err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Failed to save host: %v", err)}},
			IsError: true,
		}, nil, nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Host '%s' saved successfully.\nYou can now connect using: hostname=%s", name, name)}},
	}, nil, nil
}

// handleSSHRemoveHost handles the ssh_remove_host tool
func (s *Server) handleSSHRemoveHost(ctx context.Context, req *mcp.CallToolRequest, args map[string]any) (*mcp.CallToolResult, any, error) {
	if s.hostManager == nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: "Host manager is not available"}},
			IsError: true,
		}, nil, nil
	}

	name, _ := args["name"].(string)

	if name == "" {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: "Host name is required"}},
			IsError: true,
		}, nil, nil
	}

	if err := s.hostManager.RemoveHost(name); err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Failed to remove host: %v", err)}},
			IsError: true,
		}, nil, nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Host '%s' removed successfully", name)}},
	}, nil, nil
}

// handleSSHShellStatus handles the ssh_shell_status tool
func (s *Server) handleSSHShellStatus(ctx context.Context, req *mcp.CallToolRequest, args map[string]any) (*mcp.CallToolResult, any, error) {
	sessionID, _ := args["session_id"].(string)

	session, err := s.sessionManager.GetSessionByIDOrAlias(sessionID)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Session not found: %v\nHint: Use ssh_list_sessions() to see all active sessions", err)}},
			IsError: true,
		}, nil, nil
	}

	session.RLock()
	hasShell := session.ShellSession != nil
	session.RUnlock()

	if !hasShell {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("No active shell session for session_id: %s\nHint: Use ssh_shell() to start an interactive shell first", sessionID)}},
			IsError: true,
		}, nil, nil
	}

	status := session.ShellSession.GetStatus()

	// 计算缓冲区使用百分比
	bufferPercent := 0.0
	if status.BufferTotal > 0 {
		bufferPercent = float64(status.BufferUsed) / float64(status.BufferTotal) * 100
	}

	// 计算距离上次保活的时间
	lastKeepalive := "未记录"
	if !status.LastKeepAlive.IsZero() {
		lastKeepalive = fmt.Sprintf("%s 前", formatDuration(time.Since(status.LastKeepAlive)))
	}

	// 格式化输出（异步模式增强版）
	output := "🔍 Shell 会话状态\n\n"

	// === 基本信息 ===
	output += "📋 基本信息:\n"
	output += fmt.Sprintf("  会话 ID: %s\n", sessionID)
	if session.Alias != "" {
		output += fmt.Sprintf("  会话别名: %s\n", session.Alias)
	}
	output += fmt.Sprintf("  状态: %s\n", getStatusEmoji(status.IsActive))
	output += fmt.Sprintf("  当前目录: %s\n", status.CurrentDir)
	output += fmt.Sprintf("  终端: %s (%dx%d)\n", status.TerminalType, status.Rows, status.Cols)
	output += fmt.Sprintf("  模式: %s\n", status.Mode)
	output += fmt.Sprintf("  ANSI 处理: %s\n", status.ANSIMode)
	output += "\n"

	// === 活动时间 ===
	output += "⏱️ 活动时间:\n"
	output += fmt.Sprintf("  最后读取: %s\n", formatTimeAgo(status.LastReadTime))
	output += fmt.Sprintf("  最后写入: %s\n", formatTimeAgo(status.LastWriteTime))
	output += fmt.Sprintf("  会话时长: %s\n", formatDuration(time.Since(session.CreatedAt)))
	output += "\n"

	// === 缓冲区状态 ===
	output += "💾 后台缓冲区:\n"
	output += fmt.Sprintf("  使用量: %d / %d 行 (%.1f%%)\n", status.BufferUsed, status.BufferTotal, bufferPercent)
	if status.BufferUsed > 0 {
		// 估算缓冲区大小（假设平均每行 100 字节）
		estimatedSize := float64(status.BufferUsed) * 100 / 1024 / 1024
		output += fmt.Sprintf("  估算大小: ~%.2f MB\n", estimatedSize)
	}

	// 缓冲区健康度提示
	if bufferPercent > 90 {
		output += "  ⚠️ 警告: 缓冲区接近满载，建议尽快读取或清空\n"
	} else if bufferPercent > 70 {
		output += "  ⚡ 提示: 缓冲区使用较高，定期读取可避免数据丢失\n"
	} else if status.BufferUsed == 0 {
		output += "  ℹ️ 缓冲区为空，使用 ssh_write_input 发送命令后使用 ssh_read_output 读取\n"
	} else {
		output += "  ✅ 缓冲区状态正常\n"
	}
	output += "\n"

	// === 保活状态 ===
	output += "❤️ 保活机制:\n"
	output += fmt.Sprintf("  TCP Keepalive: 启用 (30秒间隔)\n")
	output += fmt.Sprintf("  SSH Keepalive: 启用 (30秒间隔)\n")
	output += fmt.Sprintf("  应用层心跳: 启用 (60秒间隔)\n")
	output += fmt.Sprintf("  上次成功: %s\n", lastKeepalive)

	// 保活健康度提示
	if status.KeepAliveFails > 0 {
		output += fmt.Sprintf("  ⚠️ 连续失败: %d 次\n", status.KeepAliveFails)
		if status.KeepAliveFails >= 3 {
			output += "  🚨 严重: 会话可能已断开，建议重新连接\n"
		} else {
			output += "  ⚡ 提示: 检测到网络不稳定，监控中...\n"
		}
	} else {
		output += "  ✅ 保活状态正常\n"
	}
	output += "\n"

	// === 推荐操作 ===
	output += "🎯 推荐操作:\n"
	if !status.IsActive {
		output += "  ❌ 会话已断开，请使用 ssh_disconnect 断开后重新连接\n"
	} else if status.BufferUsed > 0 {
		output += fmt.Sprintf("  📖 读取输出: ssh_read_output(session_id=\"%s\", strategy=\"latest_lines\", limit=20)\n", sessionID)
	}
	if status.LastWriteTime.IsZero() || time.Since(status.LastWriteTime) > 5*time.Minute {
		output += fmt.Sprintf("  ⌨️ 发送命令: ssh_write_input(session_id=\"%s\", input=\"your_command\")\n", sessionID)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: output}},
	}, nil, nil
}

// handleSSHHistory handles the ssh_history tool
func (s *Server) handleSSHHistory(ctx context.Context, req *mcp.CallToolRequest, args map[string]any) (*mcp.CallToolResult, any, error) {
	sessionID, _ := args["session_id"].(string)
	limitVal, _ := args["limit"].(float64)
	sourceFilter, _ := args["source"].(string) // "exec", "shell", 或 "" (全部)

	session, err := s.sessionManager.GetSessionByIDOrAlias(sessionID)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Session not found: %v\nHint: Use ssh_list_sessions() to see all active sessions", err)}},
			IsError: true,
		}, nil, nil
	}

	session.RLock()
	history := session.CommandHistory
	session.RUnlock()

	// 根据 source 过滤
	var filteredHistory []sshmcp.CommandHistoryEntry
	if sourceFilter != "" {
		for _, entry := range history {
			if entry.Source == sourceFilter {
				filteredHistory = append(filteredHistory, entry)
			}
		}
	} else {
		filteredHistory = history
	}

	limit := int(limitVal)
	if limit <= 0 {
		limit = len(filteredHistory)
	}

	// 获取最近的 N 条记录（从后往前）
	start := len(filteredHistory) - limit
	if start < 0 {
		start = 0
	}
	recentHistory := filteredHistory[start:]

	if len(recentHistory) == 0 {
		sourceMsg := ""
		if sourceFilter != "" {
			sourceMsg = fmt.Sprintf(" (source: %s)", sourceFilter)
		}
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("No command history yet%s. Execute some commands first using ssh_exec or ssh_exec_batch.\n", sourceMsg)}},
		}, nil, nil
	}

	// 格式化输出
	sourceInfo := ""
	if sourceFilter != "" {
		sourceInfo = fmt.Sprintf(" [source: %s]", sourceFilter)
	}
	output := fmt.Sprintf("Command History%s (showing %d of %d total):\n\n", sourceInfo, len(recentHistory), len(filteredHistory))
	for i, entry := range recentHistory {
		status := "✓"
		if !entry.Success {
			status = "✗"
		}
		sourceLabel := entry.Source
		if sourceLabel == "" {
			sourceLabel = "unknown"
		}
		output += fmt.Sprintf("%d. [%s] %s [source: %s]\n", i+1, status, entry.Command, sourceLabel)
		output += fmt.Sprintf("   Exit Code: %d\n", entry.ExitCode)
		output += fmt.Sprintf("   Time: %s\n", entry.Timestamp.Format("2006-01-02 15:04:05"))
		output += fmt.Sprintf("   Duration: %s\n\n", entry.ExecutionTime)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: output}},
	}, nil, nil
}

// Helper functions for enhanced status display

// getStatusEmoji returns a status indicator with emoji
func getStatusEmoji(isActive bool) string {
	if isActive {
		return "✅ 活动"
	}
	return "❌ 未活动"
}

// formatTimeAgo formats a time as "X time ago" or "never"
func formatTimeAgo(t time.Time) string {
	if t.IsZero() {
		return "从未"
	}
	return formatDuration(time.Since(t)) + " 前"
}

// formatDuration formats a duration in human-readable format
func formatDuration(d time.Duration) string {
	// Handle negative durations
	if d < 0 {
		d = -d
	}

	// Break down into components
	seconds := int(d.Seconds())
	minutes := seconds / 60
	seconds = seconds % 60
	hours := minutes / 60
	minutes = minutes % 60
	days := hours / 24
	hours = hours % 24

	// Build human-readable string
	var parts []string
	if days > 0 {
		parts = append(parts, fmt.Sprintf("%d天", days))
	}
	if hours > 0 {
		parts = append(parts, fmt.Sprintf("%d小时", hours))
	}
	if minutes > 0 {
		parts = append(parts, fmt.Sprintf("%d分钟", minutes))
	}
	if seconds > 0 || len(parts) == 0 {
		parts = append(parts, fmt.Sprintf("%d秒", seconds))
	}

	// Join components (max 2 for brevity)
	if len(parts) > 2 {
		parts = parts[:2]
	}

	result := ""
	for i, part := range parts {
		if i > 0 {
			result += " "
		}
		result += part
	}

	return result
}
