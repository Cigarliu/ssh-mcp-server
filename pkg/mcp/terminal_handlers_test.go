package mcp

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cigar/sshmcp/internal/state"
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
	stateStore, err := state.Open(filepath.Join(t.TempDir(), "state.db"))
	if err != nil {
		t.Fatalf("open state store: %v", err)
	}
	t.Cleanup(func() { stateStore.Close() })
	hosts, err := sshmcp.NewHostManagerWithStore(map[string]sshmcp.HostConfig{
		"local": {Host: "127.0.0.1", Port: 22, Username: "user", Password: "not-for-model-output", Description: "Local test connection"},
	}, "", logger, stateStore)
	if err != nil {
		t.Fatalf("new persistent host manager: %v", err)
	}
	server, err := NewServerWithStore(manager, hosts, stateStore, logger)
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
	for _, secret := range []string{"not-for-model-output", "127.0.0.1", "\"user\""} {
		if strings.Contains(string(encoded), secret) {
			t.Fatalf("connection_list exposed saved connection data %q: %s", secret, encoded)
		}
	}
	payload, ok := output.(map[string]any)
	if !ok {
		t.Fatalf("connection_list output type = %T, want map[string]any", output)
	}
	ports, ok := payload["serial_ports"].([]string)
	if !ok {
		t.Fatalf("serial_ports type = %T, want []string", payload["serial_ports"])
	}
	if ports == nil {
		t.Fatal("serial_ports must be an empty array, not null")
	}
	saved, ok := payload["saved_ssh_hosts"].([]map[string]any)
	if !ok || len(saved) != 1 {
		t.Fatalf("saved_ssh_hosts = %#v, want one sanitized profile", payload["saved_ssh_hosts"])
	}
	if saved[0]["connection_id"] != "local" || saved[0]["description"] != "Local test connection" {
		t.Fatalf("unexpected saved connection summary: %#v", saved[0])
	}
}

func TestConnectionHistoryReturnsPersistentRecordsWithoutTargetDetails(t *testing.T) {
	logger := setupTestLogger()
	manager := sshmcp.NewSessionManager(sshmcp.ManagerConfig{MaxSessions: 10, SessionTimeout: time.Minute, IdleTimeout: time.Minute, CleanupInterval: time.Minute, Logger: logger})
	t.Cleanup(manager.Close)
	stateStore, err := state.Open(filepath.Join(t.TempDir(), "state.db"))
	if err != nil {
		t.Fatalf("open state store: %v", err)
	}
	t.Cleanup(func() { stateStore.Close() })
	hosts, err := sshmcp.NewHostManagerWithStore(map[string]sshmcp.HostConfig{
		"prod-web": {Host: "10.22.33.44", Port: 22, Username: "deploy", Password: "secret", Description: "Production web server"},
	}, "", logger, stateStore)
	if err != nil {
		t.Fatalf("new persistent host manager: %v", err)
	}
	server, err := NewServerWithStore(manager, hosts, stateStore, logger)
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	if err := stateStore.RecordHistory(state.HistoryEntry{ConnectionID: "prod-web", DescriptionSnapshot: "Production web server", Kind: "exec", Input: "uname -a", Output: "Linux", State: "success"}); err != nil {
		t.Fatalf("record history: %v", err)
	}
	_, output, err := server.handleConnectionHistory(context.Background(), nil, map[string]any{"connection_id": "prod-web"})
	if err != nil {
		t.Fatalf("connection_history: %v", err)
	}
	encoded, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("encode history: %v", err)
	}
	for _, secret := range []string{"10.22.33.44", "deploy", "secret"} {
		if strings.Contains(string(encoded), secret) {
			t.Fatalf("connection_history exposed saved connection data %q: %s", secret, encoded)
		}
	}
}

func TestPersistentDirectSSHRequiresModelChosenIDAndDescription(t *testing.T) {
	logger := setupTestLogger()
	manager := sshmcp.NewSessionManager(sshmcp.ManagerConfig{MaxSessions: 10, SessionTimeout: time.Minute, IdleTimeout: time.Minute, CleanupInterval: time.Minute, Logger: logger})
	t.Cleanup(manager.Close)
	stateStore, err := state.Open(filepath.Join(t.TempDir(), "state.db"))
	if err != nil {
		t.Fatalf("open state store: %v", err)
	}
	t.Cleanup(func() { stateStore.Close() })
	hosts, err := sshmcp.NewHostManagerWithStore(map[string]sshmcp.HostConfig{}, "", logger, stateStore)
	if err != nil {
		t.Fatalf("new persistent host manager: %v", err)
	}
	server, err := NewServerWithStore(manager, hosts, stateStore, logger)
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	base := map[string]any{"transport": "ssh", "host": "203.0.113.1", "username": "deploy", "password": "secret"}
	for _, args := range []map[string]any{
		base,
		{"transport": "ssh", "host": "203.0.113.1", "username": "deploy", "password": "secret", "connection_id": "prod-web"},
		{"transport": "ssh", "host": "203.0.113.1", "username": "deploy", "password": "secret", "connection_id": "Prod-Web", "description": "Production web server"},
	} {
		result, _, err := server.handleConnectionOpen(context.Background(), nil, args)
		if err != nil {
			t.Fatalf("connection_open returned handler error: %v", err)
		}
		if result == nil || !result.IsError {
			t.Fatalf("connection_open accepted incomplete persistent metadata: %#v", args)
		}
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
		"transport": "ssh", "connection_id": "local", "port": float64(22),
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

	properties := terminalInteractSchema()["properties"].(map[string]any)
	if _, found := properties["pattern"]; found {
		t.Fatal("terminal_interact must not expose the redundant pattern field")
	}
	if _, found := properties["until"]; !found {
		t.Fatal("terminal_interact must expose until for literal matching")
	}

	connectionProperties := connectionOpenSchema()["properties"].(map[string]any)
	for _, deprecated := range []string{"session_id", "hostname", "alias"} {
		if _, found := connectionProperties[deprecated]; found {
			t.Fatalf("connection_open must not expose deprecated %q: %#v", deprecated, connectionProperties)
		}
	}
	connectionID := connectionProperties["connection_id"].(map[string]any)
	if connectionID["pattern"] != "^[a-z][a-z0-9][a-z0-9_-]{1,62}$" {
		t.Fatalf("connection_id pattern = %#v, want stable lowercase ID pattern", connectionID["pattern"])
	}

	listProperties := connectionListOutputSchema()["properties"].(map[string]any)
	serialPorts := listProperties["serial_ports"].(map[string]any)
	if serialPorts["type"] != "array" {
		t.Fatalf("serial_ports schema type = %#v, want array", serialPorts["type"])
	}
}

func TestSSHCapabilitiesIncludeUnifiedFileOperations(t *testing.T) {
	server, manager := setupTestServer(t)
	t.Cleanup(manager.Close)
	if !strings.Contains(strings.Join(server.sshCapabilities(), ","), "files") {
		t.Fatal("unified server must advertise file capabilities")
	}
}
