package mcp

// getCommonJSONSchema creates a common JSON schema structure.
func getCommonJSONSchema(properties map[string]any, required []string) map[string]any {
	return map[string]any{
		"type":       "object",
		"properties": properties,
		"required":   required,
	}
}

func connectionOpenSchema() map[string]any {
	return getCommonJSONSchema(map[string]any{
		"transport":     map[string]any{"type": "string", "enum": []string{"ssh", "serial"}, "description": "Choose exactly one transport."},
		"connection_id": map[string]any{"type": "string", "pattern": "^[a-z][a-z0-9][a-z0-9_-]{1,62}$", "minLength": 3, "maxLength": 64, "description": "Stable persistent SSH connection ID. For a first direct SSH connection, choose a readable lowercase ID such as prod-web-hk and include description. To reopen a saved connection, send only this ID with transport=ssh."},
		"description":   map[string]any{"type": "string", "description": "Required when first registering a direct SSH connection. A concise user-facing explanation used in future connection lists."},
		"host":          map[string]any{"type": "string", "description": "Direct SSH address. First-time direct connections require connection_id, description, username, and exactly one of password or private_key."},
		"port":          map[string]any{"type": "integer", "description": "Direct SSH port; omit for 22. Do not combine with a saved connection ID or serial fields.", "minimum": 1, "maximum": 65535},
		"username":      map[string]any{"type": "string", "description": "Username for a first direct SSH connection. Do not combine with a saved connection ID or serial fields."},
		"password":      map[string]any{"type": "string", "description": "Password for a first direct SSH connection. Do not send with private_key, a saved connection ID, or serial fields."},
		"private_key":   map[string]any{"type": "string", "description": "Private key path for a first direct SSH connection. Do not send with password, a saved connection ID, or serial fields."},
		"passphrase":    map[string]any{"type": "string", "description": "Passphrase for direct private_key or for a saved connection with a key. Do not send with password or serial fields."},
		"sudo_password": map[string]any{"type": "string", "description": "Optional password supplied to sudo commands executed through SSH. Do not send for serial."},
		"device":        map[string]any{"type": "string", "description": "Serial device path, for example /dev/ttyUSB0. Required when transport is serial; do not send with SSH fields."},
		"baud_rate":     map[string]any{"type": "integer", "description": "Serial baud rate; omit for 115200. Do not send for SSH.", "minimum": 1, "maximum": 4000000},
		"data_bits":     map[string]any{"type": "integer", "description": "Serial data bits; omit for 8. Do not send for SSH.", "enum": []int{5, 6, 7, 8}},
		"parity":        map[string]any{"type": "string", "description": "Serial parity; omit for none. Do not send for SSH.", "enum": []string{"none", "odd", "even", "mark", "space"}},
		"stop_bits":     map[string]any{"type": "string", "description": "Serial stop bits; omit for 1. Do not send for SSH.", "enum": []string{"1", "1.5", "2"}},
	}, []string{"transport"})
}

func connectionOpenOutputSchema() map[string]any {
	return getCommonJSONSchema(map[string]any{
		"connection_id": map[string]any{"type": "string", "description": "Use this stable ID for subsequent SSH commands, files, and reconnection."},
		"transport":     map[string]any{"type": "string", "enum": []string{"ssh", "serial"}, "description": "Opened transport."},
		"capabilities":  map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "Operations available on this active connection."},
	}, []string{"connection_id", "transport", "capabilities"})
}

func connectionCloseSchema() map[string]any {
	return getCommonJSONSchema(map[string]any{
		"connection_id": map[string]any{"type": "string", "description": "Active SSH or serial connection ID. This closes the active connection, not a saved SSH profile."},
	}, []string{"connection_id"})
}

func connectionCloseOutputSchema() map[string]any {
	return getCommonJSONSchema(map[string]any{
		"connection_id": map[string]any{"type": "string", "description": "Connection that was closed."},
		"closed":        map[string]any{"type": "boolean", "description": "Whether the active connection was closed."},
	}, []string{"connection_id", "closed"})
}

func connectionListSchema() map[string]any { return getCommonJSONSchema(map[string]any{}, []string{}) }

