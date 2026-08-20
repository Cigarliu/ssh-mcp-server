# SSH MCP Server

[简体中文](README.md)

An MCP server for SSH and serial terminals. It separates connection management, terminal byte streams, and TUI screen projection so AI clients can operate remote commands, interactive programs, and serial consoles through a small, deterministic tool surface.

## Capabilities

- Persistent SSH connection IDs: name a connection and description once, then later conversations and instances operate by ID without re-exposing the target or username.
- SQLite execution history for `ssh_exec` and `terminal_interact`, queryable across restarts and MCP instances.
- Non-interactive command execution plus SFTP transfer and directory operations.
- A shared SSH/serial terminal abstraction with atomic write/read turns, stream offsets, bounded output, and explicit completion states.
- SSH TUI screen projection. Serial connections expose raw byte streams for device CLIs, REPLs, and logs.
- MCP protocol data is written only to stdout; logs stay on stderr so they cannot corrupt the handshake.

## Quick Start

Go `1.24.4` or newer is required.

```bash
git clone https://github.com/Cigarliu/ssh-mcp-server.git
cd ssh-mcp-server
go build -o bin/sshmcp ./cmd/server
cp config.example.yaml .sshmcp.yaml
```

Configure the binary as an stdio MCP server:

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

The server searches `-config`, `.mcp.yaml`, `.sshmcp.yaml`, and `~/.sshmcp/config.yaml` in that order. It generates a default configuration when none exists.

## Tool Surface

The default `files` profile exposes eleven tools to the model.

| Profile | Tools | Purpose |
| --- | ---: | --- |
| `core` | 9 | Everyday SSH, serial, persistent-history, and terminal workflows |
| `files` | 11 | `core` plus task-oriented SFTP transfer and management |
| `advanced` | 28 | Legacy granular SSH, SFTP, migration, and diagnostic tools |

Default `files` tools:

| Tool | Purpose |
| --- | --- |
| `connection_list` | List active connections, persistent SSH IDs/descriptions, and local serial ports without returning saved targets or usernames |
| `connection_open` | Register a model-named SSH `connection_id` and description on first direct use; later open it by ID only |
| `connection_close` | Close a connection and its attached terminal |
| `connection_history` | Read durable command and terminal interaction history across instances and restarts |
| `ssh_exec` | Run a non-interactive SSH command |
| `sftp_transfer` | Upload or download one file |
| `sftp_manage` | List directories, create directories, or delete paths |
| `terminal_open` | Create a transport-neutral terminal on an open connection |
| `terminal_interact` | Atomically write input and wait for a literal `until` value or quiet |
| `terminal_view` | Read an SSH TUI screen projection, not ordinary command output |
| `terminal_close` | Close a terminal; serial terminals also release the device |

In the `files` profile, use `sftp_transfer` for upload/download and `sftp_manage` for listing, creating, or deleting paths.

## Recommended Workflows

When first registering an SSH connection, the model must choose a short, stable, readable ID (lowercase letter first; digits, `-`, and `_` are allowed) and give it a user-facing description:

```json
{
  "transport": "ssh",
  "connection_id": "prod-web-hk",
  "description": "Hong Kong production web server for deploys and log triage",
  "host": "203.0.113.10",
  "username": "deploy",
  "private_key": "~/.ssh/id_ed25519"
}
```

After a successful connection, any later conversation or MCP instance opens it with only:

```json
{ "transport": "ssh", "connection_id": "prod-web-hk" }
```

Use `connection_open -> ssh_exec -> connection_close` for ordinary SSH commands. Query durable records with `connection_history(connection_id="prod-web-hk")`.

For stateful shells, REPLs, or interactive programs:

```text
connection_open -> terminal_open -> terminal_interact -> terminal_close -> connection_close
```

`terminal_interact` returns a structured state rather than depending on arbitrary sleeps:

- `matched`: the literal `until` value was found.
- `stable`: output arrived and then reached the requested quiet period.
- `limit_reached`: output reached `max_bytes`; continue with `next_offset`.
- `timeout` or `closed`: inspect `stop_reason` before deciding whether to continue.

Use the default `wait: "quiet"` for normal commands. Use `wait: "until"` with `until` only for a known complete prompt or delimiter; it is not a regular expression. Use `wait: "none"` only when a write does not need a response yet. For a TUI, open an SSH terminal with `profile: "tui"` and call `terminal_view`. Serial terminals use `profile: "shell"` and `terminal_interact` for raw data and have no screen projection.

## Configuration

Minimal configuration:

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

The default SQLite path is `~/.sshmcp/state.db`; override it when needed:

```yaml
state:
  database_path: "/absolute/path/to/sshmcp-state.db"
```

The database uses WAL and a write wait, so multiple MCP instances on the same machine can share it. It stores registered connection parameters and execution history; keep it in a protected local directory and never place it in OneDrive, Dropbox, a network share, or source control. `hosts` remains a compatibility import source: only missing IDs are imported at startup, and SQLite owns later changes.

`connection_list` returns only connection IDs and descriptions, never saved passwords, private-key paths, addresses, or usernames. The legacy `hostname` argument and `advanced` host-management tools remain for compatibility, but new calls should use `connection_id`.

See [config.example.yaml](config.example.yaml) for all fields and defaults.

## Serial

Call `connection_list` to discover serial ports, then open one with parameters such as:

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

On Linux, the account running the MCP server must have device access. A common setup is to add it to `dialout` and sign in again:

```bash
sudo usermod -aG dialout "$USER"
```

Do not make an MCP stdio service request a sudo password at startup: it consumes stdin and breaks the MCP handshake.

## Development and Verification

```bash
go test ./...
go vet ./...
go test -race ./pkg/terminal ./pkg/serialmcp ./pkg/mcp
```

Local SSH and serial integration tests are explicitly opt-in so the default suite never connects to a device or needs credentials. Their environment variables are defined in `pkg/mcp/terminal_integration_test.go`.

## Releases

Pushing a version tag beginning with `v` automatically creates a GitHub Release with Linux (amd64, arm64), macOS (amd64, arm64), and Windows (amd64) binary archives:

```bash
git tag v1.0.0
git push origin v1.0.0
```

Each archive contains `sshmcp`, the example configuration, both READMEs, and the license. The `Release` workflow can also be started manually from GitHub Actions by entering a tag.

## License

See [LICENSE](LICENSE).
