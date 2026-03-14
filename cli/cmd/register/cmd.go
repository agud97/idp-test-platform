package main

import (
	"log/slog"
	"os"

	"github.com/agud97/idp-platform/cli/internal/cliapp"
	"github.com/spf13/cobra"
)

func newCommand() *cobra.Command {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	return cliapp.NewRegisterLegacyCommand(logger)
}
