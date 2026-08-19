// Package cmd implements task update and delete commands.
package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/spf13/cobra"
	"github.com/vincentsch/asana-cli/internal/api"
	"github.com/vincentsch/asana-cli/internal/config"
	"github.com/vincentsch/asana-cli/internal/resolve"
)

var taskUpdateCmd = &cobra.Command{
	Use:   "update <gid>",
	Short: "Update task fields",
	Long: `Update one or more fields of a task.

Supports updating name, notes, due date, and assignee. Use --clear-* flags
to remove values. At least one field must be specified.

See also: task view, task move, task done`,
	Example: `  # Rename a task
  asana task update 1234567890123456 --name "Updated title"

  # Set due date and assignee
  asana task update 1234567890123456 --due 2024-12-31 --assignee me

  # Clear the due date
  asana task update 1234567890123456 --clear-due

  # Update notes
  asana task update 1234567890123456 --notes "New description"

  # Preview changes
  asana task update 1234567890123456 --name "New name" --dry-run

  # Workflow: Reassign and reschedule
  asana task update 1234567890123456 --assignee me --due 2024-12-31
  asana task comment 1234567890123456 "Taking over this task"`,
	Args: cobra.ExactArgs(1),
	RunE: runTaskUpdate,
}

var taskDeleteCmd = &cobra.Command{
	Use:   "delete <gid>",
	Short: "Delete a task",
	Long: `Permanently delete a task.

This action cannot be undone. Use --dry-run to preview. Consider using
task done instead if you want to preserve the task history.

See also: task done, task view`,
	Example: `  # Delete a task
  asana task delete 1234567890123456 --confirm

  # Preview deletion
  asana task delete 1234567890123456 --dry-run`,
	Args: cobra.ExactArgs(1),
	RunE: runTaskDelete,
}

func init() {
	taskUpdateCmd.Flags().String("name", "", "New task name")
	taskUpdateCmd.Flags().String("notes", "", "New task notes")
	taskUpdateCmd.Flags().String("due", "", "Due date (YYYY-MM-DD or RFC3339 with timezone)")
	taskUpdateCmd.Flags().String("assignee", "", "Assignee (me, GID, or email)")
	taskUpdateCmd.Flags().Bool("clear-due", false, "Clear due date")
	taskUpdateCmd.Flags().Bool("clear-assignee", false, "Clear assignee")
	taskUpdateCmd.Flags().Bool("clear-notes", false, "Clear notes")

	taskCmd.AddCommand(taskUpdateCmd)
	taskCmd.AddCommand(taskDeleteCmd)
}

