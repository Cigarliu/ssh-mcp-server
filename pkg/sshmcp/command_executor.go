package sshmcp

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

// prepareCommandWithSudo 检测命令中的 sudo 并自动注入密码
func (s *Session) prepareCommandWithSudo(command string) string {
	// 检查命令是否包含 sudo
	trimmedCmd := strings.TrimSpace(command)

	// 检测是否是 sudo 命令
	if !strings.HasPrefix(trimmedCmd, "sudo ") && !strings.HasPrefix(trimmedCmd, "sudo\t") {
		return command
	}

	// 如果没有配置 sudo 密码，直接返回原命令
	if s.AuthConfig == nil || s.AuthConfig.SudoPassword == "" {
		return command
	}

	// 使用 echo + sudo -S 方式注入密码
	// 这会将密码通过 stdin 传递给 sudo
	return fmt.Sprintf("echo '%s' | sudo -S %s", s.AuthConfig.SudoPassword, strings.TrimPrefix(trimmedCmd, "sudo"))
}

// addToHistory adds a command execution entry to the session's history
func (s *Session) addToHistory(command string, exitCode int, executionTime time.Duration, source string) {
	if s.MaxHistorySize <= 0 {
		s.MaxHistorySize = 100 // 默认保存 100 条历史
	}

	// 创建历史条目
	entry := CommandHistoryEntry{
		Command:       command,
		ExitCode:      exitCode,
		ExecutionTime: executionTime,
		Timestamp:     time.Now(),
		Success:       exitCode == 0,
		Source:        source, // "exec" 或 "shell"
	}

	// 添加到历史记录
	s.CommandHistory = append(s.CommandHistory, entry)

	// 如果超过最大历史记录数，删除最旧的记录
	if len(s.CommandHistory) > s.MaxHistorySize {
		s.CommandHistory = s.CommandHistory[1:]
	}
}

// recordCommandResult records command execution result to history
func (s *Session) recordCommandResult(command string, result *CommandResult) {
	executionTime, _ := time.ParseDuration(result.ExecutionTime)
	s.addToHistory(command, result.ExitCode, executionTime, "exec")
}


// ExecuteCommand executes a single command on the remote host
func (s *Session) ExecuteCommand(command string, timeout time.Duration) (*CommandResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 更新最后使用时间
	s.LastUsedAt = time.Now()

	startTime := time.Now()

	// 处理 sudo 密码注入
	finalCommand := s.prepareCommandWithSudo(command)

	// 创建新的 SSH session
	session, err := s.SSHClient.NewSession()
	if err != nil {
		return nil, fmt.Errorf("create SSH session: %w", err)
	}
	defer session.Close()

	// 设置输出缓冲区（必须在执行命令之前）
	var stdoutBuf, stderrBuf bytes.Buffer
	session.Stdout = &stdoutBuf
	session.Stderr = &stderrBuf

	// 设置超时
	if timeout > 0 {
		done := make(chan error, 1)
		go func() {
			done <- session.Run(finalCommand)
		}()

		select {
		case <-time.After(timeout):
			// 超时，关闭 session
			session.Signal(ssh.SIGTERM)
			result := &CommandResult{
				ExitCode:     -1,
				Stdout:       stdoutBuf.String(),
				Stderr:       stderrBuf.String(),
				ExecutionTime: timeout.String(),
				Error:        fmt.Errorf("command timeout"),
			}
			s.addToHistory(command, result.ExitCode, timeout, "exec")
			return result, nil
		case err := <-done:
			executionTime := time.Since(startTime)
			if err != nil {
				exitErr, ok := err.(*ssh.ExitError)
				if ok {
					result := &CommandResult{
						ExitCode:     exitErr.ExitStatus(),
						Stdout:       stdoutBuf.String(),
						Stderr:       stderrBuf.String(),
						ExecutionTime: executionTime.String(),
						Error:        err,
					}
					s.addToHistory(command, result.ExitCode, executionTime, "exec")
					return result, nil
				}
				return nil, err
			}
		}
	} else {
		// 无超时限制
		if err := session.Run(finalCommand); err != nil {
			executionTime := time.Since(startTime)
			exitErr, ok := err.(*ssh.ExitError)
			if ok {
				result := &CommandResult{
					ExitCode:     exitErr.ExitStatus(),
					Stdout:       stdoutBuf.String(),
					Stderr:       stderrBuf.String(),
					ExecutionTime: executionTime.String(),
					Error:        err,
				}
				s.addToHistory(command, result.ExitCode, executionTime, "exec")
				return result, nil
			}
			return nil, err
		}
	}

	result := &CommandResult{
		ExitCode:     0,
		Stdout:       stdoutBuf.String(),
		Stderr:       stderrBuf.String(),
		ExecutionTime: time.Since(startTime).String(),
		Error:        nil,
	}

	// 记录到历史
	s.addToHistory(command, 0, time.Since(startTime), "exec")

	return result, nil
}

