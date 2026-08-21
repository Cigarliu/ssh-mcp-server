package mcp

import (
	"context"
	"fmt"

	"github.com/cigar/sshmcp/internal/buildinfo"
	"github.com/cigar/sshmcp/internal/state"

	"github.com/cigar/sshmcp/pkg/serialmcp"
	"github.com/cigar/sshmcp/pkg/sshmcp"
	"github.com/cigar/sshmcp/pkg/terminal"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rs/zerolog"
)

// Server wraps the MCP server
type Server struct {
	mcpServer      *mcp.Server
	sessionManager *sshmcp.SessionManager
	hostManager    *sshmcp.HostManager
	serialManager  *serialmcp.Manager
	terminals      *terminal.Registry
	logger         *zerolog.Logger
	stateStore     *state.Store
}

// NewServer creates a new MCP server
func NewServer(sessionManager *sshmcp.SessionManager, hostManager *sshmcp.HostManager, logger *zerolog.Logger) (*Server, error) {
	return NewServerWithStore(sessionManager, hostManager, nil, logger)
}

// NewServerWithStore creates a server backed by a shared state store.
func NewServerWithStore(sessionManager *sshmcp.SessionManager, hostManager *sshmcp.HostManager, stateStore *state.Store, logger *zerolog.Logger) (*Server, error) {
	mcpServer := mcp.NewServer(&mcp.Implementation{
		Name:    "ssh-mcp-server",
		Version: buildinfo.Version,
	}, &mcp.ServerOptions{
		Instructions: toolInstructions(),
	})

	s := &Server{
		mcpServer:      mcpServer,
		sessionManager: sessionManager,
		hostManager:    hostManager,
		serialManager:  serialmcp.NewManager(),
		terminals:      terminal.NewRegistry(),
		logger:         logger,
		stateStore:     stateStore,
	}

	s.registerTools()

	return s, nil
}

func toolInstructions() string {
	instructions := "Use one connection vocabulary. connection_id is the durable, model-chosen identifier for a saved SSH connection; it is not a secret and can be reused across MCP instances. An active SSH connection is process-local, and terminal_id is a separate process-local handle returned by terminal_open. Call connection_list only to discover available saved IDs, active connections, or serial ports; when an explicit saved connection_id is already known, reopen it directly with connection_open. For a first direct SSH connection, choose a readable lowercase connection_id, provide a concise description, and provide host, username, and exactly one authentication method. The server persists the profile only after opening succeeds. Use connection_history for durable ssh_exec and terminal_interact records; it never returns saved addresses or credentials. Use ssh_exec for one-shot commands. For an interactive shell, REPL, device console, or full-screen program, open one terminal and reuse its terminal_id with terminal_interact. Default to wait=quiet; use wait=until only with a known literal, continue with next_offset after limit_reached, and inspect timeout output before deciding whether to continue. Call terminal_view only for a terminal opened with profile=tui and screen=true. Confirm the target and intent before sftp_manage delete or sftp_transfer overwrite=true."
	return instructions + " Use sftp_transfer for upload/download and sftp_manage for directory operations."
}

// registerTools registers all SSH MCP tools
func (s *Server) registerTools() {
	// Core tools are deliberately transport-neutral after connection bootstrap.
	mcp.AddTool(s.mcpServer, &mcp.Tool{Name: "connection_open", Description: "Open an SSH or serial connection. Reopen a saved SSH profile with transport=ssh and its known connection_id only. For a first direct SSH connection, choose a readable connection_id, add a description, host, username, and exactly one authentication method; the profile is persisted only after success. Returns the stable connection_id and capabilities, never credentials.", InputSchema: connectionOpenSchema(), OutputSchema: connectionOpenOutputSchema()}, s.handleConnectionOpen)
	mcp.AddTool(s.mcpServer, &mcp.Tool{Name: "connection_close", Description: "Close an active SSH or serial connection by connection_id and close any terminal attached to it. Closing does not delete a saved SSH profile or its durable history.", InputSchema: connectionCloseSchema(), OutputSchema: connectionCloseOutputSchema()}, s.handleConnectionClose)
	mcp.AddTool(s.mcpServer, &mcp.Tool{Name: "connection_list", Description: "Discover active connections, saved SSH connection IDs with descriptions, and locally available serial ports. Use it when no suitable ID is known. Saved SSH targets, usernames, credentials, and key paths are never returned.", InputSchema: connectionListSchema(), OutputSchema: connectionListOutputSchema()}, s.handleConnectionList)
	mcp.AddTool(s.mcpServer, &mcp.Tool{Name: "connection_history", Description: "Read durable ssh_exec and terminal_interact history for a saved SSH connection_id. Use it to recover prior work across MCP instances. Results contain the stable ID and description, never the stored target or credentials.", InputSchema: connectionHistorySchema(), OutputSchema: connectionHistoryOutputSchema()}, s.handleConnectionHistory)
	mcp.AddTool(s.mcpServer, &mcp.Tool{Name: "terminal_open", Description: "Create or return the process-local terminal_id for an already open connection. Use shell for command-line interaction and tui only for SSH full-screen programs. Reuse the returned terminal_id; close it before changing profiles. Serial supports shell only.", InputSchema: terminalOpenSchema(), OutputSchema: terminalOpenOutputSchema()}, s.handleTerminalOpen)
	mcp.AddTool(s.mcpServer, &mcp.Tool{Name: "terminal_interact", Description: "Send input and read a terminal stream atomically. Reuse terminal_id, normally use input_mode=line and wait=quiet, and use wait=until only with a known literal. Branch on state and stop_reason: continue from next_offset after limit_reached, inspect returned data after timeout, and do not blindly resend input.", InputSchema: terminalInteractSchema(), OutputSchema: terminalInteractOutputSchema()}, s.handleTerminalInteract)
	mcp.AddTool(s.mcpServer, &mcp.Tool{Name: "terminal_view", Description: "Read the projected screen of a terminal opened with profile=tui. Use it only when terminal_open returned screen=true; use terminal_interact for normal shell or serial output.", InputSchema: terminalViewSchema(), OutputSchema: terminalViewOutputSchema()}, s.handleTerminalView)
	mcp.AddTool(s.mcpServer, &mcp.Tool{Name: "terminal_close", Description: "Close a process-local terminal_id. The SSH connection remains open for ssh_exec; closing a serial terminal also releases its serial port.", InputSchema: terminalCloseSchema(), OutputSchema: terminalCloseOutputSchema()}, s.handleTerminalClose)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:         "ssh_exec",
		Description:  "Run one non-interactive SSH command on an active connection_id. Use it for independent commands; use terminal_open and terminal_interact for stateful, prompt-driven, or full-screen programs. Check exit_code, stderr, and truncated before deciding the next action.",
		InputSchema:  sshExecSchema(),
		OutputSchema: sshExecOutputSchema(),
	}, s.handleSSHExec)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "sftp_transfer",
		Description: "Upload or download one file on an active SSH connection_id. Confirm both paths before transfer. Existing destination files are preserved unless overwrite=true is explicitly requested; inspect the result before retrying a failed or partial transfer.",
		InputSchema: sftpTransferSchema(),
	}, s.handleSFTPTransfer)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "sftp_manage",
		Description: "List directories, create directories, or delete a remote path on an active SSH connection_id. Prefer list to inspect a target first. delete is irreversible: require clear user intent, verify remote_path, and set recursive only when the user explicitly intends subtree removal.",
		InputSchema: sftpManageSchema(),
	}, s.handleSFTPManage)
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
