# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

cfl is a command-line interface for Atlassian Confluence Cloud written in Go. It provides markdown-first page management with automatic conversion between Markdown and Confluence's XHTML storage format.

## Commands

```bash
# Build
make build                    # Build to bin/cfl with version info from git

# Test
make test                     # Run all tests with race detection
make test-cover               # Tests with coverage (generates coverage.html)
go test -v -race ./api/...    # Run tests for a specific package

# Lint & Format
make lint                     # Run golangci-lint
make fmt                      # Format with gofmt and goimports

# Development
make run ARGS="page list"     # Run CLI directly without building
```

## Architecture

```
cmd/cfl/main.go          → Entry point, creates root command
api/                     → Confluence REST API client (pages, spaces, attachments)
internal/cmd/            → Cobra command implementations
  root/                  → Root command with global flags
  page/                  → page list|view|create|edit|delete
  space/                 → space list
  attachment/            → attachment list|upload|download
  init/                  → Configuration wizard
internal/config/         → YAML config loading with env var overrides
internal/view/           → Output formatting (table/json/plain)
pkg/md/                  → Bidirectional Markdown ↔ XHTML conversion
```

**Data Flow:** Commands load config from `~/.config/cfl/config.yml` → instantiate `api.Client` → call API methods → format output via `view.Renderer`

## Key Patterns

- **Command factories:** `NewCmd{Name}() *cobra.Command` in each command file
- **Options structs:** Commands collect flags into `*Options` structs before execution
- **Run functions:** `run{Action}(opts *Options) error` contains command logic
- **Import ordering:** Standard library, external deps, then `github.com/rianjs/confluence-cli/...` (enforced by goimports)

## Markdown Conversion

The `pkg/md` package handles format conversion:
- `converter.go`: Markdown → XHTML (uses goldmark)
- `from_html.go`: XHTML → Markdown (uses html-to-markdown)

Format auto-detection: `.md` files → markdown, `.html/.xhtml` → storage format, stdin/editor → markdown by default.
