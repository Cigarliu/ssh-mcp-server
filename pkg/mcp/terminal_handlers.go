package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/cigar/sshmcp/pkg/serialmcp"
	"github.com/cigar/sshmcp/pkg/sshmcp"
	"github.com/cigar/sshmcp/pkg/terminal"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func (s *Server) handleConnectionOpen(_ context.Context, _ *mcp.CallToolRequest, args map[string]any) (*mcp.CallToolResult, any, error) {
	transportName, _ := args["transport"].(string)
	switch transportName {
	case "ssh":
		if hasAnyArg(args, "device", "baud_rate", "data_bits", "parity", "stop_bits") {
			return terminalError(fmt.Errorf("SSH connections cannot include serial fields"))
		}
		session, err := s.openSSHConnection(args)
		if err != nil {
			return terminalError(err)
		}
		return terminalJSON(map[string]any{
			"connection_id": session.ID,
			"transport":     "ssh",
			"capabilities":  s.sshCapabilities(),
		})
	case "serial":
		if hasAnyArg(args, "hostname", "host", "port", "username", "password", "private_key", "passphrase", "sudo_password", "alias") {
			return terminalError(fmt.Errorf("serial connections cannot include SSH fields"))
		}
		device := stringArg(args, "device")
		if device == "" {
			return terminalError(fmt.Errorf("device is required for a serial connection"))
		}
		connection, err := s.serialManager.Open(serialmcp.Config{
			Device:   device,
			BaudRate: intArg(args, "baud_rate", 115200),
			DataBits: intArg(args, "data_bits", 8),
			Parity:   stringArgDefault(args, "parity", "none"),
			StopBits: stringArgDefault(args, "stop_bits", "1"),
		})
		if err != nil {
			return terminalError(err)
		}
		return terminalJSON(map[string]any{
			"connection_id": connection.ID,
			"transport":     "serial",
			"capabilities":  []string{"terminal"},
		})
	default:
		return terminalError(fmt.Errorf("transport must be ssh or serial"))
	}
}

func (s *Server) openSSHConnection(args map[string]any) (*sshmcp.Session, error) {
	var host, username, password, privateKey string
	port := 22
	if hostname := stringArg(args, "hostname"); hostname != "" {
		if hasAnyArg(args, "host", "port", "username", "password", "private_key") {
			return nil, fmt.Errorf("hostname cannot be combined with direct SSH address, port, username, or authentication fields")
		}
		saved, err := s.hostManager.GetHost(hostname)
		if err != nil {
			return nil, err
		}
		host = saved.Host
		username = saved.Username
		port = saved.Port
		if port == 0 {
			port = 22
		}
		password = saved.Password
		privateKey = saved.PrivateKeyPath
		if privateKey == "" && stringArg(args, "passphrase") != "" {
			return nil, fmt.Errorf("passphrase requires a saved SSH host with private_key_path")
		}
	} else {
		host = stringArg(args, "host")
		username = stringArg(args, "username")
		password = stringArg(args, "password")
		privateKey = stringArg(args, "private_key")
		if host == "" || username == "" {
			return nil, fmt.Errorf("host and username are required for a direct SSH connection")
		}
		if _, present := args["port"]; present {
			port = intArg(args, "port", 0)
		}
		if password != "" && privateKey != "" {
			return nil, fmt.Errorf("provide password or private_key for direct SSH, not both")
		}
		if privateKey == "" && stringArg(args, "passphrase") != "" {
			return nil, fmt.Errorf("passphrase requires private_key for direct SSH")
		}
	}
	if port < 1 || port > 65535 {
		return nil, fmt.Errorf("port must be between 1 and 65535")
	}
	auth := &sshmcp.AuthConfig{SudoPassword: stringArg(args, "sudo_password")}
	if privateKey != "" {
		auth.Type = sshmcp.AuthTypePrivateKey
		auth.PrivateKey = privateKey
		auth.Passphrase = stringArg(args, "passphrase")
	} else {
		auth.Type = sshmcp.AuthTypePassword
		auth.Password = password
		if auth.Password == "" {
			return nil, fmt.Errorf("password or private_key is required for an SSH connection")
		}
	}
	return s.sessionManager.CreateSession(host, port, username, auth, stringArg(args, "alias"))
}

func hasAnyArg(args map[string]any, keys ...string) bool {
	for _, key := range keys {
		if _, present := args[key]; present {
			return true
		}
	}
	return false
}

