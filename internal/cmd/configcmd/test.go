package configcmd

import (
	"fmt"
	"net/http"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/open-cli-collective/confluence-cli/internal/config"
)

// NewCmdTest creates the config test command.
func NewCmdTest() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "test",
		Short: "Test connectivity with configured credentials",
		Long:  `Test that cfl can connect to your Confluence instance with the current configuration.`,
		Example: `  # Test connection
  cfl config test`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			noColor, _ := cmd.Flags().GetBool("no-color")
			return runTest(noColor, nil)
		},
	}

	return cmd
}

func runTest(noColor bool, httpClient *http.Client, cfgs ...*config.Config) error {
	if noColor {
		color.NoColor = true
	}

	var cfg *config.Config
	if len(cfgs) > 0 && cfgs[0] != nil {
		cfg = cfgs[0]
	} else {
		var err error
		cfg, err = config.LoadWithEnv(config.DefaultConfigPath())
		if err != nil {
			return fmt.Errorf("failed to load config: %w (run 'cfl init' to configure)", err)
		}

		if err := cfg.Validate(); err != nil {
			return fmt.Errorf("invalid config: %w (run 'cfl init' to configure)", err)
		}
	}

	green := color.New(color.FgGreen)
	red := color.New(color.FgRed)

	fmt.Printf("Testing connection to %s...\n", cfg.URL)

	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}

	req, err := http.NewRequest("GET", cfg.URL+"/api/v2/spaces?limit=1", nil)
	if err != nil {
		return err
	}

	req.SetBasicAuth(cfg.Email, cfg.APIToken)
	req.Header.Set("Accept", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		red.Println("✗ Connection failed:", err)
		fmt.Println("\nCheck your URL with: cfl config show")
		fmt.Println("Reconfigure with: cfl init")
		return fmt.Errorf("connection failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == 401 {
		red.Println("✗ Authentication failed: 401 Unauthorized")
		fmt.Println("\nCheck your credentials with: cfl config show")
		fmt.Println("Reconfigure with: cfl init")
		return fmt.Errorf("authentication failed")
	}
	if resp.StatusCode == 403 {
		red.Println("✗ Access denied: 403 Forbidden")
		fmt.Println("\nCheck your permissions.")
		return fmt.Errorf("access denied")
	}
	if resp.StatusCode != 200 {
		red.Printf("✗ Unexpected response: %d\n", resp.StatusCode)
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	green.Println("✓ Authentication successful")
	green.Println("✓ API access verified")
	fmt.Printf("\nAuthenticated as: %s\n", cfg.Email)

	return nil
}
