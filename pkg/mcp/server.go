package mcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/cigar/sshmcp/pkg/serialmcp"
	"github.com/cigar/sshmcp/pkg/sshmcp"
	"github.com/cigar/sshmcp/pkg/terminal"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rs/zerolog"
)

// ToolProfile controls the amount of MCP surface exposed to a model.
type ToolProfile string

const (
	ToolProfileCore     ToolProfile = "core"
	ToolProfileFiles    ToolProfile = "files"
	ToolProfileAdvanced ToolProfile = "advanced"
)

// Server wraps the MCP server
type Server struct {
	mcpServer      *mcp.Server
	sessionManager *sshmcp.SessionManager
	hostManager    *sshmcp.HostManager
	serialManager  *serialmcp.Manager
	terminals      *terminal.Registry
	logger         *zerolog.Logger
}

// NewServer creates a new MCP server
func NewServer(sessionManager *sshmcp.SessionManager, hostManager *sshmcp.HostManager, logger *zerolog.Logger) (*Server, error) {
	return NewServerWithProfile(sessionManager, hostManager, logger, string(ToolProfileCore))
}

// NewServerWithProfile creates an MCP server with the requested tool profile.
func NewServerWithProfile(sessionManager *sshmcp.SessionManager, hostManager *sshmcp.HostManager, logger *zerolog.Logger, profileName string) (*Server, error) {
	profile, err := parseToolProfile(profileName)
	if err != nil {
		return nil, err
	}

	mcpServer := mcp.NewServer(&mcp.Implementation{
		Name:    "ssh-mcp-server",
		Version: "1.0.0",
	}, &mcp.ServerOptions{
		Instructions: toolInstructions(profile),
	})

	s := &Server{
		mcpServer:      mcpServer,
		sessionManager: sessionManager,
		hostManager:    hostManager,
		serialManager:  serialmcp.NewManager(),
		terminals:      terminal.NewRegistry(),
		logger:         logger,
	}

	s.registerTools(profile)

	return s, nil
}

func parseToolProfile(profileName string) (ToolProfile, error) {
	switch ToolProfile(strings.ToLower(strings.TrimSpace(profileName))) {
	case "", ToolProfileCore:
		return ToolProfileCore, nil
	case ToolProfileFiles, "basic":
		return ToolProfileFiles, nil
	case ToolProfileAdvanced:
		return ToolProfileAdvanced, nil
	default:
		return "", fmt.Errorf("unknown tool profile %q: use core, files, or advanced", profileName)
	}
}

func toolInstructions(profile ToolProfile) string {
	instructions := "Use connection_open to create an SSH or serial connection. Use ssh_exec only for non-interactive SSH commands. For any continuous terminal or REPL, call terminal_open then terminal_interact; every terminal result reports a completion state, bounded output, and stream offsets. Use terminal_view only for TUI screens."
	switch profile {
	case ToolProfileCore:
		return instructions + " This server uses the core profile; file transfer and advanced session controls are unavailable."
	case ToolProfileFiles:
		return instructions + " This server uses the files profile; use sftp_transfer for upload/download and sftp_manage for directory operations."
	default:
		return instructions
	}
}

