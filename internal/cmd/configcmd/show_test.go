package configcmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/open-cli-collective/confluence-cli/internal/config"
)

func TestRunShow_WithConfigFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yml")

	cfg := &config.Config{
		URL:          "https://test.atlassian.net/wiki",
		Email:        "test@example.com",
		APIToken:     "test-token-value",
		DefaultSpace: "DEV",
	}
	require.NoError(t, cfg.Save(configPath))

	// Override default config path for test
	origXDG := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer os.Setenv("XDG_CONFIG_HOME", origXDG)

	// Ensure XDG path matches
	xdgDir := filepath.Join(tmpDir, "cfl")
	os.MkdirAll(xdgDir, 0755)
	xdgPath := filepath.Join(xdgDir, "config.yml")
	require.NoError(t, cfg.Save(xdgPath))

	err := runShow(true)
	require.NoError(t, err)
}

func TestRunShow_NoConfigFile(t *testing.T) {
	// Clear env vars
	for _, v := range []string{"CFL_URL", "CFL_EMAIL", "CFL_API_TOKEN", "CFL_DEFAULT_SPACE",
		"ATLASSIAN_URL", "ATLASSIAN_EMAIL", "ATLASSIAN_API_TOKEN"} {
		orig := os.Getenv(v)
		os.Unsetenv(v)
		defer os.Setenv(v, orig)
	}

	origXDG := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", t.TempDir())
	defer os.Setenv("XDG_CONFIG_HOME", origXDG)

	err := runShow(true)
	require.NoError(t, err)
}
