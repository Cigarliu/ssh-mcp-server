package sshmcp

import (
	"bytes"
	"fmt"
	"io"
	"time"

	"golang.org/x/crypto/ssh"
)

// CreateShell creates an interactive shell session
func (s *Session) CreateShell(term string, rows, cols uint16) (*SSHShellSession, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, err := s.SSHClient.NewSession()
	if err != nil {
		return nil, fmt.Errorf("create SSH session: %w", err)
	}

	// 设置默认终端类型
	if term == "" {
		term = "xterm-256color"
	}

	// 请求 PTY
	if err := session.RequestPty(term, int(rows), int(cols), ssh.TerminalModes{
		ssh.ECHO:          1,     // 启用回显
		ssh.TTY_OP_ISPEED: 14400, // 输入速度 = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // 输出速度 = 14.4kbaud
	}); err != nil {
		session.Close()
		return nil, fmt.Errorf("request PTY: %w", err)
	}

	// 创建 stdin/stdout/stderr pipes
	stdin, err := session.StdinPipe()
	if err != nil {
		session.Close()
		return nil, fmt.Errorf("create stdin pipe: %w", err)
	}

	stdout, err := session.StdoutPipe()
	if err != nil {
		session.Close()
		return nil, fmt.Errorf("create stdout pipe: %w", err)
	}

	stderr, err := session.StderrPipe()
	if err != nil {
		session.Close()
		return nil, fmt.Errorf("create stderr pipe: %w", err)
	}

	// 启动 shell
	if err := session.Shell(); err != nil {
		session.Close()
		return nil, fmt.Errorf("start shell: %w", err)
	}

	shellSession := &SSHShellSession{
		Session: session,
		Stdin:   stdin,
		Stdout:  stdout,
		Stderr:  stderr,
		PTY:     true,
		TerminalInfo: TerminalInfo{
			Term: term,
			Rows: rows,
			Cols: cols,
		},
	}

	s.ShellSession = shellSession
	s.State = SessionStateActive

	return shellSession, nil
}

// WriteInput writes input to the shell
func (ss *SSHShellSession) WriteInput(input string) error {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	if ss.Stdin == nil {
		return fmt.Errorf("stdin is not available")
	}

	_, err := ss.Stdin.Write([]byte(input))
	return err
}

// ReadOutput reads output from the shell with timeout
func (ss *SSHShellSession) ReadOutput(timeout time.Duration) (string, string, error) {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	if ss.Stdout == nil || ss.Stderr == nil {
		return "", "", fmt.Errorf("stdout/stderr is not available")
	}

	// 创建缓冲区
	var stdoutBuf, stderrBuf bytes.Buffer
	done := make(chan error, 2)

	// 读取 stdout
	go func() {
		_, err := io.Copy(&stdoutBuf, ss.Stdout)
		done <- err
	}()

	// 读取 stderr
	go func() {
		_, err := io.Copy(&stderrBuf, ss.Stderr)
		done <- err
	}()

	// 等待结果或超时
	var stdoutErr, stderrErr error
	timeoutChan := time.After(timeout)

	for i := 0; i < 2; i++ {
		select {
		case err := <-done:
			if i == 0 {
				stdoutErr = err
			} else {
				stderrErr = err
			}
		case <-timeoutChan:
			return stdoutBuf.String(), stderrBuf.String(), nil
		}
	}

	return stdoutBuf.String(), stderrBuf.String(), fmt.Errorf("stdout: %v, stderr: %v", stdoutErr, stderrErr)
}

// Resize changes the terminal window size
func (ss *SSHShellSession) Resize(rows, cols uint16) error {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	if ss.Session == nil {
		return fmt.Errorf("session is not available")
	}

	err := ss.Session.WindowChange(int(rows), int(cols))
	if err == nil {
		ss.TerminalInfo.Rows = rows
		ss.TerminalInfo.Cols = cols
	}

	return err
}

// Close closes the shell session
func (ss *SSHShellSession) Close() error {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	var errs []error

	// 关闭 stdin
	if ss.Stdin != nil {
		if err := ss.Stdin.Close(); err != nil {
			errs = append(errs, fmt.Errorf("close stdin: %w", err))
		}
	}

	// 关闭 session
	if ss.Session != nil {
		if err := ss.Session.Close(); err != nil {
			errs = append(errs, fmt.Errorf("close session: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("close shell session: %v", errs)
	}

	return nil
}

// IsAlive checks if the shell session is still alive
func (ss *SSHShellSession) IsAlive() bool {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	if ss.Session == nil {
		return false
	}

	// 发送 keepalive 信号
	_, err := ss.Session.SendRequest("keepalive", true, nil)
	return err == nil
}
