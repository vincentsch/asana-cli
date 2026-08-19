// Package cmd implements section view, create, update, delete, and move commands.
package cmd

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/spf13/cobra"
	"github.com/vincentsch/asana-cli/internal/api"
	"github.com/vincentsch/asana-cli/internal/config"
	"github.com/vincentsch/asana-cli/internal/resolve"
)

var sectionViewCmd = &cobra.Command{
	Use:   "view <section-gid>",
	Short: "View section details",
	Long: `View detailed information about a section.

Shows section name, GID, creation time, and parent project.

See also: section list, task list`,
	Example: `  # View section by GID
  asana section view 1234567890123456

  # Output as JSON
  asana section view 1234567890123456 --json`,
	Args: cobra.ExactArgs(1),
	RunE: runSectionView,
}

var sectionCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new section",
	Long: `Create a new section in a project.

Optionally position the section before or after an existing section.

See also: section list, section delete, task move`,
	Example: `  # Create a section
  asana section create "In Progress" -p "My Project"

  # Create at a specific position
  asana section create "Review" -p "My Project" --insert-after "In Progress"

  # Preview without creating
  asana section create "Test" -p "My Project" --dry-run

  # Workflow: Set up a kanban board
  asana section create "Backlog" -p "My Project"
  asana section create "In Progress" -p "My Project"
  asana section create "Review" -p "My Project"
  asana section create "Done" -p "My Project"`,
	Args: cobra.ExactArgs(1),
	RunE: runSectionCreate,
}

var sectionUpdateCmd = &cobra.Command{
	Use:   "update <section-gid>",
	Short: "Update a section",
	Long: `Update a section's name.

See also: section view, section list`,
	Example: `  # Rename a section
  asana section update 1234567890123456 --name "Done"

  # Preview change
  asana section update 1234567890123456 --name "Done" --dry-run`,
	Args: cobra.ExactArgs(1),
	RunE: runSectionUpdate,
}

var sectionDeleteCmd = &cobra.Command{
	Use:   "delete <section-gid>",
	Short: "Delete a section",
	Long: `Permanently delete a section.

Tasks in the section are not deleted; they become sectionless.

See also: section list, section create`,
	Example: `  # Delete a section
  asana section delete 1234567890123456 --confirm

  # Preview deletion
  asana section delete 1234567890123456 --dry-run`,
	Args: cobra.ExactArgs(1),
	RunE: runSectionDelete,
}

var sectionMoveCmd = &cobra.Command{
	Use:   "move <section-gid>",
	Short: "Move a section within a project",
	Long: `Reorder a section within its project.

Use --before or --after to position the section relative to another.

See also: section list, task move`,
	Example: `  # Move section before another
  asana section move 1234567890123456 -p "My Project" --before "Done"

  # Move section after another
  asana section move 1234567890123456 -p "My Project" --after "To Do"

  # Preview move
  asana section move 1234567890123456 -p "My Project" --before "Done" --dry-run`,
	Args: cobra.ExactArgs(1),
	RunE: runSectionMove,
}

// sectionDetailFields are fields requested when viewing a section.
const sectionDetailFields = "gid,name,created_at,project.gid,project.name"

func init() {
	// Create flags
	sectionCreateCmd.Flags().StringP("project", "p", "", "Project name")
	sectionCreateCmd.Flags().String("project-gid", "", "Project GID")
	sectionCreateCmd.Flags().StringP("workspace", "w", "", "Workspace name (scopes project lookup)")
	sectionCreateCmd.Flags().String("workspace-gid", "", "Workspace GID (scopes project lookup)")
	sectionCreateCmd.Flags().String("insert-before", "", "Insert before this section (name or GID)")
	sectionCreateCmd.Flags().String("insert-after", "", "Insert after this section (name or GID)")

	// Update flags
	sectionUpdateCmd.Flags().String("name", "", "New section name")

	// Delete flags

	// Move flags
	sectionMoveCmd.Flags().StringP("project", "p", "", "Target project name")
	sectionMoveCmd.Flags().String("project-gid", "", "Target project GID")
	sectionMoveCmd.Flags().StringP("workspace", "w", "", "Workspace name (scopes project lookup)")
	sectionMoveCmd.Flags().String("workspace-gid", "", "Workspace GID (scopes project lookup)")
	sectionMoveCmd.Flags().String("before", "", "Move before this section (name or GID)")
	sectionMoveCmd.Flags().String("after", "", "Move after this section (name or GID)")

	sectionCmd.AddCommand(sectionViewCmd)
	sectionCmd.AddCommand(sectionCreateCmd)
	sectionCmd.AddCommand(sectionUpdateCmd)
	sectionCmd.AddCommand(sectionDeleteCmd)
	sectionCmd.AddCommand(sectionMoveCmd)
}

