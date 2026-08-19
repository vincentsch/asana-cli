// Package cmd implements section-related commands.
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
)

var sectionCmd = &cobra.Command{
	Use:   "section",
	Short: "Manage sections",
	Long: `Manage project sections.

Sections are used to organize tasks within a project, typically representing
columns in a kanban board or phases in a workflow.

See also: project, task list, task move`,
}

var sectionListCmd = &cobra.Command{
	Use:   "list",
	Short: "List sections in a project",
	Long: `List all sections in a project.

Sections are shown in their project order. Use --json for structured output.

See also: section view, section create, task list`,
	Example: `  # List sections in a project
  asana section list -p "My Project"

  # Limit output
  asana section list -p "My Project" --limit 5

  # Output as JSON
  asana section list -p "My Project" --json

  # Workflow: List sections then tasks in each
  asana section list -p "My Project"
  asana task list -p "My Project" -s "In Progress"`,
	RunE: runSectionList,
}

func init() {
	sectionListCmd.Flags().StringP("project", "p", "", "Project name")
	sectionListCmd.Flags().String("project-gid", "", "Project GID")
	sectionListCmd.Flags().StringP("workspace", "w", "", "Workspace name (scopes project lookup)")
	sectionListCmd.Flags().String("workspace-gid", "", "Workspace GID (scopes project lookup)")
	sectionListCmd.Flags().Int("limit", 0, "Limit number of sections in output")

	sectionCmd.AddCommand(sectionListCmd)
}

func runSectionList(cmd *cobra.Command, args []string) error {
	projectName, _ := cmd.Flags().GetString("project")
	projectGID, _ := cmd.Flags().GetString("project-gid")
	workspaceName, _ := cmd.Flags().GetString("workspace")
	workspaceGID, _ := cmd.Flags().GetString("workspace-gid")
	limit, _ := cmd.Flags().GetInt("limit")

	if projectName != "" && projectGID != "" {
		return fmt.Errorf("use only one of --project or --project-gid")
	}
	if workspaceName != "" && workspaceGID != "" {
		return fmt.Errorf("use only one of --workspace or --workspace-gid")
	}
	if cmd.Flags().Changed("limit") && limit < 1 {
		return fmt.Errorf("--limit must be >= 1")
	}

	needConfig := false
	if projectName == "" && projectGID == "" {
		needConfig = true
	}
	if projectName != "" && workspaceName == "" && workspaceGID == "" {
		needConfig = true
	}
	if needConfig {
		cfg, err := config.LoadConfig(runtimeConfigPath(cmd))
		if err != nil {
			return err
		}
		if workspaceName == "" && workspaceGID == "" {
			workspaceGID = cfg.Defaults.WorkspaceGID
		}
		if projectName == "" && projectGID == "" {
			projectGID = cfg.Defaults.ProjectGID
		}
	}

	resolvedWorkspace := workspaceGID
	if workspaceName != "" {
		gid, err := resolveWorkspaceWithPrompt(cmd.Context(), runtimeClient(cmd), workspaceName, runtimeNoPrompt(cmd) || runtimeOutputJSON(cmd))
		if err != nil {
			return err
		}
		resolvedWorkspace = gid
	}

	resolvedProject := projectGID
	if projectName != "" {
		gid, err := resolveProjectWithPrompt(cmd.Context(), runtimeClient(cmd), resolvedWorkspace, projectName, runtimeNoPrompt(cmd) || runtimeOutputJSON(cmd))
		if err != nil {
			return err
		}
		resolvedProject = gid
	}
	if resolvedProject == "" && interactive.IsInteractive(runtimeNoPrompt(cmd) || runtimeOutputJSON(cmd)) {
		if resolvedWorkspace == "" {
			workspace, err := interactive.SelectWorkspace(cmd.Context(), runtimeClient(cmd))
			if err != nil {
				return err
			}
			resolvedWorkspace = workspace.GID
		}
		project, err := interactive.SelectProject(cmd.Context(), runtimeClient(cmd), resolvedWorkspace)
		if err != nil {
			return err
		}
		resolvedProject = project.GID
	}
	if resolvedProject == "" {
		return fmt.Errorf("either --project or --project-gid is required")
	}

	query := url.Values{}
	query.Set("opt_fields", "gid,name")
	sections, err := api.Paginate[api.Section](cmd.Context(), runtimeClient(cmd), "/projects/"+resolvedProject+"/sections", query)
	if err != nil {
		return err
	}

	sort.Slice(sections, func(i, j int) bool {
		left := strings.ToLower(sections[i].Name)
		right := strings.ToLower(sections[j].Name)
		if left != right {
			return left < right
		}
		return sections[i].GID < sections[j].GID
	})

	if limit > 0 && len(sections) > limit {
		sections = sections[:limit]
	}

	if runtimeOutputJSON(cmd) {
		return writeSectionsJSON(cmd, sections)
	}

	return writeSectionsTable(cmd, sections)
}

func writeSectionsJSON(cmd *cobra.Command, sections []api.Section) error {
	encoder := json.NewEncoder(cmd.OutOrStdout())
	encoder.SetIndent("", "  ")
	return encoder.Encode(sections)
}

func writeSectionsTable(cmd *cobra.Command, sections []api.Section) error {
	writer := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(writer, "NAME\tGID"); err != nil {
		return err
	}
	for _, section := range sections {
		if _, err := fmt.Fprintf(writer, "%s\t%s\n", section.Name, section.GID); err != nil {
			return err
		}
	}
	return writer.Flush()
}
