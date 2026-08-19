// Package cmd implements task parent, project, tag, and follower commands.
package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/vincentsch/asana-cli/internal/api"
	"github.com/vincentsch/asana-cli/internal/config"
	"github.com/vincentsch/asana-cli/internal/resolve"
)

var taskParentCmd = &cobra.Command{
	Use:   "parent",
	Short: "Manage task parent",
	Long: `Manage the parent task, converting a task into a subtask.

See also: task subtask, task view`,
}

var taskParentSetCmd = &cobra.Command{
	Use:   "set <gid>",
	Short: "Set or change task parent",
	Long: `Set the parent of a task, making it a subtask.

Optionally position the task among siblings using --insert-before or --insert-after.

See also: task subtask create, task subtask list`,
	Example: `  # Make task a subtask of another
  asana task parent set 1234567890123456 --parent 9876543210987654

  # Position among siblings
  asana task parent set 1234567890123456 --parent 9876543210987654 --insert-after 5555555555555555

  # Preview without changing
  asana task parent set 1234567890123456 --parent 9876543210987654 --dry-run`,
	Args: cobra.ExactArgs(1),
	RunE: runTaskParentSet,
}

var taskProjectCmd = &cobra.Command{
	Use:   "project",
	Short: "Manage task project membership",
	Long: `Add or remove a task from projects. Tasks can belong to multiple projects.

See also: task move, project list`,
}

var taskProjectAddCmd = &cobra.Command{
	Use:   "add <gid>",
	Short: "Add task to a project",
	Long: `Add a task to a project, optionally in a specific section.

If no section is specified, the task is added to the project without a section.

See also: task project remove, task move, project list`,
	Example: `  # Add task to a project
  asana task project add 1234567890123456 -p "My Project"

  # Add to a specific section
  asana task project add 1234567890123456 -p "My Project" -s "In Progress"

  # Workflow: Add task to multiple projects
  asana task project add 1234567890123456 -p "Sprint 1"
  asana task project add 1234567890123456 -p "Q1 Goals"`,
	Args: cobra.ExactArgs(1),
	RunE: runTaskProjectAdd,
}

var taskProjectRemoveCmd = &cobra.Command{
	Use:   "remove <gid>",
	Short: "Remove task from a project",
	Long: `Remove a task from a project. The task is not deleted, just unlinked.

See also: task project add, task view`,
	Example: `  # Remove task from a project
  asana task project remove 1234567890123456 -p "My Project" --confirm

  # Preview without removing
  asana task project remove 1234567890123456 -p "My Project" --dry-run`,
	Args: cobra.ExactArgs(1),
	RunE: runTaskProjectRemove,
}

var taskTagCmd = &cobra.Command{
	Use:   "tag",
	Short: "Manage task tags",
	Long: `Add or remove tags from a task. Tags help categorize tasks across projects.

See also: tag list, tag tasks`,
}

var taskTagAddCmd = &cobra.Command{
	Use:   "add <gid>",
	Short: "Add tag to task",
	Long: `Add a tag to a task.

See also: task tag remove, tag list, tag tasks`,
	Example: `  # Add tag by name
  asana task tag add 1234567890123456 --tag "urgent" -w "My Workspace"

  # Add tag by GID
  asana task tag add 1234567890123456 --tag-gid 9876543210987654

  # Workflow: Categorize a task with multiple tags
  asana task tag add 1234567890123456 --tag "urgent" -w "My Workspace"
  asana task tag add 1234567890123456 --tag "bug" -w "My Workspace"`,
	Args: cobra.ExactArgs(1),
	RunE: runTaskTagAdd,
}

var taskTagRemoveCmd = &cobra.Command{
	Use:   "remove <gid>",
	Short: "Remove tag from task",
	Long: `Remove a tag from a task.

See also: task tag add, tag list`,
	Example: `  # Remove tag
  asana task tag remove 1234567890123456 --tag "urgent" -w "My Workspace" --confirm

  # Preview without removing
  asana task tag remove 1234567890123456 --tag "urgent" -w "My Workspace" --dry-run`,
	Args: cobra.ExactArgs(1),
	RunE: runTaskTagRemove,
}

var taskFollowerCmd = &cobra.Command{
	Use:   "follower",
	Short: "Manage task followers",
	Long: `Add or remove followers from a task. Followers receive notifications about task updates.

See also: user list, task view`,
}

