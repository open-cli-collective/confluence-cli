package space

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/rianjs/confluence-cli/api"
	"github.com/rianjs/confluence-cli/internal/config"
	"github.com/rianjs/confluence-cli/internal/view"
)

type listOptions struct {
	limit  int
	spaceType string
	output string
	noColor bool
}

// NewCmdList creates the space list command.
func NewCmdList() *cobra.Command {
	opts := &listOptions{}

	cmd := &cobra.Command{
		Use:   "list",
		Aliases: []string{"ls"},
		Short: "List Confluence spaces",
		Long:  `List all Confluence spaces you have access to.`,
		Example: `  # List all spaces
  cfl space list

  # List only global spaces
  cfl space list --type global

  # Output as JSON
  cfl space list -o json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get global flags
			opts.output, _ = cmd.Flags().GetString("output")
			opts.noColor, _ = cmd.Flags().GetBool("no-color")
			return runList(opts)
		},
	}

	cmd.Flags().IntVarP(&opts.limit, "limit", "l", 25, "Maximum number of spaces to return")
	cmd.Flags().StringVarP(&opts.spaceType, "type", "t", "", "Filter by space type (global, personal)")

	return cmd
}

func runList(opts *listOptions) error {
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

	// List spaces
	apiOpts := &api.ListSpacesOptions{
		Limit: opts.limit,
		Type:  opts.spaceType,
	}

	result, err := client.ListSpaces(context.Background(), apiOpts)
	if err != nil {
		return fmt.Errorf("failed to list spaces: %w", err)
	}

	// Render output
	renderer := view.NewRenderer(view.Format(opts.output), opts.noColor)

	if len(result.Results) == 0 {
		renderer.RenderText("No spaces found.")
		return nil
	}

	headers := []string{"KEY", "NAME", "TYPE", "DESCRIPTION"}
	var rows [][]string

	for _, space := range result.Results {
		desc := ""
		if space.Description != nil && space.Description.Plain != nil {
			desc = view.Truncate(space.Description.Plain.Value, 50)
		}
		rows = append(rows, []string{
			space.Key,
			space.Name,
			space.Type,
			desc,
		})
	}

	renderer.RenderTable(headers, rows)

	if result.HasMore() {
		fmt.Printf("\n(showing first %d results, use --limit to see more)\n", len(result.Results))
	}

	return nil
}
