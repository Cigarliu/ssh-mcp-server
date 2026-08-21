package mcp

import (
	"context"
	"testing"
	"time"

	"github.com/cigar/sshmcp/pkg/sshmcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestServer(t *testing.T) (*Server, *sshmcp.SessionManager) {
	t.Helper()
	logger := setupTestLogger()
	manager := sshmcp.NewSessionManager(sshmcp.ManagerConfig{
		MaxSessions:        50,
		MaxSessionsPerHost: 30,
		SessionTimeout:     5 * time.Minute,
		IdleTimeout:        2 * time.Minute,
		CleanupInterval:    10 * time.Second,
		Logger:             logger,
	})
	hosts := sshmcp.NewHostManager(map[string]sshmcp.HostConfig{}, "", logger)
	server, err := NewServer(manager, hosts, logger)
	require.NoError(t, err)
	return server, manager
}

func TestBoundedOutputLimit(t *testing.T) {
	assert.Equal(t, defaultMaxOutputChars, boundedOutputLimit(0))
	assert.Equal(t, 100, boundedOutputLimit(100))
	assert.Equal(t, defaultMaxOutputChars, boundedOutputLimit(defaultMaxOutputChars+1))
}

func TestDirectoryEntryLimit(t *testing.T) {
	assert.Equal(t, 100, directoryEntryLimit(0))
	assert.Equal(t, 100, directoryEntryLimit(100))
	assert.Equal(t, maxDirectoryEntries, directoryEntryLimit(maxDirectoryEntries+1))
}

func TestTruncateCommandOutput(t *testing.T) {
	stdout, stderr, truncated := truncateCommandOutput("abcdef", "ghijkl", 8)
	assert.Equal(t, "abcdef", stdout)
	assert.Equal(t, "gh", stderr)
	assert.True(t, truncated)
}

func TestDirectoryMode(t *testing.T) {
	mode, err := directoryMode("0750")
	assert.NoError(t, err)
	assert.Equal(t, uint32(0750), uint32(mode))
	_, err = directoryMode("invalid")
	assert.Error(t, err)
}

func TestCompactSFTPToolsRejectInvalidOperations(t *testing.T) {
	server, manager := setupTestServer(t)
	t.Cleanup(manager.Close)

	result, _, err := server.handleSFTPTransfer(context.Background(), nil, map[string]any{"connection_id": "missing", "operation": "invalid"})
	assert.NoError(t, err)
	assert.True(t, result.IsError)

	result, _, err = server.handleSFTPManage(context.Background(), nil, map[string]any{"connection_id": "missing", "operation": "invalid"})
	assert.NoError(t, err)
	assert.True(t, result.IsError)
}