var taskFollowerAddCmd = &cobra.Command{
	Use:   "add <gid>",
	Short: "Add followers to task",
	Long: `Add one or more followers to a task.

See also: task follower remove, user list`,
	Example: `  # Add yourself as follower
  asana task follower add 1234567890123456 --user me

  # Add by email
  asana task follower add 1234567890123456 --user jane@example.com -w "My Workspace"

  # Add multiple users
  asana task follower add 1234567890123456 --user me --user 9876543210987654

  # Workflow: Create task and add team as followers
  asana task create "New feature" -p "Sprint"
  asana task follower add <new-gid> --user me --user dev@example.com`,
	Args: cobra.ExactArgs(1),
	RunE: runTaskFollowerAdd,
}

var taskFollowerRemoveCmd = &cobra.Command{
	Use:   "remove <gid>",
	Short: "Remove followers from task",
	Long: `Remove one or more followers from a task.

See also: task follower add, task view`,
	Example: `  # Remove a follower
  asana task follower remove 1234567890123456 --user 9876543210987654 --confirm

  # Preview without removing
  asana task follower remove 1234567890123456 --user me --dry-run`,
	Args: cobra.ExactArgs(1),
	RunE: runTaskFollowerRemove,
}

func init() {
	// Parent commands
	taskParentSetCmd.Flags().String("parent", "", "New parent task GID (required)")
	taskParentSetCmd.Flags().String("insert-before", "", "Insert before this sibling task GID")
	taskParentSetCmd.Flags().String("insert-after", "", "Insert after this sibling task GID")

	taskParentCmd.AddCommand(taskParentSetCmd)
	taskCmd.AddCommand(taskParentCmd)

	// Project commands
	taskProjectAddCmd.Flags().StringP("project", "p", "", "Project name")
	taskProjectAddCmd.Flags().String("project-gid", "", "Project GID")
	taskProjectAddCmd.Flags().StringP("section", "s", "", "Section name within project")
	taskProjectAddCmd.Flags().String("section-gid", "", "Section GID")
	taskProjectAddCmd.Flags().StringP("workspace", "w", "", "Workspace name (for project/section name resolution)")
	taskProjectAddCmd.Flags().String("workspace-gid", "", "Workspace GID")
	taskProjectAddCmd.Flags().String("insert-before", "", "Insert before this task GID")
	taskProjectAddCmd.Flags().String("insert-after", "", "Insert after this task GID")

	taskProjectRemoveCmd.Flags().StringP("project", "p", "", "Project name")
	taskProjectRemoveCmd.Flags().String("project-gid", "", "Project GID")
	taskProjectRemoveCmd.Flags().StringP("workspace", "w", "", "Workspace name (for project name resolution)")
	taskProjectRemoveCmd.Flags().String("workspace-gid", "", "Workspace GID")

	taskProjectCmd.AddCommand(taskProjectAddCmd)
	taskProjectCmd.AddCommand(taskProjectRemoveCmd)
	taskCmd.AddCommand(taskProjectCmd)

	// Tag commands
	taskTagAddCmd.Flags().String("tag", "", "Tag name")
	taskTagAddCmd.Flags().String("tag-gid", "", "Tag GID")
	taskTagAddCmd.Flags().StringP("workspace", "w", "", "Workspace name (for tag name resolution)")
	taskTagAddCmd.Flags().String("workspace-gid", "", "Workspace GID")

	taskTagRemoveCmd.Flags().String("tag", "", "Tag name")
	taskTagRemoveCmd.Flags().String("tag-gid", "", "Tag GID")
	taskTagRemoveCmd.Flags().StringP("workspace", "w", "", "Workspace name (for tag name resolution)")
	taskTagRemoveCmd.Flags().String("workspace-gid", "", "Workspace GID")

	taskTagCmd.AddCommand(taskTagAddCmd)
	taskTagCmd.AddCommand(taskTagRemoveCmd)
	taskCmd.AddCommand(taskTagCmd)

	// Follower commands
	taskFollowerAddCmd.Flags().StringSlice("user", nil, "User(s) to add (me, GID, or email)")
	taskFollowerAddCmd.Flags().StringP("workspace", "w", "", "Workspace name (for email resolution)")
	taskFollowerAddCmd.Flags().String("workspace-gid", "", "Workspace GID")

	taskFollowerRemoveCmd.Flags().StringSlice("user", nil, "User(s) to remove (me, GID, or email)")
	taskFollowerRemoveCmd.Flags().StringP("workspace", "w", "", "Workspace name (for email resolution)")
	taskFollowerRemoveCmd.Flags().String("workspace-gid", "", "Workspace GID")

	taskFollowerCmd.AddCommand(taskFollowerAddCmd)
	taskFollowerCmd.AddCommand(taskFollowerRemoveCmd)
	taskCmd.AddCommand(taskFollowerCmd)
}

