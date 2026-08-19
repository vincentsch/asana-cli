// Package cmd implements portfolio project commands: list, add, remove.
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
)

var portfolioProjectCmd = &cobra.Command{
	Use:   "project",
	Short: "Manage portfolio projects",
	Long: `List, add, or remove projects from a portfolio.

See also: portfolio view, project list`,
}

var portfolioProjectListCmd = &cobra.Command{
	Use:   "list <portfolio-gid>",
	Short: "List projects in portfolio",
	Long: `List all projects in a portfolio.

See also: portfolio project add, portfolio view`,
	Example: `  # List projects in portfolio
  asana portfolio project list 1234567890123456

  # Limit output
  asana portfolio project list 1234567890123456 --limit 10

  # Output as JSON
  asana portfolio project list 1234567890123456 --json`,
	Args: cobra.ExactArgs(1),
	RunE: runPortfolioProjectList,
}

var portfolioProjectAddCmd = &cobra.Command{
	Use:   "add <portfolio-gid>",
	Short: "Add project to portfolio",
	Long: `Add a project to a portfolio.

See also: portfolio project list, portfolio project remove`,
	Example: `  # Add project by GID
  asana portfolio project add 1234567890123456 --project-gid 9876543210987654

  # Add project by name (requires workspace)
  asana portfolio project add 1234567890123456 --project "Q1 Sprint" -w "My Workspace"

  # Preview without adding
  asana portfolio project add 1234567890123456 --project-gid 9876543210987654 --dry-run`,
	Args: cobra.ExactArgs(1),
	RunE: runPortfolioProjectAdd,
}

var portfolioProjectRemoveCmd = &cobra.Command{
	Use:   "remove <portfolio-gid>",
	Short: "Remove project from portfolio",
	Long: `Remove a project from a portfolio.

The project is not deleted, just unlinked from the portfolio.

See also: portfolio project list, portfolio project add`,
	Example: `  # Remove project by GID
  asana portfolio project remove 1234567890123456 --project-gid 9876543210987654 --confirm

  # Preview without removing
  asana portfolio project remove 1234567890123456 --project-gid 9876543210987654 --dry-run`,
	Args: cobra.ExactArgs(1),
	RunE: runPortfolioProjectRemove,
}

func init() {
	portfolioProjectListCmd.Flags().Int("limit", 0, "Limit number of projects in output")

	portfolioProjectAddCmd.Flags().String("project", "", "Project name")
	portfolioProjectAddCmd.Flags().String("project-gid", "", "Project GID")
	portfolioProjectAddCmd.Flags().StringP("workspace", "w", "", "Workspace name (for project resolution)")
	portfolioProjectAddCmd.Flags().String("workspace-gid", "", "Workspace GID")

	portfolioProjectRemoveCmd.Flags().String("project", "", "Project name")
	portfolioProjectRemoveCmd.Flags().String("project-gid", "", "Project GID")
	portfolioProjectRemoveCmd.Flags().StringP("workspace", "w", "", "Workspace name (for project resolution)")
	portfolioProjectRemoveCmd.Flags().String("workspace-gid", "", "Workspace GID")

	portfolioProjectCmd.AddCommand(portfolioProjectListCmd)
	portfolioProjectCmd.AddCommand(portfolioProjectAddCmd)
	portfolioProjectCmd.AddCommand(portfolioProjectRemoveCmd)
	portfolioCmd.AddCommand(portfolioProjectCmd)
}

type portfolioProjectListItem struct {
	GID  string `json:"gid"`
	Name string `json:"name"`
}

type portfolioProjectOutput struct {
	Action    string  `json:"action"`
	DryRun    bool    `json:"dry_run"`
	Portfolio gidRef  `json:"portfolio"`
	Project   *gidRef `json:"project,omitempty"`
}