func (s *Server) sshCapabilities() []string {
	capabilities := []string{"exec", "terminal", "tui"}
	if s.profile == ToolProfileFiles || s.profile == ToolProfileAdvanced {
		capabilities = append(capabilities, "files")
	}
	return capabilities
}

func (s *Server) handleConnectionClose(_ context.Context, _ *mcp.CallToolRequest, args map[string]any) (*mcp.CallToolResult, any, error) {
	connectionID := stringArg(args, "connection_id")
	if connectionID == "" {
		return terminalError(fmt.Errorf("connection_id is required"))
	}
	if strings.HasPrefix(connectionID, "serial-") {
		s.terminals.RemoveByConnection(connectionID)
		if err := s.serialManager.Close(connectionID); err != nil {
			return terminalError(err)
		}
	} else {
		session, err := s.sessionManager.GetSessionByIDOrAlias(connectionID)
		if err != nil {
			return terminalError(err)
		}
		s.terminals.RemoveByConnection(session.ID)
		if err := s.sessionManager.RemoveSession(session.ID); err != nil {
			return terminalError(err)
		}
		connectionID = session.ID
	}
	return terminalJSON(map[string]any{"connection_id": connectionID, "closed": true})
}

func (s *Server) handleConnectionList(_ context.Context, _ *mcp.CallToolRequest, _ map[string]any) (*mcp.CallToolResult, any, error) {
	connections := make([]map[string]any, 0)
	for _, session := range s.sessionManager.ListSessions() {
		connections = append(connections, map[string]any{
			"connection_id": session.ID,
			"transport":     "ssh",
			"host":          session.Host,
			"username":      session.Username,
			"alias":         session.Alias,
			"capabilities":  s.sshCapabilities(),
		})
	}
	for _, connection := range s.serialManager.List() {
		connections = append(connections, map[string]any{
			"connection_id": connection.ID,
			"transport":     "serial",
			"device":        connection.Device,
			"baud_rate":     connection.BaudRate,
			"capabilities":  []string{"terminal"},
		})
	}
	sort.Slice(connections, func(i, j int) bool {
		leftTransport := connections[i]["transport"].(string)
		rightTransport := connections[j]["transport"].(string)
		if leftTransport != rightTransport {
			return leftTransport < rightTransport
		}
		return connections[i]["connection_id"].(string) < connections[j]["connection_id"].(string)
	})
	savedHosts := make([]map[string]any, 0)
	for name, host := range s.hostManager.ListHosts() {
		savedHosts = append(savedHosts, map[string]any{
			"name": name, "host": host.Host, "port": host.Port, "username": host.Username, "description": host.Description,
		})
	}
	sort.Slice(savedHosts, func(i, j int) bool { return savedHosts[i]["name"].(string) < savedHosts[j]["name"].(string) })
	ports, err := serialmcp.ListPorts()
	payload := map[string]any{"connections": connections, "saved_ssh_hosts": savedHosts, "serial_ports": ports}
	if err != nil {
		payload["serial_error"] = err.Error()
	}
	return terminalJSON(payload)
}

