# SSH MCP Server

一个面向 AI 客户端的 SSH 与串口 MCP Server。它把连接管理、终端字节流和 TUI 屏幕投影分层，使模型通过少量稳定工具完成远程命令、交互式终端和串口控制台操作。

## 能力

- SSH 密码和私钥认证，支持本地保存的主机配置与连接别名。
- 非交互命令执行、SFTP 传输和目录操作。
- 统一的 SSH/串口终端：写入与读取原子化，输出有偏移量、大小上限和明确的完成状态。
- SSH TUI 屏幕投影；串口保留原始字节流，适合设备 CLI、REPL 和日志控制台。
- MCP 协议只使用 stdout；所有日志写入 stderr，避免破坏握手。

## 快速开始

要求 Go `1.24.4` 或更高版本。

```bash
git clone https://github.com/Cigarliu/ssh-mcp-server.git
cd ssh-mcp-server
go build -o bin/sshmcp ./cmd/server
cp config.example.yaml .sshmcp.yaml
```

在 MCP 客户端中配置标准输入输出服务：

```json
{
  "mcpServers": {
    "ssh-mcp": {
      "command": "/absolute/path/ssh-mcp-server/bin/sshmcp",
      "args": ["-config", "/absolute/path/ssh-mcp-server/.sshmcp.yaml"]
    }
  }
}
```

服务也会依次查找 `-config`、当前目录的 `.mcp.yaml` / `.sshmcp.yaml` 和 `~/.sshmcp/config.yaml`。找不到配置时会生成默认配置。

## 工具面

默认使用 `core` profile，只向模型暴露 8 个工具。

| Profile | 工具数 | 用途 |
| --- | ---: | --- |
| `core` | 8 | SSH、串口与终端的日常操作 |
| `files` | 10 | `core` 加聚合的 SFTP 上传、下载和目录操作 |
| `advanced` | 27 | 兼容旧的细粒度 SSH/SFTP/诊断工具 |

`core` 工具：

| 工具 | 说明 |
| --- | --- |
| `connection_list` | 列出活跃连接、已保存 SSH 主机和本机可见串口 |
| `connection_open` | 创建 SSH 或串口连接；SSH 可传 `hostname` 使用已保存主机 |
| `connection_close` | 关闭连接及附属终端 |
| `ssh_exec` | 执行非交互 SSH 命令 |
| `terminal_open` | 在已打开连接上创建通用终端 |
| `terminal_interact` | 原子写入并等待输出、提示符、模式或静默窗口 |
| `terminal_view` | 获取 SSH TUI 屏幕投影，不用于普通命令输出 |
| `terminal_close` | 关闭终端；串口终端同时释放设备 |

在 `files` profile 中，使用 `sftp_transfer` 处理上传/下载，使用 `sftp_manage` 处理列表、创建目录和删除。

## 推荐调用路径

普通 SSH 命令使用 `connection_open -> ssh_exec -> connection_close`。

需要保持上下文、使用 REPL 或运行交互程序时：

```text
connection_open -> terminal_open -> terminal_interact -> terminal_close -> connection_close
```

`terminal_interact` 返回结构化状态，而不是依赖任意 sleep：

- `matched`: 找到指定提示符或文本。
- `stable`: 收到输出后达到静默窗口。
- `limit_reached`: 输出达到 `max_bytes`，使用 `next_offset` 继续读取。
- `timeout` 或 `closed`: 根据 `stop_reason` 决定下一步，不要盲目重试。

对于 TUI，创建 `profile: "tui"` 的 SSH 终端，并用 `terminal_view` 读取屏幕。对于串口，使用 `terminal_interact` 读取原始数据；串口没有屏幕投影。

## 配置

最小配置示例：

```yaml
tools:
  profile: core

logging:
  level: info
  format: console
  output: stderr

hosts:
  lab:
    host: "192.168.1.100"
    port: 22
    username: "operator"
    private_key_path: "~/.ssh/id_ed25519"
    description: "Lab host"
```

主机名称会通过 `connection_list` 返回给模型，但不会返回密码或私钥。使用 `connection_open` 的 `hostname: "lab"` 建立连接。需要从 MCP 保存或删除主机配置时，使用 `advanced` profile 下的旧管理工具。

完整字段和默认值见 [config.example.yaml](config.example.yaml)。

## 串口

先使用 `connection_list` 查询本机可见串口，然后使用下列参数建立连接：

```json
{
  "transport": "serial",
  "device": "/dev/ttyUSB0",
  "baud_rate": 115200,
  "data_bits": 8,
  "parity": "none",
  "stop_bits": "1"
}
```

Linux 上运行 MCP 服务的用户必须拥有设备访问权限。通常做法是加入 `dialout` 组并重新登录：

```bash
sudo usermod -aG dialout "$USER"
```

不要让 MCP stdio 服务在启动时请求 sudo 密码；这会占用标准输入并破坏 MCP 握手。

## 开发与验证

```bash
go test ./...
go vet ./...
go test -race ./pkg/terminal ./pkg/serialmcp ./pkg/mcp
```

本机 SSH 与串口集成测试均为显式 opt-in，避免默认测试连接设备或使用凭据。相关环境变量定义在 `pkg/mcp/terminal_integration_test.go`。

## English

SSH MCP Server provides transport-neutral MCP tools for SSH and serial terminals. The default `core` profile exposes eight tools for connection management, non-interactive SSH commands, and deterministic terminal interaction. Use `terminal_interact` for byte-stream output and `terminal_view` only for SSH TUI screens.

Build with `go build -o bin/sshmcp ./cmd/server`, configure it as an stdio MCP server, and keep logging on stderr. See the Chinese sections above and [config.example.yaml](config.example.yaml) for setup, tool profiles, serial permissions, and test commands.

## License

See [LICENSE](LICENSE).
