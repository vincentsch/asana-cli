// Package cmd implements workspace-related commands.
package cmd

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/vincentsch/asana-cli/internal/api"
)

var workspaceCmd = &cobra.Command{
	Use:   "workspace",
	Short: "Manage workspaces",
	Long: `Manage Asana workspaces.

Workspaces are the top-level organizational unit in Asana. They contain projects,
tasks, teams, and users. Organization workspaces have additional features like teams.

See also: project, team, config set`,
}

var workspaceListCmd = &cobra.Command{
	Use:   "list",
	Short: "List workspaces",
	Long: `List all workspaces you have access to.

Shows workspace name and GID. Workspaces are sorted alphabetically.

See also: project list, team list, config set`,
	Example: `  # List all workspaces
  asana workspace list

  # Output as JSON
  asana workspace list --json

  # Workflow: Find workspace and set as default
  asana workspace list
  asana config set workspace "My Workspace"`,
	RunE: runWorkspaceList,
}

func init() {
	workspaceCmd.AddCommand(workspaceListCmd)
}

func runWorkspaceList(cmd *cobra.Command, args []string) error {
	workspaces, err := api.Paginate[api.Workspace](cmd.Context(), runtimeClient(cmd), "/workspaces", nil)
	if err != nil {
		return err
	}

	sort.Slice(workspaces, func(i, j int) bool {
		left := strings.ToLower(workspaces[i].Name)
		right := strings.ToLower(workspaces[j].Name)
		if left != right {
			return left < right
		}
		return workspaces[i].GID < workspaces[j].GID
	})

	if runtimeOutputJSON(cmd) {
		return writeWorkspacesJSON(cmd, workspaces)
	}

	return writeWorkspacesTable(cmd, workspaces)
}

func writeWorkspacesJSON(cmd *cobra.Command, workspaces []api.Workspace) error {
	encoder := json.NewEncoder(cmd.OutOrStdout())
	encoder.SetIndent("", "  ")
	return encoder.Encode(workspaces)
}

func writeWorkspacesTable(cmd *cobra.Command, workspaces []api.Workspace) error {
	writer := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(writer, "NAME\tGID"); err != nil {
		return err
	}
	for _, workspace := range workspaces {
		if _, err := fmt.Fprintf(writer, "%s\t%s\n", workspace.Name, workspace.GID); err != nil {
			return err
		}
	}
	return writer.Flush()
}
