package sshmcp

import (
	"github.com/vito/vt100"
)

// VT100Adapter 适配 vito/vt100 库到 TerminalEmulator 接口
type VT100Adapter struct {
	vt *vt100.VT100
}

// NewVT100Emulator 创建 VT100 终端模拟器
func NewVT100Emulator(width, height int) (*VT100Adapter, error) {
	vt := vt100.NewVT100(height, width)
	return &VT100Adapter{vt: vt}, nil
}

// Write 实现 TerminalEmulator 接口
func (a *VT100Adapter) Write(data []byte) (int, error) {
	return a.vt.Write(data)
}

// GetScreenContent 实现 TerminalEmulator 接口
func (a *VT100Adapter) GetScreenContent() [][]rune {
	return a.vt.Content
}

// GetScreenContentWithFormat 实现 TerminalEmulator 接口
func (a *VT100Adapter) GetScreenContentWithFormat() ([][]rune, [][]Format) {
	content := a.vt.Content
	vtFormat := a.vt.Format

	// 转换格式
	format := make([][]Format, len(content))
	for y := range content {
		format[y] = make([]Format, len(content[y]))
		for x := range content[y] {
			if x < len(vtFormat) && y < len(vtFormat) {
				vtCellFmt := vtFormat[y][x]
				format[y][x] = Format{
					Fg:        vtCellFmt.Fg,
					Bg:        vtCellFmt.Bg,
					Bold:      vtCellFmt.Intensity == 1,
					Italic:    vtCellFmt.Italic,
					Underline: vtCellFmt.Underline,
					Blink:     vtCellFmt.Blink,
					Reverse:   vtCellFmt.Reverse,
				}
			}
		}
	}

	return content, format
}

// GetCursorPosition 实现 TerminalEmulator 接口
func (a *VT100Adapter) GetCursorPosition() (int, int) {
	return int(a.vt.Cursor.X), int(a.vt.Cursor.Y)
}

// GetSize 实现 TerminalEmulator 接口
func (a *VT100Adapter) GetSize() (int, int) {
	return a.vt.Width, a.vt.Height
}

// Resize 实现 TerminalEmulator 接口
func (a *VT100Adapter) Resize(width, height int) {
	a.vt.Resize(height, width)
}

// Close 实现 TerminalEmulator 接口
func (a *VT100Adapter) Close() error {
	// vt100 不需要显式关闭
	return nil
}
