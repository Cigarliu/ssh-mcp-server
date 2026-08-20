package mcp

// Tool input schemas for all SSH MCP tools

// getCommonJSONSchema creates a common JSON schema structure
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
		"connection_id": map[string]any{"type": "string", "description": "Stable persistent SSH connection ID. To reopen a saved connection, send only this ID with transport=ssh. For a first direct SSH connection, choose this readable ID yourself and include description."},
		"description":   map[string]any{"type": "string", "description": "Required when first registering a direct SSH connection. A concise user-facing explanation used in future connection lists."},
		"hostname":      map[string]any{"type": "string", "description": "Deprecated compatibility name for a saved SSH connection ID. Do not combine with connection_id or direct SSH fields."},
		"host":          map[string]any{"type": "string", "description": "Direct SSH address. First-time direct connections require connection_id, description, username, and exactly one of password or private_key."},
		"port":          map[string]any{"type": "integer", "description": "Direct SSH port; omit for 22. Do not combine with a saved connection ID or serial fields."},
		"username":      map[string]any{"type": "string", "description": "Username for a first direct SSH connection. Do not combine with a saved connection ID or serial fields."},
		"password":      map[string]any{"type": "string", "description": "Password for a first direct SSH connection. Do not send with private_key, a saved connection ID, or serial fields."},
		"private_key":   map[string]any{"type": "string", "description": "Private key path for a first direct SSH connection. Do not send with password, a saved connection ID, or serial fields."},
		"passphrase":    map[string]any{"type": "string", "description": "Passphrase for direct private_key or for a saved host with private_key_path. Do not send with password or serial fields."},
		"sudo_password": map[string]any{"type": "string", "description": "Optional password supplied to sudo commands executed through SSH. Do not send for serial."},
		"alias":         map[string]any{"type": "string", "description": "Deprecated. Persistent connection_id is used as the SSH session alias."},
		"device":        map[string]any{"type": "string", "description": "Serial device path, for example /dev/ttyUSB0. Required when transport is serial; do not send with SSH fields."},
		"baud_rate":     map[string]any{"type": "integer", "description": "Serial baud rate; omit for 115200. Do not send for SSH.", "minimum": 1, "maximum": 4000000},
		"data_bits":     map[string]any{"type": "integer", "description": "Serial data bits; omit for 8. Do not send for SSH.", "enum": []int{5, 6, 7, 8}},
		"parity":        map[string]any{"type": "string", "description": "Serial parity; omit for none. Do not send for SSH.", "enum": []string{"none", "odd", "even", "mark", "space"}},
		"stop_bits":     map[string]any{"type": "string", "description": "Serial stop bits; omit for 1. Do not send for SSH.", "enum": []string{"1", "1.5", "2"}},
	}, []string{"transport"})
}

func connectionOpenOutputSchema() map[string]any {
	return getCommonJSONSchema(map[string]any{
		"connection_id": map[string]any{"type": "string"},
		"transport":     map[string]any{"type": "string"},
		"capabilities":  map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
	}, []string{"connection_id", "transport", "capabilities"})
}

func connectionCloseSchema() map[string]any {
	return getCommonJSONSchema(map[string]any{"connection_id": map[string]any{"type": "string", "description": "SSH connection ID or alias, or a serial connection ID."}}, []string{"connection_id"})
}

func connectionCloseOutputSchema() map[string]any {
	return getCommonJSONSchema(map[string]any{"connection_id": map[string]any{"type": "string"}, "closed": map[string]any{"type": "boolean"}}, []string{"connection_id", "closed"})
}

func connectionListSchema() map[string]any { return getCommonJSONSchema(map[string]any{}, []string{}) }

func connectionListOutputSchema() map[string]any {
	return getCommonJSONSchema(map[string]any{
		"connections":     map[string]any{"type": "array", "items": map[string]any{"type": "object"}},
		"saved_ssh_hosts": map[string]any{"type": "array", "items": map[string]any{"type": "object"}},
		"serial_ports":    map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
	}, []string{"connections", "saved_ssh_hosts", "serial_ports"})
}