func runPortfolioProjectList(cmd *cobra.Command, args []string) error {
	portfolioGID := args[0]
	limit, _ := cmd.Flags().GetInt("limit")

	if err := validateGID(portfolioGID, "portfolio gid"); err != nil {
		return err
	}
	if cmd.Flags().Changed("limit") && limit < 1 {
		return fmt.Errorf("--limit must be >= 1")
	}

	query := url.Values{}
	query.Set("opt_fields", "gid,name,resource_type")

	items, err := api.Paginate[api.PortfolioItem](cmd.Context(), runtimeClient(cmd), "/portfolios/"+portfolioGID+"/items", query)
	if err != nil {
		return err
	}

	// Filter to only projects
	var projects []api.PortfolioItem
	for _, item := range items {
		if item.ResourceType == "project" {
			projects = append(projects, item)
		}
	}

	// Sort by name (case-insensitive), GID tiebreaker
	sort.Slice(projects, func(i, j int) bool {
		cmp := strings.Compare(strings.ToLower(projects[i].Name), strings.ToLower(projects[j].Name))
		if cmp != 0 {
			return cmp < 0
		}
		return projects[i].GID < projects[j].GID
	})

	// Convert to output format
	output := make([]portfolioProjectListItem, len(projects))
	for i, p := range projects {
		output[i] = portfolioProjectListItem{
			GID:  p.GID,
			Name: p.Name,
		}
	}

	if limit > 0 && len(output) > limit {
		output = output[:limit]
	}

	if runtimeOutputJSON(cmd) {
		encoder := json.NewEncoder(cmd.OutOrStdout())
		encoder.SetIndent("", "  ")
		return encoder.Encode(output)
	}

	if len(output) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No projects found")
		return nil
	}

	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tGID")
	for _, p := range output {
		fmt.Fprintf(w, "%s\t%s\n", p.Name, p.GID)
	}
	return w.Flush()
}

func runPortfolioProjectAdd(cmd *cobra.Command, args []string) error {
	portfolioGID := args[0]
	projectName, _ := cmd.Flags().GetString("project")
	projectGID, _ := cmd.Flags().GetString("project-gid")
	workspaceName, _ := cmd.Flags().GetString("workspace")
	workspaceGID, _ := cmd.Flags().GetString("workspace-gid")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	if err := validateGID(portfolioGID, "portfolio gid"); err != nil {
		return err
	}
	if projectName == "" && projectGID == "" {
		return fmt.Errorf("--project or --project-gid is required")
	}
	if projectName != "" && projectGID != "" {
		return fmt.Errorf("use only one of --project or --project-gid")
	}
	if workspaceName != "" && workspaceGID != "" {
		return fmt.Errorf("use only one of --workspace or --workspace-gid")
	}

	// Resolve project
	resolvedProject := projectGID
	if projectName != "" {
		resolvedWorkspace, err := resolveWorkspaceFromFlags(cmd, workspaceName, workspaceGID)
		if err != nil {
			return fmt.Errorf("workspace is required for project name resolution: %w", err)
		}
		gid, err := resolveProjectWithPrompt(cmd.Context(), runtimeClient(cmd), resolvedWorkspace, projectName, runtimeNoPrompt(cmd) || runtimeOutputJSON(cmd))
		if err != nil {
			return err
		}
		resolvedProject = gid
	}

	if dryRun {
		// Validate portfolio exists
		query := url.Values{}
		query.Set("opt_fields", "gid")
		if _, err := runtimeClient(cmd).Get(cmd.Context(), "/portfolios/"+portfolioGID, query); err != nil {
			return err
		}
		// Validate project exists
		if _, err := runtimeClient(cmd).Get(cmd.Context(), "/projects/"+resolvedProject, query); err != nil {
			return err
		}

		output := portfolioProjectOutput{
			Action:    "added",
			DryRun:    true,
			Portfolio: gidRef{GID: portfolioGID},
			Project:   &gidRef{GID: resolvedProject},
		}
		if runtimeOutputJSON(cmd) {
			encoder := json.NewEncoder(cmd.OutOrStdout())
			encoder.SetIndent("", "  ")
			return encoder.Encode(output)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "[dry-run] Would add project %s to portfolio %s\n", resolvedProject, portfolioGID)
		return nil
	}

	_, err := runtimeClient(cmd).Post(cmd.Context(), "/portfolios/"+portfolioGID+"/addItem", api.RequestBody{
		Data: map[string]string{"item": resolvedProject},
	})
	if err != nil {
		return err
	}

	output := portfolioProjectOutput{
		Action:    "added",
		DryRun:    false,
		Portfolio: gidRef{GID: portfolioGID},
		Project:   &gidRef{GID: resolvedProject},
	}

	if runtimeOutputJSON(cmd) {
		encoder := json.NewEncoder(cmd.OutOrStdout())
		encoder.SetIndent("", "  ")
		return encoder.Encode(output)
	}

	fmt.Fprintln(cmd.OutOrStdout(), "Project added to portfolio")
	return nil
}