type membershipOutput struct {
	Action  string   `json:"action"`
	DryRun  bool     `json:"dry_run"`
	Task    gidRef   `json:"task"`
	Project *gidRef  `json:"project,omitempty"`
	Section *gidRef  `json:"section,omitempty"`
	Tag     *gidRef  `json:"tag,omitempty"`
	Users   []string `json:"users,omitempty"`
}

type parentOutput struct {
	Action       string  `json:"action"`
	DryRun       bool    `json:"dry_run"`
	Task         gidRef  `json:"task"`
	Parent       gidRef  `json:"parent"`
	InsertBefore *string `json:"insert_before"`
	InsertAfter  *string `json:"insert_after"`
}

func runTaskParentSet(cmd *cobra.Command, args []string) error {
	taskGID := args[0]
	parentGID, _ := cmd.Flags().GetString("parent")
	insertBefore, _ := cmd.Flags().GetString("insert-before")
	insertAfter, _ := cmd.Flags().GetString("insert-after")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	if err := validateGID(taskGID, "gid"); err != nil {
		return err
	}
	if parentGID == "" {
		return fmt.Errorf("--parent is required")
	}
	if err := validateGID(parentGID, "parent"); err != nil {
		return err
	}
	if insertBefore != "" && insertAfter != "" {
		return fmt.Errorf("cannot use both --insert-before and --insert-after")
	}
	if insertBefore != "" {
		if err := validateGID(insertBefore, "insert-before"); err != nil {
			return err
		}
	}
	if insertAfter != "" {
		if err := validateGID(insertAfter, "insert-after"); err != nil {
			return err
		}
	}

	if dryRun {
		// Validate both tasks exist
		if _, err := fetchTaskWithFields(cmd.Context(), taskGID, "gid"); err != nil {
			return err
		}
		if _, err := fetchTaskWithFields(cmd.Context(), parentGID, "gid"); err != nil {
			return err
		}
		output := parentOutput{
			Action:       "set_parent",
			DryRun:       true,
			Task:         gidRef{GID: taskGID},
			Parent:       gidRef{GID: parentGID},
			InsertBefore: optionalString(insertBefore),
			InsertAfter:  optionalString(insertAfter),
		}
		if runtimeOutputJSON(cmd) {
			encoder := json.NewEncoder(cmd.OutOrStdout())
			encoder.SetIndent("", "  ")
			return encoder.Encode(output)
		}
		_, err := fmt.Fprintf(cmd.OutOrStdout(), "[dry-run] Would set parent of task %s to %s\n", taskGID, parentGID)
		return err
	}

	data := map[string]any{"parent": parentGID}
	if insertBefore != "" {
		data["insert_before"] = insertBefore
	}
	if insertAfter != "" {
		data["insert_after"] = insertAfter
	}

	_, err := runtimeClient(cmd).Post(cmd.Context(), "/tasks/"+taskGID+"/setParent", api.RequestBody{
		Data: data,
	})
	if err != nil {
		return err
	}

	output := parentOutput{
		Action:       "set_parent",
		DryRun:       false,
		Task:         gidRef{GID: taskGID},
		Parent:       gidRef{GID: parentGID},
		InsertBefore: optionalString(insertBefore),
		InsertAfter:  optionalString(insertAfter),
	}
	if runtimeOutputJSON(cmd) {
		encoder := json.NewEncoder(cmd.OutOrStdout())
		encoder.SetIndent("", "  ")
		return encoder.Encode(output)
	}
	_, err = fmt.Fprintln(cmd.OutOrStdout(), "Parent set")
	return err
}

