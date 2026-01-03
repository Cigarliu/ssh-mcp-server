# SSH MCP Server

基于模型上下文协议（MCP）的 SSH 服务器实现，使 AI 助手能够执行远程命令、传输文件和管理交互式 SSH 会话。

[English Version](README.md) | 中文版

## 特性

- **多实例支持** - 每个项目可以使用独立的配置
- **智能配置发现** - 自动从项目/用户/系统目录查找配置文件
- **多种认证方式** - 支持密码、私钥和 SSH agent 认证
- **命令执行** - 支持单个和批量命令执行，可控制工作目录和超时时间
- **文件传输** - 完整的 SFTP 支持，支持大文件分块传输
- **交互式 Shell** - 基于 PTY 的交互式终端，支持窗口大小调整
  - **非阻塞 I/O** - 实时输出读取，无需等待（解决 EOF 阻塞问题）
  - **终端模式控制** - Raw/Cooked 模式支持，适配不同类型的程序
  - **ANSI 转义序列处理** - 剥离、解析或透传终端控制码
  - **特殊字符输入** - 支持控制键（Ctrl+C、Ctrl+D 等）和方向键
  - **交互式程序检测** - 自动检测 vim、top、gdb 等 20+ 种交互式程序
- **会话管理** - 连接池，自动清理空闲会话
- **会话别名** - 易读的别名，方便引用会话
- **高并发支持** - 基于 Goroutine 的并发处理
- **AI 友好输出** - 优化的纯文本输出，适合 AI/LLM 理解

## 安装

```bash
git clone https://github.com/Cigarliu/ssh-mcp-server.git
cd ssh-mcp-server
go build -o bin/sshmcp ./cmd/server
```

### Claude MCP 集成

```bash
claude mcp add -s user ssh /path/to/sshmcp/bin/sshmcp
```

验证安装：

```bash
claude mcp list | grep ssh
```

## 配置

### 首次运行

首次运行时，如果未找到配置文件，SSH MCP Server 将自动在 `~/.sshmcp/config.yaml` 生成默认配置文件。您可以编辑此文件以自定义设置。

### 配置文件查找顺序

SSH MCP Server 按以下顺序查找配置文件：

1. 通过 `--config` 标志指定的路径
2. 当前目录下的 `.mcp.yaml`
3. 当前目录下的 `.sshmcp.yaml`
4. 用户主目录下的 `~/.sshmcp.yaml`
5. 系统默认配置 `/etc/sshmcp/config.yaml`

### 配置示例

在项目根目录创建 `.mcp.yaml`：

```yaml
server:
  name: "my-project-ssh"
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
  max_file_size: 1073741824  # 1GB（字节数）
  chunk_size: 4194304        # 4MB（字节数）
  transfer_timeout: 5m

logging:
  level: info  # debug, info, warn, error
  format: console  # json, console
  output: stdout
```

完整配置选项请参考 `config.example.yaml`。

### 多实例配置

不同项目可以同时使用 SSH MCP，各自使用独立配置：

**项目 A** (`.mcp.yaml`):
```yaml
server:
  name: "project-a"
session:
  max_sessions: 50
logging:
  level: debug
```

**项目 B** (`.mcp.yaml`):
```yaml
server:
  name: "project-b"
session:
  max_sessions: 200
logging:
  level: info
```

## MCP 工具

### 连接管理

| 工具 | 描述 |
|------|------|
| `ssh_connect` | 建立 SSH 连接，可选设置别名 |
| `ssh_disconnect` | 关闭 SSH 会话 |
| `ssh_list_sessions` | 列出所有活动会话 |

**会话别名**：使用 `ssh_connect` 的 `alias` 参数创建易记的标识符。后续操作可以使用别名引用会话，无需使用 UUID。

### 命令执行

| 工具 | 描述 |
|------|------|
| `ssh_exec` | 执行单个命令 |
| `ssh_exec_batch` | 按顺序执行多个命令 |
| `ssh_shell` | 启动交互式 shell，可配置终端模式（raw/cooked）、ANSI 处理和超时设置 |

### 文件传输

| 工具 | 描述 |
|------|------|
| `sftp_upload` | 上传文件到远程服务器 |
| `sftp_download` | 从远程服务器下载文件 |
| `sftp_list_dir` | 列出远程目录内容 |
| `sftp_mkdir` | 创建远程目录 |
| `sftp_delete` | 删除远程文件或目录 |

