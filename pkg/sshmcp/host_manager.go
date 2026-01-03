package sshmcp

import (
	"fmt"
	"os"
	"sync"

	"github.com/rs/zerolog"
	"gopkg.in/yaml.v3"
)

// HostConfig represents a predefined SSH host configuration
type HostConfig struct {
	Host            string `mapstructure:"host" yaml:"host"`
	Port            int    `mapstructure:"port" yaml:"port"`
	Username        string `mapstructure:"username" yaml:"username"`
	Password        string `mapstructure:"password,omitempty" yaml:"password,omitempty"`
	PrivateKeyPath  string `mapstructure:"private_key_path,omitempty" yaml:"private_key_path,omitempty"`
	Description     string `mapstructure:"description,omitempty" yaml:"description,omitempty"`
}

// HostManager manages predefined SSH hosts
type HostManager struct {
	hosts      map[string]HostConfig
	configPath string
	mu         sync.RWMutex
	logger     *zerolog.Logger
}

// NewHostManager creates a new host manager
func NewHostManager(hostsConfig map[string]HostConfig, configPath string, logger *zerolog.Logger) *HostManager {
	hm := &HostManager{
		hosts:      make(map[string]HostConfig),
		configPath: configPath,
		logger:     logger,
	}

	// Load predefined hosts from config
	for name, hostCfg := range hostsConfig {
		hm.hosts[name] = hostCfg
	}

	return hm
}

// ListHosts returns all predefined hosts
func (hm *HostManager) ListHosts() map[string]HostConfig {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	result := make(map[string]HostConfig)
	for name, host := range hm.hosts {
		result[name] = host
	}

	return result
}

// GetHost retrieves a host configuration by name
func (hm *HostManager) GetHost(name string) (HostConfig, error) {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	host, ok := hm.hosts[name]
	if !ok {
		return HostConfig{}, fmt.Errorf("host '%s' not found", name)
	}

	return host, nil
}

// HostExists checks if a host name exists
func (hm *HostManager) HostExists(name string) bool {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	_, exists := hm.hosts[name]
	return exists
}

// SaveHost saves a new host configuration
func (hm *HostManager) SaveHost(name string, hostCfg HostConfig) error {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	// Check if host already exists
	if _, exists := hm.hosts[name]; exists {
		return fmt.Errorf("host '%s' already exists", name)
	}

	// Validate host configuration
	if hostCfg.Host == "" {
		return fmt.Errorf("host address cannot be empty")
	}

	if hostCfg.Username == "" {
		return fmt.Errorf("username cannot be empty")
	}

	if hostCfg.Port == 0 {
		hostCfg.Port = 22 // Default to port 22
	}

	// Add to memory
	hm.hosts[name] = hostCfg

	// Persist to config file
	if err := hm.persist(); err != nil {
		// Rollback on error
		delete(hm.hosts, name)
		return fmt.Errorf("failed to persist host: %w", err)
	}

	hm.logger.Info().
		Str("name", name).
		Str("host", hostCfg.Host).
		Int("port", hostCfg.Port).
		Str("username", hostCfg.Username).
		Msg("Saved host configuration")

	return nil
}

// RemoveHost removes a host configuration
func (hm *HostManager) RemoveHost(name string) error {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	if _, exists := hm.hosts[name]; !exists {
		return fmt.Errorf("host '%s' not found", name)
	}

	// Remove from memory
	delete(hm.hosts, name)

	// Persist to config file
	if err := hm.persist(); err != nil {
		// Rollback on error
		hm.hosts[name] = HostConfig{}
		return fmt.Errorf("failed to persist changes: %w", err)
	}

	hm.logger.Info().
		Str("name", name).
		Msg("Removed host configuration")

	return nil
}

// persist saves the current hosts configuration to the config file
func (hm *HostManager) persist() error {
	if hm.configPath == "" {
		return fmt.Errorf("config path not set, cannot persist hosts")
	}

	// Read the current config file
	configData, err := os.ReadFile(hm.configPath)
	if err != nil {
		return fmt.Errorf("read config file: %w", err)
	}

	// Parse the YAML
	var configMap map[string]interface{}
	if err := yaml.Unmarshal(configData, &configMap); err != nil {
		return fmt.Errorf("parse config: %w", err)
	}

	// Update the hosts section
	hostsMap := make(map[string]interface{})
	for name, host := range hm.hosts {
		hostMap := map[string]interface{}{
			"host":     host.Host,
			"port":     host.Port,
			"username": host.Username,
		}
		if host.Password != "" {
			hostMap["password"] = host.Password
		}
		if host.PrivateKeyPath != "" {
			hostMap["private_key_path"] = host.PrivateKeyPath
		}
		if host.Description != "" {
			hostMap["description"] = host.Description
		}
		hostsMap[name] = hostMap
	}

	configMap["hosts"] = hostsMap

	// Marshal back to YAML
	updatedData, err := yaml.Marshal(configMap)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	// Write back to file
	if err := os.WriteFile(hm.configPath, updatedData, 0644); err != nil {
		return fmt.Errorf("write config file: %w", err)
	}

	return nil
}

// FormatHostList formats the host list for display
func (hm *HostManager) FormatHostList() string {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	if len(hm.hosts) == 0 {
		return "No predefined hosts configured"
	}

	result := "Predefined hosts:\n"
	for name, host := range hm.hosts {
		result += fmt.Sprintf("  %s:\n", name)
		result += fmt.Sprintf("    Host: %s:%d\n", host.Host, host.Port)
		result += fmt.Sprintf("    Username: %s\n", host.Username)
		if host.Description != "" {
			result += fmt.Sprintf("    Description: %s\n", host.Description)
		}
		if host.Password != "" {
			result += "    Auth: password\n"
		} else if host.PrivateKeyPath != "" {
			result += fmt.Sprintf("    Auth: private key (%s)\n", host.PrivateKeyPath)
		}
		result += "\n"
	}

	return result
}
