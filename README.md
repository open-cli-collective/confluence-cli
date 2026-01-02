# cfl - Confluence CLI

A command-line interface for Atlassian Confluence Cloud, inspired by [jira-cli](https://github.com/ankitpokhrel/jira-cli).

## Features

- Manage Confluence pages from the command line
- List and browse spaces
- Create and edit pages
- Multiple output formats (table, JSON, plain)
- Markdown-first approach for content editing

## Installation

### Homebrew (macOS/Linux)

```bash
brew tap rianjs/confluence-cli
brew install cfl
```

### Go Install

```bash
go install github.com/rianjs/confluence-cli/cmd/cfl@latest
```

### Binary Download

Download the latest release from the [Releases page](https://github.com/rianjs/confluence-cli/releases).

## Quick Start

### 1. Configure cfl

```bash
cfl init
```

This will prompt you for:
- Your Confluence URL (e.g., `https://mycompany.atlassian.net`)
- Your email address
- An API token

**To generate an API token:**
1. Go to https://id.atlassian.com/manage-profile/security/api-tokens
2. Click "Create API token"
3. Copy the token (it won't be shown again)

### 2. List Spaces

```bash
cfl space list
```

### 3. List Pages in a Space

```bash
cfl page list --space DEV
```

### 4. View a Page

```bash
cfl page view 12345
```

### 5. Create a Page

```bash
cfl page create --space DEV --title "My New Page"
```

## Commands

### Spaces

```bash
cfl space list              # List all spaces
cfl space list --type global # List only global spaces
cfl space list -o json      # Output as JSON
```

### Pages

```bash
cfl page list -s DEV        # List pages in a space
cfl page view 12345         # View a page
cfl page view 12345 --web   # Open in browser
cfl page create -s DEV -t "Title"  # Create a page
```

## Configuration

Configuration is stored in `~/.config/cfl/config.yml`:

```yaml
url: https://mycompany.atlassian.net/wiki
email: you@example.com
api_token: your-api-token
default_space: DEV
output_format: table
```

### Environment Variables

You can override configuration with environment variables:

- `CFL_URL` - Confluence URL
- `CFL_EMAIL` - Email address
- `CFL_API_TOKEN` - API token
- `CFL_DEFAULT_SPACE` - Default space key

## Output Formats

Use the `--output` or `-o` flag to change output format:

```bash
cfl space list -o table  # Default table format
cfl space list -o json   # JSON format
cfl space list -o plain  # Plain text (for scripting)
```

## Development

### Prerequisites

- Go 1.22 or later
- golangci-lint (for linting)

### Build

```bash
make build
```

### Test

```bash
make test
```

### Lint

```bash
make lint
```

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

MIT License - see [LICENSE](LICENSE) for details.