func runTaskProjectAdd(cmd *cobra.Command, args []string) error {
	taskGID := args[0]
	projectName, _ := cmd.Flags().GetString("project")
	projectGID, _ := cmd.Flags().GetString("project-gid")
	sectionName, _ := cmd.Flags().GetString("section")
	sectionGID, _ := cmd.Flags().GetString("section-gid")
	workspaceName, _ := cmd.Flags().GetString("workspace")
	workspaceGID, _ := cmd.Flags().GetString("workspace-gid")
	insertBefore, _ := cmd.Flags().GetString("insert-before")
	insertAfter, _ := cmd.Flags().GetString("insert-after")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	if err := validateGID(taskGID, "gid"); err != nil {
		return err
	}
	if projectName == "" && projectGID == "" {
		return fmt.Errorf("either --project or --project-gid is required")
	}
	if projectName != "" && projectGID != "" {
		return fmt.Errorf("use only one of --project or --project-gid")
	}
	if sectionName != "" && sectionGID != "" {
		return fmt.Errorf("use only one of --section or --section-gid")
	}
	if workspaceName != "" && workspaceGID != "" {
		return fmt.Errorf("use only one of --workspace or --workspace-gid")
	}
	if insertBefore != "" && insertAfter != "" {
		return fmt.Errorf("cannot use both --insert-before and --insert-after")
	}

	// Load config for workspace default if needed
	if projectName != "" && workspaceName == "" && workspaceGID == "" {
		cfg, err := config.LoadConfig(runtimeConfigPath(cmd))
		if err == nil && cfg.Defaults.WorkspaceGID != "" {
			workspaceGID = cfg.Defaults.WorkspaceGID
		}
	}

	// Resolve workspace
	resolvedWorkspace := workspaceGID
	if workspaceName != "" {
		gid, err := resolveWorkspaceWithPrompt(cmd.Context(), runtimeClient(cmd), workspaceName, runtimeNoPrompt(cmd) || runtimeOutputJSON(cmd))
		if err != nil {
			return err
		}
		resolvedWorkspace = gid
	}

	// Resolve project
	resolvedProjectGID := projectGID
	if projectName != "" {
		gid, err := resolveProjectWithPrompt(cmd.Context(), runtimeClient(cmd), resolvedWorkspace, projectName, runtimeNoPrompt(cmd) || runtimeOutputJSON(cmd))
		if err != nil {
			return err
		}
		resolvedProjectGID = gid
	}

	// Resolve section
	resolvedSectionGID := sectionGID
	if sectionName != "" {
		gid, err := resolveSectionWithPrompt(cmd.Context(), runtimeClient(cmd), resolvedProjectGID, sectionName, runtimeNoPrompt(cmd) || runtimeOutputJSON(cmd))
		if err != nil {
			return err
		}
		resolvedSectionGID = gid
	}

	if dryRun {
		// Validate task exists
		if _, err := fetchTaskWithFields(cmd.Context(), taskGID, "gid"); err != nil {
			return err
		}
		output := membershipOutput{
			Action:  "added",
			DryRun:  true,
			Task:    gidRef{GID: taskGID},
			Project: &gidRef{GID: resolvedProjectGID},
		}
		if resolvedSectionGID != "" {
			output.Section = &gidRef{GID: resolvedSectionGID}
		}
		if runtimeOutputJSON(cmd) {
			encoder := json.NewEncoder(cmd.OutOrStdout())
			encoder.SetIndent("", "  ")
			return encoder.Encode(output)
		}
		msg := fmt.Sprintf("[dry-run] Would add task %s to project %s", taskGID, resolvedProjectGID)
		if resolvedSectionGID != "" {
			msg += fmt.Sprintf(" (section %s)", resolvedSectionGID)
		}
		_, err := fmt.Fprintln(cmd.OutOrStdout(), msg)
		return err
	}

	data := map[string]any{"project": resolvedProjectGID}
	if resolvedSectionGID != "" {
		data["section"] = resolvedSectionGID
	}
	if insertBefore != "" {
		data["insert_before"] = insertBefore
	}
	if insertAfter != "" {
		data["insert_after"] = insertAfter
	}

	_, err := runtimeClient(cmd).Post(cmd.Context(), "/tasks/"+taskGID+"/addProject", api.RequestBody{
		Data: data,
	})
	if err != nil {
		return err
	}

	output := membershipOutput{
		Action:  "added",
		DryRun:  false,
		Task:    gidRef{GID: taskGID},
		Project: &gidRef{GID: resolvedProjectGID},
	}
	if resolvedSectionGID != "" {
		output.Section = &gidRef{GID: resolvedSectionGID}
	}
	if runtimeOutputJSON(cmd) {
		encoder := json.NewEncoder(cmd.OutOrStdout())
		encoder.SetIndent("", "  ")
		return encoder.Encode(output)
	}
	_, err = fmt.Fprintln(cmd.OutOrStdout(), "Task added to project")
	return err
}