func connectionListOutputSchema() map[string]any {
	return getCommonJSONSchema(map[string]any{
		"connections": map[string]any{"type": "array", "description": "Active connections in this MCP process.", "items": map[string]any{"type": "object", "properties": map[string]any{
			"connection_id": map[string]any{"type": "string"},
			"transport":     map[string]any{"type": "string", "enum": []string{"ssh", "serial"}},
			"description":   map[string]any{"type": "string"},
			"active":        map[string]any{"type": "boolean"},
			"device":        map[string]any{"type": "string"},
			"baud_rate":     map[string]any{"type": "integer"},
			"capabilities":  map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
		}}},
		"saved_ssh_hosts": map[string]any{"type": "array", "description": "Durable SSH IDs that can be reopened in any MCP instance; target details are redacted.", "items": map[string]any{"type": "object", "properties": map[string]any{
			"connection_id": map[string]any{"type": "string"},
			"description":   map[string]any{"type": "string"},
			"transport":     map[string]any{"type": "string", "enum": []string{"ssh"}},
		}, "required": []string{"connection_id", "description", "transport"}}},
		"serial_ports": map[string]any{"type": "array", "description": "Locally discoverable serial device paths; empty when none are available.", "items": map[string]any{"type": "string"}},
		"serial_error": map[string]any{"type": "string", "description": "Optional serial discovery error. SSH connection data remains available."},
	}, []string{"connections", "saved_ssh_hosts", "serial_ports"})
}

func connectionHistorySchema() map[string]any {
	return getCommonJSONSchema(map[string]any{
		"connection_id": map[string]any{"type": "string", "description": "Saved SSH connection ID. The connection does not need to be active."},
		"limit":         map[string]any{"type": "integer", "default": 50, "minimum": 1, "maximum": 200, "description": "Maximum newest-first durable records to return."},
	}, []string{"connection_id"})
}

func connectionHistoryOutputSchema() map[string]any {
	return getCommonJSONSchema(map[string]any{
		"connection_id": map[string]any{"type": "string", "description": "Saved SSH connection ID queried."},
		"description":   map[string]any{"type": "string", "description": "Saved connection description snapshot."},
		"history": map[string]any{"type": "array", "description": "Durable command and terminal records, newest first.", "items": map[string]any{"type": "object", "properties": map[string]any{
			"id":         map[string]any{"type": "integer"},
			"kind":       map[string]any{"type": "string", "enum": []string{"exec", "terminal"}},
			"input":      map[string]any{"type": "string"},
			"output":     map[string]any{"type": "string"},
			"state":      map[string]any{"type": "string"},
			"created_at": map[string]any{"type": "string", "format": "date-time"},
			"exit_code":  map[string]any{"type": "integer"},
		}, "required": []string{"id", "kind", "input", "output", "state", "created_at"}}},
	}, []string{"connection_id", "description", "history"})
}

func terminalOpenSchema() map[string]any {
	return getCommonJSONSchema(map[string]any{
		"connection_id": map[string]any{"type": "string", "description": "An open SSH or serial connection."},
		"profile":       map[string]any{"type": "string", "enum": []string{"shell", "tui"}, "default": "shell", "description": "SSH: shell for text programs or tui for full-screen programs. Serial supports shell only."},
		"working_dir":   map[string]any{"type": "string", "description": "Optional initial directory for a new SSH terminal only."},
		"rows":          map[string]any{"type": "integer", "default": 40, "minimum": 1, "maximum": 100, "description": "Initial SSH terminal rows; ignored by serial."},
		"cols":          map[string]any{"type": "integer", "default": 160, "minimum": 1, "maximum": 240, "description": "Initial SSH terminal columns; ignored by serial."},
	}, []string{"connection_id"})
}

func terminalOpenOutputSchema() map[string]any {
	return getCommonJSONSchema(map[string]any{
		"terminal_id":   map[string]any{"type": "string", "description": "Process-local handle to reuse with terminal_interact, terminal_view, and terminal_close."},
		"connection_id": map[string]any{"type": "string", "description": "Active connection that owns this terminal."},
		"transport":     map[string]any{"type": "string", "enum": []string{"ssh", "serial"}, "description": "Terminal transport."},
		"profile":       map[string]any{"type": "string", "enum": []string{"shell", "tui"}, "description": "Terminal profile actually opened."},
		"screen":        map[string]any{"type": "boolean", "description": "Whether terminal_view can read a projected TUI screen."},
		"start_offset":  map[string]any{"type": "integer", "description": "First readable stream offset."},
		"end_offset":    map[string]any{"type": "integer", "description": "Current stream end offset."},
	}, []string{"terminal_id", "connection_id", "transport", "profile", "screen", "start_offset", "end_offset"})
}