// registerTools registers all SSH MCP tools
func (s *Server) registerTools(profile ToolProfile) {
	// Core tools are deliberately transport-neutral after connection bootstrap.
	mcp.AddTool(s.mcpServer, &mcp.Tool{Name: "connection_open", Description: "Open an SSH or serial connection and return its capabilities. For saved SSH hosts, pass hostname from connection_list instead of credentials.", InputSchema: connectionOpenSchema(), OutputSchema: connectionOpenOutputSchema()}, s.handleConnectionOpen)
	mcp.AddTool(s.mcpServer, &mcp.Tool{Name: "connection_close", Description: "Close an SSH or serial connection and any terminal attached to it.", InputSchema: connectionCloseSchema(), OutputSchema: connectionCloseOutputSchema()}, s.handleConnectionClose)
	mcp.AddTool(s.mcpServer, &mcp.Tool{Name: "connection_list", Description: "List active connections and locally discoverable serial devices.", InputSchema: connectionListSchema(), OutputSchema: connectionListOutputSchema()}, s.handleConnectionList)
	mcp.AddTool(s.mcpServer, &mcp.Tool{Name: "terminal_open", Description: "Create a transport-neutral terminal session on an open connection. Use profile shell or repl for text prompts and tui for full-screen programs.", InputSchema: terminalOpenSchema(), OutputSchema: terminalOpenOutputSchema()}, s.handleTerminalOpen)
	mcp.AddTool(s.mcpServer, &mcp.Tool{Name: "terminal_interact", Description: "Atomically capture an output cursor, send terminal input, and wait for a prompt, pattern, quiet period, or stable screen. Decide next actions from state and stop_reason; never blindly retry a timeout.", InputSchema: terminalInteractSchema(), OutputSchema: terminalInteractOutputSchema()}, s.handleTerminalInteract)
	mcp.AddTool(s.mcpServer, &mcp.Tool{Name: "terminal_view", Description: "Read the current projected screen for a TUI terminal. It is not for normal command output.", InputSchema: terminalViewSchema(), OutputSchema: terminalViewOutputSchema()}, s.handleTerminalView)
	mcp.AddTool(s.mcpServer, &mcp.Tool{Name: "terminal_close", Description: "Close a terminal session. SSH connections remain available for ssh_exec; serial terminals also release their serial port.", InputSchema: terminalCloseSchema(), OutputSchema: terminalCloseOutputSchema()}, s.handleTerminalClose)

	// Legacy transport-specific controls remain available only for migration and
	// diagnostic work in the advanced profile.
	if profile == ToolProfileAdvanced {
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
	}

	// 命令执行工具
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:         "ssh_exec",
		Description:  "执行非交互式 SSH 命令或脚本。普通运维任务优先使用；仅 TUI/交互程序使用 terminal_open。",
		InputSchema:  sshExecSchema(),
		OutputSchema: sshExecOutputSchema(),
	}, s.handleSSHExec)

	if profile == ToolProfileAdvanced {
		mcp.AddTool(s.mcpServer, &mcp.Tool{
			Name:        "ssh_exec_batch",
			Description: "顺序执行多个独立命令；默认仅返回摘要和失败项。",
			InputSchema: sshExecBatchSchema(),
		}, s.handleSSHExecBatch)
	}

	if profile == ToolProfileAdvanced {
		mcp.AddTool(s.mcpServer, &mcp.Tool{
			Name:        "ssh_shell",
			Description: "Legacy: create a raw SSH shell. Prefer terminal_open in new integrations.",
			InputSchema: sshShellSchema(),
		}, s.handleSSHShell)
	}

	// 文件工具在 files profile 中按任务合并，在 advanced profile 中保留细粒度接口。
	if profile == ToolProfileFiles {
		mcp.AddTool(s.mcpServer, &mcp.Tool{
			Name:        "sftp_transfer",
			Description: "上传或下载文件。operation 为 upload 或 download。",
			InputSchema: sftpTransferSchema(),
		}, s.handleSFTPTransfer)

		mcp.AddTool(s.mcpServer, &mcp.Tool{
			Name:        "sftp_manage",
			Description: "列目录、创建目录或删除远程路径。operation 为 list、mkdir 或 delete。",
			InputSchema: sftpManageSchema(),
		}, s.handleSFTPManage)
	}
	if profile == ToolProfileAdvanced {
		mcp.AddTool(s.mcpServer, &mcp.Tool{Name: "sftp_upload", Description: "上传文件到远程", InputSchema: sftpUploadSchema()}, s.handleSFTPUpload)
		mcp.AddTool(s.mcpServer, &mcp.Tool{Name: "sftp_download", Description: "从远程下载文件", InputSchema: sftpDownloadSchema()}, s.handleSFTPDownload)
		mcp.AddTool(s.mcpServer, &mcp.Tool{Name: "sftp_list_dir", Description: "列出远程目录", InputSchema: sftpListDirSchema()}, s.handleSFTPListDir)
		mcp.AddTool(s.mcpServer, &mcp.Tool{Name: "sftp_mkdir", Description: "创建远程目录", InputSchema: sftpMkdirSchema()}, s.handleSFTPMkdir)
		mcp.AddTool(s.mcpServer, &mcp.Tool{Name: "sftp_delete", Description: "删除远程文件或目录", InputSchema: sftpDeleteSchema()}, s.handleSFTPDelete)
	}

	if profile == ToolProfileAdvanced {
		// Legacy shell interaction tools.
		mcp.AddTool(s.mcpServer, &mcp.Tool{Name: "ssh_write_input", Description: "Legacy: write input to an SSH shell.", InputSchema: sshWriteInputSchema()}, s.handleSSHWriteInput)
		mcp.AddTool(s.mcpServer, &mcp.Tool{Name: "ssh_read_output", Description: "Legacy: read the SSH shell line buffer.", InputSchema: sshReadOutputSchema()}, s.handleSSHReadOutput)
		mcp.AddTool(s.mcpServer, &mcp.Tool{
			Name:        "ssh_resize_pty",
			Description: "调整已启动交互式终端的尺寸。",
			InputSchema: sshResizePtySchema(),
		}, s.handleSSHResizePty)
		mcp.AddTool(s.mcpServer, &mcp.Tool{
			Name:        "ssh_terminal_snapshot",
			Description: "Legacy: return an SSH terminal screen.",
			InputSchema: sshTerminalSnapshotSchema(),
		}, s.handleSSHTerminalSnapshot)
		mcp.AddTool(s.mcpServer, &mcp.Tool{
			Name:        "ssh_shell_status",
			Description: "查询交互式 shell 的活动状态和缓冲区信息。",
			InputSchema: sshShellStatusSchema(),
		}, s.handleSSHShellStatus)

		mcp.AddTool(s.mcpServer, &mcp.Tool{
			Name:        "ssh_history",
			Description: "查看通过 ssh_exec 和 ssh_exec_batch 执行的命令历史。",
			InputSchema: sshHistorySchema(),
		}, s.handleSSHHistory)
	}

	if profile == ToolProfileAdvanced {
		mcp.AddTool(s.mcpServer, &mcp.Tool{
			Name:        "ssh_list_hosts",
			Description: "List predefined SSH host configurations.",
			InputSchema: sshListHostsSchema(),
		}, s.handleSSHListHosts)
		mcp.AddTool(s.mcpServer, &mcp.Tool{
			Name:        "ssh_save_host",
			Description: "持久化保存主机连接配置。",
			InputSchema: sshSaveHostSchema(),
		}, s.handleSSHSaveHost)

		mcp.AddTool(s.mcpServer, &mcp.Tool{
			Name:        "ssh_remove_host",
			Description: "删除已保存的主机连接配置。",
			InputSchema: sshRemoveHostSchema(),
		}, s.handleSSHRemoveHost)
	}
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
