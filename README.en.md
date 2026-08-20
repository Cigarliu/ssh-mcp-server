# SSH MCP Server

[简体中文](README.md)

A stdio MCP server for AI clients that need SSH command execution, file transfer, interactive terminals, and serial consoles. Connection profiles and execution history are shared by MCP instances on the same machine. After a profile is registered, models operate through a stable connection ID instead of receiving saved targets, usernames, or authentication details again.

## Highlights

- SSH command execution, SFTP transfer, and directory management.
- SSH shell/TUI and serial terminals with controlled read/write turns, stream offsets, and explicit completion states.
- SQLite-backed connection profiles and execution history that survive restarts and are shared across local MCP instances.
- A small default surface: the `files` profile exposes eleven everyday tools.
- Strict stdio boundaries: MCP data uses stdout only; logs use stderr only.

## Install

**Recommended: use a prebuilt archive from GitHub Releases.** Download the archive for your platform, extract it, and place `sshmcp` (`sshmcp.exe` on Windows) in a stable directory. Every archive contains the example configuration, both READMEs, and the license.

Building from source requires Go `1.24.4` or newer:

```bash
git clone https://github.com/Cigarliu/ssh-mcp-server.git
cd ssh-mcp-server
go build -o sshmcp ./cmd/server
```

On first start, the server writes a default configuration when none exists:

- Windows: `%USERPROFILE%\.sshmcp\config.yaml`
- macOS/Linux: `~/.sshmcp/config.yaml`

Pass `-config <path>` to use a different file.

## MCP Client Configuration

Register the binary as a stdio MCP server. For example, on Windows:

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

On macOS or Linux, use the corresponding absolute paths. Do not make a stdio MCP server request a sudo password at startup: it consumes stdin and breaks the MCP handshake.

## Connections and History

For a first direct SSH connection, the model should create a short, stable, readable `connection_id` and a user-facing `description`:

```json
{
  "transport": "ssh",
  "connection_id": "prod-web-hk",
  "description": "Hong Kong production web server for deployments and log triage",
  "host": "203.0.113.10",
  "username": "deploy",
  "private_key": "~/.ssh/id_ed25519"
}
```

After a successful connection, later conversations and MCP instances need only:

```json
{ "transport": "ssh", "connection_id": "prod-web-hk" }
```

`connection_list` returns only IDs, descriptions, active state, and locally visible serial ports. It never returns saved addresses, usernames, passwords, or private-key paths. `connection_history` reads durable `ssh_exec` and `terminal_interact` records.

For ordinary commands:

```text
connection_open -> ssh_exec -> connection_close
```

For interactive shells, REPLs, and TUIs:

```text
connection_open -> terminal_open -> terminal_interact -> terminal_close -> connection_close
```

`terminal_interact` defaults to `wait: "quiet"`. Use `wait: "until"` only for a known complete prompt or delimiter; `until` is a literal, not a regular expression. When the result is `limit_reached`, continue with `next_offset`.

## Tool Profiles

| Profile | Tools | Use case |
| --- | ---: | --- |
| `core` | 9 | SSH, serial, history, and terminal operations |
| `files` | 11 | Default; adds task-oriented SFTP operations to `core` |
| `advanced` | 28 | Legacy granular tools, compatibility migrations, and diagnostics |

New integrations should use the `files` profile and `connection_id`-based tools. Legacy host-management tools and the old `hostname` argument remain in `advanced` only for compatibility.

## Configuration and Security

Minimal configuration:

```yaml
tools:
  profile: files

logging:
  level: info
  format: console
  output: stderr
```

The default state database is `~/.sshmcp/state.db`. Override it when needed:

```yaml
state:
  database_path: "/absolute/path/to/sshmcp-state.db"
```

SQLite uses WAL and a write wait so several MCP instances on the same machine can share state. The database contains connection parameters and execution history. Keep it in a protected local directory; never place it in OneDrive, Dropbox, a network share, or source control. `hosts` is only a one-time compatibility import source; SQLite owns subsequent state.

See [config.example.yaml](config.example.yaml) for all fields. See [SECURITY.md](SECURITY.md) for security reporting and credential handling.

## Serial

Use `connection_list` to find locally visible serial ports, then open one with `connection_open`:

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

On Linux, the account running the server needs device access, commonly by joining the `dialout` group and signing in again.

## Development and Maintenance

```bash
go test ./...
go vet ./...
go test -race ./...
```

Local SSH and serial integration tests are explicitly opt-in. Default tests do not connect to devices or read real credentials.

When a maintainer pushes a version tag, GitHub Actions builds Linux (amd64, arm64), macOS (amd64, arm64), and Windows (amd64) archives and creates the matching GitHub Release:

```bash
git tag -a vX.Y.Z -m "Release vX.Y.Z"
git push origin vX.Y.Z
```

Release builds inject the tag into the MCP server version. The GitHub Release page is the download and release-notes entry point for each version.

## License

See [LICENSE](LICENSE).
