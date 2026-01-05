package mcp

// Tool input schemas for all SSH MCP tools

// getCommonJSONSchema creates a common JSON schema structure
func getCommonJSONSchema(properties map[string]any, required []string) map[string]any {
	return map[string]any{
		"type":     "object",
		"properties": properties,
		"required": required,
	}
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
			"description": "认证类型: password, private_key, ssh_agent（使用 hostname 时会从配置读取）",
			"enum":        []string{"password", "private_key", "ssh_agent"},
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
		"session_id": map[string]any{
			"type":        "string",
			"description": "会话 ID 或别名",
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
	}, []string{"session_id", "command"})
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
			"description": "简洁输出模式，只显示摘要和失败的命令，默认 false",
			"default":     false,
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
		"terminal_type": map[string]any{
			"type":        "string",
			"description": "终端类型，默认 xterm-256color",
			"default":     "xterm-256color",
		},
		"rows": map[string]any{
			"type":        "integer",
			"description": "终端行数，默认 24。建议值：30 行适合 htop，40 行适合 vim/htop 并用，24 行适合大多数命令",
			"default":     24,
		},
		"cols": map[string]any{
			"type":        "integer",
			"description": "终端列数，默认 80。建议值：80 列适合大多数场景，120 列适合查看日志或表格数据",
			"default":     80,
		},
		"mode": map[string]any{
			"type": "string",
			"description": `⚠️ 终端模式选择（重要）：

📌 异步模式说明：ssh_shell 启动后会立即返回，shell 在后台持续运行。输出自动缓冲到内存（10000行），使用 ssh_read_output 读取。

- "cooked"：逐行缓冲，回车后执行。仅用于简单命令（ls/cat/echo/ps/grep）。
  ⚠️ 注意：如果是简单命令，强烈建议直接使用 ssh_exec 而不是 ssh_shell

- "raw"：逐字符传递，无缓冲。必须用于交互式程序（vim/vi/nano/top/htop/gdb/python/node/mysql/psql/less/more/tmux/screen）。
  ⚠️ 警告：运行交互式程序时如果使用 cooked 模式会卡住！

默认值：cooked

决策树：
1. 是 vim/top/htop/gdb 等交互式程序？ → 用 raw
2. 是简单的一次性命令？ → 用 ssh_exec，不要用 ssh_shell
3. 需要连续执行多个命令并保持状态？ → 根据命令类型选择模式`,
			"enum":    []string{"cooked", "raw"},
			"default": "cooked",
		},
		"ansi_mode": map[string]any{
			"type":        "string",
			"description": "ANSI 处理模式：strip（移除 ANSI 序列，输出纯文本，AI 友好，默认）、raw（保留所有控制码）、parse（结构化解析，未来功能）。推荐使用 strip 获得最佳可读性",
			"enum":        []string{"raw", "strip", "parse"},
			"default":     "strip",
		},
		"read_timeout": map[string]any{
			"type":        "integer",
			"description": "读取超时时间（毫秒），默认 100ms。非阻塞模式下建议使用较短的超时以快速响应",
			"default":     100,
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
			"type": "string",
			"description": "会话 ID 或别名",
		},
		"strategy": map[string]any{
			"type": "string",
			"description": `读取策略：
- "latest_lines"：读取最新 N 行（默认，推荐）
- "all_unread"：读取所有未读数据
- "latest_bytes"：读取最新 N 字节

推荐使用 "latest_lines" + limit=20-50 获取最新输出`,
			"enum": []string{"latest_lines", "all_unread", "latest_bytes"},
			"default": "latest_lines",
		},
		"limit": map[string]any{
			"type": "integer",
			"description": `读取限制（配合 strategy 使用）：
- latest_lines: 读取多少行（默认 20）
- latest_bytes: 读取多少字节（默认 4096）

建议：日常使用 20-50 行，查看大量输出时可增加到 100-200`,
			"default": 20,
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
			"description": "终端行数",
		},
		"cols": map[string]any{
			"type":        "integer",
			"description": "终端列数",
		},
	}, []string{"session_id", "rows", "cols"})
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
		"source": map[string]any{
			"type":        "string",
			"description": "过滤命令来源：'exec' (ssh_exec执行的命令), 'shell' (交互式shell中的命令), 或留空显示所有",
			"default":     "",
		},
	}, []string{"session_id"})
}
