# sshmcp

写这个工具是因为我想在 Claude Code 里直接操作远程服务器，不想每次都切终端手动敲 ssh 命令。

它是一个基于 MCP (Model Context Protocol) 的 SSH 服务器，可以让 Claude 直接帮你：
- 在远程机器上跑命令（单条或批量都行）
- 传文件（支持 SFTP）
- 开交互式 shell

## 安装

```bash
git clone https://github.com/Cigarliu/ssh-mcp-server.git
cd ssh-mcp-server
go build -o bin/sshmcp ./cmd/server
```

然后把它加到 Claude MCP：

```bash
claude mcp add -s user ssh /path/to/sshmcp/bin/sshmcp
```

## 配置

找个地方建个配置文件，按这个优先级找：
1. `--config` 指定的路径
2. 当前目录的 `.mcp.yaml` 或 `.sshmcp.yaml`
3. `~/.sshmcp.yaml`
4. `/etc/sshmcp/config.yaml`

配置示例（复制一份改改就行）：

```yaml
server:
  name: "my-project-ssh"

ssh:
  default_port: 22
  timeout: 30s

session:
  max_sessions: 100
  idle_timeout: 10m

sftp:
  max_file_size: 1073741824  # 1GB
  chunk_size: 4194304        # 4MB

logging:
  level: info
```

完整配置示例看 `config.example.yaml`。

## 能干什么

连接管理：`ssh_connect` / `ssh_disconnect` / `ssh_list_sessions`

跑命令：`ssh_exec` / `ssh_exec_batch` / `ssh_shell`

传文件：`sftp_upload` / `sftp_download` / `sftp_list_dir` / `sftp_mkdir` / `sftp_delete`

交互式会话：`ssh_write_input` / `ssh_read_output` / `ssh_resize_pty`

## 怎么用

直接跟 Claude 说就行，比如：

```
连接到 192.168.68.212，用户 root，密码 root，执行 ls -la
```

```
上传 /home/user/app.tar.gz 到远程服务器的 /tmp/
```

```
批量执行：cd /var/log -> ls -la -> tail -n 50 syslog
```

## 项目结构

```
cmd/server/          主程序
pkg/sshmcp/          SSH 核心功能
pkg/mcp/             MCP 协议实现
internal/            内部模块
```

## 多实例

不同项目可以用不同配置，互不干扰。在项目根目录放个 `.mcp.yaml` 就行。

## 开发

```bash
go test ./...
go build -o bin/sshmcp ./cmd/server
```

## License

MIT
