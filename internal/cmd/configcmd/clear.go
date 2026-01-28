package configcmd

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/open-cli-collective/confluence-cli/internal/config"
)

// NewCmdClear creates the config clear command.
func NewCmdClear() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "clear",
		Short: "Remove stored configuration",
		Long:  `Delete the cfl configuration file. Environment variables will still be used if set.`,
		Example: `  # Clear config
  cfl config clear`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			noColor, _ := cmd.Flags().GetBool("no-color")
			return runClear(noColor)
		},
	}

	return cmd
}

func runClear(noColor bool) error {
	if noColor {
		color.NoColor = true
	}

	configPath := config.DefaultConfigPath()

	err := os.Remove(configPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove config file: %w", err)
	}

	green := color.New(color.FgGreen)
	dim := color.New(color.Faint)

	if os.IsNotExist(err) {
		_, _ = green.Printf("✓ No config file to remove\n")
	} else {
		_, _ = green.Printf("✓ Configuration cleared from %s\n", configPath)
	}

	// Check if env vars are set
	envVars := []string{"CFL_URL", "CFL_EMAIL", "CFL_API_TOKEN", "CFL_DEFAULT_SPACE",
		"ATLASSIAN_URL", "ATLASSIAN_EMAIL", "ATLASSIAN_API_TOKEN"}
	var activeVars []string
	for _, v := range envVars {
		if os.Getenv(v) != "" {
			activeVars = append(activeVars, v)
		}
	}

	if len(activeVars) > 0 {
		_, _ = dim.Printf("\nNote: Environment variables will still be used: %s\n", fmt.Sprintf("%v", activeVars))
	}

	return nil
}
