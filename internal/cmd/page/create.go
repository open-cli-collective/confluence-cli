package page

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"github.com/rianjs/confluence-cli/api"
	"github.com/rianjs/confluence-cli/internal/config"
	"github.com/rianjs/confluence-cli/internal/view"
)

type createOptions struct {
	space    string
	title    string
	parent   string
	file     string
	editor   bool
	output   string
	noColor  bool
}

// NewCmdCreate creates the page create command.
func NewCmdCreate() *cobra.Command {
	opts := &createOptions{}

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new page",
		Long: `Create a new Confluence page.

Content can be provided via:
- --file flag to read from a file
- Standard input (pipe content)
- Interactive editor (default, or with --editor flag)

Content should be in Confluence storage format (XHTML).`,
		Example: `  # Create a page with title (opens editor)
  cfl page create --space DEV --title "My Page"

  # Create from file
  cfl page create -s DEV -t "My Page" --file content.html

  # Create from stdin
  echo "<p>Hello</p>" | cfl page create -s DEV -t "My Page"

  # Create as child of another page
  cfl page create -s DEV -t "Child Page" --parent 12345`,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.output, _ = cmd.Flags().GetString("output")
			opts.noColor, _ = cmd.Flags().GetBool("no-color")
			return runCreate(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.space, "space", "s", "", "Space key (required)")
	cmd.Flags().StringVarP(&opts.title, "title", "t", "", "Page title (required)")
	cmd.Flags().StringVarP(&opts.parent, "parent", "p", "", "Parent page ID")
	cmd.Flags().StringVarP(&opts.file, "file", "f", "", "Read content from file")
	cmd.Flags().BoolVar(&opts.editor, "editor", false, "Open editor for content")

	cmd.MarkFlagRequired("title")

	return cmd
}

func runCreate(opts *createOptions) error {
	// Load config
	cfg, err := config.LoadWithEnv(config.DefaultConfigPath())
	if err != nil {
		return fmt.Errorf("failed to load config: %w (run 'cfl init' to configure)", err)
	}

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w (run 'cfl init' to configure)", err)
	}

	// Determine space
	spaceKey := opts.space
	if spaceKey == "" {
		spaceKey = cfg.DefaultSpace
	}
	if spaceKey == "" {
		return fmt.Errorf("space is required: use --space flag or set default_space in config")
	}

	// Create API client
	client := api.NewClient(cfg.URL, cfg.Email, cfg.APIToken)

	// Get space ID
	space, err := client.GetSpaceByKey(context.Background(), spaceKey)
	if err != nil {
		return fmt.Errorf("failed to find space '%s': %w", spaceKey, err)
	}

	// Get content
	content, err := getContent(opts)
	if err != nil {
		return err
	}

	// Create page
	req := &api.CreatePageRequest{
		SpaceID: space.ID,
		Title:   opts.title,
		Status:  "current",
		Body: &api.Body{
			Storage: &api.BodyRepresentation{
				Representation: "storage",
				Value:          content,
			},
		},
	}

	if opts.parent != "" {
		req.ParentID = opts.parent
	}

	page, err := client.CreatePage(context.Background(), req)
	if err != nil {
		return fmt.Errorf("failed to create page: %w", err)
	}

	// Render output
	renderer := view.NewRenderer(view.Format(opts.output), opts.noColor)

	if opts.output == "json" {
		return renderer.RenderJSON(page)
	}

	renderer.Success(fmt.Sprintf("Created page: %s", page.Title))
	renderer.RenderKeyValue("ID", page.ID)
	renderer.RenderKeyValue("URL", cfg.URL+page.Links.WebUI)

	return nil
}

func getContent(opts *createOptions) (string, error) {
	// Read from file
	if opts.file != "" {
		data, err := os.ReadFile(opts.file)
		if err != nil {
			return "", fmt.Errorf("failed to read file: %w", err)
		}
		return string(data), nil
	}

	// Check if stdin has data
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", fmt.Errorf("failed to read stdin: %w", err)
		}
		return string(data), nil
	}

	// Open editor
	return openEditor("")
}

func openEditor(initialContent string) (string, error) {
	// Create temp file
	tmpfile, err := os.CreateTemp("", "cfl-*.html")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpfile.Name())

	// Write initial content
	template := initialContent
	if template == "" {
		template = `<p>Enter your page content here.</p>
`
	}
	if _, err := tmpfile.WriteString(template); err != nil {
		return "", err
	}
	tmpfile.Close()

	// Get editor
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}
	if editor == "" {
		editor = "vi"
	}

	// Open editor
	cmd := exec.Command(editor, tmpfile.Name())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("editor failed: %w", err)
	}

	// Read content
	data, err := os.ReadFile(tmpfile.Name())
	if err != nil {
		return "", fmt.Errorf("failed to read edited content: %w", err)
	}

	content := strings.TrimSpace(string(data))
	if content == "" || content == strings.TrimSpace(template) {
		return "", fmt.Errorf("no content provided (or content unchanged)")
	}

	return content, nil
}
