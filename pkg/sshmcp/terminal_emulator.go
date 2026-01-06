package sshmcp

import (
	"os"
)

// TerminalEmulator 定义终端模拟器的抽象接口
// 这个抽象层允许我们使用不同的终端模拟器实现
type TerminalEmulator interface {
	// Write 写入数据到终端（ANSI 序列）
	Write(data []byte) (int, error)

	// GetScreenContent 获取当前屏幕内容（纯文本，按行组织）
	GetScreenContent() [][]rune

	// GetScreenContentWithFormat 获取屏幕内容和格式信息（包含颜色）
	GetScreenContentWithFormat() ([][]rune, [][]Format)

	// GetCursorPosition 获取光标位置 (x, y)
	GetCursorPosition() (int, int)

	// GetSize 获取终端尺寸 (width, height)
	GetSize() (int, int)

	// Resize 调整终端大小
	Resize(width, height int)

	// Close 关闭模拟器并释放资源
	Close() error
}

// Format 表示单元格的格式信息（颜色、样式等）
type Format struct {
	Fg, Bg    interface{} // 颜色（接口类型，兼容不同库）
	Bold      bool
	Italic    bool
	Underline bool
	Blink     bool
	Reverse   bool
}

// TerminalEmulatorType 终端模拟器类型
type TerminalEmulatorType string

const (
	// EmulatorTypeVT100 使用 vito/vt100（有残留问题）
	EmulatorTypeVT100 TerminalEmulatorType = "vt100"

	// EmulatorTypeVT10x 使用 ActiveState/vt10x（推荐，跨平台）
	EmulatorTypeVT10x TerminalEmulatorType = "vt10x"
)

// GetTerminalEmulator 根据类型创建终端模拟器
func GetTerminalEmulator(emulatorType TerminalEmulatorType, width, height int) (TerminalEmulator, error) {
	switch emulatorType {
	case EmulatorTypeVT10x:
		return NewVT10xEmulator(width, height)
	case EmulatorTypeVT100:
		fallthrough
	default:
		return NewVT100Emulator(width, height)
	}
}

// NewTerminalEmulatorFromEnv 从环境变量读取模拟器类型并创建
// 环境变量：SSH_MCP_TERMINAL_EMULATOR (vt100 or vt10x)
// 如果环境变量未设置，默认使用 vt10x（跨平台，推荐）
func NewTerminalEmulatorFromEnv(width, height int) (TerminalEmulator, error) {
	emulatorType := getTerminalEmulatorTypeFromEnv()
	return GetTerminalEmulator(emulatorType, width, height)
}

// getTerminalEmulatorTypeFromEnv 从环境变量获取模拟器类型
func getTerminalEmulatorTypeFromEnv() TerminalEmulatorType {
	// 从环境变量读取
	envType := os.Getenv("SSH_MCP_TERMINAL_EMULATOR")

	if envType != "" {
		return TerminalEmulatorType(envType)
	}

	// 默认行为：使用 vt10x（跨平台，推荐）
	return EmulatorTypeVT10x
}
