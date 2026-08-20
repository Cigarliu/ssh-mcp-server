// Package serialmcp owns physical serial connections and adapts them to terminal sessions.
package serialmcp

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/cigar/sshmcp/pkg/terminal"
	"github.com/google/uuid"
	goserial "go.bug.st/serial"
)

type Config struct {
	Device   string
	BaudRate int
	DataBits int
	Parity   string
	StopBits string
}

type Connection struct {
	ID     string
	Config Config
	port   goserial.Port

	mu       sync.Mutex
	terminal *terminal.Session
}

type ConnectionInfo struct {
	ID       string `json:"connection_id"`
	Device   string `json:"device"`
	BaudRate int    `json:"baud_rate"`
	DataBits int    `json:"data_bits"`
	Parity   string `json:"parity"`
	StopBits string `json:"stop_bits"`
}

type Manager struct {
	mu          sync.RWMutex
	connections map[string]*Connection
}

var getPortsList = goserial.GetPortsList

func NewManager() *Manager {
	return &Manager{connections: make(map[string]*Connection)}
}

func ListPorts() ([]string, error) {
	ports, err := getPortsList()
	if ports == nil {
		ports = []string{}
	}
	return ports, err
}

func (m *Manager) Open(config Config) (*Connection, error) {
	config.Device = strings.TrimSpace(config.Device)
	if config.Device == "" {
		return nil, fmt.Errorf("serial device is required")
	}
	mode, normalized, err := buildMode(config)
	if err != nil {
		return nil, err
	}
	m.mu.RLock()
	for _, existing := range m.connections {
		if existing.Config.Device == normalized.Device {
			m.mu.RUnlock()
			return nil, fmt.Errorf("serial device %q is already open", normalized.Device)
		}
	}
	m.mu.RUnlock()

	port, err := goserial.Open(normalized.Device, mode)
	if err != nil {
		if errors.Is(err, os.ErrPermission) {
			return nil, fmt.Errorf("open serial device %q: permission denied; run the MCP server as an account with serial-device access", normalized.Device)
		}
		return nil, fmt.Errorf("open serial device %q: %w", normalized.Device, err)
	}
	if err := port.SetReadTimeout(250 * time.Millisecond); err != nil {
		_ = port.Close()
		return nil, fmt.Errorf("configure serial read timeout: %w", err)
	}
	connection := &Connection{
		ID:     "serial-" + uuid.NewString()[:8],
		Config: normalized,
		port:   port,
	}
	m.mu.Lock()
	m.connections[connection.ID] = connection
	m.mu.Unlock()
	return connection, nil
}

func (m *Manager) Get(id string) (*Connection, error) {
	m.mu.RLock()
	connection := m.connections[id]
	m.mu.RUnlock()
	if connection == nil {
		return nil, fmt.Errorf("serial connection %q not found", id)
	}
	return connection, nil
}

func (m *Manager) List() []ConnectionInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()
	connections := make([]ConnectionInfo, 0, len(m.connections))
	for _, connection := range m.connections {
		connections = append(connections, ConnectionInfo{
			ID:       connection.ID,
			Device:   connection.Config.Device,
			BaudRate: connection.Config.BaudRate,
			DataBits: connection.Config.DataBits,
			Parity:   connection.Config.Parity,
			StopBits: connection.Config.StopBits,
		})
	}
	return connections
}

func (m *Manager) Close(id string) error {
	m.mu.Lock()
	connection := m.connections[id]
	if connection != nil {
		delete(m.connections, id)
	}
	m.mu.Unlock()
	if connection == nil {
		return fmt.Errorf("serial connection %q not found", id)
	}
	return connection.close()
}

func (m *Manager) CloseAll() {
	m.mu.Lock()
	connections := m.connections
	m.connections = make(map[string]*Connection)
	m.mu.Unlock()
	for _, connection := range connections {
		_ = connection.close()
	}
}

func (c *Connection) Terminal() *terminal.Session {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.terminal == nil {
		c.terminal = terminal.New(c.port, c.port, terminal.Config{})
	}
	return c.terminal
}

func (c *Connection) close() error {
	c.mu.Lock()
	if c.terminal != nil {
		c.terminal.Close()
	}
	port := c.port
	c.mu.Unlock()
	if port != nil {
		return port.Close()
	}
	return nil
}

func buildMode(config Config) (*goserial.Mode, Config, error) {
	if config.BaudRate == 0 {
		config.BaudRate = 115200
	}
	if config.BaudRate < 1 || config.BaudRate > 4_000_000 {
		return nil, config, fmt.Errorf("baud_rate must be between 1 and 4000000")
	}
	if config.DataBits == 0 {
		config.DataBits = 8
	}
	if config.DataBits < 5 || config.DataBits > 8 {
		return nil, config, fmt.Errorf("data_bits must be 5, 6, 7, or 8")
	}
	if config.Parity == "" {
		config.Parity = "none"
	}
	if config.StopBits == "" {
		config.StopBits = "1"
	}
	parity, ok := map[string]goserial.Parity{
		"none":  goserial.NoParity,
		"odd":   goserial.OddParity,
		"even":  goserial.EvenParity,
		"mark":  goserial.MarkParity,
		"space": goserial.SpaceParity,
	}[strings.ToLower(config.Parity)]
	if !ok {
		return nil, config, fmt.Errorf("parity must be none, odd, even, mark, or space")
	}
	stopBits, ok := map[string]goserial.StopBits{
		"1":   goserial.OneStopBit,
		"1.5": goserial.OnePointFiveStopBits,
		"2":   goserial.TwoStopBits,
	}[config.StopBits]
	if !ok {
		return nil, config, fmt.Errorf("stop_bits must be 1, 1.5, or 2")
	}
	config.Parity = strings.ToLower(config.Parity)
	return &goserial.Mode{
		BaudRate: config.BaudRate,
		DataBits: config.DataBits,
		Parity:   parity,
		StopBits: stopBits,
	}, config, nil
}
