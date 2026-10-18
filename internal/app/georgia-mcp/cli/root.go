package cli

import (
	"errors"
	"io"

	"github.com/spf13/cobra"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/bootstrap"
)

// NewRoot returns the root command with every subcommand.
func NewRoot() *cobra.Command {
	root := &cobra.Command{
		Use:           "georgia-mcp",
		Short:         "An MCP server for Georgian public services",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.AddCommand(
		newServeCommand(),
		newStdioCommand(),
		newVersionCommand(),
	)

	return root
}

// newAppCommand returns a command that loads the configuration in PreRunE and
// builds the app with its log on logOutput.
func newAppCommand(
	use, short string,
	logOutput io.Writer,
	run func(*cobra.Command, *bootstrap.App) error,
) *cobra.Command {
	var app *bootstrap.App

	return &cobra.Command{
		Use:   use,
		Short: short,
		Args:  cobra.NoArgs,
		PreRunE: func(*cobra.Command, []string) error {
			cfg, err := bootstrap.LoadConfig()
			if err != nil {
				return err
			}

			app, err = bootstrap.New(cfg, logOutput)

			return err
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			return errors.Join(run(cmd, app), app.Close())
		},
	}
}
