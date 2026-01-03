package mcp

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/cigar/sshmcp/pkg/sshmcp"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

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
		Type: sshmcp.AuthType(authType),
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
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Session not found: %v", err)}},
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

	commands := make([]string, len(commandsInterface))
	for i, cmd := range commandsInterface {
		commands[i], _ = cmd.(string)
	}

	session, err := s.sessionManager.GetSessionByIDOrAlias(sessionID)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Session not found: %v", err)}},
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
	term, _ := args["terminal_type"].(string)
	rowsVal, _ := args["rows"].(float64)
	colsVal, _ := args["cols"].(float64)
	mode, _ := args["mode"].(string)
	ansiMode, _ := args["ansi_mode"].(string)
	readTimeoutVal, _ := args["read_timeout"].(float64)

	session, err := s.sessionManager.GetSessionByIDOrAlias(sessionID)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Session not found: %v", err)}},
			IsError: true,
		}, nil, nil
	}

	rows := uint16(rowsVal)
	cols := uint16(colsVal)
	if rows == 0 {
		rows = 24
	}
	if cols == 0 {
		cols = 80
	}

	// 创建 Shell 配置
	config := sshmcp.DefaultShellConfig()
	
	// 设置终端模式
	if mode == "raw" {
		config.Mode = sshmcp.TerminalModeRaw
	} else {
		config.Mode = sshmcp.TerminalModeCooked
	}
	
	// 设置 ANSI 处理模式
	switch ansiMode {
	case "strip":
		config.ANSIMode = sshmcp.ANSIStrip
	case "parse":
		config.ANSIMode = sshmcp.ANSIParse
	default:
		config.ANSIMode = sshmcp.ANSIRaw
	}
	
	// 设置读取超时
	if readTimeoutVal > 0 {
		config.ReadTimeout = time.Duration(readTimeoutVal) * time.Millisecond
	}
	
	// 使用配置创建 Shell
	_, err = session.CreateShellWithConfig(term, rows, cols, config)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Failed to create shell: %v", err)}},
			IsError: true,
		}, nil, nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Interactive shell started for session %s\nUse ssh_write_input to send commands and ssh_read_output to receive responses", sessionID)}},
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
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Session not found: %v", err)}},
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

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Upload successful:\nStatus: %s\nBytes transferred: %d\nDuration: %s",
			result.Status, result.BytesTransferred, result.Duration)}},
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
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Session not found: %v", err)}},
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

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Download successful:\nStatus: %s\nBytes transferred: %d\nDuration: %s",
			result.Status, result.BytesTransferred, result.Duration)}},
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
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Session not found: %v", err)}},
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
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Session not found: %v", err)}},
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
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Session not found: %v", err)}},
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
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Session not found: %v", err)}},
			IsError: true,
		}, nil, nil
	}

	session.RLock()
	hasShell := session.ShellSession != nil
	session.RUnlock()

	if !hasShell {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("No active shell session for session_id: %s", sessionID)}},
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

// handleSSHReadOutput handles the ssh_read_output tool
// handleSSHReadOutput handles the ssh_read_output tool
func (s *Server) handleSSHReadOutput(ctx context.Context, req *mcp.CallToolRequest, args map[string]any) (*mcp.CallToolResult, any, error) {
	sessionID, _ := args["session_id"].(string)
	timeoutVal, _ := args["timeout"].(float64)
	nonBlocking, _ := args["non_blocking"].(bool)

	session, err := s.sessionManager.GetSessionByIDOrAlias(sessionID)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Session not found: %v", err)}},
			IsError: true,
		}, nil, nil
	}

	session.RLock()
	hasShell := session.ShellSession != nil
	session.RUnlock()

	if !hasShell {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("No active shell session for session_id: %s", sessionID)}},
			IsError: true,
		}, nil, nil
	}

	var stdout, stderr string
	if nonBlocking {
		// Non-blocking mode: use milliseconds timeout
		readTimeout := 100 * time.Millisecond
		if timeoutVal > 0 {
			readTimeout = time.Duration(timeoutVal) * time.Millisecond
		}
		stdout, stderr, err = session.ShellSession.ReadOutputNonBlocking(readTimeout)
	} else {
		// Blocking mode: use seconds timeout
		timeout := 5 * time.Second
		if timeoutVal > 0 {
			timeout = time.Duration(timeoutVal) * time.Second
		}
		stdout, stderr, err = session.ShellSession.ReadOutput(timeout)
	}

	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Read output failed: %v", err)}},
			IsError: true,
		}, nil, nil
	}

	output := ""
	if stdout != "" {
		output += fmt.Sprintf("STDOUT:\n%s\n", stdout)
	}
	if stderr != "" {
		output += fmt.Sprintf("STDERR:\n%s\n", stderr)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: output}},
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
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Session not found: %v", err)}},
			IsError: true,
		}, nil, nil
	}

	session.RLock()
	hasShell := session.ShellSession != nil
	session.RUnlock()

	if !hasShell {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("No active shell session for session_id: %s", sessionID)}},
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
