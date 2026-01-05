package mcp

import (
	"context"
	"fmt"

	"github.com/cigar/sshmcp/pkg/sshmcp"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rs/zerolog"
)

// Server wraps the MCP server
type Server struct {
	mcpServer      *mcp.Server
	sessionManager *sshmcp.SessionManager
	hostManager    *sshmcp.HostManager
	logger         *zerolog.Logger
}

// NewServer creates a new MCP server
func NewServer(sessionManager *sshmcp.SessionManager, hostManager *sshmcp.HostManager, logger *zerolog.Logger) (*Server, error) {
	// 创建 MCP 服务器 - 使用正确的 API
	mcpServer := mcp.NewServer(&mcp.Implementation{
		Name:    "ssh-mcp-server",
		Version: "1.0.0",
	}, nil)

	s := &Server{
		mcpServer:      mcpServer,
		sessionManager: sessionManager,
		hostManager:    hostManager,
		logger:         logger,
	}

	// 注册 Tools
	s.registerTools()

	return s, nil
}

// registerTools registers all SSH MCP tools
func (s *Server) registerTools() {
	// 连接管理工具
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "ssh_connect",
		Description: "建立 SSH 连接并创建会话",
		InputSchema: sshConnectSchema(),
	}, s.handleSSHConnect)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "ssh_disconnect",
		Description: "断开 SSH 会话",
		InputSchema: sshDisconnectSchema(),
	}, s.handleSSHDisconnect)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "ssh_list_sessions",
		Description: "列出所有活跃会话",
		InputSchema: sshListSessionsSchema(),
	}, s.handleSSHListSessions)

	// 命令执行工具
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "ssh_exec",
		Description: `执行单条命令并返回结果（推荐用于大多数场景）。

✅ 适用场景：
- 一次性命令：ls、cat、grep、ps、df -h、uptime 等
- 不需要保持上下文的独立命令
- 批量执行独立命令：使用 ssh_exec_batch

⚡ 优势：
- 比 ssh_shell(cooked) 更高效
- 自动获取完整输出和退出码
- 不会卡住
- 有超时保护
- 支持工作目录设置（working_dir）

❌ 不要使用场景：
- 需要保持环境变量或目录状态 → 使用 ssh_shell
- 运行交互式程序（vim、top、htop、gdb）→ 使用 ssh_shell(mode=raw)`,
		InputSchema: sshExecSchema(),
	}, s.handleSSHExec)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "ssh_exec_batch",
		Description: "批量执行命令",
		InputSchema: sshExecBatchSchema(),
	}, s.handleSSHExecBatch)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name: "ssh_shell",
		Description: `启动交互式 shell 会话。

⚠️ 重要提示：
1. 如果只是执行简单命令（ls、cat、grep、cd、ps 等），请使用 ssh_exec，更高效且不会卡住
2. ssh_shell 用于需要保持上下文的场景（连续多个命令、环境变量、目录切换）

模式选择指南：
- mode="cooked"：仅用于简单命令（ls/cat/echo），但不如 ssh_exec 高效
- mode="raw"：必须用于交互式程序（vim/top/htop/gdb/python/mysql），否则会卡住

典型使用场景：
✅ 需要在同一会话中连续执行多个命令并保持状态
✅ 运行交互式程序（必须用 raw 模式）
✅ 需要追踪当前目录变化

❌ 不要用 ssh_shell 的场景：
- 一次性命令 → 用 ssh_exec
- 批量独立命令 → 用 ssh_exec_batch`,
		InputSchema: sshShellSchema(),
	}, s.handleSSHShell)

	// 文件传输工具
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "sftp_upload",
		Description: "上传文件到远程",
		InputSchema: sftpUploadSchema(),
	}, s.handleSFTPUpload)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "sftp_download",
		Description: "从远程下载文件",
		InputSchema: sftpDownloadSchema(),
	}, s.handleSFTPDownload)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "sftp_list_dir",
		Description: "列出远程目录",
		InputSchema: sftpListDirSchema(),
	}, s.handleSFTPListDir)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "sftp_mkdir",
		Description: "创建远程目录",
		InputSchema: sftpMkdirSchema(),
	}, s.handleSFTPMkdir)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "sftp_delete",
		Description: "删除远程文件或目录",
		InputSchema: sftpDeleteSchema(),
	}, s.handleSFTPDelete)

	// 会话交互工具
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "ssh_write_input",
		Description: "向交互式会话写入输入",
		InputSchema: sshWriteInputSchema(),
	}, s.handleSSHWriteInput)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "ssh_read_output",
		Description: "读取会话输出",
		InputSchema: sshReadOutputSchema(),
	}, s.handleSSHReadOutput)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "ssh_resize_pty",
		Description: "调整终端窗口大小",
		InputSchema: sshResizePtySchema(),
	}, s.handleSSHResizePty)

	// Shell 状态查询工具
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "ssh_shell_status",
		Description: "查询 shell 会话状态（是否活动、当前目录、是否有未读取输出等）",
		InputSchema: sshShellStatusSchema(),
	}, s.handleSSHShellStatus)

	// 命令历史工具
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "ssh_history",
		Description: "查看会话的命令执行历史（记录所有通过 ssh_exec 和 ssh_exec_batch 执行的命令）",
		InputSchema: sshHistorySchema(),
	}, s.handleSSHHistory)

	// 主机管理工具
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "ssh_list_hosts",
		Description: "列出所有预定义的主机配置",
		InputSchema: sshListHostsSchema(),
	}, s.handleSSHListHosts)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "ssh_save_host",
		Description: "保存主机配置以便后续快速连接",
		InputSchema: sshSaveHostSchema(),
	}, s.handleSSHSaveHost)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "ssh_remove_host",
		Description: "删除已保存的主机配置",
		InputSchema: sshRemoveHostSchema(),
	}, s.handleSSHRemoveHost)
}

// Start starts the MCP server
func (s *Server) Start(ctx context.Context) error {
	s.logger.Info().Msg("Starting MCP server")

	// 使用 stdio transport - 使用正确的 API
	return s.mcpServer.Run(ctx, &mcp.StdioTransport{})
}

// GetMCPServer returns the underlying MCP server
func (s *Server) GetMCPServer() *mcp.Server {
	return s.mcpServer
}

// Helper function to create text content
func textContent(text string) []mcp.Content {
	return []mcp.Content{&mcp.TextContent{Text: text}}
}

func formatResult(format string, args ...interface{}) []mcp.Content {
	return textContent(fmt.Sprintf(format, args...))
}

func formatError(err error) []mcp.Content {
	return textContent(fmt.Sprintf("Error: %v", err))
}