func connectionHistorySchema() map[string]any {
	return getCommonJSONSchema(map[string]any{
		"connection_id": map[string]any{"type": "string", "description": "Persistent SSH connection ID from connection_list."},
		"limit":         map[string]any{"type": "integer", "default": 50, "minimum": 1, "maximum": 200},
	}, []string{"connection_id"})
}

func connectionHistoryOutputSchema() map[string]any {
	return getCommonJSONSchema(map[string]any{
		"connection_id": map[string]any{"type": "string"},
		"description":   map[string]any{"type": "string"},
		"history":       map[string]any{"type": "array", "items": map[string]any{"type": "object"}},
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
		"terminal_id":   map[string]any{"type": "string"},
		"connection_id": map[string]any{"type": "string"},
		"transport":     map[string]any{"type": "string"},
		"profile":       map[string]any{"type": "string", "enum": []string{"shell", "tui"}},
		"screen":        map[string]any{"type": "boolean"},
		"start_offset":  map[string]any{"type": "integer"},
		"end_offset":    map[string]any{"type": "integer"},
	}, []string{"terminal_id", "connection_id", "transport", "profile", "screen", "start_offset", "end_offset"})
}

func terminalInteractSchema() map[string]any {
	return getCommonJSONSchema(map[string]any{
		"terminal_id": map[string]any{"type": "string"},
		"input":       map[string]any{"type": "string", "default": "", "description": "Input to send. For read-only replay, omit input and set from_offset."},
		"input_mode":  map[string]any{"type": "string", "enum": []string{"raw", "line", "key"}, "default": "raw", "description": "raw sends bytes unchanged; line appends LF; key accepts enter, tab, esc, ctrl+c, ctrl+d, ctrl+z, up, down, left, or right."},
		"wait":        map[string]any{"type": "string", "enum": []string{"none", "until", "quiet"}, "default": "quiet", "description": "none returns without waiting; until waits for the literal until text; quiet waits after output stops changing."},
		"until":       map[string]any{"type": "string", "description": "Literal text required when wait is until. This is not a regular expression."},
		"quiet_ms":    map[string]any{"type": "integer", "default": 150, "minimum": 1, "maximum": 10000, "description": "Quiet window in milliseconds. Used only when wait is quiet."},
		"timeout_ms":  map[string]any{"type": "integer", "default": 3000, "minimum": 1, "maximum": 60000, "description": "Maximum duration of this interaction call."},
		"max_bytes":   map[string]any{"type": "integer", "default": 16384, "minimum": 1, "maximum": 65536},
		"from_offset": map[string]any{"type": "integer", "minimum": 0, "description": "Replay from a prior result offset without destructive reads."},
	}, []string{"terminal_id"})
}

func terminalInteractOutputSchema() map[string]any {
	return getCommonJSONSchema(map[string]any{
		"state":              map[string]any{"type": "string", "enum": []string{"complete", "matched", "stable", "limit_reached", "timeout", "closed"}, "description": "complete means wait=none; matched means the literal until text was found; stable means the quiet window elapsed."},
		"stop_reason":        map[string]any{"type": "string", "enum": []string{"no_wait", "until_matched", "quiet", "max_bytes", "timeout", "connection_closed"}},
		"data":               map[string]any{"type": "string"},
		"encoding":           map[string]any{"type": "string", "enum": []string{"utf8", "base64"}},
		"matched":            map[string]any{"type": "boolean"},
		"from_offset":        map[string]any{"type": "integer"},
		"next_offset":        map[string]any{"type": "integer"},
		"start_offset":       map[string]any{"type": "integer"},
		"end_offset":         map[string]any{"type": "integer"},
		"bytes_read":         map[string]any{"type": "integer"},
		"bytes_lost":         map[string]any{"type": "integer"},
		"buffered_remaining": map[string]any{"type": "integer"},
		"truncated":          map[string]any{"type": "boolean"},
		"elapsed_ms":         map[string]any{"type": "integer"},
	}, []string{"state", "stop_reason", "data", "encoding", "matched", "from_offset", "next_offset", "start_offset", "end_offset", "bytes_read", "bytes_lost", "buffered_remaining", "truncated", "elapsed_ms"})
}

