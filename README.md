# SSH MCP Server

[English](README.en.md)

一个面向 AI 客户端的 SSH 与串口 MCP Server。它把连接管理、终端字节流和 TUI 屏幕投影分层，使模型通过少量稳定工具完成远程命令、交互式终端和串口控制台操作。

## 能力

- 持久 SSH 连接目录：模型首次命名 `connection_id` 并填写描述，后续对话和实例只按 ID 操作，不重新暴露目标地址或用户名。
- SQLite 执行历史：保存 `ssh_exec` 与 `terminal_interact` 的输入、结果和时间，可跨重启查询。
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

默认使用 `files` profile，向模型暴露 11 个工具。

| Profile | 工具数 | 用途 |
| --- | ---: | --- |
| `core` | 9 | SSH、串口、持久历史与终端的日常操作 |
| `files` | 11 | `core` 加聚合的 SFTP 上传、下载和目录操作 |
| `advanced` | 28 | 兼容旧的细粒度 SSH/SFTP/诊断工具 |

默认 `files` 工具：

| 工具 | 说明 |
| --- | --- |
| `connection_list` | 列出活跃连接、持久 SSH 连接 ID/描述和本机可见串口；不会返回已保存目标或用户名 |
| `connection_open` | 首次直连时登记模型命名的 SSH `connection_id` 与描述；后续只按 ID 打开 |
| `connection_close` | 关闭连接及附属终端 |
| `connection_history` | 查询跨实例、跨重启保存的命令和终端交互历史 |
| `ssh_exec` | 执行非交互 SSH 命令 |
| `sftp_transfer` | 上传或下载单个文件 |
| `sftp_manage` | 列出目录、创建目录或删除路径 |
| `terminal_open` | 在已打开连接上创建通用终端 |
| `terminal_interact` | 原子写入并按 `until` 字面量或静默窗口等待输出 |
| `terminal_view` | 获取 SSH TUI 屏幕投影，不用于普通命令输出 |
| `terminal_close` | 关闭终端；串口终端同时释放设备 |

在 `files` profile 中，使用 `sftp_transfer` 处理上传/下载，使用 `sftp_manage` 处理列表、创建目录和删除。

## 推荐调用路径

首次登记 SSH 连接时，模型应选择一个简短、稳定、可读的 ID（小写字母开头，可使用数字、`-` 和 `_`），并附带用户可识别的描述：

```json
{
  "transport": "ssh",
  "connection_id": "prod-web-hk",
  "description": "香港生产 Web 服务器，部署与日志排查",
  "host": "203.0.113.10",
  "username": "deploy",
  "private_key": "~/.ssh/id_ed25519"
}
```

连接成功后，之后的任意对话或 MCP 实例只需调用：

```json
{ "transport": "ssh", "connection_id": "prod-web-hk" }
```

普通 SSH 命令使用 `connection_open -> ssh_exec -> connection_close`。使用 `connection_history(connection_id="prod-web-hk")` 可调出持久执行记录。

需要保持上下文、使用 REPL 或运行交互程序时：

```text
connection_open -> terminal_open -> terminal_interact -> terminal_close -> connection_close
```

`terminal_interact` 返回结构化状态，而不是依赖任意 sleep：

- `matched`: 找到指定的 `until` 字面量文本。
- `stable`: 收到输出后达到静默窗口。
- `limit_reached`: 输出达到 `max_bytes`，使用 `next_offset` 继续读取。
- `timeout` 或 `closed`: 根据 `stop_reason` 决定下一步，不要盲目重试。

通常使用默认的 `wait: "quiet"`。只有已知完整的提示符或分隔文本时才使用 `wait: "until"` 和 `until`；它不是正则表达式。`wait: "none"` 仅用于暂不需要响应的写入。对于 TUI，创建 `profile: "tui"` 的 SSH 终端，并用 `terminal_view` 读取屏幕。对于串口，使用 `profile: "shell"` 和 `terminal_interact` 读取原始数据；串口没有屏幕投影。

## 配置

最小配置示例：

```yaml
tools:
  profile: files

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

默认 SQLite 路径为 `~/.sshmcp/state.db`；可通过配置覆盖：

```yaml
state:
  database_path: "/absolute/path/to/sshmcp-state.db"
```

数据库启用 WAL 和写入等待，适用于同一台机器上的多个 MCP 实例。它保存已登记连接的真实参数及执行历史，因此应放在本机受保护目录，不要放入 OneDrive、Dropbox、网络共享或版本控制。`hosts` 配置仅作为兼容导入源：启动时只导入不存在的 ID，之后 SQLite 是权威来源。

`connection_list` 只返回连接 ID 和描述，不会返回已保存的密码、私钥路径、地址或用户名。旧的 `hostname` 参数和 `advanced` host 管理工具仍保留兼容性，但新调用应使用 `connection_id`。

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

## 发布

推送以 `v` 开头的版本 tag 会自动创建 GitHub Release，并附带 Linux（amd64、arm64）、macOS（amd64、arm64）和 Windows（amd64）二进制包：

```bash
git tag v1.0.0
git push origin v1.0.0
```

每个压缩包包含 `sshmcp`、示例配置、双语 README 和许可证。也可以在 GitHub Actions 的 `Release` 工作流中手动填写 tag 触发发布。

## License

See [LICENSE](LICENSE).