### 交互式会话

| 工具 | 描述 |
|------|------|
| `ssh_write_input` | 向交互式会话写入输入或发送特殊字符（Ctrl+C、方向键等） |
| `ssh_read_output` | 从交互式会话读取输出，支持非阻塞模式用于实时 AI 交互 |
| `ssh_resize_pty` | 调整终端窗口大小 |

## 使用示例

### 连接并执行命令

```
连接到 192.168.68.212，用户名 root，密码 root，执行 ls -la
```

### 批量命令执行

```
执行以下命令：
1. cd /var/log
2. ls -la
3. tail -n 50 syslog
```

### 文件上传

```
上传 /home/user/app.tar.gz 到远程服务器的 /tmp/app.tar.gz
```

### 使用会话别名

```
使用别名 "prod" 连接到生产服务器
检查 "prod" 服务器的磁盘空间
从 "prod" 服务器上传日志
```

### 交互式 Shell

**基础交互式 Shell：**

```
启动到 192.168.68.212 的交互式 shell，终端类型 xterm-256color，24 行，80 列
```

**交互式程序（vim、top、gdb 等）：**

Shell 现在支持完整的交互式程序控制，具有非阻塞 I/O 能力：

```
1. 连接到 SSH 服务器
2. 启动交互式 shell，使用 Raw 模式（适用于 vim、top 等程序）
3. 启动 top 命令
4. 按 CPU 使用率排序（按 P 键）、内存（按 M 键）或时间（按 T 键）
5. 使用方向键导航
6. 实时读取输出，无需阻塞等待
7. 发送控制键（Ctrl+C 中断，Ctrl+D 退出）
```

**终端模式：**

- **Cooked 模式**（默认）：行缓冲，适合简单命令（ls、cat、echo）
- **Raw 模式**：字符缓冲，交互式程序必需（vim、top、gdb、htop）

**ANSI 处理模式：**

- **Raw**：透传所有控制码（默认）
- **Strip**：移除 ANSI 序列，输出纯文本（AI 友好）
- **Parse**：结构化 ANSI 解析（未来功能）

**示例：完整控制使用 Top 命令**

交互式终端已通过真实 SSH 连接测试，运行 `top` 命令时支持：
- CPU/内存/时间排序（P/M/T 键）
- 方向键导航
- 切换完整命令行显示（c 键）
- 非阻塞实时输出读取
- 优雅退出（q 键）

完整技术细节请参阅 [docs/interactive-terminal-implementation.md](docs/interactive-terminal-implementation.md)。

## 项目结构

```
sshmcp/
├── cmd/
│   └── server/
│       ├── main.go              # 入口点
│       └── main_autoconfig.go   # 配置发现
├── pkg/
│   ├── sshmcp/                  # SSH 核心功能
│   │   ├── types.go             # 数据结构
│   │   ├── ssh_client.go        # SSH 客户端
│   │   ├── session_manager.go   # 会话管理
│   │   ├── shell_session.go     # 交互式 shell（已增强）
│   │   ├── shell_session_test.go    # Shell 单元测试
│   │   ├── interactive_test.go      # 集成测试
│   │   ├── command_executor.go  # 命令执行
│   │   └── sftp_client.go       # SFTP 客户端
│   └── mcp/                     # MCP 协议实现
│       ├── server.go            # MCP 服务器
│       ├── handlers.go          # 工具处理器
│       └── schemas.go           # 工具架构
├── internal/
│   ├── config/                  # 配置管理
│   └── logger/                  # 日志系统
├── docs/
│   ├── interactive-terminal-research.md       # 技术研究
│   └── interactive-terminal-implementation.md # 实现指南
├── config.example.yaml          # 配置示例
├── go.mod
├── go.sum
├── README.md                    # 英文文档
└── README_CN.md                 # 中文文档
```

## 开发

### 运行测试

```bash
# 运行所有测试
go test ./...

# 运行测试并生成覆盖率报告
go test -cover ./...

# 运行集成测试（需要 SSH 服务器）
SSH_HOST=192.168.68.212 SSH_USER=root SSH_PASSWORD=root go test ./pkg/sshmcp -v

# 仅运行单元测试
go test ./... -short

# 运行特定的交互式终端测试
go test ./pkg/sshmcp -run TestInteractiveShell -v

# 运行 top 命令交互测试
go run cmd/test-top/main.go
```