func terminalViewSchema() map[string]any {
	return getCommonJSONSchema(map[string]any{
		"terminal_id": map[string]any{"type": "string"},
		"max_chars":   map[string]any{"type": "integer", "default": 8000, "minimum": 1, "maximum": 12000},
	}, []string{"terminal_id"})
}

func terminalViewOutputSchema() map[string]any {
	return getCommonJSONSchema(map[string]any{
		"screen":      map[string]any{"type": "string"},
		"screen_hash": map[string]any{"type": "string"},
		"cursor":      map[string]any{"type": "object"},
		"size":        map[string]any{"type": "object"},
		"truncated":   map[string]any{"type": "boolean"},
	}, []string{"screen", "screen_hash", "cursor", "size", "truncated"})
}

func terminalCloseSchema() map[string]any {
	return getCommonJSONSchema(map[string]any{"terminal_id": map[string]any{"type": "string"}}, []string{"terminal_id"})
}

func terminalCloseOutputSchema() map[string]any {
	return getCommonJSONSchema(map[string]any{"terminal_id": map[string]any{"type": "string"}, "closed": map[string]any{"type": "boolean"}}, []string{"terminal_id", "closed"})
}

// sshConnectSchema returns the input schema for ssh_connect
func sshConnectSchema() map[string]any {
	return getCommonJSONSchema(map[string]any{
		"hostname": map[string]any{
			"type":        "string",
			"description": "预定义主机名称（配置文件中 hosts 下定义的名称），比如：prod, staging。如果使用此参数，会自动从配置文件读取 host、port、username、password 等信息，无需重复输入。与 host 参数二选一。连接前请先调用 ssh_list_hosts() 查看可用的预定义主机。",
		},
		"host": map[string]any{
			"type":        "string",
			"description": "SSH 服务器地址（与 hostname 二选一）",
		},
		"port": map[string]any{
			"type":        "integer",
			"description": "SSH 端口，默认 22（使用 hostname 时会从配置读取）",
			"default":     22,
		},
		"username": map[string]any{
			"type":        "string",
			"description": "SSH 用户名（使用 hostname 时会从配置读取）",
		},
		"auth_type": map[string]any{
			"type":        "string",
			"description": "认证类型：password 或 private_key（使用 hostname 时会从配置读取）",
			"enum":        []string{"password", "private_key"},
			"default":     "password",
		},
		"password": map[string]any{
			"type":        "string",
			"description": "密码（auth_type=password 时需要，使用 hostname 时会从配置读取）",
		},
		"private_key": map[string]any{
			"type":        "string",
			"description": "私钥文件路径（auth_type=private_key 时需要，使用 hostname 时会从配置读取）",
		},
		"passphrase": map[string]any{
			"type":        "string",
			"description": "私钥密码（可选）",
		},
		"sudo_password": map[string]any{
			"type":        "string",
			"description": "sudo 密码（可选）。如果提供，执行 sudo 命令时会自动注入此密码，无需手动输入。建议仅在安全环境中使用。",
		},
		"alias": map[string]any{
			"type":        "string",
			"description": "会话别名，简短易记的标识符，用于代替 session_id 引用会话。建议根据实际使用场景设置，比如：prod, staging, db, nginx, web。连接前请先调用 ssh_list_sessions() 查看已有别名，避免重复。如果发现冲突，请调整（如：prod → prod-2, web → web-01）。设置别名后，后续所有操作都可用 alias 代替 session_id。",
		},
	}, []string{})
}

// sshDisconnectSchema returns the input schema for ssh_disconnect
func sshDisconnectSchema() map[string]any {
	return getCommonJSONSchema(map[string]any{
		"session_id": map[string]any{
			"type":        "string",
			"description": "会话 ID 或别名",
		},
	}, []string{"session_id"})
}

