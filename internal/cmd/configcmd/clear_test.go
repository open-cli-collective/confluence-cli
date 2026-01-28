package configcmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-cli-collective/confluence-cli/internal/config"
)

func TestRunClear_WithExistingConfig(t *testing.T) {
	tmpDir := t.TempDir()
	xdgDir := filepath.Join(tmpDir, "cfl")
	os.MkdirAll(xdgDir, 0755)

	origXDG := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer os.Setenv("XDG_CONFIG_HOME", origXDG)

	cfg := &config.Config{
		URL:      "https://test.atlassian.net/wiki",
		Email:    "test@example.com",
		APIToken: "test-token",
	}
	configPath := filepath.Join(xdgDir, "config.yml")
	require.NoError(t, cfg.Save(configPath))

	err := runClear(true)
	require.NoError(t, err)

	// Verify file is deleted
	_, err = os.Stat(configPath)
	assert.True(t, os.IsNotExist(err))
}

func TestRunClear_NoConfigFile(t *testing.T) {
	origXDG := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", t.TempDir())
	defer os.Setenv("XDG_CONFIG_HOME", origXDG)

	// Should not error even if file doesn't exist
	err := runClear(true)
	require.NoError(t, err)
}

func TestRunClear_Idempotent(t *testing.T) {
	origXDG := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", t.TempDir())
	defer os.Setenv("XDG_CONFIG_HOME", origXDG)

	// Running twice should succeed
	require.NoError(t, runClear(true))
	require.NoError(t, runClear(true))
}