**测试覆盖率：**
- 单元测试：新交互式功能 100% 覆盖
- 集成测试：真实 SSH 连接测试，包含 top、vim、gdb 程序
- 性能测试：非阻塞读取延迟 ~20ms

### 构建

```bash
# 本地构建
go build -o bin/sshmcp ./cmd/server

# 交叉编译
GOOS=linux GOARCH=amd64 go build -o bin/sshmcp-linux-amd64 ./cmd/server
GOOS=darwin GOARCH=amd64 go build -o bin/sshmcp-darwin-amd64 ./cmd/server
GOOS=windows GOARCH=amd64 go build -o bin/sshmcp-windows-amd64.exe ./cmd/server
```

### 代码质量

```bash
# 格式化代码
go fmt ./...

# 静态分析
go vet ./...
```

## 环境变量

配置文件支持环境变量替换：

```yaml
ssh:
  password: "${SSH_PASSWORD}"
```

使用方法：
```bash
export SSH_PASSWORD="yourpassword"
./bin/sshmcp
```

## 故障排除

### 连接失败

- 检查网络连接和防火墙设置
- 验证 SSH 服务端口（默认 22）
- 增加配置中的 `timeout` 值

### 认证失败

- 验证用户名和密码/私钥
- 检查私钥文件权限（应为 600）
- 确认远程主机支持该认证方法

### MCP 连接问题

- 使用 `claude mcp list` 检查服务器注册状态
- 验证配置文件中的 YAML 语法
- 确认二进制文件路径正确且可执行

## 性能指标

- **二进制大小**：12MB
- **内存使用**：~20MB（空闲时）
- **最大并发会话**：100+
- **文件传输**：分块传输（默认 4MB 块大小）

## 安全建议

1. 生产环境使用私钥认证
2. 遵循最小权限原则配置 SSH 用户
3. 配置适当的会话超时时间
4. 启用详细的操作日志
5. 使用环境变量存储敏感信息

## 许可证

MIT License

## 作者

[cigar](https://github.com/Cigarliu)

## 更新日志

### [未发布版本]

**新增 - MCP 工具描述更新 (2025-01-04)**

- ✨ **增强工具架构**：更新 `ssh_shell`，新增 mode、ansi_mode 和 read_timeout 参数
- ✨ **特殊字符支持**：更新 `ssh_write_input`，新增 special_char 参数用于控制键
- ✨ **非阻塞读取**：更新 `ssh_read_output`，新增 non_blocking 参数实现 AI 友好的轮询
- 📝 **文档更新**：更新 README 工具描述以反映新的交互能力
- ✅ **AI 优化描述**：所有工具描述现在清晰说明交互式终端功能

**优势：**
- AI 助手现在可以通过工具描述发现和使用交互式终端功能
- 明确指导何时使用 Raw vs Cooked 模式
- 非阻塞模式突出显示，用于实时 AI 交互
- 特殊字符输入已文档化，支持完整的交互式程序控制

**新增 - 交互式终端支持 (2025-01-03)**

- ✨ **非阻塞 I/O**：新增 `ReadOutputNonBlocking()` 方法解决 EOF 阻塞问题
- ✨ **终端模式控制**：Raw/Cooked 模式支持，适配不同类型的程序
- ✨ **ANSI 处理**：Strip/Parse/Pass-through 模式处理终端控制码
- ✨ **特殊字符输入**：控制键（Ctrl+C、Ctrl+D、Ctrl+Z、Ctrl+L）和方向键
- ✨ **交互式程序检测**：自动检测 20+ 种交互式程序（vim、top、gdb、htop 等）
- ✨ **AI 友好输出**：优化的纯文本模式，适合 AI/LLM 使用
- ✨ **增强配置**：`ShellConfig` 结构提供细粒度控制
- 📝 **文档**：完整的研究和实现文档
- 🧪 **测试**：完整的单元和集成测试套件，真实 SSH 验证

**改进：**
- 更好地支持基于 ncurses 的程序（top、htop、iotop）
- 实时输出读取，无需阻塞
- 可配置的超时和读取行为
- 向后兼容 - 现有 API 不变

**性能：**
- 非阻塞读取延迟：平均 ~20ms
- 50 次连续读取：总时间约 1 秒
- 适合实时交互式应用

**测试：**
- 所有新功能均通过真实 SSH 连接测试
- Top 命令集成测试，完整交互（排序、导航、退出）
- 新交互式功能 100% 测试覆盖率
