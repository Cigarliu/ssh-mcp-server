package sshmcp

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/acarl005/stripansi"
	"golang.org/x/crypto/ssh"
)

// Common interactive programs that require raw mode
var interactivePrograms = []string{
	"vim", "vi", "nano", "emacs",
	"gdb", "lldb",
	"top", "htop", "iotop",
	"python", "python3", "node", "irb",
	"mysql", "psql", "mongosh",
	"tmux", "screen",
	"less", "more", "most",
}

// IsInteractiveProgram detects if a command is an interactive program
func IsInteractiveProgram(cmd string) bool {
	cmdLower := strings.ToLower(cmd)
	for _, prog := range interactivePrograms {
		if strings.Contains(cmdLower, prog) {
			return true
		}
	}
	return false
}

// CreateShell creates an interactive shell session with default config
func (s *Session) CreateShell(term string, rows, cols uint16) (*SSHShellSession, error) {
	return s.CreateShellWithConfig(term, rows, cols, DefaultShellConfig())
}

// CreateShellWithConfig creates an interactive shell session with custom configuration
func (s *Session) CreateShellWithConfig(term string, rows, cols uint16, config *ShellConfig) (*SSHShellSession, error) {
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

	// 根据配置设置终端模式
	var termModes ssh.TerminalModes
	if config.Mode == TerminalModeRaw {
		// Raw mode: minimal processing
		termModes = ssh.TerminalModes{
			ssh.ECHO:          0, // 禁用回显
			ssh.TTY_OP_ISPEED: 14400,
			ssh.TTY_OP_OSPEED: 14400,
			ssh.VINTR:         0, // 禁用中断字符
			ssh.VQUIT:         0, // 禁用退出字符
			ssh.VERASE:        0, // 禁用擦除字符
			ssh.VKILL:         0, // 禁用杀死字符
			ssh.VEOF:           0, // 禁用 EOF 字符
		}
	} else {
		// Cooked mode: normal processing
		termModes = ssh.TerminalModes{
			ssh.ECHO:          1, // 启用回显
			ssh.TTY_OP_ISPEED: 14400,
			ssh.TTY_OP_OSPEED: 14400,
		}
	}

	// 请求 PTY
	if err := session.RequestPty(term, int(rows), int(cols), termModes); err != nil {
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
		Config:  config,
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
	if err == nil {
		ss.LastWriteTime = time.Now()
	}
	return err
}

// extractCurrentDir 从输出中提取当前目录
// 支持常见的提示符格式：
// - user@host:path$  (Ubuntu/Debian)
// - [user@host path]# (RHEL/CentOS)
// - path$          (简单格式)
func extractCurrentDir(output string) string {
	lines := strings.Split(output, "\n")
	if len(lines) == 0 {
		return ""
	}

	// 检查最后一行（通常是提示符）
	lastLine := lines[len(lines)-1]

	// 尝试匹配 user@host:path 格式（Ubuntu/Debian）
	if matches := regexp.MustCompile(`[\w-]+@[\w-]+:([~$\/\w\-\.\{\}]+)[\$%#]`).FindStringSubmatch(lastLine); len(matches) > 1 {
		dir := matches[1]
		// 展开波浪号
		if dir == "~" {
			return "/home/" + os.Getenv("USER")
		}
		return dir
	}

	// 尝试匹配 [user@host path] 格式（RHEL/CentOS）
	if matches := regexp.MustCompile(`\[[\w-]+@[\w-]+ ([~$\/\w\-\.\{\}]+)\][\$%#]`).FindStringSubmatch(lastLine); len(matches) > 1 {
		dir := matches[1]
		if dir == "~" {
			return "/home/" + os.Getenv("USER")
		}
		return dir
	}

	// 尝试匹配简单的 path$ 格式
	if matches := regexp.MustCompile(`^([~$\/\w\-\.\{\}]+)[\$%#]$`).FindStringSubmatch(strings.TrimSpace(lastLine)); len(matches) > 1 {
		dir := matches[1]
		if dir == "~" {
			return "/home/" + os.Getenv("USER")
		}
		return dir
	}

	return ""
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
			stdoutStr := stdoutBuf.String()
			stderrStr := stderrBuf.String()
			// 更新状态
			ss.LastReadTime = time.Now()
			ss.hasUnreadData = stdoutBuf.Len() > 0 || stderrBuf.Len() > 0
			return stdoutStr, stderrStr, nil
		}
	}

	stdoutStr := stdoutBuf.String()
	stderrStr := stderrBuf.String()

	// 更新状态
	ss.LastReadTime = time.Now()
	ss.hasUnreadData = false

	return stdoutStr, stderrStr, fmt.Errorf("stdout: %v, stderr: %v", stdoutErr, stderrErr)
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
	session := ss.Session
	ss.mu.Unlock()

	if session == nil {
		return false
	}

	// 使用 channel 和 goroutine 实现 keepalive 超时
	type result struct {
		alive bool
	}
	resultChan := make(chan result, 1)

	go func() {
		_, err := session.SendRequest("keepalive", true, nil)
		resultChan <- result{alive: err == nil}
	}()

	// 等待结果，最多 1 秒
	select {
	case res := <-resultChan:
		return res.alive
	case <-time.After(1 * time.Second):
		// 超时，认为 session 不活跃
		return false
	}
}

