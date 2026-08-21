package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestStdioInitializeHasNoLogLines(t *testing.T) {
	if os.Getenv("SSHMCP_SERVER_TEST_HELPER") == "1" {
		main()
		return
	}

	assertStdioTools(t, []string{"connection_open", "connection_close", "connection_list", "connection_history", "ssh_exec", "sftp_transfer", "sftp_manage", "terminal_open", "terminal_interact", "terminal_view", "terminal_close"})
}

func assertStdioTools(t *testing.T, expectedTools []string) {
	t.Helper()
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	statePath := filepath.ToSlash(filepath.Join(filepath.Dir(configPath), "state.db"))
	config := fmt.Sprintf(`session:
  max_sessions: 100
  max_sessions_per_host: 10
  idle_timeout: 10m
  session_timeout: 30m
  cleanup_interval: 1m
sftp:
  max_file_size: 1073741824
  chunk_size: 4194304
  transfer_timeout: 5m
state:
  database_path: %q
logging:
  level: info
  format: console
  output: stdout
`, statePath)
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

	initializeResult := receiveResponse(t, lineCh, errCh, &stderr, 1)
	if len(initializeResult) == 0 {
		t.Fatal("initialize response has no result")
	}

	if _, err := fmt.Fprintln(stdin, `{"jsonrpc":"2.0","method":"notifications/initialized"}`); err != nil {
		t.Fatalf("send initialized notification: %v", err)
	}
	if _, err := fmt.Fprintln(stdin, `{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}`); err != nil {
		t.Fatalf("send tools/list request: %v", err)
	}

	var toolsResponse struct {
		Tools []struct {
			Name         string          `json:"name"`
			Description  string          `json:"description"`
			InputSchema  json.RawMessage `json:"inputSchema"`
			OutputSchema json.RawMessage `json:"outputSchema"`
		} `json:"tools"`
	}
	if err := json.Unmarshal(receiveResponse(t, lineCh, errCh, &stderr, 2), &toolsResponse); err != nil {
		t.Fatalf("decode tools/list response: %v", err)
	}

	type toolMetadata struct {
		description  string
		inputSchema  json.RawMessage
		outputSchema json.RawMessage
	}
	tools := make(map[string]toolMetadata, len(toolsResponse.Tools))
	for _, tool := range toolsResponse.Tools {
		if tool.Description == "" {
			t.Errorf("%s has an empty model-facing description", tool.Name)
		}
		if len(tool.InputSchema) == 0 {
			t.Errorf("%s has no input schema", tool.Name)
		}
		if strings.Contains(string(tool.InputSchema), `"session_id"`) {
			t.Errorf("%s input schema unexpectedly exposes session_id: %s", tool.Name, tool.InputSchema)
		}
		tools[tool.Name] = toolMetadata{description: tool.Description, inputSchema: tool.InputSchema, outputSchema: tool.OutputSchema}
	}
	if len(tools) != len(expectedTools) {
		t.Fatalf("server returned %d tools, want %d", len(tools), len(expectedTools))
	}
	for _, name := range expectedTools {
		if _, found := tools[name]; !found {
			t.Errorf("server is missing %s", name)
		}
	}
	expected := make(map[string]struct{}, len(expectedTools))
	for _, name := range expectedTools {
		expected[name] = struct{}{}
	}
	for name := range tools {
		if _, found := expected[name]; !found {
			t.Errorf("server unexpectedly exposes %s", name)
		}
	}
	if len(tools["ssh_exec"].outputSchema) == 0 {
		t.Error("ssh_exec must expose an output schema")
	}
	if len(tools["terminal_interact"].outputSchema) == 0 {
		t.Error("terminal_interact must expose an output schema")
	}
	for name, phrase := range map[string]string{
		"connection_open":   "persisted only after success",
		"terminal_interact": "next_offset",
		"sftp_transfer":     "overwrite=true",
		"sftp_manage":       "irreversible",
	} {
		if !strings.Contains(tools[name].description, phrase) {
			t.Errorf("%s description must include %q, got %q", name, phrase, tools[name].description)
		}
	}

	if _, err := fmt.Fprintln(stdin, `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"connection_list","arguments":{}}}`); err != nil {
		t.Fatalf("send connection_list request: %v", err)
	}
	var callResponse struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		IsError bool `json:"isError"`
	}
	if err := json.Unmarshal(receiveResponse(t, lineCh, errCh, &stderr, 3), &callResponse); err != nil {
		t.Fatalf("decode connection_list response: %v", err)
	}
	if callResponse.IsError || len(callResponse.Content) != 1 || callResponse.Content[0].Type != "text" {
		t.Fatalf("unexpected connection_list result: %+v", callResponse)
	}
	var connections map[string]json.RawMessage
	if err := json.Unmarshal([]byte(callResponse.Content[0].Text), &connections); err != nil {
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

func receiveResponse(t *testing.T, lineCh <-chan string, errCh <-chan error, stderr *bytes.Buffer, expectedID int) json.RawMessage {
	t.Helper()
	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()

	for {
		select {
		case line := <-lineCh:
			var response struct {
				JSONRPC string          `json:"jsonrpc"`
				ID      json.RawMessage `json:"id"`
				Result  json.RawMessage `json:"result"`
				Error   json.RawMessage `json:"error"`
			}
			if err := json.Unmarshal([]byte(line), &response); err != nil {
				t.Fatalf("stdout contains non-JSON-RPC output %q: %v", line, err)
			}
			if response.JSONRPC != "2.0" || len(response.ID) == 0 {
				continue
			}
			var id int
			if err := json.Unmarshal(response.ID, &id); err != nil || id != expectedID {
				continue
			}
			if len(response.Error) != 0 && string(response.Error) != "null" {
				t.Fatalf("request %d failed: %s", expectedID, response.Error)
			}
			return response.Result
		case err := <-errCh:
			t.Fatalf("server closed stdout before response %d: %v\nstderr:\n%s", expectedID, err, stderr.String())
		case <-timer.C:
			t.Fatalf("timed out waiting for response %d", expectedID)
		}
	}
}
