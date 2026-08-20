package sshmcp

import (
	"fmt"
	"os"
	"regexp"
	"sync"

	"github.com/cigar/sshmcp/internal/state"
	"github.com/rs/zerolog"
	"gopkg.in/yaml.v3"
)

var connectionIDPattern = regexp.MustCompile(`^[a-z][a-z0-9][a-z0-9_-]{1,62}$`)

// HostConfig represents a predefined SSH host configuration
type HostConfig struct {
	Host           string `mapstructure:"host" yaml:"host"`
	Port           int    `mapstructure:"port" yaml:"port"`
	Username       string `mapstructure:"username" yaml:"username"`
	Password       string `mapstructure:"password,omitempty" yaml:"password,omitempty"`
	PrivateKeyPath string `mapstructure:"private_key_path,omitempty" yaml:"private_key_path,omitempty"`
	Description    string `mapstructure:"description,omitempty" yaml:"description,omitempty"`
}

// HostManager manages predefined SSH hosts
type HostManager struct {
	hosts      map[string]HostConfig
	configPath string
	mu         sync.RWMutex
	logger     *zerolog.Logger
	stateStore *state.Store
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

// NewHostManagerWithStore makes SQLite the source of truth for durable
// connection profiles. YAML hosts are imported without overwriting records
// created by another MCP instance.
func NewHostManagerWithStore(hostsConfig map[string]HostConfig, configPath string, logger *zerolog.Logger, store *state.Store) (*HostManager, error) {
	hm := NewHostManager(hostsConfig, configPath, logger)
	hm.stateStore = store
	if store == nil {
		return hm, nil
	}
	profiles := make([]state.ConnectionProfile, 0, len(hostsConfig))
	for id, host := range hostsConfig {
		profiles = append(profiles, state.ConnectionProfile{
			ID: id, Description: host.Description, Host: host.Host, Port: host.Port,
			Username: host.Username, Password: host.Password, PrivateKeyPath: host.PrivateKeyPath,
		})
	}
	if err := store.SeedProfiles(profiles); err != nil {
		return nil, err
	}
	return hm, nil
}

// ValidateConnectionID requires a readable, stable identifier selected by the
// MCP client rather than a generated runtime UUID.
func ValidateConnectionID(id string) error {
	if !connectionIDPattern.MatchString(id) {
		return fmt.Errorf("connection_id must match %s", connectionIDPattern.String())
	}
	return nil
}

// ListHosts returns all predefined hosts
func (hm *HostManager) ListHosts() map[string]HostConfig {
	if hm.stateStore != nil {
		hosts, err := hm.ListPersistentHosts()
		if err != nil {
			if hm.logger != nil {
				hm.logger.Error().Err(err).Msg("List persistent connection profiles")
			}
			return map[string]HostConfig{}
		}
		return hosts
	}
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	result := make(map[string]HostConfig)
	for name, host := range hm.hosts {
		result[name] = host
	}

	return result
}

// ListPersistentHosts reads from the shared SQLite directory when configured.
func (hm *HostManager) ListPersistentHosts() (map[string]HostConfig, error) {
	if hm.stateStore == nil {
		return hm.ListHosts(), nil
	}
	profiles, err := hm.stateStore.ListProfiles()
	if err != nil {
		return nil, err
	}
	hosts := make(map[string]HostConfig, len(profiles))
	for _, profile := range profiles {
		hosts[profile.ID] = hostConfigFromProfile(profile)
	}
	return hosts, nil
}

// GetHost retrieves a host configuration by name
func (hm *HostManager) GetHost(name string) (HostConfig, error) {
	if hm.stateStore != nil {
		profile, err := hm.stateStore.GetProfile(name)
		if err != nil {
			return HostConfig{}, err
		}
		return hostConfigFromProfile(profile), nil
	}
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
	if hm.stateStore != nil {
		_, err := hm.stateStore.GetProfile(name)
		return err == nil
	}
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	_, exists := hm.hosts[name]
	return exists
}

// SaveHost saves a new host configuration
func (hm *HostManager) SaveHost(name string, hostCfg HostConfig) error {
	if hm.stateStore != nil {
		if err := ValidateConnectionID(name); err != nil {
			return err
		}
		if err := validateHostConfig(&hostCfg); err != nil {
			return err
		}
		if err := hm.stateStore.CreateProfile(profileFromHostConfig(name, hostCfg)); err != nil {
			return err
		}
		if hm.logger != nil {
			hm.logger.Info().Str("connection_id", name).Msg("Saved persistent connection profile")
		}
		return nil
	}
	hm.mu.Lock()
	defer hm.mu.Unlock()

	// Check if host already exists
	if _, exists := hm.hosts[name]; exists {
		return fmt.Errorf("host '%s' already exists", name)
	}
	if hostCfg.Host == "" {
		return fmt.Errorf("host address cannot be empty")
	}
	if hostCfg.Username == "" {
		return fmt.Errorf("username cannot be empty")
	}
	if hostCfg.Port == 0 {
		hostCfg.Port = 22
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
	if hm.stateStore != nil {
		return hm.stateStore.DeleteProfile(name)
	}
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

func validateHostConfig(hostCfg *HostConfig) error {
	if hostCfg.Host == "" {
		return fmt.Errorf("host address cannot be empty")
	}
	if hostCfg.Username == "" {
		return fmt.Errorf("username cannot be empty")
	}
	if hostCfg.Port == 0 {
		hostCfg.Port = 22
	}
	if hostCfg.Description == "" {
		return fmt.Errorf("connection description cannot be empty")
	}
	if hostCfg.Password != "" && hostCfg.PrivateKeyPath != "" {
		return fmt.Errorf("provide password or private_key_path, not both")
	}
	if hostCfg.Password == "" && hostCfg.PrivateKeyPath == "" {
		return fmt.Errorf("password or private_key_path is required")
	}
	return nil
}

func profileFromHostConfig(id string, host HostConfig) state.ConnectionProfile {
	return state.ConnectionProfile{
		ID: id, Description: host.Description, Host: host.Host, Port: host.Port,
		Username: host.Username, Password: host.Password, PrivateKeyPath: host.PrivateKeyPath,
	}
}

func hostConfigFromProfile(profile state.ConnectionProfile) HostConfig {
	return HostConfig{
		Host: profile.Host, Port: profile.Port, Username: profile.Username,
		Password: profile.Password, PrivateKeyPath: profile.PrivateKeyPath, Description: profile.Description,
	}
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
