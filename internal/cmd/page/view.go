package page

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/rianjs/confluence-cli/api"
	"github.com/rianjs/confluence-cli/internal/config"
	"github.com/rianjs/confluence-cli/internal/view"
)

type viewOptions struct {
	raw     bool
	web     bool
	output  string
	noColor bool
}

// NewCmdView creates the page view command.
func NewCmdView() *cobra.Command {
	opts := &viewOptions{}

	cmd := &cobra.Command{
		Use:   "view <page-id>",
		Short: "View a page",
		Long:  `View a Confluence page content.`,
		Example: `  # View a page
  cfl page view 12345

  # View raw storage format
  cfl page view 12345 --raw

  # Open in browser
  cfl page view 12345 --web`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.output, _ = cmd.Flags().GetString("output")
			opts.noColor, _ = cmd.Flags().GetBool("no-color")
			return runView(args[0], opts)
		},
	}

	cmd.Flags().BoolVar(&opts.raw, "raw", false, "Show raw Confluence storage format")
	cmd.Flags().BoolVarP(&opts.web, "web", "w", false, "Open in browser instead of displaying")

	return cmd
}

func runView(pageID string, opts *viewOptions) error {
	// Load config
	cfg, err := config.LoadWithEnv(config.DefaultConfigPath())
	if err != nil {
		return fmt.Errorf("failed to load config: %w (run 'cfl init' to configure)", err)
	}

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w (run 'cfl init' to configure)", err)
	}

	// Create API client
	client := api.NewClient(cfg.URL, cfg.Email, cfg.APIToken)

	// Get page with body
	apiOpts := &api.GetPageOptions{
		BodyFormat: "storage",
	}

	page, err := client.GetPage(context.Background(), pageID, apiOpts)
	if err != nil {
		return fmt.Errorf("failed to get page: %w", err)
	}

	// Open in browser if requested
	if opts.web {
		url := cfg.URL + page.Links.WebUI
		return openBrowser(url)
	}

	// Render output
	renderer := view.NewRenderer(view.Format(opts.output), opts.noColor)

	if opts.output == "json" {
		return renderer.RenderJSON(page)
	}

	// Show page info
	renderer.RenderKeyValue("Title", page.Title)
	renderer.RenderKeyValue("ID", page.ID)
	if page.Version != nil {
		renderer.RenderKeyValue("Version", fmt.Sprintf("%d", page.Version.Number))
	}
	fmt.Println()

	// Show content
	if page.Body != nil && page.Body.Storage != nil {
		content := page.Body.Storage.Value
		if opts.raw {
			fmt.Println(content)
		} else {
			// TODO: Convert storage format to markdown
			// For now, show a simplified version
			fmt.Println("(Content in Confluence storage format - use --raw to see full HTML)")
			fmt.Println()
			fmt.Println(content)
		}
	} else {
		fmt.Println("(No content)")
	}

	return nil
}

func openBrowser(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return fmt.Errorf("unsupported platform")
	}

	return cmd.Start()
}