// sectionOutput is used for JSON output of section commands.
type sectionOutput struct {
	Action  string               `json:"action"`
	DryRun  bool                 `json:"dry_run"`
	Section *sectionOutputDetail `json:"section,omitempty"`
}

type sectionOutputDetail struct {
	GID       string      `json:"gid"`
	Name      string      `json:"name"`
	CreatedAt string      `json:"created_at,omitempty"`
	Project   *projectRef `json:"project,omitempty"`
}

func runSectionView(cmd *cobra.Command, args []string) error {
	sectionGID := args[0]

	if err := validateGID(sectionGID, "section gid"); err != nil {
		return err
	}

	query := url.Values{}
	query.Set("opt_fields", sectionDetailFields)

	payload, err := runtimeClient(cmd).Get(cmd.Context(), "/sections/"+sectionGID, query)
	if err != nil {
		return err
	}

	var response api.Response[api.SectionDetail]
	if err := json.Unmarshal(payload, &response); err != nil {
		return &api.ResponseError{Err: err}
	}

	section := response.Data
	output := sectionOutputDetail{
		GID:       section.GID,
		Name:      section.Name,
		CreatedAt: section.CreatedAt,
	}
	if section.Project != nil {
		output.Project = &projectRef{GID: section.Project.GID, Name: &section.Project.Name}
	}

	if runtimeOutputJSON(cmd) {
		encoder := json.NewEncoder(cmd.OutOrStdout())
		encoder.SetIndent("", "  ")
		return encoder.Encode(output)
	}

	// Text output
	fmt.Fprintf(cmd.OutOrStdout(), "GID:       %s\n", section.GID)
	fmt.Fprintf(cmd.OutOrStdout(), "Name:      %s\n", section.Name)
	if section.CreatedAt != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "Created:   %s\n", section.CreatedAt)
	}
	if section.Project != nil {
		fmt.Fprintf(cmd.OutOrStdout(), "Project:   %s (%s)\n", section.Project.Name, section.Project.GID)
	}

	return nil
}