func runPortfolioProjectRemove(cmd *cobra.Command, args []string) error {
	portfolioGID := args[0]
	projectName, _ := cmd.Flags().GetString("project")
	projectGID, _ := cmd.Flags().GetString("project-gid")
	workspaceName, _ := cmd.Flags().GetString("workspace")
	workspaceGID, _ := cmd.Flags().GetString("workspace-gid")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	if err := validateGID(portfolioGID, "portfolio gid"); err != nil {
		return err
	}
	if projectName == "" && projectGID == "" {
		return fmt.Errorf("--project or --project-gid is required")
	}
	if projectName != "" && projectGID != "" {
		return fmt.Errorf("use only one of --project or --project-gid")
	}
	if workspaceName != "" && workspaceGID != "" {
		return fmt.Errorf("use only one of --workspace or --workspace-gid")
	}

	// Resolve project
	resolvedProject := projectGID
	if projectName != "" {
		resolvedWorkspace, err := resolveWorkspaceFromFlags(cmd, workspaceName, workspaceGID)
		if err != nil {
			return fmt.Errorf("workspace is required for project name resolution: %w", err)
		}
		gid, err := resolveProjectWithPrompt(cmd.Context(), runtimeClient(cmd), resolvedWorkspace, projectName, runtimeNoPrompt(cmd) || runtimeOutputJSON(cmd))
		if err != nil {
			return err
		}
		resolvedProject = gid
	}

	if dryRun {
		// Validate portfolio exists
		query := url.Values{}
		query.Set("opt_fields", "gid")
		if _, err := runtimeClient(cmd).Get(cmd.Context(), "/portfolios/"+portfolioGID, query); err != nil {
			return err
		}
		// Validate project exists
		if _, err := runtimeClient(cmd).Get(cmd.Context(), "/projects/"+resolvedProject, query); err != nil {
			return err
		}

		output := portfolioProjectOutput{
			Action:    "removed",
			DryRun:    true,
			Portfolio: gidRef{GID: portfolioGID},
			Project:   &gidRef{GID: resolvedProject},
		}
		if runtimeOutputJSON(cmd) {
			encoder := json.NewEncoder(cmd.OutOrStdout())
			encoder.SetIndent("", "  ")
			return encoder.Encode(output)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "[dry-run] Would remove project %s from portfolio %s\n", resolvedProject, portfolioGID)
		return nil
	}

	_, err := runtimeClient(cmd).Post(cmd.Context(), "/portfolios/"+portfolioGID+"/removeItem", api.RequestBody{
		Data: map[string]string{"item": resolvedProject},
	})
	if err != nil {
		return err
	}

	output := portfolioProjectOutput{
		Action:    "removed",
		DryRun:    false,
		Portfolio: gidRef{GID: portfolioGID},
		Project:   &gidRef{GID: resolvedProject},
	}

	if runtimeOutputJSON(cmd) {
		encoder := json.NewEncoder(cmd.OutOrStdout())
		encoder.SetIndent("", "  ")
		return encoder.Encode(output)
	}

	fmt.Fprintln(cmd.OutOrStdout(), "Project removed from portfolio")
	return nil
}