// ExecuteCommandOutput executes a command and returns combined output
func (s *Session) ExecuteCommandOutput(command string, timeout time.Duration) (string, error) {
	result, err := s.ExecuteCommand(command, timeout)
	if err != nil {
		return "", err
	}

	// 合并 stdout 和 stderr
	output := result.Stdout
	if result.Stderr != "" {
		if output != "" {
			output += "\n"
		}
		output += result.Stderr
	}

	return output, nil
}

// ExecuteCommandWithWorkingDir executes a command in a specific working directory
func (s *Session) ExecuteCommandWithWorkingDir(command, workingDir string, timeout time.Duration) (*CommandResult, error) {
	// 构造带工作目录的命令
	fullCommand := fmt.Sprintf("cd %s && %s", workingDir, command)
	return s.ExecuteCommand(fullCommand, timeout)
}

// ExecuteBatchCommands executes multiple commands in sequence
func (s *Session) ExecuteBatchCommands(commands []string, stopOnError bool, timeout time.Duration) ([]*CommandResult, *BatchResultSummary, error) {
	results := make([]*CommandResult, len(commands))
	summary := &BatchResultSummary{
		Total:   len(commands),
		Success: 0,
		Failed:  0,
	}

	for i, cmd := range commands {
		result, err := s.ExecuteCommand(cmd, timeout)
		if err != nil {
			// 执行出错
			results[i] = &CommandResult{
				ExitCode:     -1,
				Stdout:       "",
				Stderr:       err.Error(),
				ExecutionTime: "0s",
				Error:        err,
			}
			summary.Failed++

			if stopOnError {
				return results, summary, fmt.Errorf("command %d failed: %w", i+1, err)
			}
		} else {
			// 执行成功
			results[i] = result
			if result.ExitCode == 0 {
				summary.Success++
			} else {
				summary.Failed++
				if stopOnError {
					return results, summary, fmt.Errorf("command %d failed with exit code %d", i+1, result.ExitCode)
				}
			}
		}
	}

	return results, summary, nil
}

// BatchResultSummary represents the summary of batch command execution
type BatchResultSummary struct {
	Total   int `json:"total"`
	Success int `json:"success"`
	Failed  int `json:"failed"`
}

// ExecuteScript executes a multi-line script
func (s *Session) ExecuteScript(script string, timeout time.Duration) (*CommandResult, error) {
	// 将脚本拆分成命令
	lines := strings.Split(script, "\n")
	var commands []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		// 跳过空行和注释
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		commands = append(commands, line)
	}

	if len(commands) == 0 {
		return &CommandResult{
			ExitCode:     0,
			Stdout:       "",
			Stderr:       "No commands to execute",
			ExecutionTime: "0s",
		}, nil
	}

	// 批量执行命令
	results, _, err := s.ExecuteBatchCommands(commands, true, timeout)
	if err != nil {
		return results[0], err
	}

	// 合并所有输出
	var stdout, stderr strings.Builder
	for _, result := range results {
		if result.Stdout != "" {
			stdout.WriteString(result.Stdout)
			stdout.WriteString("\n")
		}
		if result.Stderr != "" {
			stderr.WriteString(result.Stderr)
			stderr.WriteString("\n")
		}
	}

	return &CommandResult{
		ExitCode:     0,
		Stdout:       stdout.String(),
		Stderr:       stderr.String(),
		ExecutionTime: "0s",
	}, nil
}
