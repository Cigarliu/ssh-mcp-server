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