// sshListSessionsSchema returns the input schema for ssh_list_sessions
func sshListSessionsSchema() map[string]any {
	return getCommonJSONSchema(map[string]any{}, []string{})
}

// sshExecSchema returns the input schema for ssh_exec
func sshExecSchema() map[string]any {
	return getCommonJSONSchema(map[string]any{
		"connection_id": map[string]any{
			"type":        "string",
			"description": "SSH connection ID or alias returned by connection_open",
		},
		"command": map[string]any{
			"type":        "string",
			"description": "要执行的命令",
		},
		"timeout": map[string]any{
			"type":        "integer",
			"description": "超时时间（秒），默认 30",
			"default":     30,
		},
		"working_dir": map[string]any{
			"type":        "string",
			"description": "工作目录（可选）",
		},
		"max_output_chars": map[string]any{
			"type":        "integer",
			"description": "返回给模型的 stdout 和 stderr 总字符上限，默认 12000",
			"default":     12000,
			"minimum":     1,
			"maximum":     12000,
		},
	}, []string{"connection_id", "command"})
}

// sshExecOutputSchema returns the structured output schema for ssh_exec.
func sshExecOutputSchema() map[string]any {
	return getCommonJSONSchema(map[string]any{
		"exit_code":      map[string]any{"type": "integer"},
		"stdout":         map[string]any{"type": "string"},
		"stderr":         map[string]any{"type": "string"},
		"execution_time": map[string]any{"type": "string"},
		"truncated":      map[string]any{"type": "boolean"},
	}, []string{"exit_code", "stdout", "stderr", "execution_time", "truncated"})
}

// sshExecBatchSchema returns the input schema for ssh_exec_batch
func sshExecBatchSchema() map[string]any {
	return getCommonJSONSchema(map[string]any{
		"session_id": map[string]any{
			"type":        "string",
			"description": "会话 ID 或别名",
		},
		"commands": map[string]any{
			"type":        "array",
			"description": "命令列表",
			"maxItems":    20,
			"items": map[string]any{
				"type": "string",
			},
		},
		"stop_on_error": map[string]any{
			"type":        "boolean",
			"description": "遇到错误是否停止，默认 false",
			"default":     false,
		},
		"timeout": map[string]any{
			"type":        "integer",
			"description": "超时时间（秒），默认 30",
			"default":     30,
		},
		"compact": map[string]any{
			"type":        "boolean",
			"description": "简洁输出模式，只显示摘要和失败的命令，默认 true",
			"default":     true,
		},
		"max_output_chars": map[string]any{
			"type":        "integer",
			"description": "详细输出时返回给模型的最大字符数，默认 12000",
			"default":     12000,
			"minimum":     1,
			"maximum":     12000,
		},
	}, []string{"session_id", "commands"})
}

// sshShellSchema returns the input schema for ssh_shell
func sshShellSchema() map[string]any {
	return getCommonJSONSchema(map[string]any{
		"session_id": map[string]any{
			"type":        "string",
			"description": "会话 ID 或别名",
		},
		"rows": map[string]any{
			"type":        "integer",
			"description": "终端行数，默认 40，最大 100",
			"default":     40,
			"minimum":     1,
			"maximum":     100,
		},
		"cols": map[string]any{
			"type":        "integer",
			"description": "终端列数，默认 160，最大 240",
			"default":     160,
			"minimum":     1,
			"maximum":     240,
		},
		"working_dir": map[string]any{
			"type":        "string",
			"description": "工作目录（可选）。启动 shell 前会自动执行 cd 命令切换到此目录。例如：/home/user/projects",
		},
	}, []string{"session_id"})
}

