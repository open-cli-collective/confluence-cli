// Package configcmd provides config management commands.
package configcmd

import (
	"github.com/spf13/cobra"
)

// NewCmdConfig creates the config command.
func NewCmdConfig() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage cfl configuration",
		Long:  `Commands for viewing, testing, and clearing cfl configuration.`,
	}

	cmd.AddCommand(NewCmdShow())
	cmd.AddCommand(NewCmdTest())
	cmd.AddCommand(NewCmdClear())

	return cmd
}
