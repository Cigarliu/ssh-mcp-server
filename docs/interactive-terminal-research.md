# 交互式终端程序支持 - 技术调研报告

## 📋 需求概述

**目标**：支持所有交互式终端程序，而不只是简单的命令执行

**需要支持的程序类型**：
- **调试器**：gdb, lldb
- **编辑器**：vim, nano, emacs
- **监控工具**：top, htop, iotop
- **REPL**：python, node, irb
- **数据库客户端**：mysql, psql, mongosh
- **容器工具**：docker attach, kubectl exec
- **会话管理**：tmux, screen
- **分页器**：less, more, most
- **其他 ncurses 程序**：任何使用终端控制序列的程序

---

## 🔍 当前实现的问题

### 问题 1：阻塞式读取（致命问题）

**位置**：`pkg/sshmcp/shell_session.go:94-137`

```go
// 当前的 ReadOutput 实现
func (ss *SSHShellSession) ReadOutput(timeout time.Duration) (string, string, error) {
    // ...
    go func() {
        _, err := io.Copy(&stdoutBuf, ss.Stdout)  // ← 阻塞直到 EOF
        done <- err
    }()
    // ...
}
```

**问题描述**：
- `io.Copy` 会一直读取直到遇到 **EOF**
- 交互式程序（gdb、vim、top）**永远不会发送 EOF**
- 每次读取都必须等到 **timeout** 才能返回
- 导致用户体验极差，无法实时看到输出

### 问题 2：缺少终端模式控制

交互式程序需要两种模式：

| 模式 | 特点 | 适用场景 |
|------|------|---------|
| **Canonical Mode** (Cooked) | 逐行缓冲，按回车发送 | 简单命令（ls, echo） |
| **Raw Mode** | 逐字符发送，即时响应 | 交互式程序（vim, gdb, top） |

**当前问题**：
- 虽然请求了 PTY，但没有控制终端模式
- 无法切换到 Raw Mode
- 导致方向键、特殊按键无法正常工作

### 问题 3：没有处理 ANSI 转义序列

交互式程序大量使用 ANSI 控制序列：

```
示例：gdb --tui 输出
\033[7m       # 反色显示
\033[2J       # 清屏
\033[H        # 移动光标到左上角
\033[5;20H    # 移动光标到第 5 行第 20 列
\033[31m      # 设置红色前景色
\033[0m       # 重置所有属性
```

**当前问题**：
- 没有解析这些控制序列
- 输出会包含大量"乱码"
- 无法正确渲染界面

### 问题 4：输入输出时序问题

交互式程序的交互模式：
```
用户输入 → [等待处理] → 程序响应 → [等待输入]
     ↑                                  ↓
     └───── 需要精确的时序控制 ────────┘
```

**当前问题**：
- `WriteInput` 和 `ReadOutput` 是分离的调用
- 无法精确控制读写时序
- 可能在程序还没准备好时就读取
- 或者在程序等待输入时没有及时发送

### 问题 5：特殊字符处理

交互式程序需要的特殊字符：
- `Ctrl+C` (SIGINT) - 中断程序
- `Ctrl+D` (EOF) - 退出
- `Ctrl+Z` (SIGTSTP) - 挂起
- `Ctrl+L` - 清屏
- 方向键 - TUI 界面导航
- Tab - 自动补全

**当前问题**：
- 这些特殊字符需要通过 PTY 正确发送
- 当前只是简单写入字符串，无法正确处理

---

## 💡 解决方案调研

### 方案 A：使用 creack/pty 库（推荐 ⭐）

**仓库**：https://github.com/creack/pty

**简介**：Go 语言最流行的伪终端库，被大量项目使用

**优点**：
- ✅ 经过充分验证，稳定性高
- ✅ 跨平台支持（Linux, macOS, Windows）
- ✅ 处理了底层 PTY 细节
- ✅ 被 gotty、ttyd 等知名项目使用
- ✅ 支持 Raw Mode 切换
- ✅ 易于集成

**缺点**：
- ❌ 仍然需要自己处理 ANSI 序列（但这可以交给客户端）
- ❌ 需要重构现有代码

**基本用法**：
```go
import (
    "os/exec"
    "github.com/creack/pty"
)

// 启动命令并分配 PTY
cmd := exec.Command("vim")
ptmx, err := pty.Start(cmd)
if err != nil {
    return err
}
defer ptmx.Close()

// 非阻塞读写
go io.Copy(os.Stdout, ptmx)  // 输出 → 屏幕
go io.Copy(ptmx, os.Stdin)   // 键盘 → 输入

// 等待命令结束
cmd.Wait()
```

**适用场景**：
- ✅ 需要在服务端完整控制终端
- ✅ 需要启动独立进程
- ✅ 需要最佳兼容性

---

### 方案 B：Web 终端方案（gotty/ttyd + xterm.js）

