# SSH MCP 终端模拟器使用指南

## 概述

SSH MCP Server 现在支持**两种终端模拟器后端**，用于渲染交互式全屏程序（htop、vim、tmux 等）：

1. **VT100 (vito/vt100)** - 传统实现，兼容性好但有小 bug
2. **VT10x (ActiveState/vt10x)** - 现代实现，跨平台，ANSI 解析正确（推荐）

## 快速开始

### 默认配置（推荐）

**什么都不用做**！系统默认使用 **VT10x**（跨平台，ANSI 解析正确）

### 手动指定模拟器

如果需要手动指定，设置环境变量：

```bash
# 全平台（Linux/Mac/Windows）
export SSH_MCP_TERMINAL_EMULATOR=vt10x  # 推荐

# 或者使用 VT100（有兼容性问题）
export SSH_MCP_TERMINAL_EMULATOR=vt100

# Windows (PowerShell)
$env:SSH_MCP_TERMINAL_EMULATOR="vt10x"

# Windows (CMD)
set SSH_MCP_TERMINAL_EMULATOR=vt10x
```

## 使用场景

### 场景 1: 交互式程序监控（htop）

```go
// 连接服务器
session, _ := sm.CreateSession("example.com", 22, "user", authConfig, "monitor")

// 创建交互式 shell
shell, _ := session.CreateShellWithConfig("xterm-256color", 40, 160, shellConfig)

// 启动 htop
shell.WriteInput("htop\n")
time.Sleep(2 * time.Second)

// 获取快照（自动使用配置的模拟器）
snapshot := shell.GetTerminalSnapshot()
fmt.Println(snapshot)

// 退出
shell.WriteInput("q")
```

### 场景 2: 文本编辑（vim）

```go
// 创建 shell
shell, _ := session.CreateShellWithConfig("xterm-256color", 40, 160, shellConfig)

// 启动 vim
shell.WriteInput("vim /tmp/test.txt\n")
time.Sleep(1 * time.Second)

// 输入内容
shell.WriteInput("iHello, World!\x1b")  // i 进入插入模式，ESC 退出
time.Sleep(500 * time.Millisecond)

// 保存退出
shell.WriteInput(":wq\n")
time.Sleep(1 * time.Second)

// 获取最终状态
snapshot := shell.GetTerminalSnapshot()
```

### 场景 3: 会话管理（tmux）

```go
// 创建 shell
shell, _ := session.CreateShellWithConfig("xterm-256color", 40, 160, shellConfig)

// 启动 tmux
shell.WriteInput("tmux\n")
time.Sleep(1 * time.Second)

// 创建新窗口
shell.WriteInput("\x01c")  // Ctrl+b, c
time.Sleep(500 * time.Millisecond)

// 获取快照查看 tmux 状态
snapshot := shell.GetTerminalSnapshot()
```

## 模拟器对比

### VT100 (vito/vt100)

**优点**:
- ✅ 全平台支持（Windows、Linux、Mac）
- ✅ 成熟稳定
- ✅ 轻量级

**缺点**:
- ❌ ANSI 解析有 bug，字符伪影
- ❌ 输出质量较差

**适用场景**:
- Windows 平台（唯一选择）
- 简单命令输出
- 不在意渲染质量

### VT10x (推荐)

**优点**:
- ✅ ANSI 解析正确，无伪影（99.66% 伪影消除）
- ✅ 输出质量优秀
- ✅ 现代化设计
- ✅ 全平台支持（Windows、Linux、Mac）

**缺点**:
- ⚠️ 颜色支持待完善（低优先级）

**适用场景**:
- 所有平台（推荐）
- 复杂交互式程序（htop、vim、tmux）
- 需要高质量渲染

## 显示效果对比

### VT100 输出（有伪影）

```
  0B[B||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||100.0%B]B Tasks: B86B, B292B thrB, 78 kthrB; B1B running
  |||       |||B                                                     16.7%         90                    2
  1BB|||||||||||||||||                                                 B21.2]B Load average: B1.75 1.43 B1.22
  B 100073 root |||||B||||||||||B|||||||||||||||||||||||||||||||||||||||||||B8505M/3.82GB]B Uptime: B91 days, B2:19:00
```

**问题**: 大量 'B' 字符伪影（294 个），格式混乱