func runTaskProjectRemove(cmd *cobra.Command, args []string) error {
	taskGID := args[0]
	projectName, _ := cmd.Flags().GetString("project")
	projectGID, _ := cmd.Flags().GetString("project-gid")
	workspaceName, _ := cmd.Flags().GetString("workspace")
	workspaceGID, _ := cmd.Flags().GetString("workspace-gid")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	if err := validateGID(taskGID, "gid"); err != nil {
		return err
	}
	if projectName == "" && projectGID == "" {
		return fmt.Errorf("either --project or --project-gid is required")
	}
	if projectName != "" && projectGID != "" {
		return fmt.Errorf("use only one of --project or --project-gid")
	}
	if workspaceName != "" && workspaceGID != "" {
		return fmt.Errorf("use only one of --workspace or --workspace-gid")
	}

	// Load config for workspace default if needed
	if projectName != "" && workspaceName == "" && workspaceGID == "" {
		cfg, err := config.LoadConfig(runtimeConfigPath(cmd))
		if err == nil && cfg.Defaults.WorkspaceGID != "" {
			workspaceGID = cfg.Defaults.WorkspaceGID
		}
	}

	// Resolve workspace
	resolvedWorkspace := workspaceGID
	if workspaceName != "" {
		gid, err := resolveWorkspaceWithPrompt(cmd.Context(), runtimeClient(cmd), workspaceName, runtimeNoPrompt(cmd) || runtimeOutputJSON(cmd))
		if err != nil {
			return err
		}
		resolvedWorkspace = gid
	}

	// Resolve project
	resolvedProjectGID := projectGID
	if projectName != "" {
		gid, err := resolveProjectWithPrompt(cmd.Context(), runtimeClient(cmd), resolvedWorkspace, projectName, runtimeNoPrompt(cmd) || runtimeOutputJSON(cmd))
		if err != nil {
			return err
		}
		resolvedProjectGID = gid
	}

	if dryRun {
		// Validate task exists
		if _, err := fetchTaskWithFields(cmd.Context(), taskGID, "gid"); err != nil {
			return err
		}
		output := membershipOutput{
			Action:  "removed",
			DryRun:  true,
			Task:    gidRef{GID: taskGID},
			Project: &gidRef{GID: resolvedProjectGID},
		}
		if runtimeOutputJSON(cmd) {
			encoder := json.NewEncoder(cmd.OutOrStdout())
			encoder.SetIndent("", "  ")
			return encoder.Encode(output)
		}
		_, err := fmt.Fprintf(cmd.OutOrStdout(), "[dry-run] Would remove task %s from project %s\n", taskGID, resolvedProjectGID)
		return err
	}

	_, err := runtimeClient(cmd).Post(cmd.Context(), "/tasks/"+taskGID+"/removeProject", api.RequestBody{
		Data: map[string]string{"project": resolvedProjectGID},
	})
	if err != nil {
		return err
	}

	output := membershipOutput{
		Action:  "removed",
		DryRun:  false,
		Task:    gidRef{GID: taskGID},
		Project: &gidRef{GID: resolvedProjectGID},
	}
	if runtimeOutputJSON(cmd) {
		encoder := json.NewEncoder(cmd.OutOrStdout())
		encoder.SetIndent("", "  ")
		return encoder.Encode(output)
	}
	_, err = fmt.Fprintln(cmd.OutOrStdout(), "Task removed from project")
	return err
}

