package main

import (
	"os"

	"github.com/rianjs/confluence-cli/internal/cmd/root"
)

func main() {
	cmd := root.NewCmdRoot()
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
