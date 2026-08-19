// Package cmd implements shell completion command.
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var completionCmd = &cobra.Command{
	Use:                   "completion [bash|zsh|fish]",
	Short:                 "Generate shell completion script",
	Args:                  cobra.ExactArgs(1),
	ValidArgs:             []string{"bash", "zsh", "fish"},
	DisableFlagsInUseLine: true,
	Example:               `  asana completion bash`,
	Long: `Generate shell completion script for asana.

To load completions:

Bash:
  $ source <(asana completion bash)
  # To load completions for each session, execute once:
  # Linux:
  $ asana completion bash > /etc/bash_completion.d/asana
  # macOS:
  $ asana completion bash > /usr/local/etc/bash_completion.d/asana

Zsh:
  $ source <(asana completion zsh)
  # To load completions for each session, execute once:
  $ asana completion zsh > "${fpath[1]}/_asana"

Fish:
  $ asana completion fish | source
  # To load completions for each session, execute once:
  $ asana completion fish > ~/.config/fish/completions/asana.fish
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		root := cmd.Root()
		// Direct handler tests do not attach the command to its production root.
		if root.Name() == "" {
			root = &cobra.Command{Use: "asana"}
		}
		switch args[0] {
		case "bash":
			return root.GenBashCompletion(cmd.OutOrStdout())
		case "zsh":
			return root.GenZshCompletion(cmd.OutOrStdout())
		case "fish":
			return root.GenFishCompletion(cmd.OutOrStdout(), true)
		default:
			return fmt.Errorf("unsupported shell: %s", args[0])
		}
	},
}
