package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/cigar/sshmcp/pkg/sshmcp"
)

func TestTerminalInput(t *testing.T) {
	cases := []struct {
		input string
		mode  string
		want  string
	}{
		{input: "echo ok", mode: "raw", want: "echo ok"},
		{input: "echo ok", mode: "line", want: "echo ok\n"},
		{input: "up", mode: "key", want: "\x1b[A"},
	}
	for _, test := range cases {
		got, err := terminalInput(test.input, test.mode)
		if err != nil || string(got) != test.want {
			t.Fatalf("terminalInput(%q, %q) = %q, %v; want %q", test.input, test.mode, got, err, test.want)
		}
	}
	if _, err := terminalInput("unknown", "key"); err == nil {
		t.Fatal("unsupported key must fail")
	}
}

func TestBoundedIntArg(t *testing.T) {
	value, err := boundedIntArg(map[string]any{}, "timeout_ms", 3000, 1, 60000)
	if err != nil || value != 3000 {
		t.Fatalf("default: value=%d err=%v", value, err)
	}
	value, err = boundedIntArg(map[string]any{"timeout_ms": float64(500)}, "timeout_ms", 3000, 1, 60000)
	if err != nil || value != 500 {
		t.Fatalf("valid value: value=%d err=%v", value, err)
	}
	if _, err := boundedIntArg(map[string]any{"timeout_ms": float64(0)}, "timeout_ms", 3000, 1, 60000); err == nil {
		t.Fatal("out-of-range value must fail")
	}
}

func TestConnectionListRedactsSavedHostCredentials(t *testing.T) {
	logger := setupTestLogger()
	manager := sshmcp.NewSessionManager(sshmcp.ManagerConfig{
		MaxSessions: 10, SessionTimeout: time.Minute, IdleTimeout: time.Minute, CleanupInterval: time.Minute, Logger: logger,
	})
	t.Cleanup(manager.Close)
	hosts := sshmcp.NewHostManager(map[string]sshmcp.HostConfig{
		"local": {Host: "127.0.0.1", Port: 22, Username: "user", Password: "not-for-model-output"},
	}, "", logger)
	server, err := NewServer(manager, hosts, logger)
	if err != nil {
		t.Fatalf("new server: %v", err)
	}

	_, output, err := server.handleConnectionList(context.Background(), nil, map[string]any{})
	if err != nil {
		t.Fatalf("connection_list: %v", err)
	}
	encoded, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("encode output: %v", err)
	}
	if strings.Contains(string(encoded), "not-for-model-output") {
		t.Fatalf("connection_list exposed a saved password: %s", encoded)
	}
}

func TestConnectionOpenRejectsMixedTransportFields(t *testing.T) {
	server, manager := setupTestServer(t)
	t.Cleanup(manager.Close)

	for _, args := range []map[string]any{
		{"transport": "ssh", "host": "example.test", "username": "user", "password": "password", "device": "/dev/ttyUSB0"},
		{"transport": "serial", "device": "/dev/ttyUSB0", "host": "example.test"},
	} {
		result, _, err := server.handleConnectionOpen(context.Background(), nil, args)
		if err != nil {
			t.Fatalf("connection_open returned handler error: %v", err)
		}
		if result == nil || !result.IsError {
			t.Fatalf("connection_open accepted mixed transport fields: %#v", args)
		}
	}
}

func TestSavedSSHHostRejectsDirectFields(t *testing.T) {
	logger := setupTestLogger()
	manager := sshmcp.NewSessionManager(sshmcp.ManagerConfig{
		MaxSessions: 10, SessionTimeout: time.Minute, IdleTimeout: time.Minute, CleanupInterval: time.Minute, Logger: logger,
	})
	t.Cleanup(manager.Close)
	hosts := sshmcp.NewHostManager(map[string]sshmcp.HostConfig{
		"local": {Host: "127.0.0.1", Port: 2222, Username: "user", Password: "password"},
	}, "", logger)
	server, err := NewServer(manager, hosts, logger)
	if err != nil {
		t.Fatalf("new server: %v", err)
	}

	result, _, err := server.handleConnectionOpen(context.Background(), nil, map[string]any{
		"transport": "ssh", "hostname": "local", "port": float64(22),
	})
	if err != nil {
		t.Fatalf("connection_open returned handler error: %v", err)
	}
	if result == nil || !result.IsError {
		t.Fatal("connection_open accepted a saved host with direct SSH fields")
	}
}

func TestTerminalOpenRejectsSerialTUI(t *testing.T) {
	server, manager := setupTestServer(t)
	t.Cleanup(manager.Close)

	result, _, err := server.handleTerminalOpen(context.Background(), nil, map[string]any{
		"connection_id": "serial-not-open", "profile": "tui",
	})
	if err != nil {
		t.Fatalf("terminal_open returned handler error: %v", err)
	}
	if result == nil || !result.IsError {
		t.Fatal("terminal_open accepted TUI profile for a serial connection")
	}
}

func TestTerminalViewRequiresTUIProfile(t *testing.T) {
	server, manager := setupTestServer(t)
	t.Cleanup(manager.Close)

	entry := server.terminals.Register("connection", "ssh", "shell", nil, nil, nil)
	result, _, err := server.handleTerminalView(context.Background(), nil, map[string]any{"terminal_id": entry.ID})
	if err != nil {
		t.Fatalf("terminal_view returned handler error: %v", err)
	}
	if result == nil || !result.IsError {
		t.Fatal("terminal_view accepted a shell-profile terminal")
	}
}

func TestModelFacingSchemasUseOneConnectionVocabulary(t *testing.T) {
	for _, schema := range []map[string]any{sshExecSchema(), sftpTransferSchema(), sftpManageSchema()} {
		properties := schema["properties"].(map[string]any)
		if _, found := properties["connection_id"]; !found {
			t.Fatalf("schema is missing connection_id: %#v", properties)
		}
		if _, found := properties["session_id"]; found {
			t.Fatalf("model-facing schema unexpectedly exposes session_id: %#v", properties)
		}
	}

	legacyProperties := sshExecBatchSchema()["properties"].(map[string]any)
	if _, found := legacyProperties["session_id"]; !found {
		t.Fatalf("legacy schema is missing session_id: %#v", legacyProperties)
	}

	properties := terminalInteractSchema()["properties"].(map[string]any)
	if _, found := properties["pattern"]; found {
		t.Fatal("terminal_interact must not expose the redundant pattern field")
	}
	if _, found := properties["until"]; !found {
		t.Fatal("terminal_interact must expose until for literal matching")
	}
}

func TestSSHCapabilitiesFollowToolProfile(t *testing.T) {
	server, manager := setupTestServer(t)
	t.Cleanup(manager.Close)
	if strings.Contains(strings.Join(server.sshCapabilities(), ","), "files") {
		t.Fatal("core profile must not advertise file capabilities")
	}

	filesServer, err := NewServerWithProfile(manager, server.hostManager, server.logger, "files")
	if err != nil {
		t.Fatalf("new files-profile server: %v", err)
	}
	if !strings.Contains(strings.Join(filesServer.sshCapabilities(), ","), "files") {
		t.Fatal("files profile must advertise file capabilities")
	}
}
