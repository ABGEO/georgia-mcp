package cli

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/bootstrap"
)

// The log goes to stderr, because stdout carries the protocol.
func newStdioCommand() *cobra.Command {
	return newAppCommand("stdio", "Serve the tools over stdio", os.Stderr,
		func(cmd *cobra.Command, app *bootstrap.App) error {
			return app.RunStdio(cmd.Context())
		})
}