// sftpUploadSchema returns the input schema for sftp_upload
func sftpUploadSchema() map[string]any {
	return getCommonJSONSchema(map[string]any{
		"session_id": map[string]any{
			"type":        "string",
			"description": "会话 ID 或别名",
		},
		"local_path": map[string]any{
			"type":        "string",
			"description": "本地文件路径",
		},
		"remote_path": map[string]any{
			"type":        "string",
			"description": "远程文件路径",
		},
		"create_dirs": map[string]any{
			"type":        "boolean",
			"description": "是否创建目录，默认 true",
			"default":     true,
		},
		"overwrite": map[string]any{
			"type":        "boolean",
			"description": "是否覆盖已存在文件，默认 false。设置为 true 时会覆盖远程同名文件，请谨慎使用",
			"default":     false,
		},
	}, []string{"session_id", "local_path", "remote_path"})
}

// sftpDownloadSchema returns the input schema for sftp_download
func sftpDownloadSchema() map[string]any {
	return getCommonJSONSchema(map[string]any{
		"session_id": map[string]any{
			"type":        "string",
			"description": "会话 ID 或别名",
		},
		"remote_path": map[string]any{
			"type":        "string",
			"description": "远程文件路径",
		},
		"local_path": map[string]any{
			"type":        "string",
			"description": "本地文件路径",
		},
		"create_dirs": map[string]any{
			"type":        "boolean",
			"description": "是否创建目录，默认 true",
			"default":     true,
		},
		"overwrite": map[string]any{
			"type":        "boolean",
			"description": "是否覆盖已存在文件，默认 false。设置为 true 时会覆盖远程同名文件，请谨慎使用",
			"default":     false,
		},
	}, []string{"session_id", "remote_path", "local_path"})
}

// sftpTransferSchema returns the task-oriented schema used by the files profile.
func sftpTransferSchema() map[string]any {
	return getCommonJSONSchema(map[string]any{
		"connection_id": map[string]any{
			"type":        "string",
			"description": "SSH connection ID or alias returned by connection_open",
		},
		"operation": map[string]any{
			"type":        "string",
			"description": "传输方向",
			"enum":        []string{"upload", "download"},
		},
		"local_path": map[string]any{
			"type":        "string",
			"description": "本地文件路径",
		},
		"remote_path": map[string]any{
			"type":        "string",
			"description": "远程文件路径",
		},
		"create_dirs": map[string]any{
			"type":        "boolean",
			"description": "不存在时创建目标目录，默认 true",
			"default":     true,
		},
		"overwrite": map[string]any{
			"type":        "boolean",
			"description": "覆盖同名文件，默认 false",
			"default":     false,
		},
	}, []string{"connection_id", "operation", "local_path", "remote_path"})
}

// sftpManageSchema returns the task-oriented schema used by the files profile.
func sftpManageSchema() map[string]any {
	return getCommonJSONSchema(map[string]any{
		"connection_id": map[string]any{
			"type":        "string",
			"description": "SSH connection ID or alias returned by connection_open",
		},
		"operation": map[string]any{
			"type":        "string",
			"description": "目录或路径操作",
			"enum":        []string{"list", "mkdir", "delete"},
		},
		"remote_path": map[string]any{
			"type":        "string",
			"description": "远程路径；list 省略时使用 /",
		},
		"recursive": map[string]any{
			"type":        "boolean",
			"description": "递归列出、创建或删除，默认 false；mkdir 省略时默认 true",
			"default":     false,
		},
		"limit": map[string]any{
			"type":        "integer",
			"description": "list 最多返回的目录项，默认 100，最大 200",
			"default":     100,
			"minimum":     1,
			"maximum":     200,
		},
		"mode": map[string]any{
			"type":        "string",
			"description": "mkdir 的八进制权限，默认 0755",
			"default":     "0755",
		},
	}, []string{"connection_id", "operation"})
}

// sftpListDirSchema returns the input schema for sftp_list_dir
func sftpListDirSchema() map[string]any {
	return getCommonJSONSchema(map[string]any{
		"session_id": map[string]any{
			"type":        "string",
			"description": "会话 ID 或别名",
		},
		"remote_path": map[string]any{
			"type":        "string",
			"description": "远程目录路径，默认 /",
			"default":     "/",
		},
		"recursive": map[string]any{
			"type":        "boolean",
			"description": "是否递归列出，默认 false",
			"default":     false,
		},
		"limit": map[string]any{
			"type":        "integer",
			"description": "最多返回的目录项数量，默认 100，最大 200",
			"default":     100,
			"minimum":     1,
			"maximum":     200,
		},
	}, []string{"session_id"})
}

