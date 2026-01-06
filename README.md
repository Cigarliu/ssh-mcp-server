# SSH MCP Server

[English](#english) | [简体中文](#简体中文)

---

## English

<div align="center">

**Transform Claude into your remote operations expert** 🚀

The ultimate SSH MCP solution - the only one with **complete interactive terminal** support

[![GitHub stars](https://img.shields.io/github/stars/Cigarliu/ssh-mcp-server?style=social)](https://github.com/Cigarliu/ssh-mcp-server/stargazers)
[![GitHub forks](https://img.shields.io/github/forks/Cigarliu/ssh-mcp-server?style=social)](https://github.com/Cigarliu/ssh-mcp-server/network/members)
[![Release](https://img.shields.io/github/v/release/Cigarliu/ssh-mcp-server)](https://github.com/Cigarliu/ssh-mcp-server/releases)
[![License](https://img.shields.io/github/license/Cigarliu/ssh-mcp-server)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/Cigarliu/ssh-mcp-server)](https://goreportcard.com/report/github.com/Cigarliu/ssh-mcp-server)

</div>

---

## 💡 Why SSH MCP Server?

Most SSH MCP implementations only support basic command execution. **SSH MCP Server** is different - it provides enterprise-grade features that others don't have:

### 🌟 Key Advantages

| Your Pain Points | SSH MCP Server | Other Solutions |
|------------------|----------------|-----------------|
| ❌ Can only run simple commands | ✅ **Complete interactive terminal** (vim/top/gdb) | ❌ Unsupported |
| ❌ Commands hang/freeze | ✅ **Async mode** - returns instantly | ❌ Blocking execution |
| ❌ Output full of ANSI mess | ✅ **Clean, filtered output** | ❌ Manual cleanup needed |
| ❌ Can't remember session IDs | ✅ **Session alias system** | ❌ Only UUIDs |
| ❌ Lost characters in prompt | ✅ **ECMA-48 standard filtering** | ❌ Stripansi bugs |
| ❌ No execution history | ✅ **Full command audit trail** | ❌ No tracking |
| ❌ Manual password typing | ✅ **Auto sudo password injection** | ⚠️ Partial support |

---

## 🚀 Industry-First Features

### 1️⃣ **Async Mode (Revolutionary)** ⚡

Shell starts and **returns immediately**, runs in background with automatic output buffering!

```
✨ You: "Start shell on prod server"
🤖 Claude: [Shell started in 2ms, running in background]
✨ You: "Execute ls -la"
🤖 Claude: [Output from 3 seconds ago] ...listing...
```

**Key Features:**
- ⏱️ **Instant return** - Shell starts in ~2ms
- 💾 **10000-line circular buffer** - Automatic overflow handling
- ❤️ **3-layer keepalive** - TCP (30s) + SSH (30s) + App heartbeat (60s)
- 🎯 **3 read strategies** - latest_N_lines / all_unread / latest_N_bytes
- ✅ **90+ second long-running verified** - 0 keepalive failures

### 2️⃣ **Complete Interactive Terminal** 🖥️

Run ANY interactive program perfectly:

```bash
vim /etc/nginx.conf      # ✅ Full vim support
top                      # ✅ Real-time monitoring
htop                     # ✅ Interactive process manager
gdb ./myapp             # ✅ Debugging session
tmux attach             # ✅ Terminal multiplexer
```

**What makes it different:**
- 🎮 **Raw/Cooked modes** - Smart adaptation
- ⌨️ **Full keyboard support** - Ctrl+C/D/Z, arrow keys, special keys
- 📏 **Dynamic resize** - Adjust terminal size on the fly
- 🎨 **3 ANSI modes** - Raw/Strip/Parse for different scenarios

### 3️⃣ **ECMA-48 Standard ANSI Filtering** 🧹

The ultimate clean output - **zero pollution, zero character loss, zero duplicates**

```bash
# Before (stripansi bug):
(base) igar@cigar-dev: ~cigar@cigar-dev:~$  # ❌ Missing 'c', duplicate prompt

# After (ECMA-48 parser):
(base) cigar@cigar-dev:~$                    # ✅ Perfect!
```

**Technical highlights:**
- 📜 **ECMA-48 compliant** - Uses `charmbracelet/x/ansi` parser
- 🎯 **7 sequence types** - CSI/OSC/ESC/DCS/APC/PM/SOS
- ⚡ **Zero character loss** - No more missing prompt characters
- 🔧 **Highly maintainable** - Standards-based, community-tested

---

## 🎯 30-Second Quick Start

### Choose Your MCP Client

#### 🌟 **Claude Desktop** (Recommended ⭐⭐⭐⭐⭐)

**Easiest way to get started**

1. **Build the server:**
   ```bash
   git clone git@github.com:Cigarliu/ssh-mcp-server.git
   cd ssh-mcp-server
   go build -o bin/sshmcp ./cmd/server
   ```

2. **Configure Claude Desktop:**

   - **Windows:** Open `%APPDATA%\Claude\claude_desktop_config.json`
   - **macOS:** Open `~/Library/Application Support/Claude/claude_desktop_config.json`

   Add this configuration:
   ```json
   {
     "mcpServers": {
       "ssh-mcp": {
         "command": "D:/path/to/ssh-mcp-server/bin/sshmcp.exe",
         "args": []
       }
     }
   }
   ```

   ⚠️ **Note:** Use forward slashes `/` or double backslashes `\\` for Windows paths

3. **Restart Claude Desktop**

4. **Start using:**
   ```
   Connect to 192.168.1.100, user root, password root, execute ls -la
   ```

✅ **Advantages:** Official client, best stability, full feature support

---

#### 💻 **Cline (VSCode)** ⭐⭐⭐⭐⭐

**Developer's choice with deep VSCode integration**

1. Install [Cline extension](https://marketplace.visualstudio.com/items?itemName=saoudrizwan.cline)
2. Open VSCode Settings → Search "MCP"
3. Click "Configure MCP Servers"
4. Paste the JSON configuration (same as above)
5. Start using in Cline conversations

✅ **Advantages:** Terminal control, high transparency, perfect for dev workflows

---

#### 🔧 **Continue (VSCode)** ⭐⭐⭐⭐

**The first client with full MCP support**

1. Install [Continue extension](https://marketplace.visualstudio.com/items?itemName=Continue.continue)
2. Open Command Palette (Ctrl+Shift+P)
3. Type "Continue: Open Config"
4. Add MCP servers to config file
5. Type `@` to invoke MCP tools

✅ **Advantages:** Open-source, first to support all MCP features, active development

---

#### 🤖 **Cursor AI** ⭐⭐⭐⭐

**Next-gen AI IDE**

1. Open Cursor Settings → MCP
2. Add server configuration
3. Use directly in conversations

✅ **Advantages:** High integration, rich ecosystem (15+ MCP servers)

---

#### 🐙 **GitHub Copilot (VSCode)** ⭐⭐⭐

**Official VSCode integration**

1. Ensure GitHub Copilot is installed
2. Add MCP configuration to `settings.json`
3. Restart VSCode

✅ **Advantages:** Official support, enterprise-grade reliability

---

### 📝 Universal JSON Configuration (All Clients)

```json
{
  "mcpServers": {
    "ssh-mcp": {
      "command": "D:/path/to/ssh-mcp-server/bin/sshmcp.exe",
      "args": [],
      "env": {
        "SSH_MCP_LOG_LEVEL": "info"
      }
    }
  }
}
```

**Configuration Details:**
- `ssh-mcp`: Server name (customizable)
- `command`: Absolute path to executable
  - Windows: `D:/code/ssh-mcp-server/bin/sshmcp.exe` or `D:\\code\\...`
  - macOS/Linux: `/Users/yourname/ssh-mcp-server/bin/sshmcp`
- `args`: Command-line arguments (optional)
- `env`: Environment variables (optional)

**⚠️ Path Notes:**
- ✅ Windows: Use `/` or `\\` (e.g., `D:/code/...`)
- ✅ macOS/Linux: Use absolute paths (e.g., `/Users/...`)
- ❌ Avoid relative paths or paths with Chinese characters

---

### 🎯 Start Using Immediately

After configuration, just use natural language:

**Example 1: Basic Operations**
```
Connect to 192.168.1.100, user root, password root, execute ls -la
```

**Example 2: File Operations**
```
Upload local app.tar.gz to remote server's /tmp/ directory
```

**Example 3: Interactive Commands**
```
Start interactive shell, run top command to check CPU usage
```

That's it! 🎉

---

## 💡 Typical Use Cases

### 🚨 **Scenario 1: Emergency Troubleshooting**

```
You: "Production server CPU spiked, check what's happening with top"
Claude: [Connects → Runs top → Takes screenshot → Analyzes processes]
```

### 📦 **Scenario 2: Batch Deployment**

```
You: "Deploy these 3 packages to 10 servers, start them one by one"
Claude: [Uploads in parallel → Executes sequentially → Returns summary]
```

### 🔧 **Scenario 3: Daily Operations**

```
You: "Check disk space on all servers, list those below 20%"
Claude: [Connects to each → Runs df -h → Generates comparison table]
```

### 🐛 **Scenario 4: Debug Remote Issues**

```
You: "Attach debugger to the running process on prod server"
Claude: [Connects → Starts gdb → Loads process → Provides backtrace]
```

---

## 📖 Complete Feature List

### 🔌 **Connection Management**
- ✅ Quick connect with host/user/password
- ✅ SSH key authentication support
- ✅ Session alias system (no more UUIDs!)
- ✅ Predefined host configuration
- ✅ Auto-save hosts for quick reuse

### 🖥️ **Interactive Terminal**
- ✅ Raw/Cooked mode switching
- ✅ Non-blocking I/O (no EOF hangs)
- ✅ Full keyboard support (Ctrl+C/D/Z, arrows)
- ✅ Dynamic terminal resize (rows/cols)
- ✅ 3 ANSI processing modes (Raw/Strip/Parse)
- ✅ Support for vim/top/htop/gdb/tmux

### ⚡ **Async Mode**
- ✅ Instant shell return (~2ms)
- ✅ Background execution with output buffering
- ✅ 10000-line circular buffer
- ✅ 3-layer keepalive (TCP/SSH/App)
- ✅ 3 read strategies (latest_N/all_unread/latest_bytes)
- ✅ Enhanced shell status (buffer usage, keepalive health)

### 🎨 **ANSI Processing**
- ✅ ECMA-48 standard parser (charmbracelet/x/ansi)
- ✅ Supports 7 ANSI sequence types
- ✅ Zero character loss
- ✅ Zero duplicate prompts
- ✅ Clean, readable output

### 📊 **Command Execution**
- ✅ Single command execution
- ✅ Batch command execution
- ✅ Compact mode output
- ✅ Command history tracking
- ✅ Execution time measurement
- ✅ Exit code recording

### 🔐 **Security & Convenience**
- ✅ Auto sudo password injection
- ✅ Environment variable support
- ✅ Secure credential handling

### 📁 **Current Directory Tracking**
- ✅ Auto-parse shell prompts
- ✅ Supports Ubuntu/Debian format
- ✅ Supports RHEL/CentOS format
- ✅ Supports simple prompts

### 📝 **Audit & Debugging**
- ✅ Detailed command history
- ✅ Filter by source (exec/shell)
- ✅ Success/failure tracking
- ✅ Execution timestamps

### 📂 **File Operations (SFTP)**
- ✅ Upload files
- ✅ Download files
- ✅ List directories
- ✅ Create directories
- ✅ Delete files/directories
- ✅ Recursive operations

---

## 🔧 Technical Architecture

### ANSI Filtering Technology

**Before (stripansi library):**
```go
func filterANSI(s string) string {
    return stripansi.Strip(s)  // ❌ Bug: OSC sequences cause character loss
}
```

**After (ECMA-48 parser):**
```go
func filterANSI(s string) string {
    handler := ansi.Handler{
        Print: func(r rune) {
            if r == '\n' || r == '\t' || r >= 32 {
                text.WriteRune(r)  // ✅ Only collect printable text
            }
        },
        HandleCsi: func(cmd ansi.Cmd, params ansi.Params) {},
        HandleOsc: func(cmd int, data []byte) {},
        HandleEsc: func(cmd ansi.Cmd) {},
        // ... all 7 sequence types handled
    }
    parser.Advance(b)  // ✅ Parse byte by byte
    return text.String()
}
```

**Benefits:**
- ✅ Standards-compliant (ECMA-48)
- ✅ Handles all ANSI sequence types
- ✅ No character loss
- ✅ More maintainable
- ✅ Community-tested

### Performance

- **Compilation:** Single 12MB executable
- **Startup time:** < 100ms
- **Memory usage:** ~20MB (idle), ~50MB (active shell)
- **Async shell return:** ~2ms
- **Buffer capacity:** 10000 lines (~9MB)

---

## 📜 Changelog

### [Unreleased]

**Added (2025-01-06)**
- 🔧 **ANSI Filtering Upgrade**: ECMA-48 standard parser implementation (charmbracelet/x/ansi)
  - ✅ Full compatibility with all ANSI sequence types (CSI/OSC/ESC/DCS/APC/PM/SOS)
  - ✅ Completely resolves OSC sequence character loss (e.g., `cigar` → `igar`)
  - ✅ Better universality, reliability, and maintainability
  - ✅ Cleaner code, better performance
- ✅ **Prompt Integrity Fix**: Completely resolves missing characters and duplicate prompts
- ✅ **Full Unit Test Coverage**: 17 interactive shell tests pass, 90-second long-running test passes
- 🔧 **Windows Compatibility**: Removed Bubbleterm dependency, using VT10x emulator (cross-platform)
- 🔧 **Configuration Enhancement**: Increased MaxSessionsPerHost from 10 to 30
- ✅ **Production-Ready**: Complete real-world testing with gdb/htop/strace debugging workflows

**Added (2025-01-05)**
- 🚀 **Async Mode (Industry First)**: Shell starts and returns immediately, runs in background with automatic output buffering
- 🎯 **3 Read Strategies**: latest_N_lines / all_unread / latest_N_bytes
- 💾 **Circular Buffer**: 10000-line capacity, automatic overflow, real-time reading
- ❤️ **3-Layer Keepalive**: TCP Keepalive (30s) + SSH Keepalive (30s) + App heartbeat (60s)
- 📊 **Enhanced Status Display**: Buffer usage, keepalive status, session health at a glance
- ✅ **Long-Running Verification**: 90-second test passed, 0 keepalive failures, stable connection

**Added (2025-01-04)**
- ✅ **Current Directory Tracking**: Intelligently parse shell prompts, auto-update working directory
- ✅ **Enhanced ANSI Cleaning**: Complete removal of carriage returns and zero-width characters
- ✅ **Command History Filtering**: Filter by source (exec/shell)
- ✅ **Batch Command Compact Output**: Compact mode shows only summary and failed commands
- ✅ **File Transfer Path Optimization**: Clear display of Local/Remote paths

**Added (2025-01-03)**
- ✨ **Interactive Terminal Support**: The only complete interactive SSH terminal in the industry
- ✨ **Non-blocking I/O**: Solves EOF blocking issues, enables real-time AI interaction
- ✨ **Terminal Mode Control**: Raw/Cooked mode smart adaptation
- ✨ **ANSI Processing**: Strip/Parse/Pass-through modes
- ✨ **Special Character Input**: Full support for control keys and arrow keys
- ✨ **Interactive Program Detection**: Auto-recognize 20+ program types

---

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit your changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

---

## 📄 License

MIT License - see [LICENSE](LICENSE) file for details

---

## 👨‍💻 Author

[cigar](https://github.com/Cigarliu)

---

## ⭐ Star History

If this project helps you, please consider giving it a star! ⭐

Your support motivates me to keep improving this project.

---

---

## 简体中文

<div align="center">

**让 Claude 成为你的远程运维专家** 🚀

SSH MCP 的终极方案 - 业界唯一完整交互式终端支持

[![GitHub stars](https://img.shields.io/github/stars/Cigarliu/ssh-mcp-server?style=social)](https://github.com/Cigarliu/ssh-mcp-server/stargazers)
[![GitHub forks](https://img.shields.io/github/forks/Cigarliu/ssh-mcp-server?style=social)](https://github.com/Cigarliu/ssh-mcp-server/network/members)
[![Release](https://img.shields.io/github/v/release/Cigarliu/ssh-mcp-server)](https://github.com/Cigarliu/ssh-mcp-server/releases)
[![License](https://img.shields.io/github/license/Cigarliu/ssh-mcp-server)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/Cigarliu/ssh-mcp-server)](https://goreportcard.com/report/github.com/Cigarliu/ssh-mcp-server)

</div>

---

## 💡 为什么选择 SSH MCP Server?

市面上大多数 SSH MCP 实现只支持基础命令执行。**SSH MCP Server** 与众不同 - 它提供了其他方案没有的企业级功能：

### 🌟 核心优势

| 你的痛点 | SSH MCP Server | 其他方案 |
|---------|----------------|---------|
| ❌ 只能执行简单命令 | ✅ **完整交互式终端** (vim/top/gdb) | ❌ 不支持 |
| ❌ 命令执行会卡死 | ✅ **异步模式** - 立即返回 | ❌ 阻塞式执行 |
| ❌ 输出充满ANSI乱码 | ✅ **干净可读的输出** | ❌ 需手动清理 |
| ❌ 无法记住会话ID | ✅ **会话别名系统** | ❌ 只能用UUID |
| ❌ 提示符字符丢失 | ✅ **ECMA-48标准过滤** | ❌ Stripansi有bug |
| ❌ 没有执行历史 | ✅ **完整命令审计** | ❌ 无追踪 |
| ❌ 手动输入密码 | ✅ **自动sudo密码注入** | ⚠️ 部分支持 |

---

## 🚀 业界独家功能

### 1️⃣ **异步模式（革命性创新）** ⚡

Shell启动后**立即返回**，后台持续运行，输出自动缓冲！

```
✨ 你: "在生产服务器启动shell"
🤖 Claude: [Shell已启动，耗时2ms，后台运行中]
✨ 你: "执行 ls -la"
🤖 Claude: [3秒前的输出] ...文件列表...
```

**核心特性:**
- ⏱️ **立即返回** - Shell启动仅需~2ms
- 💾 **10000行循环缓冲区** - 自动覆盖旧数据
- ❤️ **三层保活机制** - TCP(30s) + SSH(30s) + 应用心跳(60s)
- 🎯 **三种读取策略** - latest_N_lines / all_unread / latest_N_bytes
- ✅ **90秒+长连接验证** - 0次保活失败

### 2️⃣ **完整交互式终端** 🖥️

完美运行任何交互程序：

```bash
vim /etc/nginx.conf      # ✅ 完整vim支持
top                      # ✅ 实时系统监控
htop                     # ✅ 交互式进程管理
gdb ./myapp             # ✅ 调试会话
tmux attach             # ✅ 终端复用器
```

**与众不同之处:**
- 🎮 **Raw/Cooked模式** - 智能适配
- ⌨️ **全键盘支持** - Ctrl+C/D/Z、方向键、特殊按键
- 📏 **动态调整大小** - 灵活调整终端尺寸
- 🎨 **三种ANSI模式** - Raw/Strip/Parse适应不同场景

### 3️⃣ **ECMA-48标准ANSI过滤** 🧹

终极干净输出 - **零污染、零字符丢失、零重复**

```bash
# 之前（stripansi有bug）:
(base) igar@cigar-dev: ~cigar@cigar-dev:~$  # ❌ 缺少'c'，重复提示符

# 现在（ECMA-48 parser）:
(base) cigar@cigar-dev:~$                    # ✅ 完美！
```

**技术亮点:**
- 📜 **符合ECMA-48标准** - 使用 `charmbracelet/x/ansi` 解析器
- 🎯 **支持7种序列类型** - CSI/OSC/ESC/DCS/APC/PM/SOS
- ⚡ **零字符丢失** - 不再有提示符字符缺失
- 🔧 **高可维护性** - 基于标准，社区验证

---

## 🎯 30秒开箱即用

### 选择你的MCP客户端

#### 🌟 **Claude Desktop**（推荐 ⭐⭐⭐⭐⭐）

**最简单的使用方式**

1. **编译服务器:**
   ```bash
   git clone git@github.com:Cigarliu/ssh-mcp-server.git
   cd ssh-mcp-server
   go build -o bin/sshmcp ./cmd/server
   ```

2. **配置Claude Desktop:**

   - **Windows:** 打开 `%APPDATA%\Claude\claude_desktop_config.json`
   - **macOS:** 打开 `~/Library/Application Support/Claude/claude_desktop_config.json`

   添加以下配置：
   ```json
   {
     "mcpServers": {
       "ssh-mcp": {
         "command": "D:/path/to/ssh-mcp-server/bin/sshmcp.exe",
         "args": []
       }
     }
   }
   ```

   ⚠️ **注意：** Windows路径使用 `/` 或 `\\`

3. **重启 Claude Desktop**

4. **开始使用:**
   ```
   连接到 192.168.1.100，用户 root，密码 root，执行 ls -la
   ```

✅ **优势：** 官方客户端，稳定性最佳，功能最完整

---

#### 💻 **Cline (VSCode)** ⭐⭐⭐⭐⭐

**开发者的首选，深度集成VSCode**

1. 安装 [Cline扩展](https://marketplace.visualstudio.com/items?itemName=saoudrizwan.cline)
2. 打开 VSCode 设置 → 搜索 "MCP"
3. 点击 "Configure MCP Servers"
4. 粘贴JSON配置（同上）
5. 在Cline对话中使用

✅ **优势：** 终端控制、透明度高、适合开发工作流

---

#### 🔧 **Continue (VSCode)** ⭐⭐⭐⭐

**首个完整支持MCP的客户端**

1. 安装 [Continue扩展](https://marketplace.visualstudio.com/items?itemName=Continue.continue)
2. 打开命令面板 (Ctrl+Shift+P)
3. 输入 "Continue: Open Config"
4. 在配置文件中添加MCP服务器
5. 输入 `@` 即可调用MCP工具

✅ **优势：** 开源、首个完整支持所有MCP功能、活跃开发

---

#### 🤖 **Cursor AI** ⭐⭐⭐⭐

**新一代AI IDE**

1. 打开 Cursor Settings → MCP
2. 添加服务器配置
3. 在对话中直接使用

✅ **优势：** 集成度高、生态丰富（15+ MCP服务器）

---

#### 🐙 **GitHub Copilot (VSCode)** ⭐⭐⭐

**官方VSCode集成**

1. 确保已安装 GitHub Copilot
2. 在 `settings.json` 中添加MCP配置
3. 重启 VSCode

✅ **优势：** 官方支持、企业级可靠性

---

### 📝 通用JSON配置（所有客户端适用）

```json
{
  "mcpServers": {
    "ssh-mcp": {
      "command": "D:/path/to/ssh-mcp-server/bin/sshmcp.exe",
      "args": [],
      "env": {
        "SSH_MCP_LOG_LEVEL": "info"
      }
    }
  }
}
```

**配置说明：**
- `ssh-mcp`: 服务器名称（可自定义）
- `command`: 可执行文件绝对路径
  - Windows: `D:/code/ssh-mcp-server/bin/sshmcp.exe` 或 `D:\\code\\...`
  - macOS/Linux: `/Users/yourname/ssh-mcp-server/bin/sshmcp`
- `args`: 命令行参数（可选）
- `env`: 环境变量（可选）

**⚠️ 路径注意事项：**
- ✅ Windows: 使用 `/` 或 `\\`（如 `D:/code/...`）
- ✅ macOS/Linux: 使用绝对路径（如 `/Users/...`）
- ❌ 避免使用相对路径或包含中文的路径

---

### 🎯 立即体验

配置完成后，直接用自然语言对话：

**示例1：基本操作**
```
连接到 192.168.1.100，用户 root，密码 root，执行 ls -la
```

**示例2：文件操作**
```
上传本地 app.tar.gz 到远程服务器的 /tmp/ 目录
```

**示例3：交互式命令**
```
启动交互式shell，运行 top 命令查看CPU占用
```

就这么简单！🎉

---

## 💡 典型使用场景

### 🚨 **场景1：紧急故障排查**

```
你: "生产服务器CPU爆了，用top看看哪个进程异常"
Claude: [连接 → 运行top → 截图 → 分析进程]
```

### 📦 **场景2：批量部署**

```
你: "把这3个部署包上传到10台服务器，依次启动"
Claude: [并行上传 → 依次执行 → 返回汇总结果]
```

### 🔧 **场景3：日常运维**

```
你: "检查所有服务器的磁盘空间，列出低于20%的"
Claude: [逐台连接 → 执行df -h → 生成对比表格]
```

### 🐛 **场景4：远程调试**

```
你: "在生产服务器上给运行中的进程附加调试器"
Claude: [连接 → 启动gdb → 加载进程 → 提供回溯]
```

---

## 📖 完整功能列表

### 🔌 **连接管理**
- ✅ 快速连接（主机/用户/密码）
- ✅ SSH密钥认证支持
- ✅ 会话别名系统（不再用UUID！）
- ✅ 预定义主机配置
- ✅ 自动保存主机以便快速复用

### 🖥️ **交互式终端**
- ✅ Raw/Cooked模式切换
- ✅ 非阻塞I/O（不会EOF卡死）
- ✅ 全键盘支持（Ctrl+C/D/Z、方向键）
- ✅ 动态调整终端大小（行/列）
- ✅ 三种ANSI处理模式（Raw/Strip/Parse）
- ✅ 支持vim/top/htop/gdb/tmux

### ⚡ **异步模式**
- ✅ Shell立即返回（~2ms）
- ✅ 后台执行+输出缓冲
- ✅ 10000行循环缓冲区
- ✅ 三层保活（TCP/SSH/应用）
- ✅ 三种读取策略（latest_N/all_unread/latest_bytes）
- ✅ 增强状态显示（缓冲区使用率、保活健康度）

### 🎨 **ANSI处理**
- ✅ ECMA-48标准解析器（charmbracelet/x/ansi）
- ✅ 支持7种ANSI序列类型
- ✅ 零字符丢失
- ✅ 零重复提示符
- ✅ 干净可读的输出

### 📊 **命令执行**
- ✅ 单命令执行
- ✅ 批量命令执行
- ✅ 紧凑模式输出
- ✅ 命令历史追踪
- ✅ 执行时长测量
- ✅ 退出码记录

### 🔐 **安全与便捷**
- ✅ 自动sudo密码注入
- ✅ 环境变量支持
- ✅ 安全凭证处理

### 📁 **当前目录追踪**
- ✅ 自动解析shell提示符
- ✅ 支持Ubuntu/Debian格式
- ✅ 支持RHEL/CentOS格式
- ✅ 支持简单提示符

### 📝 **审计与调试**
- ✅ 详细命令历史
- ✅ 按来源过滤（exec/shell）
- ✅ 成功/失败追踪
- ✅ 执行时间戳

### 📂 **文件操作（SFTP）**
- ✅ 上传文件
- ✅ 下载文件
- ✅ 列出目录
- ✅ 创建目录
- ✅ 删除文件/目录
- ✅ 递归操作

---

## 🔧 技术架构

### ANSI过滤技术

**之前（stripansi库）:**
```go
func filterANSI(s string) string {
    return stripansi.Strip(s)  // ❌ Bug: OSC序列导致字符丢失
}
```

**现在（ECMA-48解析器）:**
```go
func filterANSI(s string) string {
    handler := ansi.Handler{
        Print: func(r rune) {
            if r == '\n' || r == '\t' || r >= 32 {
                text.WriteRune(r)  // ✅ 只收集可打印文本
            }
        },
        HandleCsi: func(cmd ansi.Cmd, params ansi.Params) {},
        HandleOsc: func(cmd int, data []byte) {},
        HandleEsc: func(cmd ansi.Cmd) {},
        // ... 处理全部7种序列类型
    }
    parser.Advance(b)  // ✅ 逐字节解析
    return text.String()
}
```

**优势:**
- ✅ 符合标准（ECMA-48）
- ✅ 处理所有ANSI序列类型
- ✅ 无字符丢失
- ✅ 更易维护
- ✅ 社区验证

### 性能指标

- **编译：** 单个12MB可执行文件
- **启动时间：** < 100ms
- **内存占用：** ~20MB（空闲）、~50MB（活跃shell）
- **异步shell返回：** ~2ms
- **缓冲区容量：** 10000行（~9MB）

---

## 📜 更新日志

### [未发布版本]

**新增 (2025-01-06)**
- 🔧 **ANSI过滤升级**：采用ECMA-48标准parser实现（charmbracelet/x/ansi）
  - ✅ 完全兼容所有ANSI序列类型（CSI/OSC/ESC/DCS/APC/PM/SOS）
  - ✅ 彻底解决OSC序列导致的字符丢失问题（如提示符 `cigar` 变成 `igar`）
  - ✅ 通用性更强，可靠性更高，可维护性更好
  - ✅ 代码更简洁，性能更优
- ✅ **提示符完整性修复**：彻底解决字符丢失和重复提示符问题
- ✅ **单元测试全覆盖**：17个交互式shell测试通过，90秒长连接测试通过
- 🔧 **Windows兼容性**：移除Bubbleterm依赖，使用VT10x模拟器（跨平台）
- 🔧 **配置增强**：MaxSessionsPerHost从10提升到30
- ✅ **生产就绪**：完成gdb/htop/strace等真实调试场景完整测试

**新增 (2025-01-05)**
- 🚀 **异步模式（业界首创）**：Shell启动后立即返回，后台持续运行，输出自动缓冲
- 🎯 **三种读取策略**：latest_N_lines / all_unread / latest_N_bytes
- 💾 **循环缓冲区**：10000行容量，自动覆盖最旧数据，支持实时读取
- ❤️ **三层保活机制**：TCP Keepalive（30s）+ SSH Keepalive（30s）+ 应用层心跳（60s）
- 📊 **增强状态显示**：缓冲区使用率、保活状态、会话健康度一目了然
- ✅ **长连接验证**：90秒测试通过，0次保活失败，连接稳定可靠

**新增 (2025-01-04)**
- ✅ **当前目录追踪**：智能解析shell提示符，自动更新工作目录
- ✅ **ANSI清理增强**：彻底移除carriage return和零宽字符
- ✅ **命令历史过滤**：支持按来源过滤（exec/shell）
- ✅ **批量命令紧凑输出**：简洁模式只显示摘要和失败命令
- ✅ **文件传输路径优化**：明确显示Local/Remote路径

**新增 (2025-01-03)**
- ✨ **交互式终端支持**：业界唯一完整的交互式SSH终端
- ✨ **非阻塞I/O**：解决EOF阻塞问题，支持实时AI交互
- ✨ **终端模式控制**：Raw/Cooked模式智能适配
- ✨ **ANSI处理**：Strip/Parse/Pass-through三种模式
- ✨ **特殊字符输入**：完整支持控制键和方向键
- ✨ **交互式程序检测**：自动识别20+程序类型

---

## 🤝 贡献

欢迎贡献！请随时提交Pull Request。

1. Fork 本仓库
2. 创建你的特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 打开 Pull Request

---

## 📄 许可证

MIT License - 详见 [LICENSE](LICENSE) 文件

---

## 👨‍💻 作者

[cigar](https://github.com/Cigarliu)

---

## ⭐ 给个Star吧

如果这个项目对你有帮助，请给它一个star！⭐

你的支持是我持续改进的动力。
