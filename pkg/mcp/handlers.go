package mcp

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/cigar/sshmcp/pkg/sshmcp"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	defaultMaxOutputChars = 12000
	maxTerminalRows       = 100
	maxTerminalCols       = 240
	maxDirectoryEntries   = 200
)

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

// handleSSHExec executes a single, independent command on an active connection.
func (s *Server) handleSSHExec(_ context.Context, _ *mcp.CallToolRequest, args map[string]any) (*mcp.CallToolResult, any, error) {
	connectionID := stringArg(args, "connection_id")
	command := stringArg(args, "command")
	timeoutVal, _ := args["timeout"].(float64)
	workingDir := stringArg(args, "working_dir")
	maxOutputCharsVal, _ := args["max_output_chars"].(float64)

	session, err := s.sessionManager.GetSessionByIDOrAlias(connectionID)
	if err != nil {
		return &mcp.CallToolResult{
			Content: textContent(fmt.Sprintf("Connection not found: %v\nHint: Use connection_list to see active connections", err)),
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
		s.recordSessionHistory(session, "exec", command, err.Error(), "error", nil)
		return &mcp.CallToolResult{Content: textContent(fmt.Sprintf("Command execution failed: %v", err)), IsError: true}, nil, nil
	}

	exitCode := result.ExitCode
	state := "success"
	if exitCode != 0 {
		state = "failed"
	}
	s.recordSessionHistory(session, "exec", command, result.Stdout+result.Stderr, state, &exitCode)

	stdout, stderr, truncated := truncateCommandOutput(result.Stdout, result.Stderr, boundedOutputLimit(maxOutputCharsVal))
	output := fmt.Sprintf("Exit Code: %d\n\n", exitCode)
	if stdout != "" {
		output += fmt.Sprintf("STDOUT:\n%s\n\n", stdout)
	}
	if stderr != "" {
		output += fmt.Sprintf("STDERR:\n%s\n\n", stderr)
	}
	output += fmt.Sprintf("Execution Time: %s", result.ExecutionTime)
	if truncated {
		output += fmt.Sprintf("\nOutput truncated to %d characters. Narrow the remote command to inspect more.", boundedOutputLimit(maxOutputCharsVal))
	}

	return &mcp.CallToolResult{Content: textContent(output)}, map[string]any{
		"exit_code":      exitCode,
		"stdout":         stdout,
		"stderr":         stderr,
		"execution_time": result.ExecutionTime,
		"truncated":      truncated,
	}, nil
}

func truncateCommandOutput(stdout, stderr string, limit int) (string, string, bool) {
	stdout, stdoutTruncated := truncateText(stdout, limit)
	remaining := limit - len([]rune(stdout))
	if remaining < 0 {
		remaining = 0
	}
	stderr, stderrTruncated := truncateText(stderr, remaining)
	return stdout, stderr, stdoutTruncated || stderrTruncated
}

func truncateText(text string, limit int) (string, bool) {
	runes := []rune(text)
	if len(runes) <= limit {
		return text, false
	}
	return string(runes[:limit]), true
}

func boundedOutputLimit(value float64) int {
	if value >= 1 && value <= defaultMaxOutputChars && value == float64(int(value)) {
		return int(value)
	}
	return defaultMaxOutputChars
}

func directoryEntryLimit(value float64) int {
	if value < 1 {
		return 100
	}
	if value > maxDirectoryEntries {
		return maxDirectoryEntries
	}
	return int(value)
}

func terminalDimension(name string, value float64, defaultValue, maximum uint16) (uint16, error) {
	if value == 0 && defaultValue > 0 {
		return defaultValue, nil
	}
	if value < 1 || value > float64(maximum) || value != float64(int(value)) {
		return 0, fmt.Errorf("%s must be an integer from 1 to %d", name, maximum)
	}
	return uint16(value), nil
}

func (s *Server) activeSSHSession(connectionID string) (*sshmcp.Session, *mcp.CallToolResult) {
	session, err := s.sessionManager.GetSessionByIDOrAlias(connectionID)
	if err != nil {
		return nil, &mcp.CallToolResult{
			Content: textContent(fmt.Sprintf("Connection not found: %v\nHint: Use connection_list to see active connections", err)),
			IsError: true,
		}
	}
	return session, nil
}

// handleSFTPTransfer performs one upload or download through a unified connection_id.
func (s *Server) handleSFTPTransfer(_ context.Context, _ *mcp.CallToolRequest, args map[string]any) (*mcp.CallToolResult, any, error) {
	operation := stringArg(args, "operation")
	if operation != "upload" && operation != "download" {
		return &mcp.CallToolResult{Content: textContent("operation must be upload or download"), IsError: true}, nil, nil
	}
	session, failure := s.activeSSHSession(stringArg(args, "connection_id"))
	if failure != nil {
		return failure, nil, nil
	}

	createDirs := boolArgDefault(args, "create_dirs", true)
	overwrite := boolArgDefault(args, "overwrite", false)
	localPath := stringArg(args, "local_path")
	remotePath := stringArg(args, "remote_path")

	switch operation {
	case "upload":
		result, err := session.UploadFile(localPath, remotePath, createDirs, overwrite)
		if err != nil {
			return &mcp.CallToolResult{Content: textContent(fmt.Sprintf("Upload failed: %v", err)), IsError: true}, nil, nil
		}
		return sftpTransferResult("Upload", localPath, remotePath, result), nil, nil
	case "download":
		result, err := session.DownloadFile(remotePath, localPath, createDirs, overwrite)
		if err != nil {
			return &mcp.CallToolResult{Content: textContent(fmt.Sprintf("Download failed: %v", err)), IsError: true}, nil, nil
		}
		return sftpTransferResult("Download", localPath, remotePath, result), nil, nil
	}

	return &mcp.CallToolResult{Content: textContent("operation must be upload or download"), IsError: true}, nil, nil
}

func sftpTransferResult(operation, localPath, remotePath string, result *sshmcp.FileTransferResult) *mcp.CallToolResult {
	output := fmt.Sprintf("%s successful:\n", operation)
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
	return &mcp.CallToolResult{Content: textContent(output)}
}

// handleSFTPManage performs one task-oriented remote path operation.
func (s *Server) handleSFTPManage(_ context.Context, _ *mcp.CallToolRequest, args map[string]any) (*mcp.CallToolResult, any, error) {
	operation := stringArg(args, "operation")
	if operation != "list" && operation != "mkdir" && operation != "delete" {
		return &mcp.CallToolResult{Content: textContent("operation must be list, mkdir, or delete"), IsError: true}, nil, nil
	}
	session, failure := s.activeSSHSession(stringArg(args, "connection_id"))
	if failure != nil {
		return failure, nil, nil
	}

	remotePath := stringArg(args, "remote_path")
	switch operation {
	case "list":
		if remotePath == "" {
			remotePath = "/"
		}
		files, err := session.ListDirectory(remotePath, boolArgDefault(args, "recursive", false))
		if err != nil {
			return &mcp.CallToolResult{Content: textContent(fmt.Sprintf("List directory failed: %v", err)), IsError: true}, nil, nil
		}
		limitValue, _ := args["limit"].(float64)
		limit := directoryEntryLimit(limitValue)
		truncated := len(files) > limit
		if truncated {
			files = files[:limit]
		}
		output := fmt.Sprintf("Directory listing for: %s\nShowing %d entries\n\n", remotePath, len(files))
		for _, file := range files {
			output += fmt.Sprintf("- %s (%s, %d bytes)\n", file.Name, file.Type, file.Size)
		}
		if truncated {
			output += fmt.Sprintf("\nResult truncated to %d entries. List a narrower path to inspect more.", limit)
		}
		output, outputTruncated := truncateText(output, defaultMaxOutputChars)
		if outputTruncated {
			output += fmt.Sprintf("\nOutput truncated to %d characters. List a narrower path to inspect more.", defaultMaxOutputChars)
		}
		return &mcp.CallToolResult{Content: textContent(output)}, nil, nil
	case "mkdir":
		if remotePath == "" {
			return &mcp.CallToolResult{Content: textContent("remote_path is required for mkdir"), IsError: true}, nil, nil
		}
		mode, err := directoryMode(stringArgDefault(args, "mode", "0755"))
		if err != nil {
			return &mcp.CallToolResult{Content: textContent(err.Error()), IsError: true}, nil, nil
		}
		if err := session.MakeDirectory(remotePath, boolArgDefault(args, "recursive", true), mode); err != nil {
			return &mcp.CallToolResult{Content: textContent(fmt.Sprintf("Mkdir failed: %v", err)), IsError: true}, nil, nil
		}
		return &mcp.CallToolResult{Content: textContent(fmt.Sprintf("Directory created: %s", remotePath))}, nil, nil
	case "delete":
		if remotePath == "" {
			return &mcp.CallToolResult{Content: textContent("remote_path is required for delete"), IsError: true}, nil, nil
		}
		if err := session.RemoveFile(remotePath, boolArgDefault(args, "recursive", false)); err != nil {
			return &mcp.CallToolResult{Content: textContent(fmt.Sprintf("Delete failed: %v", err)), IsError: true}, nil, nil
		}
		return &mcp.CallToolResult{Content: textContent(fmt.Sprintf("Deleted: %s", remotePath))}, nil, nil
	}

	return &mcp.CallToolResult{Content: textContent("operation must be list, mkdir, or delete"), IsError: true}, nil, nil
}

func directoryMode(value string) (os.FileMode, error) {
	mode, err := strconv.ParseUint(value, 8, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid directory mode %q: use an octal value such as 0755", value)
	}
	return os.FileMode(mode), nil
}

func boolArgDefault(args map[string]any, key string, defaultValue bool) bool {
	value, found := args[key].(bool)
	if !found {
		return defaultValue
	}
	return value
}
