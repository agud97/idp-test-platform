package cliapp

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// Execute runs a minimal command entrypoint for the requested subcommand.
func Execute(name string) int {
	cmd := &cobra.Command{
		Use:   name,
		Short: fmt.Sprintf("%s command placeholder", name),
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	return 0
}

