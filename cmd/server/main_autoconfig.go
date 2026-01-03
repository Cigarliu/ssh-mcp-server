package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/cigar/sshmcp/internal/config"
)

// loadConfig loads configuration from multiple locations with priority:
// 1. --config flag (highest priority)
// 2. .mcp.yaml in current directory
// 3. .sshmcp.yaml in current directory
// 4. ~/.sshmcp.yaml (home directory)
// 5. ~/.sshmcp/config.yaml (auto-generated if not exists)
func loadConfig() (*config.Config, error) {
	cfg, _, err := loadConfigWithPath()
	return cfg, err
}

// loadConfigWithPath loads configuration and returns both config and config file path
func loadConfigWithPath() (*config.Config, string, error) {
	// Check for --config flag
	args := os.Args
	for i, arg := range args {
		if arg == "-config" && i+1 < len(args) {
			configPath := args[i+1]
			fmt.Fprintf(os.Stderr, "Loading config from: %s\n", configPath)
			cfg, err := config.LoadConfig(configPath)
			return cfg, configPath, err
		}
	}

	// Check for .mcp.yaml in current directory
	if _, err := os.Stat(".mcp.yaml"); err == nil {
		configPath := ".mcp.yaml"
		fmt.Fprintf(os.Stderr, "Loading config from: %s (current directory)\n", configPath)
		cfg, err := config.LoadConfig(configPath)
		return cfg, configPath, err
	}

	// Check for .sshmcp.yaml in current directory
	if _, err := os.Stat(".sshmcp.yaml"); err == nil {
		configPath := ".sshmcp.yaml"
		fmt.Fprintf(os.Stderr, "Loading config from: %s (current directory)\n", configPath)
		cfg, err := config.LoadConfig(configPath)
		return cfg, configPath, err
	}

	// Check for ~/.sshmcp.yaml
	homeDir, err := os.UserHomeDir()
	if err == nil {
		homeConfig := filepath.Join(homeDir, ".sshmcp.yaml")
		if _, err := os.Stat(homeConfig); err == nil {
			fmt.Fprintf(os.Stderr, "Loading config from: %s (home directory)\n", homeConfig)
			cfg, err := config.LoadConfig(homeConfig)
			return cfg, homeConfig, err
		}
	}

	// No config found, let LoadConfig auto-generate default config
	fmt.Fprintln(os.Stderr, "No configuration file found in standard locations")
	cfg, err := config.LoadConfig("")
	if err != nil {
		return nil, "", err
	}

	// Return the auto-generated config path
	configPath := filepath.Join(homeDir, ".sshmcp", "config.yaml")
	return cfg, configPath, nil
}

// getProjectRoot returns the current working directory
func getProjectRoot() (string, error) {
	return os.Getwd()
}
