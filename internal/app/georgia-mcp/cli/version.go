package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/abgeo/georgia-mcp/internal/pkg/buildinfo"
)

func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version, the commit, and the Go version",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			info := buildinfo.Get()

			_, err := fmt.Fprintf(cmd.OutOrStdout(), "version %s\ncommit %s\ngo %s\n",
				info.Version, info.Commit, info.GoVersion)
			if err != nil {
				return fmt.Errorf("printing the version: %w", err)
			}

			return nil
		},
	}
}
