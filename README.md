# SSH MCP Server

SSH MCP Server 是一个基于 Model Context Protocol (MCP) 的 SSH 服务器实现，支持远程命令执行、文件传输和交互式 Shell。

## 功能特性

- **多实例支持** - 每个项目可使用独立配置，多个项目同时运行互不干扰
- **智能配置发现** - 自动从项目目录、用户目录、系统目录查找配置文件
- **多种认证方式** - 支持密码、私钥、SSH Agent 认证
- **命令执行** - 单条/批量命令执行，支持工作目录和超时控制
- **文件传输** - 完整的 SFTP 功能，支持大文件分块传输
- **交互式 Shell** - PTY 交互式终端，支持窗口大小调整
- **会话管理** - 连接池复用，自动清理空闲会话
- **高并发处理** - 基于协程的高性能并发处理

## 安装

```bash
# 克隆仓库
git clone https://github.com/cigar/sshmcp.git
cd sshmcp

# 编译
go build -o bin/sshmcp ./cmd/server
```

### Claude MCP 配置

```bash
# 添加到 Claude MCP
claude mcp add -s user ssh /path/to/sshmcp/bin/sshmcp

# 验证安装
claude mcp list | grep ssh
```

## 配置

### 配置文件发现机制

SSH MCP Server 按以下优先级查找配置文件：

1. `--config` 标志指定的路径
2. `.mcp.yaml` (当前目录)
3. `.sshmcp.yaml` (当前目录)
4. `~/.sshmcp.yaml` (用户主目录)
5. `/etc/sshmcp/config.yaml` (系统默认)

### 项目级配置示例

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
  max_file_size: 1073741824  # 1GB in bytes
  chunk_size: 4194304        # 4MB in bytes
  transfer_timeout: 5m

logging:
  level: info  # debug, info, warn, error
  format: console  # json, console
  output: stdout
```

## MCP 工具

### 连接管理
| 工具 | 描述 |
|------|------|
| `ssh_connect` | 建立 SSH 连接并创建会话 |
| `ssh_disconnect` | 断开 SSH 会话 |
| `ssh_list_sessions` | 列出所有活跃会话 |

### 命令执行
| 工具 | 描述 |
|------|------|
| `ssh_exec` | 执行单条命令 |
| `ssh_exec_batch` | 批量执行命令 |
| `ssh_shell` | 启动交互式 shell |

### 文件传输
| 工具 | 描述 |
|------|------|
| `sftp_upload` | 上传文件到远程 |
| `sftp_download` | 从远程下载文件 |
| `sftp_list_dir` | 列出远程目录 |
| `sftp_mkdir` | 创建远程目录 |
| `sftp_delete` | 删除远程文件/目录 |

### 交互式会话
| 工具 | 描述 |
|------|------|
| `ssh_write_input` | 向交互式会话写入输入 |
| `ssh_read_output` | 读取会话输出 |
| `ssh_resize_pty` | 调整终端窗口大小 |

## 使用示例

### 连接并执行命令
```
连接到 192.168.68.212，用户名 root，密码 root，执行命令 df -h
```

### 批量执行命令
```
批量执行以下命令：
1. cd /var/log
2. ls -la
3. tail -n 50 syslog
```

### 文件上传
```
上传本地文件 /home/user/app.tar.gz 到远程服务器 /tmp/app.tar.gz
```

### 交互式 Shell
```
启动到 192.168.68.212 的交互式 shell，终端类型 xterm-256color，24行80列
```

## 项目结构

```
sshmcp/
├── cmd/
│   └── server/
│       ├── main.go              # 主程序入口
│       └── main_autoconfig.go   # 智能配置发现
├── pkg/
│   ├── sshmcp/                  # 核心 SSH 功能
│   │   ├── types.go             # 数据结构
│   │   ├── ssh_client.go        # SSH 客户端
│   │   ├── session_manager.go   # 会话管理器
│   │   ├── shell_session.go     # 交互式 Shell
│   │   ├── command_executor.go  # 命令执行器
│   │   └── sftp_client.go       # SFTP 客户端
│   └── mcp/                     # MCP 协议实现
│       ├── server.go            # MCP 服务器
│       └── handlers.go          # 工具处理函数
├── internal/
│   ├── config/                  # 配置管理
│   └── logger/                  # 日志系统
├── config.example.yaml          # 配置示例
├── go.mod
├── go.sum
└── README.md
```

## 开发

```bash
# 运行测试
go test ./...

# 查看覆盖率
go test -cover ./...

# 本地构建
go build -o bin/sshmcp ./cmd/server

# 交叉编译
GOOS=linux GOARCH=amd64 go build -o bin/sshmcp-linux-amd64 ./cmd/server
GOOS=darwin GOARCH=amd64 go build -o bin/sshmcp-darwin-amd64 ./cmd/server
GOOS=windows GOARCH=amd64 go build -o bin/sshmcp-windows-amd64.exe ./cmd/server
```

## 多实例配置

不同的项目可以同时使用 SSH MCP，每个项目拥有独立的配置。

项目 A (`/path/project-a/.mcp.yaml`):
```yaml
server:
  name: "project-a"
session:
  max_sessions: 50
logging:
  level: debug
```

项目 B (`/path/project-b/.mcp.yaml`):
```yaml
server:
  name: "project-b"
session:
  max_sessions: 200
logging:
  level: info
```

## 环境变量支持

配置文件支持环境变量替换：

```yaml
ssh:
  password: "${SSH_PASSWORD}"
```

## 故障排查

### 连接失败
检查网络连接和防火墙设置，确认 SSH 服务端口（默认 22），增加配置中的 `timeout` 值。

### 认证失败
验证用户名和密码/私钥是否正确，检查私钥文件权限（应为 600）。

### MCP 连接失败
检查 `claude mcp list` 确认服务已添加，查看配置文件 YAML 语法是否正确。

## 性能指标

- 二进制大小: 12MB
- 内存占用: ~20MB (空闲)
- 并发连接: 支持 100+ 并发会话
- 文件传输: 支持大文件分块传输 (默认 4MB chunks)

## 安全建议

1. 在生产环境中优先使用私钥认证
2. 限制 SSH 用户的权限，使用最小权限原则
3. 配置合理的会话超时时间
4. 敏感信息使用环境变量，不要硬编码在配置文件中

## 许可证

MIT License

## 作者

[cigar](https://github.com/cigar)
