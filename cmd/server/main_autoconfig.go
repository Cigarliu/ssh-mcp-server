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
// 5. /home/cigar/tools/sshmcp/config.yaml (default)
func loadConfig() (*config.Config, error) {
	// Check for --config flag
	args := os.Args
	for i, arg := range args {
		if arg == "-config" && i+1 < len(args) {
			configPath := args[i+1]
			fmt.Fprintf(os.Stderr, "Loading config from: %s\n", configPath)
			return config.LoadConfig(configPath)
		}
	}

	// Check for .mcp.yaml in current directory
	if _, err := os.Stat(".mcp.yaml"); err == nil {
		fmt.Fprintf(os.Stderr, "Loading config from: .mcp.yaml (current directory)\n")
		return config.LoadConfig(".mcp.yaml")
	}

	// Check for .sshmcp.yaml in current directory
	if _, err := os.Stat(".sshmcp.yaml"); err == nil {
		fmt.Fprintf(os.Stderr, "Loading config from: .sshmcp.yaml (current directory)\n")
		return config.LoadConfig(".sshmcp.yaml")
	}

	// Check for ~/.sshmcp.yaml
	homeDir, err := os.UserHomeDir()
	if err == nil {
		homeConfig := filepath.Join(homeDir, ".sshmcp.yaml")
		if _, err := os.Stat(homeConfig); err == nil {
			fmt.Fprintf(os.Stderr, "Loading config from: %s (home directory)\n", homeConfig)
			return config.LoadConfig(homeConfig)
		}
	}

	// Use default config
	defaultConfig := "/home/cigar/tools/sshmcp/config.yaml"
	fmt.Fprintf(os.Stderr, "Loading config from: %s (default)\n", defaultConfig)
	return config.LoadConfig(defaultConfig)
}

// getProjectRoot returns the current working directory
func getProjectRoot() (string, error) {
	return os.Getwd()
}