func (s *Server) handleTerminalOpen(_ context.Context, _ *mcp.CallToolRequest, args map[string]any) (*mcp.CallToolResult, any, error) {
	connectionID := stringArg(args, "connection_id")
	profile := stringArgDefault(args, "profile", "shell")
	if profile != "shell" && profile != "tui" {
		return terminalError(fmt.Errorf("profile must be shell or tui"))
	}
	if strings.HasPrefix(connectionID, "serial-") {
		if profile == "tui" {
			return terminalError(fmt.Errorf("serial terminals do not provide a TUI screen; use profile shell"))
		}
		if workingDir := stringArg(args, "working_dir"); workingDir != "" {
			return terminalError(fmt.Errorf("working_dir is supported only for SSH terminals"))
		}
		if existing := s.terminals.FindByConnection(connectionID); existing != nil {
			if existing.Profile != profile {
				return terminalError(fmt.Errorf("terminal %q is already open with profile %q; close it before reopening with %q", existing.ID, existing.Profile, profile))
			}
			return terminalOpenResult(existing)
		}
		connection, err := s.serialManager.Get(connectionID)
		if err != nil {
			return terminalError(err)
		}
		entry := s.terminals.Register(connectionID, "serial", "shell", connection.Terminal(), func() error {
			return s.serialManager.Close(connectionID)
		}, nil)
		return terminalOpenResult(entry)
	}

	session, err := s.sessionManager.GetSessionByIDOrAlias(connectionID)
	if err != nil {
		return terminalError(err)
	}
	connectionID = session.ID
	if existing := s.terminals.FindByConnection(connectionID); existing != nil {
		if existing.Profile != profile {
			return terminalError(fmt.Errorf("terminal %q is already open with profile %q; close it before reopening with %q", existing.ID, existing.Profile, profile))
		}
		if stringArg(args, "working_dir") != "" {
			return terminalError(fmt.Errorf("working_dir can only be set when opening a new terminal; use terminal_interact to change an existing shell directory"))
		}
		return terminalOpenResult(existing)
	}
	rows, err := terminalDimension("rows", numberArg(args, "rows"), 40, maxTerminalRows)
	if err != nil {
		return terminalError(err)
	}
	cols, err := terminalDimension("cols", numberArg(args, "cols"), 160, maxTerminalCols)
	if err != nil {
		return terminalError(err)
	}

	shell := session.GetShellSession()
	if shell != nil && shell.Terminal != nil && !shell.Terminal.Status().Closed {
		isRaw := shell.Config != nil && shell.Config.Mode == sshmcp.TerminalModeRaw
		if (profile == "tui") != isRaw {
			return terminalError(fmt.Errorf("an existing SSH shell uses a different terminal mode; close the connection before opening profile %q", profile))
		}
	}
	if shell == nil || shell.Terminal == nil || shell.Terminal.Status().Closed {
		config := sshmcp.DefaultShellConfig()
		config.ANSIMode = sshmcp.ANSIRaw
		if profile == "tui" {
			config.Mode = sshmcp.TerminalModeRaw
		}
		shell, err = session.CreateShellWithConfig("xterm-256color", rows, cols, config)
		if err != nil {
			return terminalError(err)
		}
	}
	if workingDir := stringArg(args, "working_dir"); workingDir != "" {
		if err := shell.WriteInput("cd -- " + shellQuote(workingDir) + "\n"); err != nil {
			return terminalError(err)
		}
	}
	entry := s.terminals.Register(session.ID, "ssh", profile, shell.Terminal, shell.Close, func(width, height int) error {
		return shell.Resize(uint16(height), uint16(width))
	})
	return terminalOpenResult(entry)
}

func (s *Server) handleTerminalInteract(ctx context.Context, _ *mcp.CallToolRequest, args map[string]any) (*mcp.CallToolResult, any, error) {
	entry, err := s.terminals.Get(stringArg(args, "terminal_id"))
	if err != nil {
		return terminalError(err)
	}
	input, err := terminalInput(stringArg(args, "input"), stringArgDefault(args, "input_mode", "raw"))
	if err != nil {
		return terminalError(err)
	}
	waitKind := terminal.WaitKind(stringArgDefault(args, "wait", string(terminal.WaitQuiet)))
	switch waitKind {
	case terminal.WaitNone, terminal.WaitUntil, terminal.WaitQuiet:
	default:
		return terminalError(fmt.Errorf("invalid wait value %q", waitKind))
	}
	quietMS, err := boundedIntArg(args, "quiet_ms", 150, 1, 10000)
	if err != nil {
		return terminalError(err)
	}
	maxBytes, err := boundedIntArg(args, "max_bytes", terminal.DefaultMaxBytes, 1, 65536)
	if err != nil {
		return terminalError(err)
	}
	timeoutMS, err := boundedIntArg(args, "timeout_ms", 3000, 1, 60000)
	if err != nil {
		return terminalError(err)
	}
	request := terminal.InteractRequest{
		Input: input,
		Wait: terminal.Wait{
			Kind:  waitKind,
			Until: stringArg(args, "until"),
			Quiet: time.Duration(quietMS) * time.Millisecond,
		},
		MaxBytes: maxBytes,
	}
	if offset, ok := uintArg(args, "from_offset"); ok {
		request.FromOffset = &offset
	}
	timeout := time.Duration(timeoutMS) * time.Millisecond
	callCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	result, err := entry.Session.Interact(callCtx, request)
	if err != nil {
		return terminalError(err)
	}
	return terminalJSON(result)
}

