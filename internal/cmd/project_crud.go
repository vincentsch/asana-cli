// Package cmd implements project view, create, update, and delete commands.
package cmd

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/spf13/cobra"
	"github.com/vincentsch/asana-cli/internal/api"
	"github.com/vincentsch/asana-cli/internal/config"
)

var projectViewCmd = &cobra.Command{
	Use:   "view <project-gid>",
	Short: "View project details",
	Long: `View detailed information about a project.

Shows name, owner, team, due date, task counts, and other metadata.

See also: project list, section list, task list`,
	Example: `  # View project by GID
  asana project view 1234567890123456

  # Output as JSON
  asana project view 1234567890123456 --json

  # Workflow: View project then list its contents
  asana project view 1234567890123456
  asana section list --project-gid 1234567890123456
  asana task list --project-gid 1234567890123456`,
	Args: cobra.ExactArgs(1),
	RunE: runProjectView,
}

var projectCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new project",
	Long: `Create a new project in a workspace.

For organization workspaces, a team is required. Set color, due date, and
privacy settings as needed.

See also: project list, project view, section create`,
	Example: `  # Create a simple project
  asana project create "My Project" -w "My Workspace"

  # Create in a team with color
  asana project create "Q1 Goals" -w "Company" -t "Engineering" --color dark-blue

  # Create with due date and privacy
  asana project create "Sprint 1" -w "Company" -t "Dev" --due-on 2024-12-31 --privacy private_to_team

  # Preview without creating
  asana project create "Test" -w "My Workspace" --dry-run

  # Workflow: Create project and add sections
  asana project create "New Sprint" -w "My Workspace"
  asana section create "To Do" --project-gid <new-gid>
  asana section create "In Progress" --project-gid <new-gid>
  asana section create "Done" --project-gid <new-gid>`,
	Args: cobra.ExactArgs(1),
	RunE: runProjectCreate,
}

var projectUpdateCmd = &cobra.Command{
	Use:   "update <project-gid>",
	Short: "Update a project",
	Long: `Update project properties.

Supports renaming, updating notes, color, due date, and archive status.

See also: project view, project delete`,
	Example: `  # Rename a project
  asana project update 1234567890123456 --name "New Name"

  # Archive a project
  asana project update 1234567890123456 --archived true

  # Update multiple fields
  asana project update 1234567890123456 --color light-green --due-on 2025-01-15

  # Preview changes
  asana project update 1234567890123456 --name "Test" --dry-run`,
	Args: cobra.ExactArgs(1),
	RunE: runProjectUpdate,
}

var projectDeleteCmd = &cobra.Command{
	Use:   "delete <project-gid>",
	Short: "Delete a project",
	Long: `Permanently delete a project.

This action cannot be undone. All tasks in the project will also be deleted.
Use --dry-run to preview. Consider archiving instead with project update --archived true.

See also: project update, project list`,
	Example: `  # Delete a project
  asana project delete 1234567890123456 --confirm

  # Preview deletion
  asana project delete 1234567890123456 --dry-run`,
	Args: cobra.ExactArgs(1),
	RunE: runProjectDelete,
}

// projectDetailFields are fields requested when viewing a project.
const projectDetailFields = "gid,name,archived,color,notes,due_on,privacy_setting,owner.gid,owner.name,team.gid,team.name"

func init() {
	// View flags
	// (none needed)

	// Create flags
	projectCreateCmd.Flags().StringP("workspace", "w", "", "Workspace name")
	projectCreateCmd.Flags().String("workspace-gid", "", "Workspace GID")
	projectCreateCmd.Flags().StringP("team", "t", "", "Team name (required for organizations)")
	projectCreateCmd.Flags().String("team-gid", "", "Team GID")
	projectCreateCmd.Flags().String("color", "", "Project color")
	projectCreateCmd.Flags().String("due-on", "", "Project due date (YYYY-MM-DD)")
	projectCreateCmd.Flags().String("privacy", "", "Privacy setting: public_to_workspace, private_to_team, or private")

	// Update flags
	projectUpdateCmd.Flags().String("name", "", "New project name")
	projectUpdateCmd.Flags().String("notes", "", "Project notes/description")
	projectUpdateCmd.Flags().String("color", "", "Project color")
	projectUpdateCmd.Flags().String("due-on", "", "Project due date (YYYY-MM-DD)")
	projectUpdateCmd.Flags().String("archived", "", "Archive status: true or false")

	// Delete flags

	projectCmd.AddCommand(projectViewCmd)
	projectCmd.AddCommand(projectCreateCmd)
	projectCmd.AddCommand(projectUpdateCmd)
	projectCmd.AddCommand(projectDeleteCmd)
}