// ReadOutputNonBlocking reads output with non-blocking I/O
// This is the NEW method that solves the EOF blocking issue
func (ss *SSHShellSession) ReadOutputNonBlocking(timeout time.Duration) (string, string, error) {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	if ss.Stdout == nil || ss.Stderr == nil {
		return "", "", fmt.Errorf("stdout/stderr is not available")
	}

	// 使用实际配置的 timeout 或传入的 timeout
	readTimeout := timeout
	if readTimeout <= 0 && ss.Config != nil {
		readTimeout = ss.Config.ReadTimeout
	}
	if readTimeout <= 0 {
		readTimeout = 100 * time.Millisecond
	}

	// 创建缓冲区
	var stdoutBuf, stderrBuf bytes.Buffer
	var wg sync.WaitGroup
	var stdoutErr, stderrErr error

	// 读取 stdout with timeout
	wg.Add(1)
	go func() {
		defer wg.Done()
		buf := make([]byte, 4096)

		// 尝试设置读取超时（如果 stdout 支持）
		if stdoutFile, ok := ss.Stdout.(interface{ SetReadDeadline(time.Time) error }); ok {
			stdoutFile.SetReadDeadline(time.Now().Add(readTimeout))
		}

		n, err := ss.Stdout.Read(buf)
		if err != nil && err != io.EOF {
			if os.IsTimeout(err) || err.Error() == "deadline exceeded" {
				// 超时不是错误，返回已读取的部分
				stdoutBuf.Write(buf[:n])
				return
			}
			stdoutErr = err
			return
		}
		stdoutBuf.Write(buf[:n])
	}()

	// 读取 stderr with timeout
	wg.Add(1)
	go func() {
		defer wg.Done()
		buf := make([]byte, 4096)

		// 尝试设置读取超时
		if stderrFile, ok := ss.Stderr.(interface{ SetReadDeadline(time.Time) error }); ok {
			stderrFile.SetReadDeadline(time.Now().Add(readTimeout))
		}

		n, err := ss.Stderr.Read(buf)
		if err != nil && err != io.EOF {
			if os.IsTimeout(err) || err.Error() == "deadline exceeded" {
				// 超时不是错误，返回已读取的部分
				stderrBuf.Write(buf[:n])
				return
			}
			stderrErr = err
			return
		}
		stderrBuf.Write(buf[:n])
	}()

	// 等待读取完成或超时
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// 读取完成
	case <-time.After(readTimeout + 10*time.Millisecond):
		// 超时，返回已读取的部分
	}

	stdoutStr := stdoutBuf.String()
	stderrStr := stderrBuf.String()

	// 根据 ANSIMode 处理输出
	if ss.Config != nil {
		switch ss.Config.ANSIMode {
		case ANSIStrip:
			stdoutStr = stripANSI(stdoutStr)
			stderrStr = stripANSI(stderrStr)
		}
	}

	// 更新状态
	ss.LastReadTime = time.Now()
	ss.hasUnreadData = false

	// 尝试从输出中提取当前目录
	if stdoutStr != "" {
		if dir := extractCurrentDir(stdoutStr); dir != "" {
			ss.currentDir = dir
		}
	}

	if stdoutErr != nil || stderrErr != nil {
		return stdoutStr, stderrStr, fmt.Errorf("stdout: %v, stderr: %v", stdoutErr, stderrErr)
	}

	return stdoutStr, stderrStr, nil
}

