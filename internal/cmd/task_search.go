// Package cmd implements task search command.
package cmd

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/spf13/cobra"
	"github.com/vincentsch/asana-cli/internal/api"
	"github.com/vincentsch/asana-cli/internal/config"
	"github.com/vincentsch/asana-cli/internal/resolve"
)

var taskSearchCmd = &cobra.Command{
	Use:   "search",
	Short: "Search tasks in a workspace (premium feature)",
	Long: `Search for tasks across a workspace using various filters.

This is a premium Asana feature. Search by text, assignee, project, completion
status, or due date. Results are limited to 100 by the API.

See also: task list, task view`,
	Example: `  # Search by text
  asana task search -w "My Workspace" --text "login bug"

  # Find your incomplete tasks
  asana task search -w "My Workspace" --assignee me --completed false

  # Find tasks due before a date
  asana task search -w "My Workspace" --due-before 2024-12-31

  # Find tasks in a specific project
  asana task search -w "My Workspace" -p "Sprint 1"

  # Combine filters
  asana task search -w "My Workspace" --assignee me --text "urgent" --limit 10

  # Workflow: Find and process overdue tasks
  asana task search -w "My Workspace" --assignee me --due-before 2024-01-01
  asana task update <gid> --due 2024-02-01`,
	RunE: runTaskSearch,
}

func init() {
	taskSearchCmd.Flags().StringP("workspace", "w", "", "Workspace name (required)")
	taskSearchCmd.Flags().String("workspace-gid", "", "Workspace GID")
	taskSearchCmd.Flags().String("text", "", "Search text")
	taskSearchCmd.Flags().String("assignee", "", "Filter by assignee (me, GID, or email)")
	taskSearchCmd.Flags().StringP("project", "p", "", "Filter by project name")
	taskSearchCmd.Flags().String("project-gid", "", "Filter by project GID")
	taskSearchCmd.Flags().String("completed", "", "Filter by completion status (true or false)")
	taskSearchCmd.Flags().String("due-before", "", "Tasks due before date (YYYY-MM-DD)")
	taskSearchCmd.Flags().String("due-after", "", "Tasks due after date (YYYY-MM-DD)")
	taskSearchCmd.Flags().Int("limit", 20, "Maximum results (capped at 100 by API)")

	taskCmd.AddCommand(taskSearchCmd)
}

func runTaskSearch(cmd *cobra.Command, args []string) error {
	workspaceName, _ := cmd.Flags().GetString("workspace")
	workspaceGID, _ := cmd.Flags().GetString("workspace-gid")
	text, _ := cmd.Flags().GetString("text")
	assignee, _ := cmd.Flags().GetString("assignee")
	projectName, _ := cmd.Flags().GetString("project")
	projectGID, _ := cmd.Flags().GetString("project-gid")
	completed, _ := cmd.Flags().GetString("completed")
	dueBefore, _ := cmd.Flags().GetString("due-before")
	dueAfter, _ := cmd.Flags().GetString("due-after")
	limit, _ := cmd.Flags().GetInt("limit")

	if workspaceName != "" && workspaceGID != "" {
		return fmt.Errorf("use only one of --workspace or --workspace-gid")
	}
	if projectName != "" && projectGID != "" {
		return fmt.Errorf("use only one of --project or --project-gid")
	}
	if limit < 1 || limit > 100 {
		return fmt.Errorf("--limit must be between 1 and 100")
	}

	// Load config for workspace default if needed
	if workspaceName == "" && workspaceGID == "" {
		cfg, err := config.LoadConfig(runtimeConfigPath(cmd))
		if err == nil && cfg.Defaults.WorkspaceGID != "" {
			workspaceGID = cfg.Defaults.WorkspaceGID
		}
	}

	if workspaceName == "" && workspaceGID == "" {
		return fmt.Errorf("--workspace or --workspace-gid is required")
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

	// Build query parameters
	query := url.Values{}
	query.Set("opt_fields", taskListFields)
	query.Set("limit", fmt.Sprintf("%d", limit))

	if text != "" {
		query.Set("text", text)
	}

	if assignee != "" {
		var resolvedAssignee string
		switch {
		case strings.EqualFold(assignee, "me"):
			user, err := fetchCurrentUser(cmd.Context())
			if err != nil {
				return err
			}
			resolvedAssignee = user.GID
		case resolve.IsGID(assignee):
			resolvedAssignee = assignee
		default:
			gid, err := resolve.UserInWorkspace(cmd.Context(), runtimeClient(cmd), resolvedWorkspace, assignee)
			if err != nil {
				return err
			}
			resolvedAssignee = gid
		}
		query.Set("assignee.any", resolvedAssignee)
	}

	if projectName != "" || projectGID != "" {
		resolvedProject := projectGID
		if projectName != "" {
			gid, err := resolveProjectWithPrompt(cmd.Context(), runtimeClient(cmd), resolvedWorkspace, projectName, runtimeNoPrompt(cmd) || runtimeOutputJSON(cmd))
			if err != nil {
				return err
			}
			resolvedProject = gid
		}
		query.Set("projects.any", resolvedProject)
	}

	if completed != "" {
		switch strings.ToLower(completed) {
		case "true":
			query.Set("completed", "true")
		case "false":
			query.Set("completed", "false")
		default:
			return fmt.Errorf("--completed must be true or false")
		}
	}

	if dueBefore != "" {
		// Validate date format
		if _, err := parseDateOnly(dueBefore); err != nil {
			return fmt.Errorf("--due-before: %w", err)
		}
		query.Set("due_on.before", dueBefore)
	}

	if dueAfter != "" {
		// Validate date format
		if _, err := parseDateOnly(dueAfter); err != nil {
			return fmt.Errorf("--due-after: %w", err)
		}
		query.Set("due_on.after", dueAfter)
	}

	// Call search API - note: this does NOT use pagination, results are unstable
	payload, err := runtimeClient(cmd).Get(cmd.Context(), "/workspaces/"+resolvedWorkspace+"/tasks/search", query)
	if err != nil {
		// Check for premium feature error
		var apiErr *api.APIError
		if ok := isAPIError(err, &apiErr); ok && apiErr.StatusCode == 402 {
			return fmt.Errorf("task search is a premium feature - requires Asana Premium or higher")
		}
		return err
	}

	var response api.Response[[]api.Task]
	if err := json.Unmarshal(payload, &response); err != nil {
		return &api.ResponseError{Err: err}
	}

	// Convert to list items - preserve API ordering
	output := make([]taskListItem, 0, len(response.Data))
	for _, task := range response.Data {
		output = append(output, newTaskListItem(task, nil))
	}

	return writeTaskListOutput(cmd, output)
}

func parseDateOnly(input string) (string, error) {
	// Simple validation - expect YYYY-MM-DD
	if len(input) != 10 {
		return "", fmt.Errorf("invalid date format %q: use YYYY-MM-DD", input)
	}
	if input[4] != '-' || input[7] != '-' {
		return "", fmt.Errorf("invalid date format %q: use YYYY-MM-DD", input)
	}
	return input, nil
}

func isAPIError(err error, target **api.APIError) bool {
	if apiErr, ok := err.(*api.APIError); ok {
		*target = apiErr
		return true
	}
	return false
}
