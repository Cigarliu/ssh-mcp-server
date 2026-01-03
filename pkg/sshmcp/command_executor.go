package sshmcp

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

// ExecuteCommand executes a single command on the remote host
func (s *Session) ExecuteCommand(command string, timeout time.Duration) (*CommandResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 更新最后使用时间
	s.LastUsedAt = time.Now()

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

	// 记录开始时间
	startTime := time.Now()

	// 设置超时
	if timeout > 0 {
		done := make(chan error, 1)
		go func() {
			done <- session.Run(command)
		}()

		select {
		case <-time.After(timeout):
			// 超时，关闭 session
			session.Signal(ssh.SIGTERM)
			return &CommandResult{
				ExitCode:     -1,
				Stdout:       stdoutBuf.String(),
				Stderr:       stderrBuf.String(),
				ExecutionTime: timeout.String(),
				Error:        fmt.Errorf("command timeout"),
			}, nil
		case err := <-done:
			if err != nil {
				exitErr, ok := err.(*ssh.ExitError)
				if ok {
					return &CommandResult{
						ExitCode:     exitErr.ExitStatus(),
						Stdout:       stdoutBuf.String(),
						Stderr:       stderrBuf.String(),
						ExecutionTime: time.Since(startTime).String(),
						Error:        err,
					}, nil
				}
				return nil, err
			}
		}
	} else {
		// 无超时限制
		if err := session.Run(command); err != nil {
			exitErr, ok := err.(*ssh.ExitError)
			if ok {
				return &CommandResult{
					ExitCode:     exitErr.ExitStatus(),
					Stdout:       stdoutBuf.String(),
					Stderr:       stderrBuf.String(),
					ExecutionTime: time.Since(startTime).String(),
					Error:        err,
				}, nil
			}
			return nil, err
		}
	}

	return &CommandResult{
		ExitCode:     0,
		Stdout:       stdoutBuf.String(),
		Stderr:       stderrBuf.String(),
		ExecutionTime: time.Since(startTime).String(),
		Error:        nil,
	}, nil
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
