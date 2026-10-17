package cli

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/bootstrap"
)

func newServeCommand() *cobra.Command {
	return newAppCommand("serve", "Serve the tools over streamable HTTP", os.Stdout,
		func(cmd *cobra.Command, app *bootstrap.App) error {
			return app.RunHTTP(cmd.Context())
		})
}
