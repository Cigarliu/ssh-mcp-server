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
		"host": map[string]any{
			"type":        "string",
			"description": "SSH 服务器地址",
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
		"auth_type": map[string]any{
			"type":        "string",
			"description": "认证类型: password, private_key, ssh_agent",
			"enum":        []string{"password", "private_key", "ssh_agent"},
			"default":     "password",
		},
		"password": map[string]any{
			"type":        "string",
			"description": "密码（auth_type=password 时需要）",
		},
		"private_key": map[string]any{
			"type":        "string",
			"description": "私钥文件路径（auth_type=private_key 时需要）",
		},
		"passphrase": map[string]any{
			"type":        "string",
			"description": "私钥密码（可选）",
		},
		"alias": map[string]any{
			"type":        "string",
			"description": "会话别名，简短易记的标识符，用于代替 session_id 引用会话。建议根据实际使用场景设置，比如：prod, staging, db, nginx, web。连接前请先调用 ssh_list_sessions() 查看已有别名，避免重复。如果发现冲突，请调整（如：prod → prod-2, web → web-01）。设置别名后，后续所有操作都可用 alias 代替 session_id。",
		},
	}, []string{"host", "username"})
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
			"description": "终端行数，默认 24",
			"default":     24,
		},
		"cols": map[string]any{
			"type":        "integer",
			"description": "终端列数，默认 80",
			"default":     80,
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
			"description": "是否覆盖已存在文件，默认 true",
			"default":     true,
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
			"description": "是否覆盖已存在文件，默认 true",
			"default":     true,
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
			"description": "要写入的输入内容",
		},
	}, []string{"session_id", "input"})
}

// sshReadOutputSchema returns the input schema for ssh_read_output
func sshReadOutputSchema() map[string]any {
	return getCommonJSONSchema(map[string]any{
		"session_id": map[string]any{
			"type":        "string",
			"description": "会话 ID 或别名",
		},
		"timeout": map[string]any{
			"type":        "integer",
			"description": "超时时间（秒），默认 1",
			"default":     1,
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