**核心思路**：
```
SSH → PTY → WebSocket → Browser (xterm.js)
```

**相关项目**：
- **gotty**：https://github.com/sorenisanerd/gotty - Go 实现的 Web 终端
- **ttyd**：C 语言实现的 gotty 替代品
- **xterm.js**：https://github.com/xtermjs/xterm.js - 浏览器端终端模拟器

**工作原理**：
1. 服务端使用 PTY 启动程序
2. 通过 WebSocket 双向传输数据
3. 浏览器端的 xterm.js 处理：
   - ANSI 转义序列解析
   - 终端渲染
   - 用户输入
   - 特殊按键

**优点**：
- ✅ 完美支持所有交互式程序
- ✅ xterm.js 经过大量验证
- ✅ 客户端处理渲染，服务端负载低
- ✅ 支持颜色、光标移动等所有终端特性

**缺点**：
- ❌ 需要客户端支持（浏览器或 WebSocket）
- ❌ 架构复杂度增加
- ❌ 不适合纯 CLI 使用场景

**适用场景**：
- ✅ Web SSH 服务
- ✅ 远程终端访问
- ✅ 需要图形化界面的场景

---

### 方案 C：自己实现终端模拟器（不推荐）

**需要实现**：

1. **ANSI 转义序列解析器**
   - VT100/VT220 标准
   - xterm 扩展
   - 解析状态机
   - 参考：https://vt100.net/emu/dec_ansi_parser

2. **虚拟终端缓冲区**
   - 字符矩阵（通常 24x80 或更大）
   - 光标位置跟踪
   - 滚动区域管理
   - 属性存储（颜色、粗体、下划线等）

3. **输入/输出管理**
   - 非阻塞 I/O
   - 事件驱动架构
   - 原始模式支持
   - 特殊字符映射

**复杂度**：非常高（需要数千行代码）

**优点**：
- ✅ 完全控制
- ✅ 可以定制功能
- ✅ 不依赖外部库

**缺点**：
- ❌ 开发成本极高
- ❌ 容易出现边缘情况 bug
- ❌ 维护成本高
- ❌ 重复造轮子

**适用场景**：
- ⚠️ 仅当有特殊需求且上述方案都不满足时

---

### 方案 D：轻量级适配层（折中方案 ⭐⭐）

**核心思路**：不实现完整的终端模拟器，而是提供基础支持

**需要实现**：

1. **非阻塞 I/O**
   ```go
   func (ss *SSHShellSession) ReadNonBlocking(timeout time.Duration) (string, error) {
       buf := make([]byte, 4096)
       ss.Stdout.SetReadDeadline(time.Now().Add(timeout))

       n, err := ss.Stdout.Read(buf)
       if errors.Is(err, os.ErrDeadlineExceeded) {
           return string(buf[:n]), nil  // 返回已读取的部分
       }
       return string(buf[:n]), err
   }
   ```

2. **交互式程序检测**
   ```go
   var interactivePrograms = []string{
       "vim", "nano", "emacs",
       "gdb", "lldb",
       "top", "htop",
       "python", "node", "irb",
       "mysql", "psql",
   }

   func IsInteractiveProgram(cmd string) bool {
       for _, prog := range interactivePrograms {
           if strings.Contains(cmd, prog) {
               return true
           }
       }
       return false
   }
   ```

3. **模式切换**
   ```go
   type TerminalMode int
   const (
       ModeRaw      TerminalMode = iota  // 原始模式（交互式程序）
       ModeCooked                        // 熟模式（简单命令）
   )

   func (ss *SSHShellSession) SetMode(mode TerminalMode) error {
       // 设置终端模式
   }
   ```

4. **可选的 ANSI 过滤**
   ```go
   type ANSIMode int
   const (
       ANSIRaw      ANSIMode = iota  // 原始透传
       ANSIFiltered                   // 过滤 ANSI 序列
       ANSIParsed                     // 解析为结构化数据
   )
   ```

**优点**：
- ✅ 实现成本较低
- ✅ 保持当前架构
- ✅ 向后兼容
- ✅ 可以逐步完善

**缺点**：
- ❌ 功能不完整
- ❌ 可能无法支持所有程序

**适用场景**：
- ✅ 快速改进现有实现
- ✅ 主要使用场景是简单命令
- ✅ 交互式程序使用频率不高

---

## 🎯 推荐方案

基于你的 SSH MCP 架构和使用场景，我建议：

### **阶段 1：基础改进（立即可做）**

1. **实现非阻塞读取**
   - 解决 EOF 阻塞问题
   - 提供实时响应

2. **添加 Raw Mode 选项**
   - 支持需要逐字符输入的程序
   - 保持向后兼容

3. **改进错误处理**
   - 更好的超时控制
   - 更清晰的错误信息

### **阶段 2：根据使用场景选择方向**

