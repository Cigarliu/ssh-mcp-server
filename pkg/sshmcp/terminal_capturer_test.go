package sshmcp

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTerminalCapturer_NewTerminalCapturer 测试创建终端捕获器
func TestTerminalCapturer_NewTerminalCapturer(t *testing.T) {
	capturer, err := NewTerminalCapturer(80, 24)
	require.NoError(t, err)
	require.NotNil(t, capturer)

	assert.NotNil(t, capturer.Emulator)
	assert.False(t, capturer.closed)
}

// TestTerminalCapturer_GetScreenSnapshot 测试获取屏幕快照
func TestTerminalCapturer_GetScreenSnapshot(t *testing.T) {
	capturer, err := NewTerminalCapturer(80, 24)
	require.NoError(t, err)

	// 写入一些简单文本（不包含ANSI序列）
	testData := []byte("Hello, World!\n")
	capturer.Emulator.Write(testData)

	// 获取快照
	snapshot := capturer.GetScreenSnapshot()

	assert.Contains(t, snapshot, "Hello, World!")
}

// TestTerminalCapturer_ANSISequences 测试ANSI序列处理
func TestTerminalCapturer_ANSISequences(t *testing.T) {
	capturer, err := NewTerminalCapturer(80, 24)
	require.NoError(t, err)

	// 测试颜色序列
	testData := []byte("\x1b[31mRed Text\x1b[0m\n")
	capturer.Emulator.Write(testData)

	// 获取纯文本快照
	snapshot := capturer.GetScreenSnapshot()
	assert.Contains(t, snapshot, "Red Text")

	// 获取带颜色的快照
	coloredSnapshot := capturer.GetScreenSnapshotWithColor()
	assert.Contains(t, coloredSnapshot, "Red Text")
	assert.Contains(t, coloredSnapshot, "\x1b[") // 应该包含ANSI序列
}

// TestTerminalCapturer_CursorPosition 测试光标位置
func TestTerminalCapturer_CursorPosition(t *testing.T) {
	capturer, err := NewTerminalCapturer(80, 24)
	require.NoError(t, err)

	x, y := capturer.GetCursorPosition()
	// 初始位置应该是 (0, 0)
	assert.Equal(t, 0, x)
	assert.Equal(t, 0, y)
}

// TestTerminalCapturer_Size 测试终端尺寸
func TestTerminalCapturer_Size(t *testing.T) {
	width, height := 120, 40

	capturer, err := NewTerminalCapturer(width, height)
	require.NoError(t, err)

	w, h := capturer.GetSize()
	assert.Equal(t, width, w)
	assert.Equal(t, height, h)
}

// TestTerminalCapturer_Resize 测试调整尺寸
func TestTerminalCapturer_Resize(t *testing.T) {
	capturer, err := NewTerminalCapturer(80, 24)
	require.NoError(t, err)

	// 调整尺寸
	capturer.Resize(100, 30)

	w, h := capturer.GetSize()
	assert.Equal(t, 100, w)
	assert.Equal(t, 30, h)
}

// TestTerminalCapturer_MultipleLines 测试多行文本
func TestTerminalCapturer_MultipleLines(t *testing.T) {
	capturer, err := NewTerminalCapturer(80, 24)
	require.NoError(t, err)

	// 写入多行文本
	testData := []byte("Line 1\nLine 2\nLine 3\n")
	capturer.Emulator.Write(testData)

	snapshot := capturer.GetScreenSnapshot()
	lines := strings.Split(snapshot, "\n")

	// 应该至少包含3行
	assert.GreaterOrEqual(t, len(lines), 3)
	assert.Contains(t, snapshot, "Line 1")
	assert.Contains(t, snapshot, "Line 2")
	assert.Contains(t, snapshot, "Line 3")
}

// TestTerminalCapturer_ClearScreen 测试清屏
func TestTerminalCapturer_ClearScreen(t *testing.T) {
	capturer, err := NewTerminalCapturer(80, 24)
	require.NoError(t, err)

	// 写入一些文本
	testData := []byte("Old Content\n")
	capturer.Emulator.Write(testData)

	// 清屏
	clearSeq := []byte("\x1b[2J")
	capturer.Emulator.Write(clearSeq)

	snapshot := capturer.GetScreenSnapshot()
	// 内容应该被清除了（大部分是空格）
	assert.NotContains(t, snapshot, "Old Content")
}

// TestTerminalCapturer_CursorMovement 测试光标移动
func TestTerminalCapturer_CursorMovement(t *testing.T) {
	capturer, err := NewTerminalCapturer(80, 24)
	require.NoError(t, err)

	// 写入文本并移动光标
	testData := []byte("ABC\x1b[3D") // 移动光标向左3个位置
	capturer.Emulator.Write(testData)

	x, _ := capturer.GetCursorPosition()
	// 光标应该向左移动
	assert.Equal(t, 0, x) // 应该回到起始位置
}

// TestTerminalCapturer_ThreadSafety 测试线程安全
func TestTerminalCapturer_ThreadSafety(t *testing.T) {
	capturer, err := NewTerminalCapturer(80, 24)
	require.NoError(t, err)

	done := make(chan bool)

	// goroutine 1: 持续写入
	go func() {
		for i := 0; i < 100; i++ {
			testData := []byte(fmt.Sprintf("Test line %d\n", i))
			capturer.Emulator.Write(testData)
			time.Sleep(1 * time.Millisecond)
		}
		done <- true
	}()

	// goroutine 2: 持续读取快照
	go func() {
		for i := 0; i < 50; i++ {
			_ = capturer.GetScreenSnapshot()
			_, _ = capturer.GetCursorPosition()
			_, _ = capturer.GetSize()
			time.Sleep(2 * time.Millisecond)
		}
		done <- true
	}()

	// 等待两个goroutine完成
	<-done
	<-done

	// 验证最终状态
	snapshot := capturer.GetScreenSnapshot()
	assert.NotEmpty(t, snapshot)
}

// TestTerminalCapturer_Close 测试关闭
func TestTerminalCapturer_Close(t *testing.T) {
	capturer, err := NewTerminalCapturer(80, 24)
	require.NoError(t, err)

	err = capturer.Close()
	assert.NoError(t, err)

	// 再次关闭应该也不报错
	err = capturer.Close()
	assert.NoError(t, err)

	assert.True(t, capturer.closed)
}