func runSectionCreate(cmd *cobra.Command, args []string) error {
	name := args[0]
	projectName, _ := cmd.Flags().GetString("project")
	projectGID, _ := cmd.Flags().GetString("project-gid")
	workspaceName, _ := cmd.Flags().GetString("workspace")
	workspaceGID, _ := cmd.Flags().GetString("workspace-gid")
	insertBefore, _ := cmd.Flags().GetString("insert-before")
	insertAfter, _ := cmd.Flags().GetString("insert-after")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	if projectName != "" && projectGID != "" {
		return fmt.Errorf("use only one of --project or --project-gid")
	}
	if workspaceName != "" && workspaceGID != "" {
		return fmt.Errorf("use only one of --workspace or --workspace-gid")
	}
	if insertBefore != "" && insertAfter != "" {
		return fmt.Errorf("use only one of --insert-before or --insert-after")
	}

	// Resolve project
	resolvedProject, err := resolveSectionProject(cmd, projectName, projectGID, workspaceName, workspaceGID)
	if err != nil {
		return err
	}

	resolvedInsertBefore := insertBefore
	if insertBefore != "" {
		if resolve.IsGID(insertBefore) {
			if err := validateGID(insertBefore, "insert-before"); err != nil {
				return err
			}
		} else {
			gid, err := resolveSectionWithPrompt(cmd.Context(), runtimeClient(cmd), resolvedProject, insertBefore, runtimeNoPrompt(cmd) || runtimeOutputJSON(cmd))
			if err != nil {
				return err
			}
			resolvedInsertBefore = gid
		}
	}
	resolvedInsertAfter := insertAfter
	if insertAfter != "" {
		if resolve.IsGID(insertAfter) {
			if err := validateGID(insertAfter, "insert-after"); err != nil {
				return err
			}
		} else {
			gid, err := resolveSectionWithPrompt(cmd.Context(), runtimeClient(cmd), resolvedProject, insertAfter, runtimeNoPrompt(cmd) || runtimeOutputJSON(cmd))
			if err != nil {
				return err
			}
			resolvedInsertAfter = gid
		}
	}

	if dryRun {
		output := sectionOutput{
			Action: "created",
			DryRun: true,
			Section: &sectionOutputDetail{
				Name: name,
			},
		}
		if runtimeOutputJSON(cmd) {
			encoder := json.NewEncoder(cmd.OutOrStdout())
			encoder.SetIndent("", "  ")
			return encoder.Encode(output)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "[dry-run] Would create section %q in project %s\n", name, resolvedProject)
		return nil
	}

	// Build request body
	data := map[string]interface{}{
		"name": name,
	}
	if resolvedInsertBefore != "" {
		data["insert_before"] = resolvedInsertBefore
	}
	if resolvedInsertAfter != "" {
		data["insert_after"] = resolvedInsertAfter
	}

	payload, err := runtimeClient(cmd).Post(cmd.Context(), "/projects/"+resolvedProject+"/sections", api.RequestBody{
		Data: data,
		Options: &api.RequestOptions{
			Fields: splitFields(sectionDetailFields),
		},
	})
	if err != nil {
		return err
	}

	var response api.Response[api.SectionDetail]
	if err := json.Unmarshal(payload, &response); err != nil {
		return &api.ResponseError{Err: err}
	}

	section := response.Data
	output := sectionOutput{
		Action: "created",
		DryRun: false,
		Section: &sectionOutputDetail{
			GID:  section.GID,
			Name: section.Name,
		},
	}
	if section.Project != nil {
		output.Section.Project = &projectRef{GID: section.Project.GID, Name: &section.Project.Name}
	}

	if runtimeOutputJSON(cmd) {
		encoder := json.NewEncoder(cmd.OutOrStdout())
		encoder.SetIndent("", "  ")
		return encoder.Encode(output)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Created section: %s (%s)\n", section.Name, section.GID)
	return nil
}

func runSectionUpdate(cmd *cobra.Command, args []string) error {
	sectionGID := args[0]
	name, _ := cmd.Flags().GetString("name")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	if err := validateGID(sectionGID, "section gid"); err != nil {
		return err
	}

	if name == "" {
		return fmt.Errorf("--name is required for update")
	}

	if dryRun {
		output := sectionOutput{
			Action: "updated",
			DryRun: true,
			Section: &sectionOutputDetail{
				GID:  sectionGID,
				Name: name,
			},
		}
		if runtimeOutputJSON(cmd) {
			encoder := json.NewEncoder(cmd.OutOrStdout())
			encoder.SetIndent("", "  ")
			return encoder.Encode(output)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "[dry-run] Would update section %s name to %q\n", sectionGID, name)
		return nil
	}

	payload, err := runtimeClient(cmd).Put(cmd.Context(), "/sections/"+sectionGID, api.RequestBody{
		Data: map[string]string{"name": name},
		Options: &api.RequestOptions{
			Fields: splitFields(sectionDetailFields),
		},
	})
	if err != nil {
		return err
	}

	var response api.Response[api.SectionDetail]
	if err := json.Unmarshal(payload, &response); err != nil {
		return &api.ResponseError{Err: err}
	}

	section := response.Data
	output := sectionOutput{
		Action: "updated",
		DryRun: false,
		Section: &sectionOutputDetail{
			GID:  section.GID,
			Name: section.Name,
		},
	}
	if section.Project != nil {
		output.Section.Project = &projectRef{GID: section.Project.GID, Name: &section.Project.Name}
	}

	if runtimeOutputJSON(cmd) {
		encoder := json.NewEncoder(cmd.OutOrStdout())
		encoder.SetIndent("", "  ")
		return encoder.Encode(output)
	}

	fmt.Fprintln(cmd.OutOrStdout(), "Section updated")
	return nil
}

func runSectionDelete(cmd *cobra.Command, args []string) error {
	sectionGID := args[0]
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	if err := validateGID(sectionGID, "section gid"); err != nil {
		return err
	}

	if dryRun {
		output := sectionOutput{
			Action: "deleted",
			DryRun: true,
			Section: &sectionOutputDetail{
				GID: sectionGID,
			},
		}
		if runtimeOutputJSON(cmd) {
			encoder := json.NewEncoder(cmd.OutOrStdout())
			encoder.SetIndent("", "  ")
			return encoder.Encode(output)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "[dry-run] Would delete section %s\n", sectionGID)
		return nil
	}

	if err := runtimeClient(cmd).Delete(cmd.Context(), "/sections/"+sectionGID); err != nil {
		return err
	}

	output := sectionOutput{
		Action: "deleted",
		DryRun: false,
		Section: &sectionOutputDetail{
			GID: sectionGID,
		},
	}
	if runtimeOutputJSON(cmd) {
		encoder := json.NewEncoder(cmd.OutOrStdout())
		encoder.SetIndent("", "  ")
		return encoder.Encode(output)
	}

	fmt.Fprintln(cmd.OutOrStdout(), "Section deleted")
	return nil
}

func runSectionMove(cmd *cobra.Command, args []string) error {
	sectionGID := args[0]
	projectName, _ := cmd.Flags().GetString("project")
	projectGID, _ := cmd.Flags().GetString("project-gid")
	workspaceName, _ := cmd.Flags().GetString("workspace")
	workspaceGID, _ := cmd.Flags().GetString("workspace-gid")
	beforeGID, _ := cmd.Flags().GetString("before")
	afterGID, _ := cmd.Flags().GetString("after")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	if err := validateGID(sectionGID, "section gid"); err != nil {
		return err
	}

	if projectName != "" && projectGID != "" {
		return fmt.Errorf("use only one of --project or --project-gid")
	}
	if workspaceName != "" && workspaceGID != "" {
		return fmt.Errorf("use only one of --workspace or --workspace-gid")
	}
	if beforeGID != "" && afterGID != "" {
		return fmt.Errorf("use only one of --before or --after")
	}
	if beforeGID == "" && afterGID == "" {
		return fmt.Errorf("one of --before or --after is required")
	}

	// Resolve project - required for the insertInProject endpoint
	resolvedProject, err := resolveSectionProject(cmd, projectName, projectGID, workspaceName, workspaceGID)
	if err != nil {
		return err
	}

	resolvedBefore := beforeGID
	if beforeGID != "" {
		if resolve.IsGID(beforeGID) {
			if err := validateGID(beforeGID, "before"); err != nil {
				return err
			}
		} else {
			gid, err := resolveSectionWithPrompt(cmd.Context(), runtimeClient(cmd), resolvedProject, beforeGID, runtimeNoPrompt(cmd) || runtimeOutputJSON(cmd))
			if err != nil {
				return err
			}
			resolvedBefore = gid
		}
	}
	resolvedAfter := afterGID
	if afterGID != "" {
		if resolve.IsGID(afterGID) {
			if err := validateGID(afterGID, "after"); err != nil {
				return err
			}
		} else {
			gid, err := resolveSectionWithPrompt(cmd.Context(), runtimeClient(cmd), resolvedProject, afterGID, runtimeNoPrompt(cmd) || runtimeOutputJSON(cmd))
			if err != nil {
				return err
			}
			resolvedAfter = gid
		}
	}

	if dryRun {
		output := sectionOutput{
			Action: "moved",
			DryRun: true,
			Section: &sectionOutputDetail{
				GID: sectionGID,
			},
		}
		if runtimeOutputJSON(cmd) {
			encoder := json.NewEncoder(cmd.OutOrStdout())
			encoder.SetIndent("", "  ")
			return encoder.Encode(output)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "[dry-run] Would move section %s in project %s\n", sectionGID, resolvedProject)
		return nil
	}

	// Build request body
	data := map[string]interface{}{
		"section": sectionGID,
		"project": resolvedProject,
	}
	if resolvedBefore != "" {
		data["before_section"] = resolvedBefore
	}
	if resolvedAfter != "" {
		data["after_section"] = resolvedAfter
	}

	_, err = runtimeClient(cmd).Post(cmd.Context(), "/projects/"+resolvedProject+"/sections/insert", api.RequestBody{
		Data: data,
	})
	if err != nil {
		return err
	}

	output := sectionOutput{
		Action: "moved",
		DryRun: false,
		Section: &sectionOutputDetail{
			GID: sectionGID,
		},
	}
	if runtimeOutputJSON(cmd) {
		encoder := json.NewEncoder(cmd.OutOrStdout())
		encoder.SetIndent("", "  ")
		return encoder.Encode(output)
	}

	fmt.Fprintln(cmd.OutOrStdout(), "Section moved")
	return nil
}

// resolveSectionProject resolves a project from flags or config for section commands.
func resolveSectionProject(cmd *cobra.Command, projectName, projectGID, workspaceName, workspaceGID string) (string, error) {
	if projectGID != "" {
		return projectGID, nil
	}

	// Get workspace for project resolution
	resolvedWorkspace := workspaceGID
	if workspaceName != "" {
		gid, err := resolveWorkspaceWithPrompt(cmd.Context(), runtimeClient(cmd), workspaceName, runtimeNoPrompt(cmd) || runtimeOutputJSON(cmd))
		if err != nil {
			return "", err
		}
		resolvedWorkspace = gid
	}

	// Try to get from config if needed
	if projectName == "" && resolvedWorkspace == "" {
		cfg, err := config.LoadConfig(runtimeConfigPath(cmd))
		if err == nil {
			if cfg.Defaults.ProjectGID != "" {
				return cfg.Defaults.ProjectGID, nil
			}
			resolvedWorkspace = cfg.Defaults.WorkspaceGID
		}
	}

	if projectName != "" {
		if resolvedWorkspace == "" {
			cfg, err := config.LoadConfig(runtimeConfigPath(cmd))
			if err == nil {
				resolvedWorkspace = cfg.Defaults.WorkspaceGID
			}
		}
		if resolvedWorkspace == "" {
			return "", fmt.Errorf("--workspace or --workspace-gid required when using --project")
		}
		return resolveProjectWithPrompt(cmd.Context(), runtimeClient(cmd), resolvedWorkspace, projectName, runtimeNoPrompt(cmd) || runtimeOutputJSON(cmd))
	}

	return "", fmt.Errorf("--project or --project-gid is required")
}
