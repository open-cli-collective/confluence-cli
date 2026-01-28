package configcmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/open-cli-collective/confluence-cli/internal/config"
)

// NewCmdShow creates the config show command.
func NewCmdShow() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show",
		Short: "Display current configuration",
		Long:  `Display the current cfl configuration with credential source indicators.`,
		Example: `  # Show current config
  cfl config show`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			noColor, _ := cmd.Flags().GetBool("no-color")
			return runShow(noColor)
		},
	}

	return cmd
}

func runShow(noColor bool) error {
	if noColor {
		color.NoColor = true
	}

	configPath := config.DefaultConfigPath()

	// Load file config (may not exist)
	fileCfg, fileErr := config.Load(configPath)
	if fileErr != nil {
		fileCfg = &config.Config{}
	}

	// Load full config with env overrides
	cfg, _ := config.LoadWithEnv(configPath)

	bold := color.New(color.Bold)
	dim := color.New(color.Faint)

	printField := func(label, value, fileValue string, envVars ...string) {
		_, _ = bold.Printf("%-12s", label+":")
		if value == "" {
			_, _ = dim.Println("-")
			return
		}

		// Mask tokens
		display := value
		if strings.Contains(strings.ToLower(label), "token") && len(value) > 8 {
			display = value[:4] + strings.Repeat("*", len(value)-8) + value[len(value)-4:]
		}

		fmt.Print(display)

		// Determine source
		source := "config"
		if fileErr != nil {
			source = "-"
		}
		for _, envVar := range envVars {
			if v := os.Getenv(envVar); v != "" && v == value {
				source = envVar
				break
			}
		}
		if fileValue != value && source == "config" {
			source = "-"
		}

		_, _ = dim.Printf("  (source: %s)\n", source)
	}

	printField("URL", cfg.URL, fileCfg.URL, "CFL_URL", "ATLASSIAN_URL")
	printField("Email", cfg.Email, fileCfg.Email, "CFL_EMAIL", "ATLASSIAN_EMAIL")
	printField("API Token", cfg.APIToken, fileCfg.APIToken, "CFL_API_TOKEN", "ATLASSIAN_API_TOKEN")
	printField("Space", cfg.DefaultSpace, fileCfg.DefaultSpace, "CFL_DEFAULT_SPACE")

	fmt.Println()
	_, _ = dim.Printf("Config file: %s\n", configPath)
	if fileErr != nil {
		_, _ = dim.Println("(file not found)")
	}

	return nil
}