func runTaskUpdate(cmd *cobra.Command, args []string) error {
	taskGID := args[0]
	name, _ := cmd.Flags().GetString("name")
	notes, _ := cmd.Flags().GetString("notes")
	due, _ := cmd.Flags().GetString("due")
	assignee, _ := cmd.Flags().GetString("assignee")
	clearDue, _ := cmd.Flags().GetBool("clear-due")
	clearAssignee, _ := cmd.Flags().GetBool("clear-assignee")
	clearNotes, _ := cmd.Flags().GetBool("clear-notes")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	if err := validateGID(taskGID, "gid"); err != nil {
		return err
	}

	// Check mutual exclusion
	if due != "" && clearDue {
		return fmt.Errorf("cannot use both --due and --clear-due")
	}
	if assignee != "" && clearAssignee {
		return fmt.Errorf("cannot use both --assignee and --clear-assignee")
	}
	if notes != "" && clearNotes {
		return fmt.Errorf("cannot use both --notes and --clear-notes")
	}

	// Check at least one field
	hasUpdate := name != "" || notes != "" || due != "" || assignee != "" || clearDue || clearAssignee || clearNotes
	if !hasUpdate {
		return fmt.Errorf("no fields to update (use --name, --notes, --due, --assignee, --clear-due, --clear-assignee, or --clear-notes)")
	}

	// For email-based assignee resolution, we need the workspace from the task
	var workspaceGID string
	if assignee != "" && !resolve.IsGID(assignee) && !strings.EqualFold(assignee, "me") {
		task, err := fetchTaskWithFields(cmd.Context(), taskGID, taskWriteFields)
		if err != nil {
			return err
		}
		if len(task.Memberships) > 0 && task.Memberships[0].Project != nil {
			project, err := fetchProjectForWorkspace(cmd.Context(), task.Memberships[0].Project.GID)
			if err != nil {
				return err
			}
			workspaceGID = project.WorkspaceGID
		}
		if workspaceGID == "" {
			cfg, err := config.LoadConfig(runtimeConfigPath(cmd))
			if err == nil {
				workspaceGID = cfg.Defaults.WorkspaceGID
			}
		}
		if workspaceGID == "" {
			return fmt.Errorf("workspace gid is required to resolve assignee email %q; use a GID or set a default workspace", assignee)
		}
	}

	// Build update payload
	data := make(map[string]any)

	if name != "" {
		data["name"] = name
	}
	if notes != "" {
		data["notes"] = notes
	}
	if clearNotes {
		data["notes"] = nil
	}

	if due != "" {
		parsedOn, parsedAt, err := parseDueDate(due)
		if err != nil {
			return err
		}
		if parsedOn != "" {
			data["due_on"] = parsedOn
			data["due_at"] = nil
		} else {
			data["due_at"] = parsedAt
		}
	}
	if clearDue {
		data["due_on"] = nil
		data["due_at"] = nil
	}

	var assigneeRef *userRef
	if assignee != "" {
		resolvedAssignee, err := resolve.UserInWorkspace(cmd.Context(), runtimeClient(cmd), workspaceGID, assignee)
		if err != nil {
			return err
		}
		data["assignee"] = resolvedAssignee
		if strings.EqualFold(assignee, "me") {
			user, err := fetchCurrentUser(cmd.Context())
			if err != nil {
				return err
			}
			assigneeRef = userRefFromUser(user)
		} else {
			assigneeRef = &userRef{GID: resolvedAssignee}
		}
	}
	if clearAssignee {
		data["assignee"] = nil
	}

	if dryRun {
		task, err := fetchTaskWithFields(cmd.Context(), taskGID, taskWriteFields)
		if err != nil {
			return err
		}
		result := taskWriteResultFromTask(task)

		// Apply changes to result for preview
		if name != "" {
			result.Name = name
		}
		if due != "" {
			parsedOn, parsedAt, _ := parseDueDate(due)
			result.DueOn = optionalString(parsedOn)
			result.DueAt = optionalString(parsedAt)
		}
		if clearDue {
			result.DueOn = nil
			result.DueAt = nil
		}
		if assignee != "" {
			result.Assignee = assigneeRef
		}
		if clearAssignee {
			result.Assignee = nil
		}

		output := writeOutput{Action: "updated", DryRun: true, Task: result}
		if runtimeOutputJSON(cmd) {
			return writeWriteJSON(cmd, output)
		}
		taskLabel := task.Name
		if taskLabel == "" {
			taskLabel = task.GID
		}
		_, err = fmt.Fprintf(cmd.OutOrStdout(), "[dry-run] Would update task %q\n", taskLabel)
		return err
	}

	payload, err := runtimeClient(cmd).Put(cmd.Context(), "/tasks/"+taskGID, api.RequestBody{
		Data: data,
		Options: &api.RequestOptions{
			Fields: splitFields(taskWriteFields),
		},
	})
	if err != nil {
		return err
	}

	var response api.Response[api.Task]
	if err := json.Unmarshal(payload, &response); err != nil {
		return &api.ResponseError{Err: err}
	}

	output := writeOutput{Action: "updated", DryRun: false, Task: taskWriteResultFromTask(&response.Data)}
	if runtimeOutputJSON(cmd) {
		return writeWriteJSON(cmd, output)
	}
	_, err = fmt.Fprintln(cmd.OutOrStdout(), "Task updated")
	return err
}

func runTaskDelete(cmd *cobra.Command, args []string) error {
	taskGID := args[0]
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	if err := validateGID(taskGID, "gid"); err != nil {
		return err
	}

	if dryRun {
		task, err := fetchTaskWithFields(cmd.Context(), taskGID, "gid,name")
		if err != nil {
			return err
		}
		output := writeOutput{
			Action: "deleted",
			DryRun: true,
			Task:   &taskWriteResult{GID: task.GID, Name: task.Name},
		}
		if runtimeOutputJSON(cmd) {
			return writeWriteJSON(cmd, output)
		}
		taskLabel := task.Name
		if taskLabel == "" {
			taskLabel = task.GID
		}
		_, err = fmt.Fprintf(cmd.OutOrStdout(), "[dry-run] Would delete task %q\n", taskLabel)
		return err
	}

	if err := runtimeClient(cmd).Delete(cmd.Context(), "/tasks/"+taskGID); err != nil {
		return err
	}

	output := writeOutput{
		Action: "deleted",
		DryRun: false,
		Task:   &taskWriteResult{GID: taskGID},
	}
	if runtimeOutputJSON(cmd) {
		return writeWriteJSON(cmd, output)
	}
	_, err := fmt.Fprintln(cmd.OutOrStdout(), "Task deleted")
	return err
}

// ProjectWithWorkspace extends Project with workspace info for resolution
type projectWithWorkspace struct {
	GID          string `json:"gid"`
	Name         string `json:"name"`
	WorkspaceGID string `json:"-"`
}

func fetchProjectForWorkspace(ctx context.Context, gid string) (*projectWithWorkspace, error) {
	query := url.Values{}
	query.Set("opt_fields", "gid,name,workspace.gid")
	payload, err := runtimeFromContext(ctx).client.Get(ctx, "/projects/"+gid, query)
	if err != nil {
		return nil, err
	}

	type workspaceRef struct {
		GID string `json:"gid"`
	}
	type response struct {
		GID       string       `json:"gid"`
		Name      string       `json:"name"`
		Workspace workspaceRef `json:"workspace"`
	}
	var resp api.Response[response]
	if err := json.Unmarshal(payload, &resp); err != nil {
		return nil, &api.ResponseError{Err: err}
	}

	return &projectWithWorkspace{
		GID:          resp.Data.GID,
		Name:         resp.Data.Name,
		WorkspaceGID: resp.Data.Workspace.GID,
	}, nil
}
