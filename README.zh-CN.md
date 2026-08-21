# SSH MCP Server

[English](README.md) | **简体中文**

面向 AI 客户端的 stdio MCP Server，用于安全地执行 SSH 命令、传输文件、操作交互式终端和串口控制台。连接档案与执行历史可由同一台机器上的多个 MCP 实例共享；模型后续只按稳定的连接 ID 操作，不会再次取得已保存的目标地址、用户名或认证信息。

## 主要能力

- SSH 命令执行、SFTP 上传下载和目录管理。
- SSH Shell/TUI 与串口终端，支持受控读写、输出偏移和明确完成状态。
- 基于 SQLite 的连接档案与命令历史，跨 MCP 实例和重启保留。
- 固定、面向模型的 11 个日常工具。
- 严格 stdio 协议边界：MCP 数据只写入 stdout，日志只写入 stderr。

## 安装

**推荐：使用 GitHub Releases 的预编译包。** 下载与你的平台匹配的归档，解压后将 `sshmcp`（Windows 为 `sshmcp.exe`）放在稳定目录。每个归档包含示例配置、双语 README 和许可证。

从源码构建需要 Go `1.24.4` 或更高版本：

```bash
git clone https://github.com/Cigarliu/ssh-mcp-server.git
cd ssh-mcp-server
go build -o sshmcp ./cmd/server
```

首次启动时，如果未提供配置文件，服务会自动生成默认配置：

- Windows：`%USERPROFILE%\.sshmcp\config.yaml`
- macOS/Linux：`~/.sshmcp/config.yaml`

也可以使用 `-config <path>` 指定配置位置。

## MCP 客户端配置

将程序作为 stdio MCP Server 注册。以 Windows 为例：

```json
{
  "mcpServers": {
    "ssh-mcp": {
      "command": "C:\\Tools\\sshmcp\\sshmcp.exe",
      "args": ["-config", "C:\\Users\\you\\.sshmcp\\config.yaml"]
    }
  }
}
```

在 macOS 或 Linux 上，将 `command` 和 `-config` 参数替换为对应的绝对路径。不要让服务在启动时请求 sudo 密码；这会占用 stdin 并破坏 MCP 握手。

## 连接与历史

首次直连 SSH 时，模型应根据用途创建一个简短、稳定、可读的 `connection_id`，同时提供用户可识别的 `description`：

```json
{
  "transport": "ssh",
  "connection_id": "prod-web-hk",
  "description": "香港生产 Web 服务，用于部署与日志排查",
  "host": "203.0.113.10",
  "username": "deploy",
  "private_key": "~/.ssh/id_ed25519"
}
```

成功连接后，后续会话和其他 MCP 实例只需：

```json
{ "transport": "ssh", "connection_id": "prod-web-hk" }
```

`connection_list` 仅返回连接 ID、描述、活动状态和本机串口；不会返回地址、用户名、密码或私钥路径。`connection_history` 可查询持久化的 `ssh_exec` 与 `terminal_interact` 记录。

普通命令使用：

```text
connection_open -> ssh_exec -> connection_close
```

交互式 Shell、REPL 或 TUI 使用：

```text
connection_open -> terminal_open -> terminal_interact -> terminal_close -> connection_close
```

`terminal_interact` 默认使用 `wait: "quiet"`。仅在已知完整提示符或分隔符时使用 `wait: "until"`；`until` 是字面量，不是正则表达式。返回 `limit_reached` 时使用 `next_offset` 继续读取。

服务始终暴露同一套 11 个面向模型的工具：连接打开、关闭、列表与历史；一次性 SSH 命令；任务型 SFTP 传输与管理；以及终端打开、交互、查看和关闭。不再提供工具 profile、公开的 `session_id`、alias 或旧 host 管理 API。

## 配置与安全

最小配置：

```yaml
logging:
  level: info
  format: console
  output: stderr
```

默认状态数据库位于 `~/.sshmcp/state.db`；可通过以下配置覆盖：

```yaml
state:
  database_path: "/absolute/path/to/sshmcp-state.db"
```

SQLite 使用 WAL 和写入等待机制，允许同一台机器上的多个 MCP 实例共享状态。数据库包含连接参数和执行历史，必须存放在受保护的本地目录；不要将其放入 OneDrive、Dropbox、网络共享或版本控制。`hosts` 仅用作旧配置的一次性导入源，SQLite 是后续状态的唯一来源。

完整字段见 [config.example.yaml](config.example.yaml)。安全问题和凭据处理规则见 [SECURITY.md](SECURITY.md)。

## 串口

先调用 `connection_list` 获取本机可见串口，再通过 `connection_open` 建立连接：

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

在 Linux 上，运行服务的账户必须具有设备访问权限，通常需要加入 `dialout` 组后重新登录。

## 开发与维护

```bash
go test ./...
go vet ./...
go test -race ./...
```

本地 SSH 与串口集成测试均为显式 opt-in，不会在默认测试中访问设备或读取真实凭据。

维护者推送版本 tag 后，GitHub Actions 会构建 Linux（amd64、arm64）、macOS（amd64、arm64）和 Windows（amd64）归档，并创建对应 Release：

```bash
git tag -a vX.Y.Z -m "Release vX.Y.Z"
git push origin vX.Y.Z
```

发布构建会将 tag 注入 MCP 的服务版本。GitHub Release 页面是各版本的变更说明与下载入口。

## License

See [LICENSE](LICENSE).