**场景 A：主要用于 CLI 脚本**
- 当前方案 + 轻量级适配层
- 支持常见交互式程序
- 不追求完美兼容性

**场景 B：需要 Web UI**
- 参考 gotty 架构
- 服务端：PTY 管理
- 客户端：xterm.js 渲染
- 通过 WebSocket 通信

**场景 C：完全自动化（expect-like）**
- 添加 expect 功能
- 等待特定提示符
- 自动化交互流程

### **阶段 3：长期优化**

根据实际使用反馈：
- 优化性能
- 扩展支持的程序
- 改进用户体验

---

## 📊 技术对比总结

| 特性 | 当前实现 | creack/pty | gotty/xterm.js | 轻量适配 |
|------|---------|------------|----------------|----------|
| 非阻塞 I/O | ❌ | ✅ | ✅ | ✅ |
| Raw Mode | ❌ | ✅ | ✅ | ✅ |
| ANSI 解析 | ❌ | ⚠️ | ✅ | ⚠️ |
| 特殊字符 | ❌ | ✅ | ✅ | ⚠️ |
| 窗口大小 | ✅ | ✅ | ✅ | ✅ |
| 所有交互程序 | ❌ | ✅ | ✅ | ⚠️ |
| 实现复杂度 | 低 | 中 | 高 | 中 |
| 依赖项 | 无 | 1 个库 | 多个 | 无 |
| Web 支持 | ❌ | ⚠️ | ✅ | ❌ |
| CLI 友好 | ✅ | ✅ | ❌ | ✅ |

---

## 🔧 实施建议

### 如果选择 creack/pty：

1. **保持现有 API 兼容性**
   ```go
   // 添加新方法，不修改现有方法
   func (s *Session) CreateShellWithPty(...) (*SSHShellSession, error)
   ```

2. **渐进式迁移**
   - 先实现基础功能
   - 添加测试
   - 逐步替换

3. **添加配置选项**
   ```go
   type ShellConfig struct {
       Mode           TerminalMode  // Raw or Cooked
       ANSIMode       ANSIMode      // 如何处理 ANSI
       ReadTimeout    time.Duration
       WriteTimeout   time.Duration
   }
   ```

### 如果选择轻量级适配：

1. **最小改动原则**
   - 只修改必要部分
   - 保持现有测试通过

2. **提供开关**
   - 默认使用兼容模式
   - 用户可选择启用新功能

3. **充分测试**
   - 测试常见程序（vim, top, gdb）
   - 测试边界情况

---

## 📚 参考资料

### 技术文档
- [ANSI escape code - Wikipedia](https://en.wikipedia.org/wiki/ANSI_escape_code)
- [VT100 ANSI Parser](https://vt100.net/emu/dec_ansi_parser)
- [The TTY demystified](https://waynerv.com/posts/how-tty-system-works/)
- [Console Virtual Terminal Sequences - Microsoft](https://learn.microsoft.com/en-us/windows/console/console-virtual-terminal-sequences)

### Go 库
- [creack/pty - GitHub](https://github.com/creack/pty)
- [golang.org/x/crypto/ssh](https://pkg.go.dev/golang.org/x/crypto/ssh)

### 参考项目
- [gotty - Share your terminal as a web application](https://github.com/sorenisanerd/gotty)
- [xterm.js - A terminal for the web](https://github.com/xtermjs/xterm.js)
- [sshx - A web-based SSH client](https://github.com/novitae/sshx)

### 讨论和问答
- [Implementing nCurses over telnet/ssh - StackOverflow](https://stackoverflow.com/questions/13334827/implementing-ncurses-over-telnet-ssh)
- [How to render a remote ncurses console - StackOverflow](https://stackoverflow.com/questions/16382252/how-to-render-a-remote-ncurses-console)
- [Interactive SSH : r/golang](https://www.reddit.com/r/golang/comments/87hi86/interactive_ssh/)

### 中文资料
- [Go进阶：如何开发多彩动感的终端UI应用 - InfoQ](https://www.infoq.cn/article/jjQlJFLtfT8B4ogIJoOf)
- [使用伪终端实现golang语言中的终端功能 - CSDN](https://blog.csdn.net/HoUnix/article/details/133370676)
- [Go语言实现SSH远程终端及WebSocket - 博客园](https://www.cnblogs.com/you-men/p/13934845.html)

---

## 📝 决策记录

**调研日期**：2025-01-03
**负责人**：Claude Code
**状态**：待决策

### 待决策项：
1. 是否使用 creack/pty 库？
2. 是否需要 Web 界面支持？
3. 主要使用场景是什么（CLI / Web / 自动化）？
4. 兼容性要求（需要支持多少种交互式程序）？

### 下一步：
- [ ] 讨论并确定最终方案
- [ ] 创建实施计划
- [ ] 开始开发

---

**最后更新**：2025-01-03