// projectOutput is used for JSON output of project commands.
type projectOutput struct {
	Action  string               `json:"action"`
	DryRun  bool                 `json:"dry_run"`
	Project *projectOutputDetail `json:"project,omitempty"`
}

type projectOutputDetail struct {
	GID            string   `json:"gid"`
	Name           string   `json:"name"`
	Archived       bool     `json:"archived,omitempty"`
	Color          string   `json:"color,omitempty"`
	Notes          string   `json:"notes,omitempty"`
	DueOn          string   `json:"due_on,omitempty"`
	PrivacySetting string   `json:"privacy_setting,omitempty"`
	Owner          *userRef `json:"owner,omitempty"`
	Team           *teamRef `json:"team,omitempty"`
}

type projectViewOutput struct {
	Project    projectOutputDetail   `json:"project"`
	TaskCounts api.ProjectTaskCounts `json:"task_counts"`
}

type teamRef struct {
	GID  string `json:"gid"`
	Name string `json:"name,omitempty"`
}

func runProjectView(cmd *cobra.Command, args []string) error {
	projectGID := args[0]

	if err := validateGID(projectGID, "project gid"); err != nil {
		return err
	}

	query := url.Values{}
	query.Set("opt_fields", projectDetailFields)

	payload, err := runtimeClient(cmd).Get(cmd.Context(), "/projects/"+projectGID, query)
	if err != nil {
		return err
	}

	var response api.Response[api.ProjectDetail]
	if err := json.Unmarshal(payload, &response); err != nil {
		return &api.ResponseError{Err: err}
	}

	countPayload, err := runtimeClient(cmd).Get(cmd.Context(), "/projects/"+projectGID+"/task_counts", url.Values{})
	if err != nil {
		return err
	}
	var countResp api.Response[api.ProjectTaskCounts]
	if err := json.Unmarshal(countPayload, &countResp); err != nil {
		return &api.ResponseError{Err: err}
	}

	project := response.Data
	output := projectOutputDetail{
		GID:            project.GID,
		Name:           project.Name,
		Archived:       project.Archived,
		Color:          project.Color,
		Notes:          project.Notes,
		DueOn:          project.DueOn,
		PrivacySetting: project.PrivacySetting,
	}
	if project.Owner != nil {
		output.Owner = &userRef{GID: project.Owner.GID, Name: &project.Owner.Name}
	}
	if project.Team != nil {
		output.Team = &teamRef{GID: project.Team.GID, Name: project.Team.Name}
	}

	if runtimeOutputJSON(cmd) {
		encoder := json.NewEncoder(cmd.OutOrStdout())
		encoder.SetIndent("", "  ")
		return encoder.Encode(projectViewOutput{
			Project:    output,
			TaskCounts: countResp.Data,
		})
	}

	// Text output
	fmt.Fprintf(cmd.OutOrStdout(), "GID:      %s\n", project.GID)
	fmt.Fprintf(cmd.OutOrStdout(), "Name:     %s\n", project.Name)
	fmt.Fprintf(cmd.OutOrStdout(), "Archived: %t\n", project.Archived)
	if project.Color != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "Color:    %s\n", project.Color)
	}
	if project.DueOn != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "Due:      %s\n", project.DueOn)
	}
	if project.PrivacySetting != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "Privacy:  %s\n", project.PrivacySetting)
	}
	if project.Owner != nil {
		fmt.Fprintf(cmd.OutOrStdout(), "Owner:    %s (%s)\n", project.Owner.Name, project.Owner.GID)
	}
	if project.Team != nil {
		fmt.Fprintf(cmd.OutOrStdout(), "Team:     %s (%s)\n", project.Team.Name, project.Team.GID)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Tasks:    %d (incomplete: %d, completed: %d)\n", countResp.Data.NumTasks, countResp.Data.NumIncompleteTasks, countResp.Data.NumCompletedTasks)

	return nil
}