### VT10x 输出（推荐）

```
  [||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||]100.0% Tasks: 86, 292 thr,  78 kthr;   1 running
  |||       |||                                                      16.7%         90                    2
  1|||||||||||||||                                                    21.2% Load average: 1.75 1.43 1.22
    100073 root |||||||||||||||||||||||||||||||||||||||||||||||||||||||8505M/3.82GB Uptime: 91 days,  2:19:00
```

**改善**: 无伪影，格式清晰（99.66% 伪影消除）

## 配置建议

### 推荐配置（按平台）

#### 所有平台（推荐）

```bash
# ~/.bashrc 或 ~/.zshrc（Linux/Mac）
# 或环境变量（Windows）
export SSH_MCP_TERMINAL_EMULATOR=vt10x
```

**理由**: 最佳渲染质量，全平台支持，无伪影（99.66% 伪影消除）

### 高级配置

#### 动态切换（按会话）

```go
// 会话 1: 使用 VT10x（高质量）
os.Setenv("SSH_MCP_TERMINAL_EMULATOR", "vt10x")
session1, _ := sm.CreateSession("prod-server", 22, "user", auth, "prod")

// 会话 2: 使用 VT100（兼容性）
os.Setenv("SSH_MCP_TERMINAL_EMULATOR", "vt100")
session2, _ := sm.CreateSession("legacy-server", 22, "user", auth, "legacy")
```

#### 编程方式指定

```go
// 创建时指定模拟器类型
capturer, err := sshmcp.NewTerminalCapturerWithType(
    160, 40,
    sshmcp.EmulatorTypeVT10x,  // 或 EmulatorTypeVT100
)
```

## 故障排除

### 问题 1: 输出有字符伪影

**症状**: 大量 'B' 字符，格式混乱

**原因**: 正在使用 vt100 模拟器

**解决方案**:
```bash
# 切换到 VT10x（全平台）
export SSH_MCP_TERMINAL_EMULATOR=vt10x
```

### 问题 2: 颜色显示不正确

**症状**: 颜色信息丢失或错误

**原因**: VT10x 颜色支持待完善

**临时方案**: 使用 vt100 模拟器
```bash
export SSH_MCP_TERMINAL_EMULATOR=vt100
```

**长期方案**: 等待 VT10x 颜色支持完善（当前为低优先级）

### 问题 4: 性能问题

**症状**: 渲染速度慢，CPU 使用高

**诊断**:
```go
// 检查模拟器类型
emulatorType := os.Getenv("SSH_MCP_TERMINAL_EMULATOR")
log.Printf("当前模拟器: %s", emulatorType)
```

**解决方案**:
- 调整终端尺寸（减小 rows/cols）
- 检查网络延迟
- 更新到最新版本

## 最佳实践

### 1. 终端尺寸选择

```go
// 小屏幕（简单命令）
shell, _ := session.CreateShellWithConfig("xterm-256color", 24, 80, config)

// 标准屏幕（大多数场景）✅ 推荐
shell, _ := session.CreateShellWithConfig("xterm-256color", 40, 160, config)

// 大屏幕（复杂表格）
shell, _ := session.CreateShellWithConfig("xterm-256color", 50, 200, config)
```

### 2. 读取超时设置

```go
// 交互式程序（需要持续读取）
shellConfig := &ShellConfig{
    Mode:         TerminalModeRaw,
    ANSIMode:     ANSIRaw,
    ReadTimeout:  100 * time.Millisecond,  // ✅ 推荐
    WriteTimeout: 5 * time.Second,
}

// 简单命令（可以设置更长）
shellConfig := &ShellConfig{
    Mode:         TerminalModeRaw,
    ANSIMode:     ANSIRaw,
    ReadTimeout:  500 * time.Millisecond,  // 更宽松
    WriteTimeout: 5 * time.Second,
}
```

### 3. 退出交互式程序

**重要**: 始终正确退出程序，避免僵尸进程

```go
// ✅ 正确
shell.WriteInput("q")  // 退出 htop
time.Sleep(500 * time.Millisecond)
shell.Close()

// ❌ 错误（直接关闭，可能留下僵尸进程）
shell.Close()
```

### 4. 错误处理