func terminalInteractSchema() map[string]any {
	return getCommonJSONSchema(map[string]any{
		"terminal_id": map[string]any{"type": "string", "description": "Process-local terminal ID returned by terminal_open."},
		"input":       map[string]any{"type": "string", "default": "", "description": "Input to send. For read-only replay, omit input and set from_offset."},
		"input_mode":  map[string]any{"type": "string", "enum": []string{"raw", "line", "key"}, "default": "raw", "description": "raw sends bytes unchanged; line appends LF; key accepts enter, tab, esc, ctrl+c, ctrl+d, ctrl+z, up, down, left, or right."},
		"wait":        map[string]any{"type": "string", "enum": []string{"none", "until", "quiet"}, "default": "quiet", "description": "none returns without waiting; until waits for the literal until text; quiet waits after output stops changing."},
		"until":       map[string]any{"type": "string", "description": "Literal text required when wait is until. This is not a regular expression."},
		"quiet_ms":    map[string]any{"type": "integer", "default": 150, "minimum": 1, "maximum": 10000, "description": "Quiet window in milliseconds. Used only when wait is quiet."},
		"timeout_ms":  map[string]any{"type": "integer", "default": 3000, "minimum": 1, "maximum": 60000, "description": "Maximum duration of this interaction call."},
		"max_bytes":   map[string]any{"type": "integer", "default": 16384, "minimum": 1, "maximum": 65536, "description": "Maximum bytes to return this call. On limit_reached, continue with next_offset rather than resending input."},
		"from_offset": map[string]any{"type": "integer", "minimum": 0, "description": "Replay from a prior result offset without destructive reads."},
	}, []string{"terminal_id"})
}

func terminalInteractOutputSchema() map[string]any {
	return getCommonJSONSchema(map[string]any{
		"state":              map[string]any{"type": "string", "enum": []string{"complete", "matched", "stable", "limit_reached", "timeout", "closed"}, "description": "complete means wait=none; matched means the literal until text was found; stable means the quiet window elapsed."},
		"stop_reason":        map[string]any{"type": "string", "enum": []string{"no_wait", "until_matched", "quiet", "max_bytes", "timeout", "connection_closed"}, "description": "Why the call stopped. Use it with state to choose the next action."},
		"data":               map[string]any{"type": "string", "description": "Captured terminal output in encoding."},
		"encoding":           map[string]any{"type": "string", "enum": []string{"utf8", "base64"}, "description": "Encoding used for data."},
		"matched":            map[string]any{"type": "boolean", "description": "Whether the literal until text was observed."},
		"from_offset":        map[string]any{"type": "integer", "description": "Offset used for this read."},
		"next_offset":        map[string]any{"type": "integer", "description": "Use this to continue after limit_reached or to replay later output."},
		"start_offset":       map[string]any{"type": "integer", "description": "First retained offset in the capture buffer."},
		"end_offset":         map[string]any{"type": "integer", "description": "Current end offset in the capture buffer."},
		"bytes_read":         map[string]any{"type": "integer", "description": "Bytes returned in data."},
		"bytes_lost":         map[string]any{"type": "integer", "description": "Bytes no longer retained before from_offset."},
		"buffered_remaining": map[string]any{"type": "integer", "description": "Unread bytes still retained after this result."},
		"truncated":          map[string]any{"type": "boolean", "description": "Whether max_bytes limited this result."},
		"elapsed_ms":         map[string]any{"type": "integer", "description": "Elapsed interaction time in milliseconds."},
	}, []string{"state", "stop_reason", "data", "encoding", "matched", "from_offset", "next_offset", "start_offset", "end_offset", "bytes_read", "bytes_lost", "buffered_remaining", "truncated", "elapsed_ms"})
}

func terminalViewSchema() map[string]any {
	return getCommonJSONSchema(map[string]any{
		"terminal_id": map[string]any{"type": "string", "description": "TUI terminal ID returned by terminal_open with screen=true."},
		"max_chars":   map[string]any{"type": "integer", "default": 8000, "minimum": 1, "maximum": 12000, "description": "Maximum projected screen characters to return."},
	}, []string{"terminal_id"})
}

