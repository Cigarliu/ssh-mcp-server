# SSH MCP Server

An SSH server implementation based on the Model Context Protocol (MCP), enabling AI assistants to execute remote commands, transfer files, and manage interactive SSH sessions.

## Features

- **Multi-Instance Support** - Each project can use independent configurations
- **Smart Configuration Discovery** - Automatically finds configuration from project/user/system directories
- **Multiple Authentication Methods** - Password, private key, and SSH agent support
- **Command Execution** - Single and batch command execution with working directory and timeout control
- **File Transfer** - Full SFTP support with large file chunked transfer
- **Interactive Shell** - PTY-based interactive terminal with window size adjustment
- **Session Management** - Connection pooling with automatic cleanup of idle sessions
- **Session Aliases** - Human-readable aliases for easier session reference
- **High Concurrency** - Goroutine-based concurrent processing

## Installation

```bash
git clone https://github.com/Cigarliu/ssh-mcp-server.git
cd ssh-mcp-server
go build -o bin/sshmcp ./cmd/server
```

### Claude MCP Integration

```bash
claude mcp add -s user ssh /path/to/sshmcp/bin/sshmcp
```

Verify installation:

```bash
claude mcp list | grep ssh
```

## Configuration

### Configuration Discovery

SSH MCP Server searches for configuration files in the following order:

1. Path specified via `--config` flag
2. `.mcp.yaml` in current directory
3. `.sshmcp.yaml` in current directory
4. `~/.sshmcp.yaml` (user home directory)
5. `/etc/sshmcp/config.yaml` (system default)

### Configuration Example

Create `.mcp.yaml` in your project root:

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

See `config.example.yaml` for complete configuration options.

### Multi-Instance Configuration

Different projects can use SSH MCP simultaneously with independent configurations:

**Project A** (`.mcp.yaml`):
```yaml
server:
  name: "project-a"
session:
  max_sessions: 50
logging:
  level: debug
```

**Project B** (`.mcp.yaml`):
```yaml
server:
  name: "project-b"
session:
  max_sessions: 200
logging:
  level: info
```

## MCP Tools

### Connection Management

| Tool | Description |
|------|-------------|
| `ssh_connect` | Establish SSH connection with optional alias |
| `ssh_disconnect` | Close SSH session |
| `ssh_list_sessions` | List all active sessions |

**Session Aliases**: Use `ssh_connect` with an `alias` parameter to create a memorable identifier. Subsequent operations can reference sessions by alias instead of UUID.

### Command Execution

| Tool | Description |
|------|-------------|
| `ssh_exec` | Execute single command |
| `ssh_exec_batch` | Execute multiple commands sequentially |
| `ssh_shell` | Start interactive shell session |

### File Transfer

| Tool | Description |
|------|-------------|
| `sftp_upload` | Upload file to remote server |
| `sftp_download` | Download file from remote server |
| `sftp_list_dir` | List remote directory contents |
| `sftp_mkdir` | Create remote directory |
| `sftp_delete` | Delete remote file or directory |

### Interactive Session

| Tool | Description |
|------|-------------|
| `ssh_write_input` | Write input to interactive session |
| `ssh_read_output` | Read output from interactive session |
| `ssh_resize_pty` | Adjust terminal window size |

## Usage Examples

### Connect and Execute Commands

```
Connect to 192.168.68.212, username root, password root, execute ls -la
```

### Batch Command Execution

```
Execute the following commands:
1. cd /var/log
2. ls -la
3. tail -n 50 syslog
```

### File Upload

```
Upload /home/user/app.tar.gz to /tmp/app.tar.gz on remote server
```

### Using Session Aliases

```
Connect to production server with alias "prod"
Check disk space on "prod" server
Upload logs from "prod" server
```

### Interactive Shell

```
Start interactive shell to 192.168.68.212, terminal type xterm-256color, 24 rows, 80 columns
```

## Project Structure

```
sshmcp/
├── cmd/
│   └── server/
│       ├── main.go              # Entry point
│       └── main_autoconfig.go   # Configuration discovery
├── pkg/
│   ├── sshmcp/                  # SSH core functionality
│   │   ├── types.go             # Data structures
│   │   ├── ssh_client.go        # SSH client
│   │   ├── session_manager.go   # Session management
│   │   ├── shell_session.go     # Interactive shell
│   │   ├── command_executor.go  # Command execution
│   │   └── sftp_client.go       # SFTP client
│   └── mcp/                     # MCP protocol implementation
│       ├── server.go            # MCP server
│       ├── handlers.go          # Tool handlers
│       └── schemas.go           # Tool schemas
├── internal/
│   ├── config/                  # Configuration management
│   └── logger/                  # Logging system
├── config.example.yaml          # Configuration example
├── go.mod
├── go.sum
└── README.md
```

## Development

### Running Tests

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run integration tests (requires SSH server)
SSH_HOST=192.168.68.212 SSH_USER=root SSH_PASSWORD=root go test ./pkg/sshmcp -v

# Run unit tests only
go test ./... -short
```

### Building

```bash
# Local build
go build -o bin/sshmcp ./cmd/server

# Cross-compile
GOOS=linux GOARCH=amd64 go build -o bin/sshmcp-linux-amd64 ./cmd/server
GOOS=darwin GOARCH=amd64 go build -o bin/sshmcp-darwin-amd64 ./cmd/server
GOOS=windows GOARCH=amd64 go build -o bin/sshmcp-windows-amd64.exe ./cmd/server
```

### Code Quality

```bash
# Format code
go fmt ./...

# Static analysis
go vet ./...
```

## Environment Variables

Configuration files support environment variable substitution:

```yaml
ssh:
  password: "${SSH_PASSWORD}"
```

Usage:
```bash
export SSH_PASSWORD="yourpassword"
./bin/sshmcp
```

## Troubleshooting

### Connection Failures

- Check network connectivity and firewall settings
- Verify SSH service port (default 22)
- Increase `timeout` value in configuration

### Authentication Failures

- Verify username and password/private key
- Check private key file permissions (should be 600)
- Confirm remote host supports the authentication method

### MCP Connection Issues

- Check `claude mcp list` to verify server registration
- Validate YAML syntax in configuration files
- Verify binary path is correct and executable

## Performance

- **Binary Size**: 12MB
- **Memory Usage**: ~20MB (idle)
- **Max Concurrent Sessions**: 100+
- **File Transfer**: Chunked transfer (default 4MB chunks)

## Security Recommendations

1. Use private key authentication in production environments
2. Follow principle of least privilege for SSH users
3. Configure appropriate session timeouts
4. Enable detailed operation logging
5. Use environment variables for sensitive information

## License

MIT License

## Author

[cigar](https://github.com/Cigarliu)