```go
snapshot := shell.GetTerminalSnapshot()
if snapshot == "" {
    // 可能是：
    // 1. 程序未启动
    // 2. 读取超时
    // 3. 连接断开
    log.Error().Msg("快照为空，检查连接状态")
}
```

## API 参考

### 环境变量

| 变量名 | 值 | 平台 | 默认 |
|--------|-----|------|------|
| `SSH_MCP_TERMINAL_EMULATOR` | `vt100` \| `vt10x` | 全平台 | vt10x（推荐） |

### 核心接口

```go
type TerminalEmulator interface {
    // 写入 ANSI 序列
    Write(data []byte) (int, error)

    // 获取屏幕内容（纯文本）
    GetScreenContent() [][]rune

    // 获取屏幕内容（带格式）
    GetScreenContentWithFormat() ([][]rune, [][]Format)

    // 获取光标位置
    GetCursorPosition() (int, int)

    // 获取终端尺寸
    GetSize() (int, int)

    // 调整终端尺寸
    Resize(width, height int)

    // 关闭模拟器
    Close() error
}
```

### 工厂函数

```go
// 使用环境变量创建
func NewTerminalCapturer(width, height int) (*TerminalCapturer, error)

// 指定类型创建
func NewTerminalCapturerWithType(
    width, height int,
    emulatorType TerminalEmulatorType,
) (*TerminalCapturer, error)
```

## 示例代码

### 完整示例：htop 监控

```go
package main

import (
    "fmt"
    "os"
    "time"

    "github.com/cigar/sshmcp/pkg/sshmcp"
    "github.com/rs/zerolog"
)

func main() {
    // 配置使用 VT10x（推荐）
    os.Setenv("SSH_MCP_TERMINAL_EMULATOR", "vt10x")

    // 创建会话管理器
    logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
    config := sshmcp.ManagerConfig{
        MaxSessions:    10,
        SessionTimeout: 10 * time.Minute,
        IdleTimeout:    5 * time.Minute,
        Logger:         &logger,
    }
    sm := sshmcp.NewSessionManager(config)
    defer sm.Close()

    // 连接服务器
    session, err := sm.CreateSession("server", 22, "user", &sshmcp.AuthConfig{
        Type:     sshmcp.AuthTypePassword,
        Password: "password",
    }, "monitor")
    if err != nil {
        logger.Fatal().Err(err).Msg("连接失败")
    }

    // 创建 shell
    shellConfig := &sshmcp.ShellConfig{
        Mode:         sshmcp.TerminalModeRaw,
        ANSIMode:     sshmcp.ANSIRaw,
        ReadTimeout:  100 * time.Millisecond,
        WriteTimeout: 5 * time.Second,
    }

    shell, err := session.CreateShellWithConfig("xterm-256color", 40, 160, shellConfig)
    if err != nil {
        logger.Fatal().Err(err).Msg("创建 shell 失败")
    }
    defer shell.Close()

    // 启动 htop
    shell.WriteInput("htop\n")
    time.Sleep(2 * time.Second)

    // 获取快照
    snapshot := shell.GetTerminalSnapshot()
    fmt.Println(snapshot)

    // 退出 htop
    shell.WriteInput("q")
    time.Sleep(500 * time.Millisecond)
}
```

## 参考资源

- [测试报告](TESTING_REPORT.md) - 测试结果和性能对比
- [API 文档](https://godoc.org/github.com/cigar/sshmcp/pkg/sshmcp) - GoDoc 文档

## 更新日志

### v1.2.0 (2026-01-06)

- ✅ 新增 VT10x 终端模拟器支持（ActiveState/vt10x）
- ✅ 创建抽象接口层
- ✅ 修复 ANSI 伪影问题（99.66% 伪影消除）
- ✅ 全平台支持（Windows/Linux/Mac）
- ✅ 移除 Bubbleterm（仅支持 Linux/Mac）

### v1.1.0

- ✅ VT100 终端模拟器
- ✅ 基础交互式 shell 功能
- ✅ 终端快照功能

### v1.0.0 (初始版本)

- ✅ VT100 终端模拟器
- ✅ 基础交互式 shell 功能
- ✅ 终端快照功能

---

**文档版本**: 1.1.0
**最后更新**: 2026-01-06
**维护者**: [您的名字]