func runProjectCreate(cmd *cobra.Command, args []string) error {
	name := args[0]
	workspaceName, _ := cmd.Flags().GetString("workspace")
	workspaceGID, _ := cmd.Flags().GetString("workspace-gid")
	teamName, _ := cmd.Flags().GetString("team")
	teamGID, _ := cmd.Flags().GetString("team-gid")
	color, _ := cmd.Flags().GetString("color")
	dueOn, _ := cmd.Flags().GetString("due-on")
	privacy, _ := cmd.Flags().GetString("privacy")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	if workspaceName != "" && workspaceGID != "" {
		return fmt.Errorf("use only one of --workspace or --workspace-gid")
	}
	if teamName != "" && teamGID != "" {
		return fmt.Errorf("use only one of --team or --team-gid")
	}

	needsWorkspace := teamName != "" || (teamName == "" && teamGID == "")
	resolvedWorkspace := ""
	if needsWorkspace || workspaceName != "" || workspaceGID != "" {
		workspace, err := resolveWorkspaceFromFlags(cmd, workspaceName, workspaceGID)
		if err != nil {
			return err
		}
		resolvedWorkspace = workspace
	}

	// Resolve team if provided
	resolvedTeam := teamGID
	if teamName != "" {
		gid, err := resolveTeamWithPrompt(cmd.Context(), runtimeClient(cmd), resolvedWorkspace, teamName, runtimeNoPrompt(cmd) || runtimeOutputJSON(cmd))
		if err != nil {
			return err
		}
		resolvedTeam = gid
	}

	// Validate due date if provided
	if dueOn != "" {
		if _, err := parseDateOnly(dueOn); err != nil {
			return fmt.Errorf("--due-on: %w", err)
		}
	}

	// Validate privacy if provided
	if privacy != "" && privacy != "public_to_workspace" && privacy != "private_to_team" && privacy != "private" {
		return fmt.Errorf("--privacy must be public_to_workspace, private_to_team, or private")
	}

	if dryRun {
		output := projectOutput{
			Action: "created",
			DryRun: true,
			Project: &projectOutputDetail{
				Name:           name,
				Color:          color,
				DueOn:          dueOn,
				PrivacySetting: privacy,
			},
		}
		if runtimeOutputJSON(cmd) {
			encoder := json.NewEncoder(cmd.OutOrStdout())
			encoder.SetIndent("", "  ")
			return encoder.Encode(output)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "[dry-run] Would create project: %s\n", name)
		return nil
	}

	// Build request body
	data := map[string]interface{}{
		"name": name,
	}
	if resolvedWorkspace != "" {
		data["workspace"] = resolvedWorkspace
	}
	if resolvedTeam != "" {
		data["team"] = resolvedTeam
	}
	if color != "" {
		data["color"] = color
	}
	if dueOn != "" {
		data["due_on"] = dueOn
	}
	if privacy != "" {
		data["privacy_setting"] = privacy
	}

	payload, err := runtimeClient(cmd).Post(cmd.Context(), "/projects", api.RequestBody{
		Data: data,
		Options: &api.RequestOptions{
			Fields: splitFields(projectDetailFields),
		},
	})
	if err != nil {
		return err
	}

	var response api.Response[api.ProjectDetail]
	if err := json.Unmarshal(payload, &response); err != nil {
		return &api.ResponseError{Err: err}
	}

	project := response.Data
	output := projectOutput{
		Action: "created",
		DryRun: false,
		Project: &projectOutputDetail{
			GID:            project.GID,
			Name:           project.Name,
			Color:          project.Color,
			DueOn:          project.DueOn,
			PrivacySetting: project.PrivacySetting,
		},
	}
	if project.Owner != nil {
		output.Project.Owner = &userRef{GID: project.Owner.GID, Name: &project.Owner.Name}
	}
	if project.Team != nil {
		output.Project.Team = &teamRef{GID: project.Team.GID, Name: project.Team.Name}
	}

	if runtimeOutputJSON(cmd) {
		encoder := json.NewEncoder(cmd.OutOrStdout())
		encoder.SetIndent("", "  ")
		return encoder.Encode(output)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Created project: %s (%s)\n", project.Name, project.GID)
	return nil
}

func runProjectUpdate(cmd *cobra.Command, args []string) error {
	projectGID := args[0]
	name, _ := cmd.Flags().GetString("name")
	notes, _ := cmd.Flags().GetString("notes")
	color, _ := cmd.Flags().GetString("color")
	dueOn, _ := cmd.Flags().GetString("due-on")
	archivedValue, _ := cmd.Flags().GetString("archived")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	if err := validateGID(projectGID, "project gid"); err != nil {
		return err
	}

	// Validate due date if provided
	if dueOn != "" {
		if _, err := parseDateOnly(dueOn); err != nil {
			return fmt.Errorf("--due-on: %w", err)
		}
	}

	var archivedSet bool
	var archived bool
	if archivedValue != "" {
		switch strings.ToLower(archivedValue) {
		case "true":
			archived = true
			archivedSet = true
		case "false":
			archived = false
			archivedSet = true
		default:
			return fmt.Errorf("--archived must be true or false")
		}
	}

	// Build update data
	data := make(map[string]interface{})
	if name != "" {
		data["name"] = name
	}
	if cmd.Flags().Changed("notes") {
		data["notes"] = notes
	}
	if color != "" {
		data["color"] = color
	}
	if dueOn != "" {
		data["due_on"] = dueOn
	}
	if archivedSet {
		data["archived"] = archived
	}

	if len(data) == 0 {
		return fmt.Errorf("no update flags provided")
	}

	if dryRun {
		output := projectOutput{
			Action: "updated",
			DryRun: true,
			Project: &projectOutputDetail{
				GID: projectGID,
			},
		}
		if runtimeOutputJSON(cmd) {
			encoder := json.NewEncoder(cmd.OutOrStdout())
			encoder.SetIndent("", "  ")
			return encoder.Encode(output)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "[dry-run] Would update project %s\n", projectGID)
		return nil
	}

	payload, err := runtimeClient(cmd).Put(cmd.Context(), "/projects/"+projectGID, api.RequestBody{
		Data: data,
		Options: &api.RequestOptions{
			Fields: splitFields(projectDetailFields),
		},
	})
	if err != nil {
		return err
	}

	var response api.Response[api.ProjectDetail]
	if err := json.Unmarshal(payload, &response); err != nil {
		return &api.ResponseError{Err: err}
	}

	project := response.Data
	output := projectOutput{
		Action: "updated",
		DryRun: false,
		Project: &projectOutputDetail{
			GID:            project.GID,
			Name:           project.Name,
			Archived:       project.Archived,
			Notes:          project.Notes,
			Color:          project.Color,
			DueOn:          project.DueOn,
			PrivacySetting: project.PrivacySetting,
		},
	}
	if project.Owner != nil {
		output.Project.Owner = &userRef{GID: project.Owner.GID, Name: &project.Owner.Name}
	}
	if project.Team != nil {
		output.Project.Team = &teamRef{GID: project.Team.GID, Name: project.Team.Name}
	}

	if runtimeOutputJSON(cmd) {
		encoder := json.NewEncoder(cmd.OutOrStdout())
		encoder.SetIndent("", "  ")
		return encoder.Encode(output)
	}

	fmt.Fprintln(cmd.OutOrStdout(), "Project updated")
	return nil
}

func runProjectDelete(cmd *cobra.Command, args []string) error {
	projectGID := args[0]
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	if err := validateGID(projectGID, "project gid"); err != nil {
		return err
	}

	if dryRun {
		output := projectOutput{
			Action: "deleted",
			DryRun: true,
			Project: &projectOutputDetail{
				GID: projectGID,
			},
		}
		if runtimeOutputJSON(cmd) {
			encoder := json.NewEncoder(cmd.OutOrStdout())
			encoder.SetIndent("", "  ")
			return encoder.Encode(output)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "[dry-run] Would delete project %s\n", projectGID)
		return nil
	}

	if err := runtimeClient(cmd).Delete(cmd.Context(), "/projects/"+projectGID); err != nil {
		return err
	}

	output := projectOutput{
		Action: "deleted",
		DryRun: false,
		Project: &projectOutputDetail{
			GID: projectGID,
		},
	}
	if runtimeOutputJSON(cmd) {
		encoder := json.NewEncoder(cmd.OutOrStdout())
		encoder.SetIndent("", "  ")
		return encoder.Encode(output)
	}

	fmt.Fprintln(cmd.OutOrStdout(), "Project deleted")
	return nil
}

// resolveWorkspaceFromFlags resolves workspace from flags or config.
func resolveWorkspaceFromFlags(cmd *cobra.Command, workspaceName, workspaceGID string) (string, error) {
	if workspaceGID != "" {
		return workspaceGID, nil
	}
	if workspaceName != "" {
		return resolveWorkspaceWithPrompt(cmd.Context(), runtimeClient(cmd), workspaceName, runtimeNoPrompt(cmd) || runtimeOutputJSON(cmd))
	}
	// Try config default
	cfg, err := config.LoadConfig(runtimeConfigPath(cmd))
	if err == nil && cfg.Defaults.WorkspaceGID != "" {
		return cfg.Defaults.WorkspaceGID, nil
	}
	return "", fmt.Errorf("--workspace or --workspace-gid is required")
}
