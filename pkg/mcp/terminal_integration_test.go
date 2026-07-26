package mcp

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/cigar/sshmcp/pkg/sshmcp"
	"github.com/cigar/sshmcp/pkg/terminal"
	"github.com/stretchr/testify/require"
)

// These tests exercise the model-facing tool sequence against opt-in local
// transports. Credentials and device paths come only from the test environment.
func TestLocalSSHTerminalToolPath(t *testing.T) {
	if os.Getenv("RUN_LOCAL_MCP_TERMINAL_TEST") != "1" {
		t.Skip("set RUN_LOCAL_MCP_TERMINAL_TEST=1 to run local SSH terminal integration")
	}
	host := os.Getenv("SSHMCP_LOCAL_SSH_HOST")
	username := os.Getenv("SSHMCP_LOCAL_SSH_USER")
	password := os.Getenv("SSHMCP_LOCAL_SSH_PASSWORD")
	if host == "" || username == "" || password == "" {
		t.Fatal("SSHMCP_LOCAL_SSH_HOST, SSHMCP_LOCAL_SSH_USER, and SSHMCP_LOCAL_SSH_PASSWORD are required")
	}

	server, manager := setupTestServer(t)
	t.Cleanup(manager.Close)
	server.hostManager = sshmcp.NewHostManager(map[string]sshmcp.HostConfig{
		"local": {Host: host, Port: 22, Username: username, Password: password},
	}, "", server.logger)
	const alias = "local-terminal-path"

	_, opened, err := server.handleConnectionOpen(context.Background(), nil, map[string]any{
		"transport": "ssh",
		"hostname":  "local",
		"alias":     alias,
	})
	require.NoError(t, err)
	connection, ok := opened.(map[string]any)
	require.True(t, ok, "connection_open must return structured data")
	connectionID, ok := connection["connection_id"].(string)
	require.True(t, ok)

	_, executed, err := server.handleSSHExec(context.Background(), nil, map[string]any{
		"connection_id": connectionID,
		"command":       "printf '__SSHMCP_EXEC_OK__\\n'",
	})
	require.NoError(t, err)
	execResult, ok := executed.(map[string]any)
	require.True(t, ok, "ssh_exec must return structured data")
	stdout, ok := execResult["stdout"].(string)
	require.True(t, ok)
	require.Contains(t, stdout, "__SSHMCP_EXEC_OK__")

	_, openedTerminal, err := server.handleTerminalOpen(context.Background(), nil, map[string]any{
		"connection_id": alias,
		"profile":       "shell",
	})
	require.NoError(t, err)
	terminalInfo, ok := openedTerminal.(map[string]any)
	require.True(t, ok, "terminal_open must return structured data")
	require.Equal(t, false, terminalInfo["screen"], "shell profile must not advertise a screen projection")
	terminalID, ok := terminalInfo["terminal_id"].(string)
	require.True(t, ok)

	_, interacted, err := server.handleTerminalInteract(context.Background(), nil, map[string]any{
		"terminal_id": terminalID,
		"input":       "printf '__SSHMCP_TERMINAL_OK__\\n'",
		"input_mode":  "line",
		"wait":        "quiet",
		"quiet_ms":    float64(150),
		"timeout_ms":  float64(5000),
	})
	require.NoError(t, err)
	result, ok := interacted.(terminal.Result)
	require.True(t, ok, "terminal_interact must return a terminal result")
	require.Equal(t, "stable", result.State)
	require.Equal(t, "utf8", result.Encoding)
	require.True(t, strings.Contains(result.Data, "__SSHMCP_TERMINAL_OK__"), "terminal data: %q", result.Data)

	viewResult, viewed, err := server.handleTerminalView(context.Background(), nil, map[string]any{"terminal_id": terminalID})
	require.NoError(t, err)
	require.Nil(t, viewed)
	require.True(t, viewResult.IsError, "terminal_view must reject a shell-profile terminal")

	_, closed, err := server.handleConnectionClose(context.Background(), nil, map[string]any{"connection_id": alias})
	require.NoError(t, err)
	closeResult, ok := closed.(map[string]any)
	require.True(t, ok)
	require.Equal(t, connectionID, closeResult["connection_id"])
	require.Nil(t, server.terminals.FindByConnection(connectionID), "closing an alias must remove its terminal")
}