func runTaskTagAdd(cmd *cobra.Command, args []string) error {
	taskGID := args[0]
	tagName, _ := cmd.Flags().GetString("tag")
	tagGID, _ := cmd.Flags().GetString("tag-gid")
	workspaceName, _ := cmd.Flags().GetString("workspace")
	workspaceGID, _ := cmd.Flags().GetString("workspace-gid")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	if err := validateGID(taskGID, "gid"); err != nil {
		return err
	}
	if tagName == "" && tagGID == "" {
		return fmt.Errorf("either --tag or --tag-gid is required")
	}
	if tagName != "" && tagGID != "" {
		return fmt.Errorf("use only one of --tag or --tag-gid")
	}
	if workspaceName != "" && workspaceGID != "" {
		return fmt.Errorf("use only one of --workspace or --workspace-gid")
	}

	// Load config for workspace default if needed
	if tagName != "" && workspaceName == "" && workspaceGID == "" {
		cfg, err := config.LoadConfig(runtimeConfigPath(cmd))
		if err == nil && cfg.Defaults.WorkspaceGID != "" {
			workspaceGID = cfg.Defaults.WorkspaceGID
		}
	}

	// Resolve workspace
	resolvedWorkspace := workspaceGID
	if workspaceName != "" {
		gid, err := resolveWorkspaceWithPrompt(cmd.Context(), runtimeClient(cmd), workspaceName, runtimeNoPrompt(cmd) || runtimeOutputJSON(cmd))
		if err != nil {
			return err
		}
		resolvedWorkspace = gid
	}

	// Resolve tag
	resolvedTagGID := tagGID
	if tagName != "" {
		if resolvedWorkspace == "" {
			return fmt.Errorf("--workspace or --workspace-gid is required for tag name resolution")
		}
		gid, err := resolveTagWithPrompt(cmd.Context(), runtimeClient(cmd), resolvedWorkspace, tagName, runtimeNoPrompt(cmd) || runtimeOutputJSON(cmd))
		if err != nil {
			return err
		}
		resolvedTagGID = gid
	}

	if dryRun {
		// Validate task exists
		if _, err := fetchTaskWithFields(cmd.Context(), taskGID, "gid"); err != nil {
			return err
		}
		output := membershipOutput{
			Action: "added",
			DryRun: true,
			Task:   gidRef{GID: taskGID},
			Tag:    &gidRef{GID: resolvedTagGID},
		}
		if runtimeOutputJSON(cmd) {
			encoder := json.NewEncoder(cmd.OutOrStdout())
			encoder.SetIndent("", "  ")
			return encoder.Encode(output)
		}
		_, err := fmt.Fprintf(cmd.OutOrStdout(), "[dry-run] Would add tag %s to task %s\n", resolvedTagGID, taskGID)
		return err
	}

	_, err := runtimeClient(cmd).Post(cmd.Context(), "/tasks/"+taskGID+"/addTag", api.RequestBody{
		Data: map[string]string{"tag": resolvedTagGID},
	})
	if err != nil {
		return err
	}

	output := membershipOutput{
		Action: "added",
		DryRun: false,
		Task:   gidRef{GID: taskGID},
		Tag:    &gidRef{GID: resolvedTagGID},
	}
	if runtimeOutputJSON(cmd) {
		encoder := json.NewEncoder(cmd.OutOrStdout())
		encoder.SetIndent("", "  ")
		return encoder.Encode(output)
	}
	_, err = fmt.Fprintln(cmd.OutOrStdout(), "Tag added to task")
	return err
}

func runTaskTagRemove(cmd *cobra.Command, args []string) error {
	taskGID := args[0]
	tagName, _ := cmd.Flags().GetString("tag")
	tagGID, _ := cmd.Flags().GetString("tag-gid")
	workspaceName, _ := cmd.Flags().GetString("workspace")
	workspaceGID, _ := cmd.Flags().GetString("workspace-gid")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	if err := validateGID(taskGID, "gid"); err != nil {
		return err
	}
	if tagName == "" && tagGID == "" {
		return fmt.Errorf("either --tag or --tag-gid is required")
	}
	if tagName != "" && tagGID != "" {
		return fmt.Errorf("use only one of --tag or --tag-gid")
	}
	if workspaceName != "" && workspaceGID != "" {
		return fmt.Errorf("use only one of --workspace or --workspace-gid")
	}

	// Load config for workspace default if needed
	if tagName != "" && workspaceName == "" && workspaceGID == "" {
		cfg, err := config.LoadConfig(runtimeConfigPath(cmd))
		if err == nil && cfg.Defaults.WorkspaceGID != "" {
			workspaceGID = cfg.Defaults.WorkspaceGID
		}
	}

	// Resolve workspace
	resolvedWorkspace := workspaceGID
	if workspaceName != "" {
		gid, err := resolveWorkspaceWithPrompt(cmd.Context(), runtimeClient(cmd), workspaceName, runtimeNoPrompt(cmd) || runtimeOutputJSON(cmd))
		if err != nil {
			return err
		}
		resolvedWorkspace = gid
	}

	// Resolve tag
	resolvedTagGID := tagGID
	if tagName != "" {
		if resolvedWorkspace == "" {
			return fmt.Errorf("--workspace or --workspace-gid is required for tag name resolution")
		}
		gid, err := resolveTagWithPrompt(cmd.Context(), runtimeClient(cmd), resolvedWorkspace, tagName, runtimeNoPrompt(cmd) || runtimeOutputJSON(cmd))
		if err != nil {
			return err
		}
		resolvedTagGID = gid
	}

	if dryRun {
		// Validate task exists
		if _, err := fetchTaskWithFields(cmd.Context(), taskGID, "gid"); err != nil {
			return err
		}
		output := membershipOutput{
			Action: "removed",
			DryRun: true,
			Task:   gidRef{GID: taskGID},
			Tag:    &gidRef{GID: resolvedTagGID},
		}
		if runtimeOutputJSON(cmd) {
			encoder := json.NewEncoder(cmd.OutOrStdout())
			encoder.SetIndent("", "  ")
			return encoder.Encode(output)
		}
		_, err := fmt.Fprintf(cmd.OutOrStdout(), "[dry-run] Would remove tag %s from task %s\n", resolvedTagGID, taskGID)
		return err
	}

	_, err := runtimeClient(cmd).Post(cmd.Context(), "/tasks/"+taskGID+"/removeTag", api.RequestBody{
		Data: map[string]string{"tag": resolvedTagGID},
	})
	if err != nil {
		return err
	}

	output := membershipOutput{
		Action: "removed",
		DryRun: false,
		Task:   gidRef{GID: taskGID},
		Tag:    &gidRef{GID: resolvedTagGID},
	}
	if runtimeOutputJSON(cmd) {
		encoder := json.NewEncoder(cmd.OutOrStdout())
		encoder.SetIndent("", "  ")
		return encoder.Encode(output)
	}
	_, err = fmt.Fprintln(cmd.OutOrStdout(), "Tag removed from task")
	return err
}