func (s *Server) handleTerminalView(_ context.Context, _ *mcp.CallToolRequest, args map[string]any) (*mcp.CallToolResult, any, error) {
	entry, err := s.terminals.Get(stringArg(args, "terminal_id"))
	if err != nil {
		return terminalError(err)
	}
	if entry.Profile != "tui" {
		return terminalError(fmt.Errorf("terminal %q uses profile %q; terminal_view requires profile tui", entry.ID, entry.Profile))
	}
	screen := entry.Session.Screen()
	if screen == nil {
		return terminalError(fmt.Errorf("terminal %q has no screen projection", entry.ID))
	}
	content := screen.Snapshot()
	maxChars := intArg(args, "max_chars", 8000)
	content, truncated := truncateText(content, maxChars)
	x, y := screen.Cursor()
	width, height := screen.Size()
	return terminalJSON(map[string]any{
		"screen":      content,
		"screen_hash": terminal.ScreenHash(screen),
		"cursor":      map[string]int{"x": x, "y": y},
		"size":        map[string]int{"width": width, "height": height},
		"truncated":   truncated,
	})
}

func (s *Server) handleTerminalClose(_ context.Context, _ *mcp.CallToolRequest, args map[string]any) (*mcp.CallToolResult, any, error) {
	id := stringArg(args, "terminal_id")
	if err := s.terminals.Close(id); err != nil {
		return terminalError(err)
	}
	return terminalJSON(map[string]any{"terminal_id": id, "closed": true})
}

func terminalOpenResult(entry *terminal.Entry) (*mcp.CallToolResult, any, error) {
	status := entry.Session.Status()
	return terminalJSON(map[string]any{
		"terminal_id":   entry.ID,
		"connection_id": entry.ConnectionID,
		"transport":     entry.Transport,
		"profile":       entry.Profile,
		"screen":        entry.Profile == "tui" && entry.Session.Screen() != nil,
		"start_offset":  status.StartOffset,
		"end_offset":    status.EndOffset,
	})
}

func terminalJSON(value any) (*mcp.CallToolResult, any, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return terminalError(fmt.Errorf("encode terminal result: %w", err))
	}
	return &mcp.CallToolResult{Content: textContent(string(encoded))}, value, nil
}

func terminalError(err error) (*mcp.CallToolResult, any, error) {
	return &mcp.CallToolResult{Content: textContent(err.Error()), IsError: true}, nil, nil
}

func stringArg(args map[string]any, key string) string {
	value, _ := args[key].(string)
	return value
}

func stringArgDefault(args map[string]any, key, fallback string) string {
	if value := stringArg(args, key); value != "" {
		return value
	}
	return fallback
}

func numberArg(args map[string]any, key string) float64 {
	switch value := args[key].(type) {
	case float64:
		return value
	case float32:
		return float64(value)
	case int:
		return float64(value)
	case int64:
		return float64(value)
	default:
		return 0
	}
}

func intArg(args map[string]any, key string, fallback int) int {
	if value := numberArg(args, key); value > 0 && value == float64(int(value)) {
		return int(value)
	}
	return fallback
}

func boundedIntArg(args map[string]any, key string, fallback, minimum, maximum int) (int, error) {
	if _, present := args[key]; !present {
		return fallback, nil
	}
	value := numberArg(args, key)
	if value != float64(int(value)) || value < float64(minimum) || value > float64(maximum) {
		return 0, fmt.Errorf("%s must be an integer between %d and %d", key, minimum, maximum)
	}
	return int(value), nil
}

func uintArg(args map[string]any, key string) (uint64, bool) {
	value := numberArg(args, key)
	if value < 0 || value != float64(uint64(value)) {
		return 0, false
	}
	if _, present := args[key]; !present {
		return 0, false
	}
	return uint64(value), true
}

func terminalInput(input, mode string) ([]byte, error) {
	switch mode {
	case "raw":
		return []byte(input), nil
	case "line":
		if input != "" && !strings.HasSuffix(input, "\n") {
			input += "\n"
		}
		return []byte(input), nil
	case "key":
		keys := map[string]string{
			"enter": "\r", "tab": "\t", "esc": "\x1b", "ctrl+c": "\x03", "ctrl+d": "\x04", "ctrl+z": "\x1a",
			"up": "\x1b[A", "down": "\x1b[B", "right": "\x1b[C", "left": "\x1b[D",
		}
		value, ok := keys[strings.ToLower(input)]
		if !ok {
			return nil, fmt.Errorf("unsupported key %q", input)
		}
		return []byte(value), nil
	default:
		return nil, fmt.Errorf("input_mode must be raw, line, or key")
	}
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}
