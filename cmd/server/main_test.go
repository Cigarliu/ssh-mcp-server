package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestStdioInitializeHasNoLogLines(t *testing.T) {
	if os.Getenv("SSHMCP_SERVER_TEST_HELPER") == "1" {
		main()
		return
	}

	profiles := []struct {
		name  string
		tools []string
	}{
		{
			name:  "core",
			tools: []string{"connection_open", "connection_close", "connection_list", "ssh_exec", "terminal_open", "terminal_interact", "terminal_view", "terminal_close"},
		},
		{
			name:  "files",
			tools: []string{"connection_open", "connection_close", "connection_list", "ssh_exec", "sftp_transfer", "sftp_manage", "terminal_open", "terminal_interact", "terminal_view", "terminal_close"},
		},
		{
			name:  "basic",
			tools: []string{"connection_open", "connection_close", "connection_list", "ssh_exec", "sftp_transfer", "sftp_manage", "terminal_open", "terminal_interact", "terminal_view", "terminal_close"},
		},
		{
			name:  "advanced",
			tools: []string{"connection_open", "connection_close", "connection_list", "ssh_connect", "ssh_disconnect", "ssh_list_sessions", "ssh_exec", "ssh_exec_batch", "ssh_shell", "sftp_upload", "sftp_download", "sftp_list_dir", "sftp_mkdir", "sftp_delete", "ssh_write_input", "ssh_read_output", "ssh_resize_pty", "ssh_terminal_snapshot", "ssh_shell_status", "ssh_history", "ssh_list_hosts", "ssh_save_host", "ssh_remove_host", "terminal_open", "terminal_interact", "terminal_view", "terminal_close"},
		},
	}

	for _, profile := range profiles {
		t.Run(profile.name, func(t *testing.T) {
			assertStdioProfile(t, profile.name, profile.tools)
		})
	}
}

func assertStdioProfile(t *testing.T, profile string, expectedTools []string) {
	t.Helper()
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	config := fmt.Sprintf(`server:
  name: ssh-mcp-server
  version: test
session:
  max_sessions: 100
  max_sessions_per_host: 10
  idle_timeout: 10m
  session_timeout: 30m
  cleanup_interval: 1m
sftp:
  max_file_size: 1073741824
  chunk_size: 4194304
  transfer_timeout: 5m
tools:
  profile: %s
logging:
  level: info
  format: console
  output: stdout
`, profile)
	if err := os.WriteFile(configPath, []byte(config), 0600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cmd := exec.Command(os.Args[0], "-test.run=^TestStdioInitializeHasNoLogLines$", "--", "-config", configPath)
	cmd.Env = append(os.Environ(), "SSHMCP_SERVER_TEST_HELPER=1")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("create stdin pipe: %v", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("create stdout pipe: %v", err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatalf("start server: %v", err)
	}
	t.Cleanup(func() {
		_ = stdin.Close()
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		_ = cmd.Wait()
	})

	initialize := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"test-client","version":"1.0.0"}}}`
	if _, err := fmt.Fprintln(stdin, initialize); err != nil {
		t.Fatalf("send initialize request: %v", err)
	}

	lineCh := make(chan string, 4)
	errCh := make(chan error, 1)
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			lineCh <- scanner.Text()
		}
		errCh <- scanner.Err()
	}()

	line := receiveServerLine(t, lineCh, errCh, &stderr)

	var response struct {
		JSONRPC string          `json:"jsonrpc"`
		ID      int             `json:"id"`
		Result  json.RawMessage `json:"result"`
	}
	if err := json.Unmarshal([]byte(line), &response); err != nil {
		t.Fatalf("stdout contains non-JSON-RPC output %q: %v", line, err)
	}
	if response.JSONRPC != "2.0" || response.ID != 1 || len(response.Result) == 0 {
		t.Fatalf("unexpected initialize response: %s", line)
	}

	if _, err := fmt.Fprintln(stdin, `{"jsonrpc":"2.0","method":"notifications/initialized"}`); err != nil {
		t.Fatalf("send initialized notification: %v", err)
	}
	if _, err := fmt.Fprintln(stdin, `{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}`); err != nil {
		t.Fatalf("send tools/list request: %v", err)
	}

	var toolsResponse struct {
		Result struct {
			Tools []struct {
				Name         string          `json:"name"`
				OutputSchema json.RawMessage `json:"outputSchema"`
			} `json:"tools"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(receiveServerLine(t, lineCh, errCh, &stderr)), &toolsResponse); err != nil {
		t.Fatalf("decode tools/list response: %v", err)
	}

	tools := make(map[string]json.RawMessage, len(toolsResponse.Result.Tools))
	for _, tool := range toolsResponse.Result.Tools {
		tools[tool.Name] = tool.OutputSchema
	}
	if len(tools) != len(expectedTools) {
		t.Fatalf("%s profile returned %d tools, want %d", profile, len(tools), len(expectedTools))
	}
	for _, name := range expectedTools {
		if _, found := tools[name]; !found {
			t.Errorf("%s profile is missing %s", profile, name)
		}
	}
	expected := make(map[string]struct{}, len(expectedTools))
	for _, name := range expectedTools {
		expected[name] = struct{}{}
	}
	for name := range tools {
		if _, found := expected[name]; !found {
			t.Errorf("%s profile unexpectedly exposes %s", profile, name)
		}
	}
	if len(tools["ssh_exec"]) == 0 {
		t.Error("ssh_exec must expose an output schema")
	}
	if len(tools["terminal_interact"]) == 0 {
		t.Error("terminal_interact must expose an output schema")
	}

	if _, err := fmt.Fprintln(stdin, `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"connection_list","arguments":{}}}`); err != nil {
		t.Fatalf("send connection_list request: %v", err)
	}
	var callResponse struct {
		Result struct {
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
			IsError bool `json:"isError"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(receiveServerLine(t, lineCh, errCh, &stderr)), &callResponse); err != nil {
		t.Fatalf("decode connection_list response: %v", err)
	}
	if callResponse.Result.IsError || len(callResponse.Result.Content) != 1 || callResponse.Result.Content[0].Type != "text" {
		t.Fatalf("unexpected connection_list result: %+v", callResponse.Result)
	}
	var connections map[string]json.RawMessage
	if err := json.Unmarshal([]byte(callResponse.Result.Content[0].Text), &connections); err != nil {
		t.Fatalf("connection_list text is not JSON: %v", err)
	}
	if _, found := connections["connections"]; !found {
		t.Fatal("connection_list result is missing connections")
	}
	if _, found := connections["saved_ssh_hosts"]; !found {
		t.Fatal("connection_list result is missing saved_ssh_hosts")
	}
	if _, found := connections["serial_ports"]; !found {
		t.Fatal("connection_list result is missing serial_ports")
	}
}

func receiveServerLine(t *testing.T, lineCh <-chan string, errCh <-chan error, stderr *bytes.Buffer) string {
	t.Helper()
	select {
	case line := <-lineCh:
		return line
	case err := <-errCh:
		t.Fatalf("server closed stdout before responding: %v\nstderr:\n%s", err, stderr.String())
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for server response")
	}
	return ""
}