func TestLocalSSHTUIProjectionToolPath(t *testing.T) {
	if os.Getenv("RUN_LOCAL_MCP_TERMINAL_TEST") != "1" {
		t.Skip("set RUN_LOCAL_MCP_TERMINAL_TEST=1 to run local SSH terminal integration")
	}
	host := os.Getenv("SSHMCP_LOCAL_SSH_HOST")
	username := os.Getenv("SSHMCP_LOCAL_SSH_USER")
	password := os.Getenv("SSHMCP_LOCAL_SSH_PASSWORD")
	if host == "" || username == "" || password == "" {
		t.Fatal("SSHMCP_LOCAL_SSH_HOST, SSHMCP_LOCAL_SSH_USER, and SSHMCP_LOCAL_SSH_PASSWORD are required")
	}

	logger := setupTestLogger()
	manager := sshmcp.NewSessionManager(sshmcp.ManagerConfig{
		MaxSessions: 10, MaxSessionsPerHost: 10, SessionTimeout: time.Minute, IdleTimeout: time.Minute, CleanupInterval: time.Minute, Logger: logger,
	})
	t.Cleanup(manager.Close)
	hosts := sshmcp.NewHostManager(map[string]sshmcp.HostConfig{
		"local": {Host: host, Port: 22, Username: username, Password: password},
	}, "", logger)
	server, err := NewServer(manager, hosts, logger)
	require.NoError(t, err)

	alias := "local-tui"
	_, opened, err := server.handleConnectionOpen(context.Background(), nil, map[string]any{
		"transport": "ssh", "hostname": "local", "alias": alias,
	})
	require.NoError(t, err)
	connection, ok := opened.(map[string]any)
	require.True(t, ok, "connection_open must return structured data")
	connectionID, ok := connection["connection_id"].(string)
	require.True(t, ok)

	_, openedTerminal, err := server.handleTerminalOpen(context.Background(), nil, map[string]any{
		"connection_id": alias, "profile": "tui",
	})
	require.NoError(t, err)
	terminalInfo, ok := openedTerminal.(map[string]any)
	require.True(t, ok, "terminal_open must return structured data")
	require.Equal(t, true, terminalInfo["screen"], "tui profile must advertise a screen projection")
	terminalID, ok := terminalInfo["terminal_id"].(string)
	require.True(t, ok)

	_, interacted, err := server.handleTerminalInteract(context.Background(), nil, map[string]any{
		"terminal_id": terminalID,
		"input":       "printf '__SSHMCP_TUI_OK__\\n'",
		"input_mode":  "line",
		"wait":        "quiet",
		"quiet_ms":    float64(150),
		"timeout_ms":  float64(5000),
	})
	require.NoError(t, err)
	result, ok := interacted.(terminal.Result)
	require.True(t, ok, "terminal_interact must return a terminal result")
	require.Equal(t, "stable", result.State)
	require.True(t, strings.Contains(result.Data, "__SSHMCP_TUI_OK__"), "terminal data: %q", result.Data)

	_, viewed, err := server.handleTerminalView(context.Background(), nil, map[string]any{"terminal_id": terminalID})
	require.NoError(t, err)
	view, ok := viewed.(map[string]any)
	require.True(t, ok, "terminal_view must return structured data")
	require.NotEmpty(t, view["screen_hash"])

	_, _, err = server.handleConnectionClose(context.Background(), nil, map[string]any{"connection_id": connectionID})
	require.NoError(t, err)
}

func TestLocalSerialTerminalToolPath(t *testing.T) {
	if os.Getenv("RUN_LOCAL_SERIAL_TERMINAL_TEST") != "1" {
		t.Skip("set RUN_LOCAL_SERIAL_TERMINAL_TEST=1 to run local serial integration")
	}
	device := os.Getenv("SSHMCP_LOCAL_SERIAL_DEVICE")
	if device == "" {
		t.Fatal("SSHMCP_LOCAL_SERIAL_DEVICE is required")
	}

	server, manager := setupTestServer(t)
	t.Cleanup(manager.Close)
	_, opened, err := server.handleConnectionOpen(context.Background(), nil, map[string]any{
		"transport": "serial",
		"device":    device,
		"baud_rate": float64(115200),
	})
	require.NoError(t, err)
	connection, ok := opened.(map[string]any)
	require.True(t, ok, "connection_open must return structured data")
	connectionID, ok := connection["connection_id"].(string)
	require.True(t, ok)
	t.Cleanup(func() { server.serialManager.CloseAll() })

	_, openedTerminal, err := server.handleTerminalOpen(context.Background(), nil, map[string]any{"connection_id": connectionID})
	require.NoError(t, err)
	terminalInfo, ok := openedTerminal.(map[string]any)
	require.True(t, ok, "terminal_open must return structured data")
	terminalID, ok := terminalInfo["terminal_id"].(string)
	require.True(t, ok)

	_, interacted, err := server.handleTerminalInteract(context.Background(), nil, map[string]any{
		"terminal_id": terminalID,
		"wait":        "none",
	})
	require.NoError(t, err)
	result, ok := interacted.(terminal.Result)
	require.True(t, ok)
	require.Equal(t, "complete", result.State)

	_, closed, err := server.handleConnectionClose(context.Background(), nil, map[string]any{"connection_id": connectionID})
	require.NoError(t, err)
	closeResult, ok := closed.(map[string]any)
	require.True(t, ok)
	require.Equal(t, true, closeResult["closed"])
}
