package sshmcp

import (
	"bytes"
	"io"
	"os"
	"sync"
	"time"
)

// TerminalCapturer 捕获终端输出并维护屏幕状态
type TerminalCapturer struct {
	Emulator TerminalEmulator  // 使用抽象接口
	PTY      *os.File
	mu       sync.Mutex
	closed   bool
}

// NewTerminalCapturer 创建一个新的终端捕获器
// 使用默认的终端模拟器（从环境变量读取）
func NewTerminalCapturer(width, height int) (*TerminalCapturer, error) {
	emulator, err := NewTerminalEmulatorFromEnv(width, height)
	if err != nil {
		return nil, err
	}

	return &TerminalCapturer{
		Emulator: emulator,
		closed:   false,
	}, nil
}

// NewTerminalCapturerWithType 创建指定类型的终端捕获器
func NewTerminalCapturerWithType(width, height int, emulatorType TerminalEmulatorType) (*TerminalCapturer, error) {
	emulator, err := GetTerminalEmulator(emulatorType, width, height)
	if err != nil {
		return nil, err
	}

	return &TerminalCapturer{
		Emulator: emulator,
		closed:   false,
	}, nil
}

// StartFromPTY 从现有的 PTY 启动捕获
func (tc *TerminalCapturer) StartFromPTY(pty *os.File) {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	tc.PTY = pty
	go tc.readLoop()
}

// readLoop 持续读取 PTY 输出并更新终端模拟器
func (tc *TerminalCapturer) readLoop() {
	buf := make([]byte, 4096)

	for {
		tc.mu.Lock()
		if tc.closed {
			tc.mu.Unlock()
			break
		}
		pty := tc.PTY
		tc.mu.Unlock()

		if pty == nil {
			time.Sleep(100 * time.Millisecond)
			continue
		}

		// 设置读取超时
		pty.SetReadDeadline(time.Now().Add(1 * time.Second))

		n, err := pty.Read(buf)
		if n > 0 {
			// 让终端模拟器处理原始字节流（包括所有 ANSI 序列）
			tc.mu.Lock()
			tc.Emulator.Write(buf[:n])
			tc.mu.Unlock()
		}

		if err != nil {
			if err == io.EOF {
				// EOF 是正常情况，连接关闭
				break
			}
			if os.IsTimeout(err) {
				// 超时不是错误，继续读取
				continue
			}
			// 其他错误，退出
			break
		}
	}
}

// GetScreenSnapshot 获取当前屏幕内容的快照（纯文本）
func (tc *TerminalCapturer) GetScreenSnapshot() string {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	if tc.Emulator == nil {
		return ""
	}

	var buf bytes.Buffer
	content := tc.Emulator.GetScreenContent()

	// 遍历整个屏幕
	for y := 0; y < len(content); y++ {
		row := content[y]
		for x := 0; x < len(row); x++ {
			char := row[x]
			if char != 0 {
				buf.WriteRune(char)
			} else {
				buf.WriteRune(' ')
			}
		}
		if y < len(content)-1 {
			buf.WriteByte('\n')
		}
	}

	return buf.String()
}

// GetScreenSnapshotWithColor 获取包含颜色信息的快照（使用ANSI颜色码）
func (tc *TerminalCapturer) GetScreenSnapshotWithColor() string {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	if tc.Emulator == nil {
		return ""
	}

	var buf bytes.Buffer
	content, format := tc.Emulator.GetScreenContentWithFormat()

	// 遍历整个屏幕并保留颜色
	for y := 0; y < len(content); y++ {
		row := content[y]
		rowFormat := format[y]
		lastFgSeq := ""
		lastBgSeq := ""

		for x := 0; x < len(row); x++ {
			char := row[x]
			var cellFmt Format
			if x < len(rowFormat) {
				cellFmt = rowFormat[x]
			}

			// 应用前景色（使用类型断言检查是否为 termenv.Color）
			if cellFmt.Fg != nil {
				// 尝试类型断言为 termenv.Color（vt100 适配器使用）
				if color, ok := cellFmt.Fg.(interface{ Sequence(bool) string }); ok {
					fgSeq := color.Sequence(false)
					if fgSeq != lastFgSeq && fgSeq != "" {
						buf.WriteString("\x1b[" + fgSeq + "m")
						lastFgSeq = fgSeq
					}
				}
			}

			// 应用背景色
			if cellFmt.Bg != nil {
				if color, ok := cellFmt.Bg.(interface{ Sequence(bool) string }); ok {
					bgSeq := color.Sequence(true)
					if bgSeq != lastBgSeq && bgSeq != "" {
						buf.WriteString("\x1b[" + bgSeq + "m")
						lastBgSeq = bgSeq
					}
				}
			}

			if char != 0 {
				buf.WriteRune(char)
			} else {
				buf.WriteRune(' ')
			}
		}

		// 重置颜色
		buf.WriteString("\x1b[0m")

		if y < len(content)-1 {
			buf.WriteByte('\n')
		}
	}

	return buf.String()
}

// GetCursorPosition 获取当前光标位置
func (tc *TerminalCapturer) GetCursorPosition() (int, int) {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	if tc.Emulator == nil {
		return 0, 0
	}

	return tc.Emulator.GetCursorPosition()
}

// GetSize 获取终端尺寸
func (tc *TerminalCapturer) GetSize() (int, int) {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	if tc.Emulator == nil {
		return 0, 0
	}

	return tc.Emulator.GetSize()
}

// Resize 调整终端尺寸
func (tc *TerminalCapturer) Resize(width, height int) {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	if tc.Emulator != nil {
		tc.Emulator.Resize(width, height)
	}
}

// Close 关闭捕获器
func (tc *TerminalCapturer) Close() error {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	if tc.closed {
		return nil
	}

	tc.closed = true

	if tc.PTY != nil {
		return tc.PTY.Close()
	}

	return nil
}
