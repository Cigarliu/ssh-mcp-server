package sshmcp

import (
	"github.com/ActiveState/vt10x"
)

// VT10xAdapter 适配 vt10x 库到 TerminalEmulator 接口
// vt10x 是一个跨平台的 headless terminal emulator
type VT10xAdapter struct {
	state *vt10x.State
	vt    *vt10x.VT
}

// NewVT10xEmulator 创建 VT10x 终端模拟器
func NewVT10xEmulator(width, height int) (*VT10xAdapter, error) {
	// 创建 State（存储屏幕状态）
	state := &vt10x.State{
		// RecordHistory: true, // 可选：记录滚动历史
	}

	// 创建 VT10x 终端（不需要 Reader 和 Writer，用于 headless 模式）
	vt, err := vt10x.New(state, nil, nil)
	if err != nil {
		return nil, err
	}

	adapter := &VT10xAdapter{
		state: state,
		vt:    vt,
	}

	// 初始化终端尺寸（通过写入 ANSI 序列）
	// 使用 Device Attributes 请求来设置尺寸
	adapter.state.WriteString("", height, width)

	return adapter, nil
}

// Write 实现 TerminalEmulator 接口
// 将 ANSI 序列喂给终端模拟器
func (a *VT10xAdapter) Write(data []byte) (int, error) {
	// vt10x 会解析 ANSI 序列并更新 state
	n, err := a.vt.Write(data)
	return n, err
}

// GetScreenContent 实现 TerminalEmulator 接口
// 获取当前屏幕内容的纯文本
func (a *VT10xAdapter) GetScreenContent() [][]rune {
	a.state.Lock()
	defer a.state.Unlock()

	// 获取终端尺寸
	rows, cols := a.state.Size()

	// 构建屏幕内容
	content := make([][]rune, rows)
	for y := 0; y < rows; y++ {
		content[y] = make([]rune, cols)
		for x := 0; x < cols; x++ {
			ch, _, _ := a.state.Cell(x, y)
			content[y][x] = ch
		}
	}

	return content
}

// GetScreenContentWithFormat 实现 TerminalEmulator 接口
// 获取带格式信息的内容（vt10x 支持颜色）
func (a *VT10xAdapter) GetScreenContentWithFormat() ([][]rune, [][]Format) {
	a.state.Lock()
	defer a.state.Unlock()

	rows, cols := a.state.Size()

	content := make([][]rune, rows)
	format := make([][]Format, rows)

	for y := 0; y < rows; y++ {
		content[y] = make([]rune, cols)
		format[y] = make([]Format, cols)
		for x := 0; x < cols; x++ {
			ch, fg, bg := a.state.Cell(x, y)
			content[y][x] = ch
			format[y][x] = Format{
				Fg: fg,
				Bg: bg,
			}
		}
	}

	return content, format
}

// GetCursorPosition 实现 TerminalEmulator 接口
func (a *VT10xAdapter) GetCursorPosition() (int, int) {
	a.state.Lock()
	defer a.state.Unlock()

	return a.state.Cursor()
}

// GetSize 实现 TerminalEmulator 接口
func (a *VT10xAdapter) GetSize() (int, int) {
	a.state.Lock()
	defer a.state.Unlock()

	rows, cols := a.state.Size()
	return cols, rows
}

// Resize 实现 TerminalEmulator 接口
func (a *VT10xAdapter) Resize(width, height int) {
	// 使用 State.WriteString 来更新尺寸
	a.state.Lock()
	defer a.state.Unlock()
	a.state.WriteString("", height, width)
}

// Close 实现 TerminalEmulator 接口
func (a *VT10xAdapter) Close() error {
	// vt10x 不需要显式关闭
	return nil
}
