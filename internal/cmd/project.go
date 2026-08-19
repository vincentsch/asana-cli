// Package cmd implements project-related commands.
package cmd

import (
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/vincentsch/asana-cli/internal/api"
	"github.com/vincentsch/asana-cli/internal/config"
	"github.com/vincentsch/asana-cli/internal/interactive"
	"github.com/vincentsch/asana-cli/internal/resolve"
)

var projectCmd = &cobra.Command{
	Use:   "project",
	Short: "Manage projects",
	Long: `Manage Asana projects.

Projects are containers for tasks, typically representing a goal, initiative, or
ongoing work. They can have sections for organizing tasks into columns or phases.

See also: task, section, workspace`,
}

var projectListCmd = &cobra.Command{
	Use:   "list",
	Short: "List projects in a workspace",
	Long: `List all projects in a workspace or team.

Projects are sorted alphabetically by name. Use --team to filter to a specific
team in organization workspaces.

See also: project view, project create, task list`,
	Example: `  # List all projects in a workspace
  asana project list -w "My Workspace"

  # List projects in a specific team
  asana project list -w "My Workspace" -t "Engineering"

  # Limit output
  asana project list -w "My Workspace" --limit 10

  # Output as JSON
  asana project list -w "My Workspace" --json

  # Workflow: Find project and list its tasks
  asana project list -w "My Workspace"
  asana task list -p "My Project"`,
	RunE: runProjectList,
}

func init() {
	projectListCmd.Flags().StringP("workspace", "w", "", "Workspace name")
	projectListCmd.Flags().String("workspace-gid", "", "Workspace GID")
	projectListCmd.Flags().StringP("team", "t", "", "Team name")
	projectListCmd.Flags().String("team-gid", "", "Team GID")
	projectListCmd.Flags().Int("limit", 0, "Limit number of projects in output")

	projectCmd.AddCommand(projectListCmd)
}

func runProjectList(cmd *cobra.Command, args []string) error {
	workspaceName, _ := cmd.Flags().GetString("workspace")
	workspaceGID, _ := cmd.Flags().GetString("workspace-gid")
	teamName, _ := cmd.Flags().GetString("team")
	teamGID, _ := cmd.Flags().GetString("team-gid")
	limit, _ := cmd.Flags().GetInt("limit")

	if workspaceName != "" && workspaceGID != "" {
		return fmt.Errorf("use only one of --workspace or --workspace-gid")
	}
	if teamName != "" && teamGID != "" {
		return fmt.Errorf("use only one of --team or --team-gid")
	}
	if cmd.Flags().Changed("limit") && limit < 1 {
		return fmt.Errorf("--limit must be >= 1")
	}

	if workspaceName == "" && workspaceGID == "" {
		cfg, err := config.LoadConfig(runtimeConfigPath(cmd))
		if err != nil {
			return err
		}
		workspaceGID = cfg.Defaults.WorkspaceGID
	}

	resolvedWorkspace := workspaceGID
	if workspaceName != "" {
		gid, err := resolveWorkspaceWithPrompt(cmd.Context(), runtimeClient(cmd), workspaceName, runtimeNoPrompt(cmd) || runtimeOutputJSON(cmd))
		if err != nil {
			return err
		}
		resolvedWorkspace = gid
	}
	if resolvedWorkspace == "" && interactive.IsInteractive(runtimeNoPrompt(cmd) || runtimeOutputJSON(cmd)) {
		workspace, err := interactive.SelectWorkspace(cmd.Context(), runtimeClient(cmd))
		if err != nil {
			return err
		}
		resolvedWorkspace = workspace.GID
	}
	if resolvedWorkspace == "" {
		return fmt.Errorf("either --workspace or --workspace-gid is required")
	}

	resolvedTeam := teamGID
	if teamName != "" {
		gid, err := resolve.Team(cmd.Context(), runtimeClient(cmd), resolvedWorkspace, teamName)
		if err != nil {
			return err
		}
		resolvedTeam = gid
	}

	query := url.Values{}
	query.Set("opt_fields", "gid,name")
	if resolvedTeam != "" {
		query.Set("team", resolvedTeam)
	} else {
		query.Set("workspace", resolvedWorkspace)
	}

	projects, err := api.Paginate[api.Project](cmd.Context(), runtimeClient(cmd), "/projects", query)
	if err != nil {
		return err
	}

	sort.Slice(projects, func(i, j int) bool {
		left := strings.ToLower(projects[i].Name)
		right := strings.ToLower(projects[j].Name)
		if left != right {
			return left < right
		}
		return projects[i].GID < projects[j].GID
	})

	if limit > 0 && len(projects) > limit {
		projects = projects[:limit]
	}

	if runtimeOutputJSON(cmd) {
		return writeProjectsJSON(cmd, projects)
	}

	return writeProjectsTable(cmd, projects)
}

func writeProjectsJSON(cmd *cobra.Command, projects []api.Project) error {
	encoder := json.NewEncoder(cmd.OutOrStdout())
	encoder.SetIndent("", "  ")
	return encoder.Encode(projects)
}

func writeProjectsTable(cmd *cobra.Command, projects []api.Project) error {
	writer := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(writer, "NAME\tGID"); err != nil {
		return err
	}
	for _, project := range projects {
		if _, err := fmt.Fprintf(writer, "%s\t%s\n", project.Name, project.GID); err != nil {
			return err
		}
	}
	return writer.Flush()
}