// sftpMkdirSchema returns the input schema for sftp_mkdir
func sftpMkdirSchema() map[string]any {
	return getCommonJSONSchema(map[string]any{
		"session_id": map[string]any{
			"type":        "string",
			"description": "会话 ID 或别名",
		},
		"remote_path": map[string]any{
			"type":        "string",
			"description": "要创建的目录路径",
		},
		"recursive": map[string]any{
			"type":        "boolean",
			"description": "是否递归创建，默认 true",
			"default":     true,
		},
		"mode": map[string]any{
			"type":        "string",
			"description": "目录权限，默认 0755",
			"default":     "0755",
		},
	}, []string{"session_id", "remote_path"})
}

// sftpDeleteSchema returns the input schema for sftp_delete
func sftpDeleteSchema() map[string]any {
	return getCommonJSONSchema(map[string]any{
		"session_id": map[string]any{
			"type":        "string",
			"description": "会话 ID 或别名",
		},
		"remote_path": map[string]any{
			"type":        "string",
			"description": "要删除的文件或目录路径",
		},
		"recursive": map[string]any{
			"type":        "boolean",
			"description": "是否递归删除目录，默认 false",
			"default":     false,
		},
	}, []string{"session_id", "remote_path"})
}

// sshWriteInputSchema returns the input schema for ssh_write_input
func sshWriteInputSchema() map[string]any {
	return getCommonJSONSchema(map[string]any{
		"session_id": map[string]any{
			"type":        "string",
			"description": "会话 ID 或别名",
		},
		"input": map[string]any{
			"type":        "string",
			"description": "要写入的输入内容（命令或文本）。如果要发送特殊控制字符，使用 special_char 参数",
		},
		"special_char": map[string]any{
			"type":        "string",
			"description": "特殊控制字符：ctrl+c（中断）、ctrl+d（EOF）、ctrl+z（挂起）、ctrl+l（清屏）、enter（回车）、tab（制表符）、esc（退出）、up/down/left/right（方向键）。使用特殊字符时不要同时提供 input 参数",
			"enum":        []string{"ctrl+c", "sigint", "ctrl+d", "eof", "ctrl+z", "sigtstp", "ctrl+l", "clear", "enter", "return", "tab", "esc", "up", "down", "left", "right"},
		},
	}, []string{"session_id"})
}

// sshReadOutputSchema returns the input schema for ssh_read_output (异步模式)
func sshReadOutputSchema() map[string]any {
	return getCommonJSONSchema(map[string]any{
		"session_id": map[string]any{
			"type":        "string",
			"description": "会话 ID 或别名",
		},
		"strategy": map[string]any{
			"type": "string",
			"description": `读取策略：
- "latest_lines"：读取最新 N 行（默认，推荐）
- "all_unread"：读取所有未读数据
- "latest_bytes"：读取最新 N 字节

推荐使用 "latest_lines" + limit=20-50 获取最新输出`,
			"enum":    []string{"latest_lines", "all_unread", "latest_bytes"},
			"default": "latest_lines",
		},
		"limit": map[string]any{
			"type": "integer",
			"description": `读取限制（配合 strategy 使用）：
- latest_lines: 读取多少行（默认 20）
- latest_bytes: 读取多少字节（默认 4096）

建议：日常使用 20-50 行，查看大量输出时可增加到 100-200`,
			"default": 20,
			"minimum": 1,
		},
		"max_output_chars": map[string]any{
			"type":        "integer",
			"description": "返回给模型的最大字符数，默认 12000",
			"default":     12000,
			"minimum":     1,
			"maximum":     12000,
		},
	}, []string{"session_id"})
}

