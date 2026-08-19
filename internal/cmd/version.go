// Package cmd implements the version command.
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/vincentsch/asana-cli/internal/version"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Long:  `Print the CLI version, build time, and commit hash.`,
	Example: `  # Show version
  asana version`,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Fprintln(cmd.OutOrStdout(), version.String())
	},
}
