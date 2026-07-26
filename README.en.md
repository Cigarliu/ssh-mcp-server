# SSH MCP Server

[简体中文](README.md)

An MCP server for SSH and serial terminals. It separates connection management, terminal byte streams, and TUI screen projection so AI clients can operate remote commands, interactive programs, and serial consoles through a small, deterministic tool surface.

## Capabilities

- SSH password and private-key authentication, saved host configurations, and connection aliases.
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

The default `core` profile exposes eight tools to the model.

| Profile | Tools | Purpose |
| --- | ---: | --- |
| `core` | 8 | Everyday SSH, serial, and terminal workflows |
| `files` | 10 | `core` plus task-oriented SFTP transfer and management |
| `advanced` | 27 | Legacy granular SSH, SFTP, migration, and diagnostic tools |

Core tools:

| Tool | Purpose |
| --- | --- |
| `connection_list` | List active connections, saved SSH hosts, and locally visible serial ports |
| `connection_open` | Open SSH or serial; SSH uses either a saved host or direct connection fields |
| `connection_close` | Close a connection and its attached terminal |
| `ssh_exec` | Run a non-interactive SSH command |
| `terminal_open` | Create a transport-neutral terminal on an open connection |
| `terminal_interact` | Atomically write input and wait for a literal `until` value or quiet |
| `terminal_view` | Read an SSH TUI screen projection, not ordinary command output |
| `terminal_close` | Close a terminal; serial terminals also release the device |

In the `files` profile, use `sftp_transfer` for upload/download and `sftp_manage` for listing, creating, or deleting paths.

## Recommended Workflows

Use `connection_open -> ssh_exec -> connection_close` for ordinary SSH commands.

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

Saved host names are visible to the model through `connection_list`, but passwords and private keys are never returned. Pass `hostname: "lab"` to `connection_open` to use it. Saving and removing host configurations through MCP remains in the `advanced` profile.

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

## License

See [LICENSE](LICENSE).