// sshResizePtySchema returns the input schema for ssh_resize_pty
func sshResizePtySchema() map[string]any {
	return getCommonJSONSchema(map[string]any{
		"session_id": map[string]any{
			"type":        "string",
			"description": "会话 ID 或别名",
		},
		"rows": map[string]any{
			"type":        "integer",
			"description": "终端行数，最大 100",
			"minimum":     1,
			"maximum":     100,
		},
		"cols": map[string]any{
			"type":        "integer",
			"description": "终端列数，最大 240",
			"minimum":     1,
			"maximum":     240,
		},
	}, []string{"session_id", "rows", "cols"})
}

// sshTerminalSnapshotSchema returns the input schema for ssh_terminal_snapshot
func sshTerminalSnapshotSchema() map[string]any {
	return getCommonJSONSchema(map[string]any{
		"session_id": map[string]any{
			"type":        "string",
			"description": "会话 ID 或别名",
		},
		"with_color": map[string]any{
			"type":        "boolean",
			"description": "是否包含 ANSI 颜色码（默认 false）",
			"default":     false,
		},
		"include_cursor_info": map[string]any{
			"type":        "boolean",
			"description": "是否包含光标位置信息（默认 false）",
			"default":     false,
		},
		"max_output_chars": map[string]any{
			"type":        "integer",
			"description": "返回给模型的最大字符数，默认 12000",
			"default":     12000,
			"minimum":     1,
			"maximum":     12000,
		},
	}, []string{"session_id"})
}

// sshListHostsSchema returns the input schema for ssh_list_hosts
func sshListHostsSchema() map[string]any {
	return getCommonJSONSchema(map[string]any{}, []string{})
}

// sshSaveHostSchema returns the input schema for ssh_save_host
func sshSaveHostSchema() map[string]any {
	return getCommonJSONSchema(map[string]any{
		"name": map[string]any{
			"type":        "string",
			"description": "主机名称，用于标识这个主机配置，比如：prod, staging, db-server, web-server。请先调用 ssh_list_hosts() 查看已有名称，避免重复。如果发现冲突，请调整（如：prod → prod-2）。保存后，可直接使用此名称连接，无需重复输入账号密码。",
		},
		"host": map[string]any{
			"type":        "string",
			"description": "SSH 服务器地址（IP 或域名）",
		},
		"port": map[string]any{
			"type":        "integer",
			"description": "SSH 端口，默认 22",
			"default":     22,
		},
		"username": map[string]any{
			"type":        "string",
			"description": "SSH 用户名",
		},
		"password": map[string]any{
			"type":        "string",
			"description": "密码（与 private_key 二选一）",
		},
		"private_key_path": map[string]any{
			"type":        "string",
			"description": "私钥文件路径（与 password 二选一）",
		},
		"description": map[string]any{
			"type":        "string",
			"description": "主机描述（可选）",
		},
	}, []string{"name", "host", "username"})
}

// sshRemoveHostSchema returns the input schema for ssh_remove_host
func sshRemoveHostSchema() map[string]any {
	return getCommonJSONSchema(map[string]any{
		"name": map[string]any{
			"type":        "string",
			"description": "要删除的主机名称",
		},
	}, []string{"name"})
}

// sshShellStatusSchema returns the input schema for ssh_shell_status
func sshShellStatusSchema() map[string]any {
	return getCommonJSONSchema(map[string]any{
		"session_id": map[string]any{
			"type":        "string",
			"description": "会话 ID 或别名",
		},
	}, []string{"session_id"})
}

// sshHistorySchema returns the input schema for ssh_history
func sshHistorySchema() map[string]any {
	return getCommonJSONSchema(map[string]any{
		"session_id": map[string]any{
			"type":        "string",
			"description": "会话 ID 或别名",
		},
		"limit": map[string]any{
			"type":        "integer",
			"description": "返回的最大历史记录数，默认 10。设置为 0 返回所有历史记录",
			"default":     10,
		},
	}, []string{"session_id"})
}
