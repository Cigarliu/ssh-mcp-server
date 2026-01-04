# SSH MCP Server

[English](#english) | [简体中文](#简体中文)

---

## 简体中文

基于 Model Context Protocol (MCP) 的 SSH 服务器实现，让 AI 助手能够执行远程命令、传输文件、管理交互式 SSH 会话。

[![GitHub stars](https://img.shields.io/github/stars/Cigarliu/ssh-mcp-server?style=social)](https://github.com/Cigarliu/ssh-mcp-server/stargazers)
[![Go Report Card](https://goreportcard.com/badge/github.com/Cigarliu/ssh-mcp-server)](https://goreportcard.com/report/github.com/Cigarliu/ssh-mcp-server)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

---

## 🚀 快速开始（三步上手）

### 1️⃣ 编译

```bash
git clone https://github.com/Cigarliu/ssh-mcp-server.git
cd ssh-mcp-server
go build -o bin/sshmcp ./cmd/server
```

### 2️⃣ 配置 Claude Desktop

打开 Claude Desktop 配置文件，添加以下内容：

**Windows:** `%APPDATA%\Claude\claude_desktop_config.json`
**macOS:** `~/Library/Application Support/Claude/claude_desktop_config.json`

```json
{
  "mcpServers": {
    "ssh-mcp": {
      "command": "D:\\path\\to\\ssh-mcp-server\\bin\\sshmcp.exe",
      "args": []
    }
  }
}
```

> **注意：** 将 `command` 路径改为你实际编译后的可执行文件路径
> - **Windows:** 使用 `\\` 或 `/` 作为路径分隔符，例如 `"D:/code/ssh-mcp-server/bin/sshmcp.exe"`
> - **macOS/Linux:** 使用绝对路径，例如 `"/Users/yourname/ssh-mcp-server/bin/sshmcp"`

### 3️⃣ 立即使用

重启 Claude Desktop，然后直接对话：

```
连接到 192.168.1.100，用户 root，密码 root，执行 ls -la
```

就这样！Claude 会自动调用 SSH MCP Server 完成操作。

### 📝 更多使用示例

**场景 1：使用会话别名（推荐）**
```
1. 连接生产服务器，别名设为 prod
2. 查看 prod 服务器的磁盘空间
3. 上传部署包到 prod 服务器
```

**场景 2：交互式终端**
```
1. 连接 SSH 服务器
2. 启动交互式 shell（Raw 模式）
3. 运行 top 命令查看系统资源
4. 按 P 键按 CPU 排序，M 键按内存排序
5. 按 q 退出
```

**场景 3：批量操作**
```
依次执行以下命令：
1. cd /var/log
2. ls -la
3. tail -n 50 syslog
```

**场景 4：文件传输**
```
上传本地文件 app.tar.gz 到远程服务器的 /tmp/ 目录
```

---

## ✨ 为什么选择 SSH MCP Server？

市面上已有几个 SSH MCP 实现，但它们大多只提供基础的命令执行功能。SSH MCP Server 从零设计，提供了**其他方案没有的企业级功能**：

### 🔥 核心优势对比

| 功能 | SSH MCP Server | tufantunc/ssh-mcp | classfang/ssh-mcp-server | AiondaDotCom/mcp-ssh |
|------|----------------|-------------------|-------------------------|---------------------|
| **交互式终端** | ✅ 完整支持 | ❌ 不支持 | ❌ 不支持 | ❌ 不支持 |
| **非阻塞I/O** | ✅ 支持 | ❌ 不支持 | ❌ 不支持 | ❌ 不支持 |
| **vim/top/gdb** | ✅ 完美支持 | ❌ 不支持 | ❌ 不支持 | ❌ 不支持 |
| **SFTP 操作** | ✅ 完整支持 | ❌ 仅基础 | ✅ 支持 | ✅ 基础支持 |
| **会话别名** | ✅ 支持 | ❌ 不支持 | ❌ 不支持 | ✅ 通过config |
| **批量命令** | ✅ 支持 | ❌ 不支持 | ❌ 不支持 | ✅ 支持 |
| **命令历史** | ✅ 详细追踪 | ❌ 不支持 | ❌ 不支持 | ❌ 不支持 |
| **目录追踪** | ✅ 自动追踪 | ❌ 不支持 | ❌ 不支持 | ❌ 不支持 |
| **sudo 密码** | ✅ 自动注入 | ✅ 支持 | ❌ 不支持 | ❌ 不支持 |
| **预定义主机** | ✅ 支持 | ❌ 不支持 | ✅ 支持 | ✅ 通过config |
| **紧凑输出** | ✅ 可选 | ❌ 不支持 | ❌ 不支持 | ❌ 不支持 |
| **性能** | Go 编译 | Node.js | Node.js | Node.js + ssh |

### 🎯 独家功能

#### 1. **完整的交互式终端** - 业界唯一
其他 SSH MCP 库只能执行一次性命令，无法运行交互式程序（如 vim、top、htop、gdb）。

SSH MCP Server 提供真正的 PTY（伪终端）支持：
- ✅ **非阻塞 I/O**：实时读取输出，不会因为 EOF 卡死
- ✅ **Raw/Cooked 模式**：智能适配不同类型的程序
- ✅ **终端控制**：支持 Ctrl+C、Ctrl+D、方向键等特殊按键
- ✅ **窗口调整**：动态调整终端大小（rows/cols）
- ✅ **ANSI 处理**：三种模式（Raw/Strip/Parse）处理终端控制码

#### 2. **会话持久化与智能管理**
```bash
# 创建带别名的会话
ssh_connect alias=prod host=192.168.1.100 user=admin

# 后续所有操作都可以使用别名代替 UUID
ssh_exec session_id=prod command="df -h"
sftp_upload session_id=prod local_path=app.tar.gz remote_path=/tmp/
```

其他方案需要记住长长的 UUID，使用体验极差。

#### 3. **命令历史与审计**
每次执行都会记录：
- 命令内容
- 退出码
- 执行时长
- 时间戳
- 成功/失败状态
- 命令来源（exec 或 shell）

```bash
# 查看某个会话的所有命令历史
ssh_history session_id=prod limit=20

# 只看 exec 命令
ssh_history session_id=prod source=exec
```

#### 4. **当前目录自动追踪**
执行 `cd /tmp` 后，shell 状态会自动更新当前目录：
- 支持 Ubuntu/Debian 格式：`user@host:path$`
- 支持 RHEL/CentOS 格式：`[user@host path]#`
- 支持简单格式：`path$`

#### 5. **批量命令的灵活输出**
```bash
# 紧凑模式：只显示摘要
ssh_exec_batch session_id=prod compact=true commands=["df -h", "free -h", "uptime"]
# 输出：
# ✓ Batch execution completed
#   Total: 3 | Success: 3 | Failed: 0

# 详细模式：显示每个命令的输出
ssh_exec_batch session_id=prod compact=false commands=["df -h", "free -h"]
```

#### 6. **sudo 密码自动注入**
```bash
ssh_connect ... sudo_password=your_sudo_pass
ssh_exec session_id=myserver command="sudo systemctl restart nginx"
# 自动注入密码，无需手动输入
```

#### 7. **原生编译，单文件部署**
- 用 Go 语言编写，编译后是单个可执行文件
- 13MB 大小，无 Node.js 依赖
- 跨平台编译（Linux/macOS/Windows）
- 启动速度快，内存占用低

---

## 📦 安装

### 快速安装

```bash
git clone https://github.com/Cigarliu/ssh-mcp-server.git
cd ssh-mcp-server
go build -o bin/sshmcp ./cmd/server
```

### 添加到 Claude

```bash
claude mcp add -s user ssh-mcp /path/to/sshmcp/bin/sshmcp
```

验证安装：

```bash
claude mcp list | grep ssh-mcp
```

---

## 🚀 快速开始

### 基础使用

```
连接到 192.168.1.100，用户名 root，密码 root，执行 ls -la
```

### 使用会话别名

```
1. 连接生产服务器，别名设为 prod
2. 查看 prod 服务器的磁盘空间
3. 上传文件到 prod 服务器
```

### 交互式终端

```
1. 连接 SSH 服务器
2. 启动交互式 shell（Raw 模式）
3. 运行 top 命令
4. 按 P 键按 CPU 排序，M 键按内存排序
5. 使用方向键导航
6. 实时读取输出（非阻塞）
7. 按 Ctrl+C 中断，按 q 退出
```

### 批量命令

```
依次执行以下命令：
1. cd /var/log
2. ls -la
3. tail -n 50 syslog
```

---

## 🛠️ 完整工具列表

### 连接管理
| 工具 | 描述 |
|------|------|
| `ssh_connect` | 建立 SSH 连接，支持别名 |
| `ssh_disconnect` | 关闭 SSH 会话 |
| `ssh_list_sessions` | 列出所有活跃会话 |
| `ssh_list_hosts` | 列出预定义主机配置 |
| `ssh_save_host` | 保存主机配置供快速连接 |
| `ssh_remove_host` | 删除已保存的主机配置 |

### 命令执行
| 工具 | 描述 |
|------|------|
| `ssh_exec` | 执行单个命令 |
| `ssh_exec_batch` | 批量执行命令（支持紧凑输出） |
| `ssh_shell` | 启动交互式 shell（支持 Raw/Cooked 模式） |
| `ssh_history` | 查看命令历史（支持来源过滤） |

### 文件传输
| 工具 | 描述 |
|------|------|
| `sftp_upload` | 上传文件到远程服务器 |
| `sftp_download` | 从远程服务器下载文件 |
| `sftp_list_dir` | 列出远程目录内容 |
| `sftp_mkdir` | 创建远程目录 |
| `sftp_delete` | 删除远程文件或目录 |

### 交互式会话控制
| 工具 | 描述 |
|------|------|
| `ssh_write_input` | 写入输入或发送特殊字符 |
| `ssh_read_output` | 读取输出（支持非阻塞模式） |
| `ssh_shell_status` | 查看 shell 状态（目录、活跃状态等） |
| `ssh_resize_pty` | 调整终端窗口大小 |

---

## 📊 技术亮点

### 交互式终端实现

SSH MCP Server 实现了**业界唯一的**完整交互式终端支持：

**问题背景：**
- 其他 SSH MCP 库只能执行一次性命令
- 无法运行 vim、top、htop、gdb 等交互式程序
- 输出读取会阻塞在 EOF，导致 AI 无法实时响应

**解决方案：**
1. **非阻塞 I/O**：通过 `SetReadDeadline()` 避免永久阻塞
2. **智能模式切换**：Raw 模式用于交互程序，Cooked 模式用于简单命令
3. **特殊字符映射**：完整支持 Ctrl+C、Ctrl+D、方向键等
4. **ANSI 处理**：Strip 模式提供干净的文本输出

**实测性能：**
- 非阻塞读取延迟：~20ms
- 50 次连续读取：~1 秒总时间
- 适合实时交互应用

完整技术细节见 [docs/interactive-terminal-implementation.md](docs/interactive-terminal-implementation.md)

---

## 📖 配置

### 配置文件发现顺序

1. `--config` 指定的路径
2. 当前目录的 `.mcp.yaml`
3. 当前目录的 `.sshmcp.yaml`
4. 用户目录的 `~/.sshmcp.yaml`
5. 系统默认 `/etc/sshmcp/config.yaml`

### 配置示例

创建 `.mcp.yaml`：

```yaml
server:
  name: "my-project"
  version: "1.0.0"

ssh:
  default_port: 22
  timeout: 30s
  keepalive_interval: 30s

session:
  max_sessions: 100
  max_sessions_per_host: 10
  idle_timeout: 10m
  session_timeout: 30m
  cleanup_interval: 1m

sftp:
  max_file_size: 1073741824  # 1GB
  chunk_size: 4194304        # 4MB
  transfer_timeout: 5m

logging:
  level: info
  format: console
```

---

## 🧪 测试

```bash
# 运行所有测试
go test ./...

# 运行集成测试（需要 SSH 服务器）
SSH_HOST=192.168.1.100 SSH_USER=root SSH_PASSWORD=root go test ./pkg/sshmcp -v

# 只运行单元测试
go test ./... -short
```

---

## 💻 开发

```bash
# 本地构建
go build -o bin/sshmcp ./cmd/server

# 交叉编译
GOOS=linux GOARCH=amd64 go build -o bin/sshmcp-linux-amd64 ./cmd/server
GOOS=darwin GOARCH=amd64 go build -o bin/sshmcp-darwin-amd64 ./cmd/server
GOOS=windows GOARCH=amd64 go build -o bin/sshmcp-windows-amd64.exe ./cmd/server
```

---

## 📈 性能指标

| 指标 | 数值 |
|------|------|
| 二进制大小 | 13 MB |
| 内存占用 | ~20 MB（空闲） |
| 最大并发会话 | 100+ |
| 文件传输 | 分块传输（默认 4MB） |
| 非阻塞读取延迟 | ~20 ms |

---

## 🔒 安全建议

1. 生产环境使用密钥认证
2. 遵循最小权限原则
3. 配置合适的会话超时
4. 启用详细的操作日志
5. 使用环境变量存储敏感信息

---

## 📜 更新日志

### [Unreleased]

**新增 (2025-01-04)**
- ✅ **当前目录追踪**：智能解析 shell 提示符，自动更新工作目录
- ✅ **ANSI 清理增强**：彻底移除 carriage return 和零宽字符
- ✅ **命令历史过滤**：支持按来源过滤（exec/shell）
- ✅ **批量命令紧凑输出**：简洁模式只显示摘要和失败命令
- ✅ **文件传输路径优化**：明确显示 Local/Remote 路径

**新增 (2025-01-03)**
- ✨ **交互式终端支持**：业界唯一完整的交互式 SSH 终端
- ✨ **非阻塞 I/O**：解决 EOF 阻塞问题，支持实时 AI 交互
- ✨ **终端模式控制**：Raw/Cooked 模式智能适配
- ✨ **ANSI 处理**：Strip/Parse/Pass-through 三种模式
- ✨ **特殊字符输入**：完整支持控制键和方向键
- ✨ **交互式程序检测**：自动识别 20+ 程序类型

---

## 📄 许可证

MIT License - 详见 [LICENSE](LICENSE) 文件

---

## 👨‍💻 作者

[cigar](https://github.com/Cigarliu)

---

## 🙏 致谢

感谢以下项目：
- [Model Context Protocol](https://modelcontextprotocol.io/)
- [Go SSH 客户端库](https://github.com/golang/crypto)

---

---

## English

An SSH server implementation based on the Model Context Protocol (MCP), enabling AI assistants to execute remote commands, transfer files, and manage interactive SSH sessions.

[![GitHub stars](https://img.shields.io/github/stars/Cigarliu/ssh-mcp-server?style=social)](https://github.com/Cigarliu/ssh-mcp-server/stargazers)
[![Go Report Card](https://goreportcard.com/badge/github.com/Cigarliu/ssh-mcp-server)](https://goreportcard.com/report/github.com/Cigarliu/ssh-mcp-server)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

---

## 🚀 Quick Start (3 Steps)

### 1️⃣ Build

```bash
git clone https://github.com/Cigarliu/ssh-mcp-server.git
cd ssh-mcp-server
go build -o bin/sshmcp ./cmd/server
```

### 2️⃣ Configure Claude Desktop

Open your Claude Desktop configuration file and add:

**Windows:** `%APPDATA%\Claude\claude_desktop_config.json`
**macOS:** `~/Library/Application Support/Claude/claude_desktop_config.json`

```json
{
  "mcpServers": {
    "ssh-mcp": {
      "command": "D:\\path\\to\\ssh-mcp-server\\bin\\sshmcp.exe",
      "args": []
    }
  }
}
```

> **Note:** Change `command` path to your actual compiled binary path
> - **Windows:** Use `\\` or `/` as path separator, e.g. `"D:/code/ssh-mcp-server/bin/sshmcp.exe"`
> - **macOS/Linux:** Use absolute path, e.g. `"/Users/yourname/ssh-mcp-server/bin/sshmcp"`

### 3️⃣ Start Using

Restart Claude Desktop and chat:

```
Connect to 192.168.1.100, username root, password root, execute ls -la
```

That's it! Claude will automatically use SSH MCP Server to complete the operation.

### 📝 More Usage Examples

**Scenario 1: Using Session Aliases (Recommended)**
```
1. Connect to production server with alias "prod"
2. Check disk space on "prod" server
3. Upload deployment package to "prod" server
```

**Scenario 2: Interactive Terminal**
```
1. Connect to SSH server
2. Start interactive shell (Raw Mode)
3. Run top command to view system resources
4. Press P to sort by CPU, M to sort by memory
5. Press q to quit
```

**Scenario 3: Batch Operations**
```
Execute the following commands sequentially:
1. cd /var/log
2. ls -la
3. tail -n 50 syslog
```

**Scenario 4: File Transfer**
```
Upload local file app.tar.gz to /tmp/ directory on remote server
```

---

## ✨ Why SSH MCP Server?

While several SSH MCP implementations exist, most only provide basic command execution. SSH MCP Server is built from scratch with **enterprise-grade features not found elsewhere**:

### 🔥 Core Advantages

| Feature | SSH MCP Server | tufantunc/ssh-mcp | classfang/ssh-mcp-server | AiondaDotCom/mcp-ssh |
|---------|----------------|-------------------|-------------------------|---------------------|
| **Interactive Terminal** | ✅ Full Support | ❌ No | ❌ No | ❌ No |
| **Non-blocking I/O** | ✅ Yes | ❌ No | ❌ No | ❌ No |
| **vim/top/gdb** | ✅ Perfect | ❌ No | ❌ No | ❌ No |
| **SFTP Operations** | ✅ Complete | ❌ Basic | ✅ Yes | ✅ Basic |
| **Session Aliases** | ✅ Yes | ❌ No | ❌ No | ✅ via config |
| **Batch Commands** | ✅ Yes | ❌ No | ❌ No | ✅ Yes |
| **Command History** | ✅ Detailed | ❌ No | ❌ No | ❌ No |
| **Directory Tracking** | ✅ Auto | ❌ No | ❌ No | ❌ No |
| **sudo Password** | ✅ Auto-inject | ✅ Yes | ❌ No | ❌ No |
| **Predefined Hosts** | ✅ Yes | ❌ No | ✅ Yes | ✅ via config |
| **Compact Output** | ✅ Optional | ❌ No | ❌ No | ❌ No |
| **Performance** | Go Compiled | Node.js | Node.js | Node.js + ssh |

### 🎯 Exclusive Features

#### 1. **Full Interactive Terminal** - Industry First
Other SSH MCP libraries can only execute one-shot commands, unable to run interactive programs like vim, top, htop, or gdb.

SSH MCP Server provides true PTY (pseudo-terminal) support:
- ✅ **Non-blocking I/O**: Real-time output reading without EOF blocking
- ✅ **Raw/Cooked Modes**: Smart adaptation for different program types
- ✅ **Terminal Control**: Full support for Ctrl+C, Ctrl+D, arrow keys, etc.
- ✅ **Window Resizing**: Dynamic terminal size adjustment
- ✅ **ANSI Processing**: Three modes (Raw/Strip/Parse) for terminal control codes

#### 2. **Session Persistence & Smart Management**
```bash
# Create session with alias
ssh_connect alias=prod host=192.168.1.100 user=admin

# All subsequent operations use alias instead of UUID
ssh_exec session_id=prod command="df -h"
sftp_upload session_id=prod local_path=app.tar.gz remote_path=/tmp/
```

Other solutions require remembering long UUIDs, providing poor UX.

#### 3. **Command History & Auditing`
Every execution records:
- Command content
- Exit code
- Execution duration
- Timestamp
- Success/failure status
- Command source (exec or shell)

```bash
# View command history for a session
ssh_history session_id=prod limit=20

# Filter by source
ssh_history session_id=prod source=exec
```

#### 4. **Automatic Current Directory Tracking`
After `cd /tmp`, shell status automatically updates current directory:
- Supports Ubuntu/Debian format: `user@host:path$`
- Supports RHEL/CentOS format: `[user@host path]#`
- Supports simple format: `path$`

#### 5. **Flexible Batch Command Output`
```bash
# Compact mode: summary only
ssh_exec_batch session_id=prod compact=true commands=["df -h", "free -h", "uptime"]
# Output:
# ✓ Batch execution completed
#   Total: 3 | Success: 3 | Failed: 0

# Verbose mode: full output for each command
ssh_exec_batch session_id=prod compact=false commands=["df -h", "free -h"]
```

#### 6. **Automatic sudo Password Injection`
```bash
ssh_connect ... sudo_password=your_sudo_pass
ssh_exec session_id=myserver command="sudo systemctl restart nginx"
# Password auto-injected, no manual input needed
```

#### 7. **Native Compilation, Single File Deployment**
- Written in Go, compiles to single executable
- 13MB size, no Node.js dependencies
- Cross-platform compilation (Linux/macOS/Windows)
- Fast startup, low memory usage

---

## 📦 Installation

### Quick Install

```bash
git clone https://github.com/Cigarliu/ssh-mcp-server.git
cd ssh-mcp-server
go build -o bin/sshmcp ./cmd/server
```

### Add to Claude

```bash
claude mcp add -s user ssh-mcp /path/to/sshmcp/bin/sshmcp
```

Verify installation:

```bash
claude mcp list | grep ssh-mcp
```

---

## 🚀 Quick Start

### Basic Usage

```
Connect to 192.168.1.100, username root, password root, execute ls -la
```

### Using Session Aliases

```
1. Connect to production server with alias "prod"
2. Check disk space on "prod" server
3. Upload file to "prod" server
```

### Interactive Terminal

```
1. Connect to SSH server
2. Start interactive shell (Raw Mode)
3. Run top command
4. Sort by CPU (press P), memory (press M), or time (press T)
5. Navigate with arrow keys
6. Read real-time output (non-blocking)
7. Press Ctrl+C to interrupt, q to quit
```

### Batch Commands

```
Execute the following commands sequentially:
1. cd /var/log
2. ls -la
3. tail -n 50 syslog
```

---

## 🛠️ Complete Tool List

### Connection Management
| Tool | Description |
|------|-------------|
| `ssh_connect` | Establish SSH connection with alias support |
| `ssh_disconnect` | Close SSH session |
| `ssh_list_sessions` | List all active sessions |
| `ssh_list_hosts` | List predefined host configurations |
| `ssh_save_host` | Save host configuration for quick connection |
| `ssh_remove_host` | Remove saved host configuration |

### Command Execution
| Tool | Description |
|------|-------------|
| `ssh_exec` | Execute single command |
| `ssh_exec_batch` | Execute batch commands (compact mode supported) |
| `ssh_shell` | Start interactive shell (Raw/Cooked modes) |
| `ssh_history` | View command history (source filtering) |

### File Transfer
| Tool | Description |
|------|-------------|
| `sftp_upload` | Upload file to remote server |
| `sftp_download` | Download file from remote server |
| `sftp_list_dir` | List remote directory contents |
| `sftp_mkdir` | Create remote directory |
| `sftp_delete` | Delete remote file or directory |

### Interactive Session Control
| Tool | Description |
|------|-------------|
| `ssh_write_input` | Write input or send special characters |
| `ssh_read_output` | Read output (non-blocking mode supported) |
| `ssh_shell_status` | View shell status (directory, activity, etc.) |
| `ssh_resize_pty` | Adjust terminal window size |

---

## 📊 Technical Highlights

### Interactive Terminal Implementation

SSH MCP Server implements the **industry's only** complete interactive terminal support:

**Background:**
- Other SSH MCP libraries can only execute one-shot commands
- Cannot run interactive programs like vim, top, htop, gdb
- Output reading blocks on EOF, preventing real-time AI response

**Solution:**
1. **Non-blocking I/O**: Avoid permanent blocking via `SetReadDeadline()`
2. **Smart Mode Switching**: Raw mode for interactive programs, Cooked mode for simple commands
3. **Special Character Mapping**: Full support for Ctrl+C, Ctrl+D, arrow keys
4. **ANSI Processing**: Strip mode provides clean text output

**Measured Performance:**
- Non-blocking read latency: ~20ms
- 50 consecutive reads: ~1 second total time
- Suitable for real-time interactive applications

See [docs/interactive-terminal-implementation.md](docs/interactive-terminal-implementation.md) for complete technical details.

---

## 📖 Configuration

### Configuration Discovery Order

1. Path specified by `--config` flag
2. `.mcp.yaml` in current directory
3. `.sshmcp.yaml` in current directory
4. `~/.sshmcp.yaml` in user home directory
5. `/etc/sshmcp/config.yaml` (system default)

### Configuration Example

Create `.mcp.yaml`:

```yaml
server:
  name: "my-project"
  version: "1.0.0"

ssh:
  default_port: 22
  timeout: 30s
  keepalive_interval: 30s

session:
  max_sessions: 100
  max_sessions_per_host: 10
  idle_timeout: 10m
  session_timeout: 30m
  cleanup_interval: 1m

sftp:
  max_file_size: 1073741824  # 1GB
  chunk_size: 4194304        # 4MB
  transfer_timeout: 5m

logging:
  level: info
  format: console
```

---

## 🧪 Testing

```bash
# Run all tests
go test ./...

# Run integration tests (requires SSH server)
SSH_HOST=192.168.1.100 SSH_USER=root SSH_PASSWORD=root go test ./pkg/sshmcp -v

# Run unit tests only
go test ./... -short
```

---

## 💻 Development

```bash
# Local build
go build -o bin/sshmcp ./cmd/server

# Cross-compile
GOOS=linux GOARCH=amd64 go build -o bin/sshmcp-linux-amd64 ./cmd/server
GOOS=darwin GOARCH=amd64 go build -o bin/sshmcp-darwin-amd64 ./cmd/server
GOOS=windows GOARCH=amd64 go build -o bin/sshmcp-windows-amd64.exe ./cmd/server
```

---

## 📈 Performance

| Metric | Value |
|--------|-------|
| Binary Size | 13 MB |
| Memory Usage | ~20 MB (idle) |
| Max Concurrent Sessions | 100+ |
| File Transfer | Chunked (default 4MB) |
| Non-blocking Read Latency | ~20 ms |

---

## 🔒 Security Recommendations

1. Use key authentication in production
2. Follow principle of least privilege
3. Configure appropriate session timeouts
4. Enable detailed operation logging
5. Use environment variables for sensitive data

---

## 📜 Changelog

### [Unreleased]

**Added (2025-01-04)**
- ✅ **Current Directory Tracking**: Smart shell prompt parsing for auto-updating working directory
- ✅ **Enhanced ANSI Cleaning**: Complete removal of carriage returns and zero-width characters
- ✅ **Command History Filtering**: Source-based filtering (exec/shell)
- ✅ **Compact Batch Output**: Concise mode shows summary and failed commands only
- ✅ **File Transfer Path Optimization**: Clear Local/Remote path display

**Added (2025-01-03)**
- ✨ **Interactive Terminal Support**: Industry's only complete interactive SSH terminal
- ✨ **Non-blocking I/O**: EOF blocking resolved, real-time AI interaction enabled
- ✨ **Terminal Mode Control**: Raw/Cooked smart adaptation
- ✨ **ANSI Processing**: Strip/Parse/Pass-through modes
- ✨ **Special Character Input**: Full control key and arrow key support
- ✨ **Interactive Program Detection**: Auto-recognize 20+ program types

---

## 📄 License

MIT License - see [LICENSE](LICENSE) file

---

## 👨‍💻 Author

[cigar](https://github.com/Cigarliu)

---

## 🙏 Acknowledgments

Thanks to:
- [Model Context Protocol](https://modelcontextprotocol.io/)
- [Go SSH Client Library](https://github.com/golang/crypto)