func terminalViewOutputSchema() map[string]any {
	return getCommonJSONSchema(map[string]any{
		"screen":      map[string]any{"type": "string", "description": "Current projected screen contents."},
		"screen_hash": map[string]any{"type": "string", "description": "Stable hash of the projected screen."},
		"cursor":      map[string]any{"type": "object", "description": "Projected cursor coordinates."},
		"size":        map[string]any{"type": "object", "description": "Projected screen width and height."},
		"truncated":   map[string]any{"type": "boolean", "description": "Whether screen was shortened to max_chars."},
	}, []string{"screen", "screen_hash", "cursor", "size", "truncated"})
}

func terminalCloseSchema() map[string]any {
	return getCommonJSONSchema(map[string]any{
		"terminal_id": map[string]any{"type": "string", "description": "Process-local terminal ID to close."},
	}, []string{"terminal_id"})
}

func terminalCloseOutputSchema() map[string]any {
	return getCommonJSONSchema(map[string]any{
		"terminal_id": map[string]any{"type": "string", "description": "Terminal that was closed."},
		"closed":      map[string]any{"type": "boolean", "description": "Whether the terminal was closed."},
	}, []string{"terminal_id", "closed"})
}

func sshExecSchema() map[string]any {
	return getCommonJSONSchema(map[string]any{
		"connection_id":    map[string]any{"type": "string", "description": "Active SSH connection ID returned by connection_open. Reopen the saved ID first when it is not active."},
		"command":          map[string]any{"type": "string", "description": "One independent remote command to execute."},
		"timeout":          map[string]any{"type": "integer", "description": "Command timeout in seconds; omit for 30.", "default": 30, "minimum": 1, "maximum": 3600},
		"working_dir":      map[string]any{"type": "string", "description": "Optional working directory for this command only."},
		"max_output_chars": map[string]any{"type": "integer", "description": "Maximum combined stdout and stderr characters returned to the model.", "default": 12000, "minimum": 1, "maximum": 12000},
	}, []string{"connection_id", "command"})
}

func sshExecOutputSchema() map[string]any {
	return getCommonJSONSchema(map[string]any{
		"exit_code":      map[string]any{"type": "integer", "description": "Remote command exit status; zero usually indicates success."},
		"stdout":         map[string]any{"type": "string", "description": "Captured standard output."},
		"stderr":         map[string]any{"type": "string", "description": "Captured standard error."},
		"execution_time": map[string]any{"type": "string", "description": "Elapsed command duration."},
		"truncated":      map[string]any{"type": "boolean", "description": "Whether stdout or stderr was shortened to max_output_chars."},
	}, []string{"exit_code", "stdout", "stderr", "execution_time", "truncated"})
}

func sftpTransferSchema() map[string]any {
	return getCommonJSONSchema(map[string]any{
		"connection_id": map[string]any{"type": "string", "description": "Active SSH connection ID returned by connection_open."},
		"operation":     map[string]any{"type": "string", "enum": []string{"upload", "download"}, "description": "upload copies local_path to remote_path; download copies remote_path to local_path."},
		"local_path":    map[string]any{"type": "string", "description": "Local source path for upload or local destination path for download."},
		"remote_path":   map[string]any{"type": "string", "description": "Remote destination path for upload or remote source path for download."},
		"create_dirs":   map[string]any{"type": "boolean", "description": "Create missing destination directories; defaults to true."},
		"overwrite":     map[string]any{"type": "boolean", "default": false, "description": "Overwrite an existing destination file only with explicit user intent. Defaults to false."},
	}, []string{"connection_id", "operation", "local_path", "remote_path"})
}

func sftpManageSchema() map[string]any {
	return getCommonJSONSchema(map[string]any{
		"connection_id": map[string]any{"type": "string", "description": "Active SSH connection ID returned by connection_open."},
		"operation":     map[string]any{"type": "string", "enum": []string{"list", "mkdir", "delete"}, "description": "list inspects a remote directory; mkdir creates a directory; delete removes a file or directory."},
		"remote_path":   map[string]any{"type": "string", "description": "Remote path. Omit only for list to inspect /. Required for mkdir and delete."},
		"recursive":     map[string]any{"type": "boolean", "default": false, "description": "For list, recurse into subdirectories. For mkdir, defaults to true when omitted. For delete, recurse only with explicit user intent because it removes a subtree."},
		"limit":         map[string]any{"type": "integer", "default": 100, "minimum": 1, "maximum": 200, "description": "Maximum entries returned by list."},
		"mode":          map[string]any{"type": "string", "default": "0755", "description": "Octal directory permissions for mkdir."},
	}, []string{"connection_id", "operation"})
}
