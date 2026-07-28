package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigWithPathGeneratesDefaultInHome(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("HOME", tempDir)
	t.Setenv("USERPROFILE", tempDir)
	t.Chdir(t.TempDir())

	cfg, usedPath, err := LoadConfigWithPath("")
	if err != nil {
		t.Fatalf("LoadConfigWithPath failed: %v", err)
	}

	expectedPath := filepath.Join(tempDir, ".sshmcp", "config.yaml")
	if usedPath != expectedPath {
		t.Fatalf("used path = %q, want %q", usedPath, expectedPath)
	}
	if _, err := os.Stat(expectedPath); err != nil {
		t.Fatalf("generated config was not written: %v", err)
	}
	if cfg.Tools.Profile != "files" {
		t.Fatalf("tool profile = %q, want files", cfg.Tools.Profile)
	}
	if cfg.Logging.Output != "stderr" {
		t.Fatalf("logging output = %q, want stderr", cfg.Logging.Output)
	}
}

func TestLoadConfigWithPathGeneratesExplicitMissingConfig(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "nested", ".sshmcp.yaml")

	cfg, usedPath, err := LoadConfigWithPath(configPath)
	if err != nil {
		t.Fatalf("LoadConfigWithPath failed: %v", err)
	}

	expectedPath, err := filepath.Abs(configPath)
	if err != nil {
		t.Fatalf("resolve expected path: %v", err)
	}
	if usedPath != expectedPath {
		t.Fatalf("used path = %q, want %q", usedPath, expectedPath)
	}
	if _, err := os.Stat(expectedPath); err != nil {
		t.Fatalf("generated explicit config was not written: %v", err)
	}
	if cfg.Server.Name != "ssh-mcp-server" {
		t.Fatalf("server name = %q, want ssh-mcp-server", cfg.Server.Name)
	}
}

func TestLoadConfigWithPathKeepsExistingConfig(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	content := []byte(`tools:
  profile: core
logging:
  output: stderr
`)
	if err := os.WriteFile(configPath, content, 0600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, usedPath, err := LoadConfigWithPath(configPath)
	if err != nil {
		t.Fatalf("LoadConfigWithPath failed: %v", err)
	}

	if usedPath != configPath {
		t.Fatalf("used path = %q, want %q", usedPath, configPath)
	}
	if cfg.Tools.Profile != "core" {
		t.Fatalf("tool profile = %q, want core", cfg.Tools.Profile)
	}
	after, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if string(after) != string(content) {
		t.Fatal("existing config was overwritten")
	}
}