func runTaskFollowerAdd(cmd *cobra.Command, args []string) error {
	taskGID := args[0]
	users, _ := cmd.Flags().GetStringSlice("user")
	workspaceName, _ := cmd.Flags().GetString("workspace")
	workspaceGID, _ := cmd.Flags().GetString("workspace-gid")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	if err := validateGID(taskGID, "gid"); err != nil {
		return err
	}
	if len(users) == 0 {
		return fmt.Errorf("--user is required")
	}
	if workspaceName != "" && workspaceGID != "" {
		return fmt.Errorf("use only one of --workspace or --workspace-gid")
	}

	// Determine if we need workspace for email resolution
	needsWorkspace := false
	for _, user := range users {
		user = strings.TrimSpace(user)
		if !resolve.IsGID(user) && !strings.EqualFold(user, "me") && strings.Contains(user, "@") {
			needsWorkspace = true
			break
		}
	}

	// Load config for workspace default if needed
	if needsWorkspace && workspaceName == "" && workspaceGID == "" {
		inferred, err := inferWorkspaceFromTask(cmd.Context(), taskGID)
		if err != nil {
			return err
		}
		workspaceGID = inferred
	}
	if needsWorkspace && workspaceName == "" && workspaceGID == "" {
		cfg, err := config.LoadConfig(runtimeConfigPath(cmd))
		if err == nil && cfg.Defaults.WorkspaceGID != "" {
			workspaceGID = cfg.Defaults.WorkspaceGID
		}
	}

	// Resolve workspace
	resolvedWorkspace := workspaceGID
	if workspaceName != "" {
		gid, err := resolveWorkspaceWithPrompt(cmd.Context(), runtimeClient(cmd), workspaceName, runtimeNoPrompt(cmd) || runtimeOutputJSON(cmd))
		if err != nil {
			return err
		}
		resolvedWorkspace = gid
	}
	if needsWorkspace && resolvedWorkspace == "" {
		return fmt.Errorf("workspace gid is required to resolve user email; use a GID or set a default workspace")
	}

	// Resolve users
	resolvedUsers, err := resolveUserIdentifiers(cmd.Context(), resolvedWorkspace, users)
	if err != nil {
		return err
	}

	if dryRun {
		// Validate task exists
		if _, err := fetchTaskWithFields(cmd.Context(), taskGID, "gid"); err != nil {
			return err
		}
		output := membershipOutput{
			Action: "added",
			DryRun: true,
			Task:   gidRef{GID: taskGID},
			Users:  resolvedUsers,
		}
		if runtimeOutputJSON(cmd) {
			encoder := json.NewEncoder(cmd.OutOrStdout())
			encoder.SetIndent("", "  ")
			return encoder.Encode(output)
		}
		_, err := fmt.Fprintf(cmd.OutOrStdout(), "[dry-run] Would add %d followers to task %s\n", len(resolvedUsers), taskGID)
		return err
	}

	_, err = runtimeClient(cmd).Post(cmd.Context(), "/tasks/"+taskGID+"/addFollowers", api.RequestBody{
		Data: map[string][]string{"followers": resolvedUsers},
	})
	if err != nil {
		return err
	}

	output := membershipOutput{
		Action: "added",
		DryRun: false,
		Task:   gidRef{GID: taskGID},
		Users:  resolvedUsers,
	}
	if runtimeOutputJSON(cmd) {
		encoder := json.NewEncoder(cmd.OutOrStdout())
		encoder.SetIndent("", "  ")
		return encoder.Encode(output)
	}
	_, err = fmt.Fprintf(cmd.OutOrStdout(), "Added %d followers\n", len(resolvedUsers))
	return err
}

