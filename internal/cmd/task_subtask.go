// Package cmd implements task subtask commands.
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

var taskSubtaskCmd = &cobra.Command{
	Use:   "subtask",
	Short: "Manage subtasks",
	Long: `Manage subtasks of a parent task.

Subtasks are nested tasks that help break down larger work items.
Use subcommands to list or create subtasks.

See also: task view, task parent`,
}

var taskSubtaskListCmd = &cobra.Command{
	Use:   "list <parent-gid>",
	Short: "List subtasks of a task",
	Long: `List all subtasks of a parent task.

See also: task view, task subtask create`,
	Example: `  # List subtasks
  asana task subtask list 1234567890123456

  # Limit output
  asana task subtask list 1234567890123456 --limit 5

  # Output as JSON
  asana task subtask list 1234567890123456 --json`,
	Args: cobra.ExactArgs(1),
	RunE: runTaskSubtaskList,
}

var taskSubtaskCreateCmd = &cobra.Command{
	Use:   "create <parent-gid> <title>",
	Short: "Create a subtask",
	Long: `Create a new subtask under a parent task.

The subtask inherits the parent's project membership.

See also: task subtask list, task create`,
	Example: `  # Create a subtask
  asana task subtask create 1234567890123456 "Review code"

  # Create with assignee and due date
  asana task subtask create 1234567890123456 "Write tests" --assignee me --due 2024-12-31

  # Preview without creating
  asana task subtask create 1234567890123456 "New subtask" --dry-run

  # Workflow: Break down a task into subtasks
  asana task subtask create 1234567890123456 "Design" --assignee me
  asana task subtask create 1234567890123456 "Implement" --assignee me
  asana task subtask create 1234567890123456 "Test" --assignee me`,
	Args: cobra.ExactArgs(2),
	RunE: runTaskSubtaskCreate,
}

func init() {
	taskSubtaskListCmd.Flags().Int("limit", 0, "Limit number of subtasks in output")

	taskSubtaskCreateCmd.Flags().String("assignee", "", "Assignee (me, GID, or email)")
	taskSubtaskCreateCmd.Flags().String("due", "", "Due date (YYYY-MM-DD or RFC3339 with timezone)")

	taskSubtaskCmd.AddCommand(taskSubtaskListCmd)
	taskSubtaskCmd.AddCommand(taskSubtaskCreateCmd)
	taskCmd.AddCommand(taskSubtaskCmd)
}

func runTaskSubtaskList(cmd *cobra.Command, args []string) error {
	parentGID := args[0]
	limit, _ := cmd.Flags().GetInt("limit")

	if err := validateGID(parentGID, "parent gid"); err != nil {
		return err
	}
	if cmd.Flags().Changed("limit") && limit < 1 {
		return fmt.Errorf("--limit must be >= 1")
	}

	query := url.Values{}
	query.Set("opt_fields", taskListFields)
	subtasks, err := api.Paginate[api.Task](cmd.Context(), runtimeClient(cmd), "/tasks/"+parentGID+"/subtasks", query)
	if err != nil {
		return err
	}

	output := make([]taskListItem, 0, len(subtasks))
	for _, task := range subtasks {
		output = append(output, newTaskListItem(task, nil))
	}

	if limit > 0 && len(output) > limit {
		output = output[:limit]
	}

	return writeTaskListOutput(cmd, output)
}

func runTaskSubtaskCreate(cmd *cobra.Command, args []string) error {
	parentGID := args[0]
	title := args[1]
	assigneeValue, _ := cmd.Flags().GetString("assignee")
	dueValue, _ := cmd.Flags().GetString("due")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	if err := validateGID(parentGID, "parent gid"); err != nil {
		return err
	}

	// For email-based assignee resolution, we need the workspace
	var workspaceGID string
	if assigneeValue != "" && !resolve.IsGID(assigneeValue) && !strings.EqualFold(assigneeValue, "me") {
		parent, err := fetchTaskWithFields(cmd.Context(), parentGID, "gid,memberships.project.gid")
		if err != nil {
			return err
		}
		if len(parent.Memberships) > 0 && parent.Memberships[0].Project != nil {
			project, err := fetchProjectForWorkspace(cmd.Context(), parent.Memberships[0].Project.GID)
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
			return fmt.Errorf("workspace gid is required to resolve assignee email %q; use a GID or set a default workspace", assigneeValue)
		}
	}

	var assigneeGID string
	var assigneeRef *userRef
	if assigneeValue != "" {
		switch {
		case strings.EqualFold(assigneeValue, "me"):
			user, err := fetchCurrentUser(cmd.Context())
			if err != nil {
				return err
			}
			assigneeGID = user.GID
			assigneeRef = userRefFromUser(user)
		case resolve.IsGID(assigneeValue):
			assigneeGID = assigneeValue
			assigneeRef = &userRef{GID: assigneeValue, Name: nil}
		default:
			resolvedGID, err := resolve.UserInWorkspace(cmd.Context(), runtimeClient(cmd), workspaceGID, assigneeValue)
			if err != nil {
				return err
			}
			assigneeGID = resolvedGID
			assigneeRef = &userRef{GID: resolvedGID, Name: nil}
		}
	}

	var dueOn string
	var dueAt string
	if dueValue != "" {
		parsedOn, parsedAt, err := parseDueDate(dueValue)
		if err != nil {
			return err
		}
		dueOn = parsedOn
		dueAt = parsedAt
	}

	if dryRun {
		// Validate parent exists
		if _, err := fetchTaskWithFields(cmd.Context(), parentGID, "gid"); err != nil {
			return err
		}
		output := writeOutput{
			Action: "created",
			DryRun: true,
			Task: &taskWriteResult{
				GID:       "",
				Name:      title,
				Completed: false,
				Assignee:  assigneeRef,
				DueOn:     optionalString(dueOn),
				DueAt:     optionalString(dueAt),
			},
		}
		if runtimeOutputJSON(cmd) {
			return writeWriteJSON(cmd, output)
		}
		_, err := fmt.Fprintf(cmd.OutOrStdout(), "[dry-run] Would create subtask %q under task %s\n", title, parentGID)
		return err
	}

	data := map[string]any{
		"name": title,
	}
	if assigneeGID != "" {
		data["assignee"] = assigneeGID
	}
	if dueOn != "" {
		data["due_on"] = dueOn
	}
	if dueAt != "" {
		data["due_at"] = dueAt
	}

	payload, err := runtimeClient(cmd).Post(cmd.Context(), "/tasks/"+parentGID+"/subtasks", api.RequestBody{
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

	output := writeOutput{Action: "created", DryRun: false, Task: taskWriteResultFromTask(&response.Data)}
	if runtimeOutputJSON(cmd) {
		return writeWriteJSON(cmd, output)
	}
	_, err = fmt.Fprintf(cmd.OutOrStdout(), "Created subtask %s\n", response.Data.GID)
	return err
}