// stripANSI removes ANSI escape sequences from string using the stripansi library
func stripANSI(s string) string {
	// 使用 stripansi 库移除 ANSI 转义序列
	result := stripansi.Strip(s)

	// 移除 carriage return (通常在行尾)
	result = strings.ReplaceAll(result, "\r", "")

	// 移除零宽字符和其他不可见控制字符（除了换行和制表符）
	cleaned := strings.Builder{}
	for _, r := range result {
		// 保留可打印字符、换行、制表符
		if r == '\n' || r == '\t' || r >= 32 {
			cleaned.WriteRune(r)
		}
	}

	return cleaned.String()
}

// WriteSpecialChars writes special control characters to the shell
func (ss *SSHShellSession) WriteSpecialChars(char string) error {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	if ss.Stdin == nil {
		return fmt.Errorf("stdin is not available")
	}

	var input []byte
	switch strings.ToLower(char) {
	case "ctrl+c", "sigint":
		input = []byte{0x03} // ASCII ETX (End of Text)
	case "ctrl+d", "eof":
		input = []byte{0x04} // ASCII EOT (End of Transmission)
	case "ctrl+z", "sigtstp":
		input = []byte{0x1A} // ASCII SUB (Substitute)
	case "ctrl+l", "clear":
		input = []byte{0x0C} // ASCII FF (Form Feed)
	case "enter", "return":
		input = []byte{0x0D} // ASCII CR (Carriage Return)
	case "tab":
		input = []byte{0x09} // ASCII HT (Horizontal Tab)
	case "esc":
		input = []byte{0x1B} // ASCII ESC (Escape)
	case "up":
		input = []byte{0x1B, 0x5B, 0x41} // ESC [ A
	case "down":
		input = []byte{0x1B, 0x5B, 0x42} // ESC [ B
	case "right":
		input = []byte{0x1B, 0x5B, 0x43} // ESC [ C
	case "left":
		input = []byte{0x1B, 0x5B, 0x44} // ESC [ D
	default:
		return fmt.Errorf("unsupported special character: %s", char)
	}

	_, err := ss.Stdin.Write(input)
	return err
}

// SetMode dynamically changes the terminal mode
func (ss *SSHShellSession) SetMode(mode TerminalMode) error {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	if ss.Config == nil {
		return fmt.Errorf("shell config is not set")
	}

	ss.Config.Mode = mode

	// Note: 在 SSH 中无法动态更改已建立的 PTY 模式
	// 这只是更新配置，实际的模式更改需要在创建 shell 时设置
	// 如果需要动态更改，需要重新创建 shell session
	return fmt.Errorf("cannot change mode dynamically for SSH PTY, please recreate shell with new mode")
}

// GetConfig returns the current shell configuration
func (ss *SSHShellSession) GetConfig() *ShellConfig {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	return ss.Config
}

// GetStatus returns the current status of the shell session
func (ss *SSHShellSession) GetStatus() *ShellStatus {
	ss.mu.Lock()

	// 复制需要的数据
	currentDir := ss.currentDir
	hasUnreadData := ss.hasUnreadData
	lastReadTime := ss.LastReadTime
	lastWriteTime := ss.LastWriteTime
	terminalType := ss.TerminalInfo.Term
	rows := ss.TerminalInfo.Rows
	cols := ss.TerminalInfo.Cols
	ansiMode := ss.Config.ANSIMode.String()
	mode := ss.Config.Mode

	ss.mu.Unlock()

	// 在锁外调用 IsAlive()，避免死锁
	status := &ShellStatus{
		IsActive:      ss.IsAlive(),
		CurrentDir:    currentDir,
		HasUnreadOutput: hasUnreadData,
		LastReadTime:  lastReadTime,
		LastWriteTime: lastWriteTime,
		TerminalType:  terminalType,
		Rows:          rows,
		Cols:          cols,
		ANSIMode:      ansiMode,
	}

	// Convert mode to string
	switch mode {
	case TerminalModeCooked:
		status.Mode = "cooked"
	case TerminalModeRaw:
		status.Mode = "raw"
	default:
		status.Mode = "unknown"
	}

	return status
}