func runTaskFollowerRemove(cmd *cobra.Command, args []string) error {
	taskGID := args[0]
	users, _ := cmd.Flags().GetStringSlice("user")
	workspaceName, _ := cmd.Flags().GetString("workspace")
	workspaceGID, _ := cmd.Flags().GetString("workspace-gid")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	if err := validateGID(taskGID, "gid"); err != nil {
		return err
	}
	if len(users) == 0 {
		return fmt.Errorf("--user is required")
	}
	if workspaceName != "" && workspaceGID != "" {
		return fmt.Errorf("use only one of --workspace or --workspace-gid")
	}

	// Determine if we need workspace for email resolution
	needsWorkspace := false
	for _, user := range users {
		user = strings.TrimSpace(user)
		if !resolve.IsGID(user) && !strings.EqualFold(user, "me") && strings.Contains(user, "@") {
			needsWorkspace = true
			break
		}
	}

	// Load config for workspace default if needed
	if needsWorkspace && workspaceName == "" && workspaceGID == "" {
		inferred, err := inferWorkspaceFromTask(cmd.Context(), taskGID)
		if err != nil {
			return err
		}
		workspaceGID = inferred
	}
	if needsWorkspace && workspaceName == "" && workspaceGID == "" {
		cfg, err := config.LoadConfig(runtimeConfigPath(cmd))
		if err == nil && cfg.Defaults.WorkspaceGID != "" {
			workspaceGID = cfg.Defaults.WorkspaceGID
		}
	}

	// Resolve workspace
	resolvedWorkspace := workspaceGID
	if workspaceName != "" {
		gid, err := resolveWorkspaceWithPrompt(cmd.Context(), runtimeClient(cmd), workspaceName, runtimeNoPrompt(cmd) || runtimeOutputJSON(cmd))
		if err != nil {
			return err
		}
		resolvedWorkspace = gid
	}
	if needsWorkspace && resolvedWorkspace == "" {
		return fmt.Errorf("workspace gid is required to resolve user email; use a GID or set a default workspace")
	}

	// Resolve users
	resolvedUsers, err := resolveUserIdentifiers(cmd.Context(), resolvedWorkspace, users)
	if err != nil {
		return err
	}

	if dryRun {
		// Validate task exists
		if _, err := fetchTaskWithFields(cmd.Context(), taskGID, "gid"); err != nil {
			return err
		}
		output := membershipOutput{
			Action: "removed",
			DryRun: true,
			Task:   gidRef{GID: taskGID},
			Users:  resolvedUsers,
		}
		if runtimeOutputJSON(cmd) {
			encoder := json.NewEncoder(cmd.OutOrStdout())
			encoder.SetIndent("", "  ")
			return encoder.Encode(output)
		}
		_, err := fmt.Fprintf(cmd.OutOrStdout(), "[dry-run] Would remove %d followers from task %s\n", len(resolvedUsers), taskGID)
		return err
	}

	_, err = runtimeClient(cmd).Post(cmd.Context(), "/tasks/"+taskGID+"/removeFollowers", api.RequestBody{
		Data: map[string][]string{"followers": resolvedUsers},
	})
	if err != nil {
		return err
	}

	output := membershipOutput{
		Action: "removed",
		DryRun: false,
		Task:   gidRef{GID: taskGID},
		Users:  resolvedUsers,
	}
	if runtimeOutputJSON(cmd) {
		encoder := json.NewEncoder(cmd.OutOrStdout())
		encoder.SetIndent("", "  ")
		return encoder.Encode(output)
	}
	_, err = fmt.Fprintf(cmd.OutOrStdout(), "Removed %d followers\n", len(resolvedUsers))
	return err
}

func inferWorkspaceFromTask(ctx context.Context, taskGID string) (string, error) {
	task, err := fetchTaskWithFields(ctx, taskGID, "gid,memberships.project.gid")
	if err != nil {
		return "", err
	}
	for _, membership := range task.Memberships {
		if membership.Project == nil || membership.Project.GID == "" {
			continue
		}
		project, err := fetchProjectForWorkspace(ctx, membership.Project.GID)
		if err != nil {
			return "", err
		}
		return project.WorkspaceGID, nil
	}
	return "", nil
}
