// Package cmd implements task-related commands.
package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"github.com/vincentsch/asana-cli/internal/api"
	"github.com/vincentsch/asana-cli/internal/config"
	"github.com/vincentsch/asana-cli/internal/interactive"
	"github.com/vincentsch/asana-cli/internal/resolve"
)

const (
	taskListFields       = "gid,name,completed,assignee.gid,assignee.name,due_on"
	taskViewFields       = "gid,name,completed,assignee.gid,assignee.name,due_on,due_at,notes,memberships.project.gid,memberships.project.name,memberships.section.gid,memberships.section.name"
	taskWriteFields      = "gid,name,completed,assignee.gid,assignee.name,due_on,due_at,memberships.project.gid,memberships.project.name,memberships.section.gid,memberships.section.name"
	storyFields          = "gid,resource_subtype,text,created_at,created_by.gid,created_by.name"
	storyWriteFields     = "gid,text,created_at,created_by.gid,created_by.name"
	userFields           = "gid,name"
	sectionFields        = "gid,name"
	sectionProjectFields = "gid,name,project.gid,project.name"
	projectFields        = "gid,name"
)

var taskCmd = &cobra.Command{
	Use:   "task",
	Short: "Manage tasks",
	Long: `Manage Asana tasks.

Tasks are the basic unit of work in Asana. They can be assigned, have due dates,
belong to projects, and contain subtasks. Use subcommands to list, view, create,
update, move, and organize tasks.

See also: project, section, user`,
}

var taskListCmd = &cobra.Command{
	Use:   "list",
	Short: "List tasks in a project or section",
	Long: `List tasks in a project or section.

By default, lists all tasks in the project organized by section. Use --section
to filter to a specific section. Use --assignee to filter by task owner.

Tasks are shown in their section order. Output includes task GID, name, and
completion status. Use --json for detailed output including assignee and due date.

See also: task view, task create, section list`,
	Example: `  # List all tasks in a project
  asana task list -p "My Project"

  # List tasks in a specific section
  asana task list -p "My Project" -s "In Progress"

  # List only your tasks
  asana task list -p "My Project" --assignee me

  # Output as JSON for scripting
  asana task list -p "My Project" --json

  # Limit output to first 10 tasks
  asana task list -p "My Project" --limit 10

  # Workflow: Find tasks, view details, then update
  asana task list -p "Sprint" --assignee me
  asana task view 1234567890123456
  asana task move 1234567890123456 -s "Done"`,
	RunE: runTaskList,
}

var taskViewCmd = &cobra.Command{
	Use:   "view <gid>",
	Short: "View task details",
	Long: `View detailed information about a task.

Shows task name, assignee, due date, completion status, notes, project
memberships, and recent comments. Use --comments-limit to control how
many comments are displayed.

See also: task list, task update, task comment`,
	Example: `  # View a task by GID
  asana task view 1234567890123456

  # View task with more comments
  asana task view 1234567890123456 --comments-limit 20

  # Output as JSON
  asana task view 1234567890123456 --json`,
	Args: cobra.ExactArgs(1),
	RunE: runTaskView,
}

var taskMoveCmd = &cobra.Command{
	Use:   "move <gid>",
	Short: "Move task to a section",
	Long: `Move a task to a different section within a project.

This is useful for updating task status in kanban-style boards. The task
must already belong to the project containing the target section.
Use --dry-run to preview the move without making changes.

See also: task done, task update, section list`,
	Example: `  # Move task to "In Progress" section
  asana task move 1234567890123456 -s "In Progress"

  # Move task using section GID
  asana task move 1234567890123456 --section-gid 9876543210987654

  # Preview the move first
  asana task move 1234567890123456 -s "Done" --dry-run

  # Workflow: Update status on kanban board
  asana section list -p "Sprint"           # See available sections
  asana task move 1234567890123456 -s "Review"`,
	Args: cobra.ExactArgs(1),
	RunE: runTaskMove,
}

var taskCommentCmd = &cobra.Command{
	Use:   "comment <task-gid> <message>",
	Short: "Manage task comments",
	Long: `Add a comment to a task or manage existing comments.

When called with a task GID and message, adds a new comment. Use "-" as the
message to read from stdin (useful for multi-line comments or piping).

Subcommands are available for updating and deleting comments.

See also: task view, task update`,
	Example: `  # Add a comment to a task
  asana task comment 1234567890123456 "Looking into this now"

  # Add a multi-line comment from stdin
  printf 'Line 1\nLine 2\n' | asana task comment 1234567890123456 -

  # View comments on a task
  asana task view 1234567890123456

  # Workflow: Review task, add comment, mark done
  asana task view 1234567890123456
  asana task comment 1234567890123456 "Fixed in commit abc123"
  asana task done 1234567890123456`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return cmd.Help()
		}
		if len(args) != 2 {
			return fmt.Errorf("comment requires <task-gid> <message>")
		}
		return runTaskComment(cmd, args)
	},
}

var taskDoneCmd = &cobra.Command{
	Use:   "done <gid>",
	Short: "Mark task as complete",
	Long: `Mark a task as complete.

This sets the task's completed field to true. Use 'task reopen' to undo.

See also: task reopen, task move, task update`,
	Example: `  # Complete a task
  asana task done 1234567890123456

  # Preview without changing
  asana task done 1234567890123456 --dry-run

  # Workflow: Finish work, comment, and close
  asana task comment 1234567890123456 "Completed - deployed to staging"
  asana task done 1234567890123456`,
	Args: cobra.ExactArgs(1),
	RunE: runTaskDone,
}

var taskReopenCmd = &cobra.Command{
	Use:   "reopen <gid>",
	Short: "Mark task as incomplete",
	Long: `Mark a completed task as incomplete.

This sets the task's completed field back to false. Use this to undo
an accidental completion or to reactivate work that needs more attention.

See also: task done, task move, task update`,
	Example: `  # Reopen a completed task
  asana task reopen 1234567890123456

  # Preview without changing
  asana task reopen 1234567890123456 --dry-run

  # Workflow: Reopen and reassign to a section
  asana task reopen 1234567890123456
  asana task move 1234567890123456 -s "In Progress"`,
	Args: cobra.ExactArgs(1),
	RunE: runTaskReopen,
}

var taskCreateCmd = &cobra.Command{
	Use:   "create <title>",
	Short: "Create a new task",
	Long: `Create a new task in a project.

The task is placed in the first section of the project by default. Use --section
to place it in a specific section. Optionally assign the task and set a due date.

Due dates can be specified as YYYY-MM-DD for date-only or RFC3339 format
(e.g., 2024-12-31T17:00:00Z) for datetime with timezone.

See also: task list, task view, task update, section list`,
	Example: `  # Create a task in a project
  asana task create "Fix login bug" -p "My Project"

  # Create and assign to yourself with a due date
  asana task create "Review PR" -p "Sprint" --assignee me --due 2024-12-31

  # Create in a specific section
  asana task create "New feature" -p "My Project" -s "To Do"

  # Preview without creating
  asana task create "Test task" -p "My Project" --dry-run

  # Workflow: Create task and immediately view it
  asana task create "Urgent fix" -p "Sprint" --json | jq -r '.task.gid'
  asana task view <gid-from-above>`,
	Args: cobra.ExactArgs(1),
	RunE: runTaskCreate,
}

func init() {
	taskListCmd.Flags().StringP("project", "p", "", "Project name")
	taskListCmd.Flags().String("project-gid", "", "Project GID")
	taskListCmd.Flags().StringP("section", "s", "", "Section name")
	taskListCmd.Flags().String("section-gid", "", "Section GID")
	taskListCmd.Flags().StringP("workspace", "w", "", "Workspace name (scopes project lookup)")
	taskListCmd.Flags().String("workspace-gid", "", "Workspace GID (scopes project lookup)")
	taskListCmd.Flags().String("assignee", "any", "Filter by assignee (any, me, or GID)")
	taskListCmd.Flags().Int("limit", 0, "Limit number of tasks in output")

	taskViewCmd.Flags().Int("comments-limit", 5, "Limit number of comments to show")

	taskCmd.AddCommand(taskListCmd)
	taskCmd.AddCommand(taskViewCmd)
	taskMoveCmd.Flags().StringP("section", "s", "", "Target section name")
	taskMoveCmd.Flags().String("section-gid", "", "Target section GID")
	taskMoveCmd.Flags().StringP("project", "p", "", "Project name (required if task has multiple projects)")
	taskMoveCmd.Flags().String("project-gid", "", "Project GID")

	taskCreateCmd.Flags().StringP("project", "p", "", "Project name")
	taskCreateCmd.Flags().String("project-gid", "", "Project GID")
	taskCreateCmd.Flags().StringP("section", "s", "", "Section name")
	taskCreateCmd.Flags().String("section-gid", "", "Section GID")
	taskCreateCmd.Flags().StringP("workspace", "w", "", "Workspace name (scopes project lookup)")
	taskCreateCmd.Flags().String("workspace-gid", "", "Workspace GID (scopes project lookup)")
	taskCreateCmd.Flags().String("assignee", "", "Assignee (me or GID)")
	taskCreateCmd.Flags().String("due", "", "Due date (YYYY-MM-DD or RFC3339 with timezone)")

	taskCmd.AddCommand(taskMoveCmd)
	taskCmd.AddCommand(taskCommentCmd)
	taskCmd.AddCommand(taskDoneCmd)
	taskCmd.AddCommand(taskReopenCmd)
	taskCmd.AddCommand(taskCreateCmd)
}

type taskListItem struct {
	GID       string       `json:"gid"`
	Name      string       `json:"name"`
	Completed bool         `json:"completed"`
	Assignee  *api.User    `json:"assignee"`
	DueOn     *string      `json:"due_on"`
	Section   *api.Section `json:"section"`
}

type taskViewOutput struct {
	GID         string           `json:"gid"`
	Name        string           `json:"name"`
	Completed   bool             `json:"completed"`
	Assignee    *api.User        `json:"assignee"`
	DueOn       *string          `json:"due_on"`
	DueAt       *string          `json:"due_at"`
	Notes       string           `json:"notes"`
	Memberships []api.Membership `json:"memberships"`
	Comments    []taskComment    `json:"comments"`
}

type taskComment struct {
	GID       string    `json:"gid"`
	CreatedAt string    `json:"created_at"`
	CreatedBy *api.User `json:"created_by"`
	Text      string    `json:"text"`
}

type writeOutput struct {
	Action string           `json:"action"`
	DryRun bool             `json:"dry_run"`
	Task   *taskWriteResult `json:"task,omitempty"`
	Story  *storyResult     `json:"story,omitempty"`
}

type userRef struct {
	GID  string  `json:"gid"`
	Name *string `json:"name"`
}

type projectRef struct {
	GID  string  `json:"gid"`
	Name *string `json:"name"`
}

type sectionRef struct {
	GID  string  `json:"gid"`
	Name *string `json:"name"`
}

type membershipRef struct {
	Project projectRef `json:"project"`
	Section sectionRef `json:"section"`
}

type gidRef struct {
	GID string `json:"gid"`
}

type taskWriteResult struct {
	GID         string          `json:"gid"`
	Name        string          `json:"name"`
	Completed   bool            `json:"completed"`
	Assignee    *userRef        `json:"assignee"`
	DueOn       *string         `json:"due_on"`
	DueAt       *string         `json:"due_at"`
	Memberships []membershipRef `json:"memberships"`
}

type storyResult struct {
	GID       string   `json:"gid"`
	Text      string   `json:"text"`
	CreatedAt string   `json:"created_at"`
	CreatedBy *userRef `json:"created_by"`
}

type projectOption struct {
	GID  string
	Name string
}

func runTaskList(cmd *cobra.Command, args []string) error {
	projectName, _ := cmd.Flags().GetString("project")
	projectGID, _ := cmd.Flags().GetString("project-gid")
	sectionName, _ := cmd.Flags().GetString("section")
	sectionGID, _ := cmd.Flags().GetString("section-gid")
	workspaceName, _ := cmd.Flags().GetString("workspace")
	workspaceGID, _ := cmd.Flags().GetString("workspace-gid")
	assigneeFilter, _ := cmd.Flags().GetString("assignee")
	limit, _ := cmd.Flags().GetInt("limit")

	if projectName != "" && projectGID != "" {
		return fmt.Errorf("use only one of --project or --project-gid")
	}
	if sectionName != "" && sectionGID != "" {
		return fmt.Errorf("use only one of --section or --section-gid")
	}
	if workspaceName != "" && workspaceGID != "" {
		return fmt.Errorf("use only one of --workspace or --workspace-gid")
	}
	if cmd.Flags().Changed("limit") && limit < 1 {
		return fmt.Errorf("--limit must be >= 1")
	}

	needsProject := sectionName != "" || (sectionName == "" && sectionGID == "")
	needConfig := false
	if needsProject && projectName == "" && projectGID == "" {
		needConfig = true
	}
	if needsProject && projectName != "" && workspaceName == "" && workspaceGID == "" {
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

	assigneeFilter = strings.ToLower(strings.TrimSpace(assigneeFilter))
	if assigneeFilter == "" {
		assigneeFilter = "any"
	}

	var assigneeGID string
	switch {
	case assigneeFilter == "any":
	case assigneeFilter == "me":
		user, err := fetchCurrentUser(cmd.Context())
		if err != nil {
			return err
		}
		assigneeGID = user.GID
	case resolve.IsGID(assigneeFilter):
		assigneeGID = assigneeFilter
	default:
		return fmt.Errorf("invalid --assignee value %q (use any, me, or a GID)", assigneeFilter)
	}

	matchesAssignee := func(task api.Task) bool {
		if assigneeGID == "" {
			return true
		}
		if task.Assignee == nil {
			return false
		}
		return task.Assignee.GID == assigneeGID
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

	if resolvedProject == "" && needsProject && interactive.IsInteractive(runtimeNoPrompt(cmd) || runtimeOutputJSON(cmd)) {
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

	if resolvedProject == "" && sectionName != "" {
		return fmt.Errorf("--section requires --project or --project-gid")
	}
	if resolvedProject == "" && sectionName == "" && sectionGID == "" {
		return fmt.Errorf("either --project/--project-gid or --section/--section-gid is required")
	}

	if sectionGID != "" || sectionName != "" {
		return runTaskListBySection(cmd, sectionName, sectionGID, resolvedProject, matchesAssignee, limit)
	}

	return runTaskListByProject(cmd, resolvedProject, matchesAssignee, limit)
}

func runTaskListBySection(
	cmd *cobra.Command,
	sectionName string,
	sectionGID string,
	projectGID string,
	matchesAssignee func(api.Task) bool,
	limit int,
) error {
	resolvedSectionGID := sectionGID

	if sectionName != "" {
		if projectGID == "" {
			return fmt.Errorf("--section requires --project or --project-gid")
		}
		gid, err := resolveSectionWithPrompt(cmd.Context(), runtimeClient(cmd), projectGID, sectionName, runtimeNoPrompt(cmd) || runtimeOutputJSON(cmd))
		if err != nil {
			return err
		}
		resolvedSectionGID = gid
	}

	query := url.Values{}
	query.Set("section", resolvedSectionGID)
	query.Set("opt_fields", taskListFields)
	tasks, err := api.Paginate[api.Task](cmd.Context(), runtimeClient(cmd), "/tasks", query)
	if err != nil {
		return err
	}

	var section *api.Section
	if runtimeOutputJSON(cmd) {
		section, err = fetchSection(cmd.Context(), resolvedSectionGID)
		if err != nil {
			return err
		}
	}
	output := make([]taskListItem, 0, len(tasks))
	for _, task := range tasks {
		if !matchesAssignee(task) {
			continue
		}
		output = append(output, newTaskListItem(task, section))
	}

	if limit > 0 && len(output) > limit {
		output = output[:limit]
	}

	return writeTaskListOutput(cmd, output)
}

func runTaskListByProject(
	cmd *cobra.Command,
	projectGID string,
	matchesAssignee func(api.Task) bool,
	limit int,
) error {
	sectionsQuery := url.Values{}
	sectionsQuery.Set("opt_fields", "gid,name")
	sections, err := api.Paginate[api.Section](cmd.Context(), runtimeClient(cmd), "/projects/"+projectGID+"/sections", sectionsQuery)
	if err != nil {
		return err
	}

	output := make([]taskListItem, 0)
	seen := make(map[string]struct{})

	for _, section := range sections {
		tasksQuery := url.Values{}
		tasksQuery.Set("section", section.GID)
		tasksQuery.Set("opt_fields", taskListFields)
		tasks, err := api.Paginate[api.Task](cmd.Context(), runtimeClient(cmd), "/tasks", tasksQuery)
		if err != nil {
			return err
		}

		for _, task := range tasks {
			seen[task.GID] = struct{}{}
			if !matchesAssignee(task) {
				continue
			}
			sectionRef := section
			output = append(output, newTaskListItem(task, &sectionRef))
		}
	}

	projectTasksQuery := url.Values{}
	projectTasksQuery.Set("project", projectGID)
	projectTasksQuery.Set("opt_fields", taskListFields)
	projectTasks, err := api.Paginate[api.Task](cmd.Context(), runtimeClient(cmd), "/tasks", projectTasksQuery)
	if err != nil {
		return err
	}

	for _, task := range projectTasks {
		if _, ok := seen[task.GID]; ok {
			continue
		}
		if !matchesAssignee(task) {
			continue
		}
		output = append(output, newTaskListItem(task, nil))
	}

	if limit > 0 && len(output) > limit {
		output = output[:limit]
	}

	return writeTaskListOutput(cmd, output)
}

func writeTaskListOutput(cmd *cobra.Command, tasks []taskListItem) error {
	if runtimeOutputJSON(cmd) {
		encoder := json.NewEncoder(cmd.OutOrStdout())
		encoder.SetIndent("", "  ")
		return encoder.Encode(tasks)
	}

	writer := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(writer, "GID\tNAME\tCOMPLETED"); err != nil {
		return err
	}
	for _, task := range tasks {
		if _, err := fmt.Fprintf(writer, "%s\t%s\t%t\n", task.GID, task.Name, task.Completed); err != nil {
			return err
		}
	}
	return writer.Flush()
}

func runTaskView(cmd *cobra.Command, args []string) error {
	commentsLimit, _ := cmd.Flags().GetInt("comments-limit")
	if cmd.Flags().Changed("comments-limit") && commentsLimit < 1 {
		return fmt.Errorf("--comments-limit must be >= 1")
	}

	taskGID := args[0]
	task, err := fetchTask(cmd.Context(), taskGID)
	if err != nil {
		return err
	}

	stories, err := fetchTaskStories(cmd.Context(), taskGID)
	if err != nil {
		return err
	}

	comments := filterComments(stories)
	sort.Slice(comments, func(i, j int) bool {
		return compareCommentTime(comments[i].CreatedAt, comments[j].CreatedAt)
	})
	if commentsLimit > len(comments) {
		commentsLimit = len(comments)
	}
	comments = comments[:commentsLimit]

	if runtimeOutputJSON(cmd) {
		return writeTaskViewJSON(cmd, task, comments)
	}

	return writeTaskViewTable(cmd, task, comments)
}

func runTaskMove(cmd *cobra.Command, args []string) error {
	taskGID := args[0]
	sectionName, _ := cmd.Flags().GetString("section")
	sectionGID, _ := cmd.Flags().GetString("section-gid")
	projectName, _ := cmd.Flags().GetString("project")
	projectGID, _ := cmd.Flags().GetString("project-gid")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	if err := validateGID(taskGID, "gid"); err != nil {
		return err
	}
	if sectionName == "" && sectionGID == "" {
		return fmt.Errorf("either --section or --section-gid is required")
	}
	if sectionName != "" && sectionGID != "" {
		return fmt.Errorf("use only one of --section or --section-gid")
	}
	if projectName != "" && projectGID != "" {
		return fmt.Errorf("use only one of --project or --project-gid")
	}
	if sectionGID != "" {
		if err := validateGID(sectionGID, "section gid"); err != nil {
			return err
		}
	}
	if projectGID != "" {
		if err := validateGID(projectGID, "project gid"); err != nil {
			return err
		}
	}

	task, err := fetchTaskWithFields(cmd.Context(), taskGID, taskWriteFields)
	if err != nil {
		return err
	}

	projectOptions := projectOptionsFromMemberships(task.Memberships)
	projectSpecified := projectName != "" || projectGID != ""

	var targetProject projectOption
	if !projectSpecified {
		if len(projectOptions) > 1 {
			return fmt.Errorf(formatProjectDisambiguation(projectOptions))
		}
		if len(projectOptions) == 1 {
			targetProject = projectOptions[0]
		}
	} else {
		if projectGID == "" {
			matches := matchProjectOptionsByName(projectOptions, projectName)
			if len(matches) > 1 {
				return fmt.Errorf(formatProjectDisambiguation(matches))
			}
			if len(matches) == 1 {
				targetProject = matches[0]
				projectGID = matches[0].GID
			} else {
				gid, err := resolveProjectWithPrompt(cmd.Context(), runtimeClient(cmd), "", projectName, runtimeNoPrompt(cmd) || runtimeOutputJSON(cmd))
				if err != nil {
					return err
				}
				projectGID = gid
			}
		}

		if projectGID != "" && targetProject.GID == "" {
			found := false
			for _, option := range projectOptions {
				if option.GID == projectGID {
					targetProject = option
					found = true
					break
				}
			}
			if !found && len(projectOptions) > 0 {
				name := projectName
				if name == "" {
					name = projectGID
				}
				return fmt.Errorf("task does not belong to project %q (gid: %s)", name, projectGID)
			}
			if targetProject.GID == "" {
				targetProject = projectOption{GID: projectGID, Name: projectName}
			}
		}
	}

	resolvedSectionGID := sectionGID
	if sectionName != "" {
		if targetProject.GID == "" {
			return fmt.Errorf("task does not belong to any project; use --project or --project-gid to resolve the section")
		}
		gid, err := resolveSectionWithPrompt(cmd.Context(), runtimeClient(cmd), targetProject.GID, sectionName, runtimeNoPrompt(cmd) || runtimeOutputJSON(cmd))
		if err != nil {
			return err
		}
		resolvedSectionGID = gid
	}

	section, err := fetchSectionWithProject(cmd.Context(), resolvedSectionGID)
	if err != nil {
		return err
	}
	if section.Project == nil {
		return fmt.Errorf("section %q (gid: %s) is missing a project", section.Name, section.GID)
	}
	if targetProject.GID == "" {
		targetProject = projectOption{GID: section.Project.GID, Name: section.Project.Name}
	}
	if section.Project.GID != targetProject.GID {
		projectLabel := targetProject.Name
		if projectLabel == "" {
			projectLabel = targetProject.GID
		}
		return fmt.Errorf("Section %q (gid: %s) does not belong to project %q (gid: %s)", section.Name, section.GID, projectLabel, targetProject.GID)
	}

	if dryRun {
		taskResult := taskWriteResultFromTask(task)
		taskResult.Memberships = membershipRefsWithSection(task.Memberships, targetProject.GID, section)
		output := writeOutput{Action: "moved", DryRun: true, Task: taskResult}
		if runtimeOutputJSON(cmd) {
			return writeWriteJSON(cmd, output)
		}
		taskLabel := task.Name
		if taskLabel == "" {
			taskLabel = task.GID
		}
		sectionLabel := section.Name
		if sectionLabel == "" {
			sectionLabel = section.GID
		}
		_, err := fmt.Fprintf(cmd.OutOrStdout(), "[dry-run] Would move task %q to section %q\n", taskLabel, sectionLabel)
		return err
	}

	_, err = runtimeClient(cmd).Post(cmd.Context(), "/sections/"+section.GID+"/addTask", api.RequestBody{
		Data: map[string]string{"task": taskGID},
	})
	if err != nil {
		return err
	}

	updatedTask, err := fetchTaskWithFields(cmd.Context(), taskGID, taskWriteFields)
	if err != nil {
		return err
	}
	output := writeOutput{Action: "moved", DryRun: false, Task: taskWriteResultFromTask(updatedTask)}
	if runtimeOutputJSON(cmd) {
		return writeWriteJSON(cmd, output)
	}

	sectionLabel := section.Name
	if sectionLabel == "" {
		sectionLabel = section.GID
	}
	_, err = fmt.Fprintf(cmd.OutOrStdout(), "Moved task to section %q\n", sectionLabel)
	return err
}

func runTaskComment(cmd *cobra.Command, args []string) error {
	taskGID := args[0]
	message := args[1]
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	if err := validateGID(taskGID, "gid"); err != nil {
		return err
	}

	if message == "-" {
		text, err := readCommentInput(cmd.InOrStdin())
		if err != nil {
			return err
		}
		message = text
	} else if strings.TrimSpace(message) == "" {
		return fmt.Errorf("Comment cannot be empty")
	}

	if dryRun {
		if _, err := fetchTaskWithFields(cmd.Context(), taskGID, "gid"); err != nil {
			return err
		}
		var author *userRef
		if runtimeOutputJSON(cmd) {
			user, err := fetchCurrentUser(cmd.Context())
			if err != nil {
				return err
			}
			author = userRefFromUser(user)
		}
		output := writeOutput{
			Action: "commented",
			DryRun: true,
			Story: &storyResult{
				GID:       "",
				Text:      message,
				CreatedAt: "",
				CreatedBy: author,
			},
		}
		if runtimeOutputJSON(cmd) {
			return writeWriteJSON(cmd, output)
		}
		_, err := fmt.Fprintf(cmd.OutOrStdout(), "[dry-run] Would add comment to task %s\n", taskGID)
		return err
	}

	payload, err := runtimeClient(cmd).Post(cmd.Context(), "/tasks/"+taskGID+"/stories", api.RequestBody{
		Data: map[string]string{"text": message},
		Options: &api.RequestOptions{
			Fields: splitFields(storyWriteFields),
		},
	})
	if err != nil {
		return err
	}

	var response api.Response[api.Story]
	if err := json.Unmarshal(payload, &response); err != nil {
		return &api.ResponseError{Err: err}
	}
	output := writeOutput{
		Action: "commented",
		DryRun: false,
		Story: &storyResult{
			GID:       response.Data.GID,
			Text:      response.Data.Text,
			CreatedAt: response.Data.CreatedAt,
			CreatedBy: userRefFromUser(response.Data.CreatedBy),
		},
	}
	if runtimeOutputJSON(cmd) {
		return writeWriteJSON(cmd, output)
	}
	_, err = fmt.Fprintln(cmd.OutOrStdout(), "Comment added")
	return err
}

func runTaskDone(cmd *cobra.Command, args []string) error {
	return runTaskCompletion(cmd, args[0], true, "completed", "Marked task as completed")
}

func runTaskReopen(cmd *cobra.Command, args []string) error {
	return runTaskCompletion(cmd, args[0], false, "reopened", "Marked task as incomplete")
}

func runTaskCompletion(cmd *cobra.Command, gid string, completed bool, action, message string) error {
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	if err := validateGID(gid, "gid"); err != nil {
		return err
	}

	if dryRun {
		task, err := fetchTaskWithFields(cmd.Context(), gid, taskWriteFields)
		if err != nil {
			return err
		}
		result := taskWriteResultFromTask(task)
		result.Completed = completed
		output := writeOutput{Action: action, DryRun: true, Task: result}
		if runtimeOutputJSON(cmd) {
			return writeWriteJSON(cmd, output)
		}
		taskLabel := task.Name
		if taskLabel == "" {
			taskLabel = task.GID
		}
		state := "incomplete"
		if completed {
			state = "completed"
		}
		_, err = fmt.Fprintf(cmd.OutOrStdout(), "[dry-run] Would mark task %q as %s\n", taskLabel, state)
		return err
	}

	payload, err := runtimeClient(cmd).Put(cmd.Context(), "/tasks/"+gid, api.RequestBody{
		Data: map[string]bool{"completed": completed},
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

	output := writeOutput{Action: action, DryRun: false, Task: taskWriteResultFromTask(&response.Data)}
	if runtimeOutputJSON(cmd) {
		return writeWriteJSON(cmd, output)
	}
	_, err = fmt.Fprintln(cmd.OutOrStdout(), message)
	return err
}

func runTaskCreate(cmd *cobra.Command, args []string) error {
	title := args[0]
	projectName, _ := cmd.Flags().GetString("project")
	projectGID, _ := cmd.Flags().GetString("project-gid")
	sectionName, _ := cmd.Flags().GetString("section")
	sectionGID, _ := cmd.Flags().GetString("section-gid")
	workspaceName, _ := cmd.Flags().GetString("workspace")
	workspaceGID, _ := cmd.Flags().GetString("workspace-gid")
	assigneeValue, _ := cmd.Flags().GetString("assignee")
	dueValue, _ := cmd.Flags().GetString("due")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	if projectName != "" && projectGID != "" {
		return fmt.Errorf("use only one of --project or --project-gid")
	}
	if sectionName != "" && sectionGID != "" {
		return fmt.Errorf("use only one of --section or --section-gid")
	}
	if workspaceName != "" && workspaceGID != "" {
		return fmt.Errorf("use only one of --workspace or --workspace-gid")
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

	if projectGID != "" {
		if err := validateGID(projectGID, "project gid"); err != nil {
			return err
		}
	}
	if sectionGID != "" {
		if err := validateGID(sectionGID, "section gid"); err != nil {
			return err
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

	resolvedProjectGID := projectGID
	projectLabel := projectName
	if projectName != "" {
		gid, err := resolveProjectWithPrompt(cmd.Context(), runtimeClient(cmd), resolvedWorkspace, projectName, runtimeNoPrompt(cmd) || runtimeOutputJSON(cmd))
		if err != nil {
			return err
		}
		resolvedProjectGID = gid
	}
	if resolvedProjectGID == "" && interactive.IsInteractive(runtimeNoPrompt(cmd) || runtimeOutputJSON(cmd)) {
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
		resolvedProjectGID = project.GID
		if projectLabel == "" {
			projectLabel = project.Name
		}
	}
	if resolvedProjectGID == "" {
		return fmt.Errorf("--project or --project-gid is required")
	}
	if sectionName != "" && resolvedProjectGID == "" {
		return fmt.Errorf("--section requires --project or --project-gid")
	}

	if (dryRun || runtimeOutputJSON(cmd)) && projectLabel == "" {
		fetched, err := fetchProject(cmd.Context(), resolvedProjectGID)
		if err != nil {
			return err
		}
		projectLabel = fetched.Name
	}

	resolvedSectionGID := sectionGID
	var section *api.SectionWithProject
	if sectionName != "" {
		gid, err := resolveSectionWithPrompt(cmd.Context(), runtimeClient(cmd), resolvedProjectGID, sectionName, runtimeNoPrompt(cmd) || runtimeOutputJSON(cmd))
		if err != nil {
			return err
		}
		resolvedSectionGID = gid
	}
	if resolvedSectionGID != "" {
		fetched, err := fetchSectionWithProject(cmd.Context(), resolvedSectionGID)
		if err != nil {
			return err
		}
		if fetched.Project == nil || fetched.Project.GID != resolvedProjectGID {
			projectDisplay := projectLabel
			if projectDisplay == "" {
				projectDisplay = resolvedProjectGID
			}
			return fmt.Errorf("Section %q (gid: %s) does not belong to project %q (gid: %s)", fetched.Name, fetched.GID, projectDisplay, resolvedProjectGID)
		}
		section = fetched
	} else {
		sections, err := fetchProjectSections(cmd.Context(), resolvedProjectGID)
		if err != nil {
			return err
		}
		if len(sections) > 0 {
			section = &api.SectionWithProject{
				GID:  sections[0].GID,
				Name: sections[0].Name,
				Project: &api.Project{
					GID:  resolvedProjectGID,
					Name: projectLabel,
				},
			}
			resolvedSectionGID = sections[0].GID
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
			return fmt.Errorf("invalid --assignee value %q (use me or a GID)", assigneeValue)
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
		projectOutput := projectRef{GID: resolvedProjectGID, Name: optionalString(projectLabel)}
		sectionOutput := sectionRef{GID: "", Name: nil}
		if section != nil {
			sectionOutput = sectionRefFromSection(section)
		}
		memberships := []membershipRef{{Project: projectOutput, Section: sectionOutput}}
		output := writeOutput{
			Action: "created",
			DryRun: true,
			Task: &taskWriteResult{
				GID:         "",
				Name:        title,
				Completed:   false,
				Assignee:    assigneeRef,
				DueOn:       optionalString(dueOn),
				DueAt:       optionalString(dueAt),
				Memberships: memberships,
			},
		}
		if runtimeOutputJSON(cmd) {
			return writeWriteJSON(cmd, output)
		}
		projectDisplay := projectLabel
		if projectDisplay == "" {
			projectDisplay = resolvedProjectGID
		}
		_, err := fmt.Fprintf(cmd.OutOrStdout(), "[dry-run] Would create task %q in project %q\n", title, projectDisplay)
		return err
	}

	data := map[string]any{
		"name":     title,
		"projects": []string{resolvedProjectGID},
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
	if resolvedSectionGID != "" {
		data["memberships"] = []map[string]string{{
			"project": resolvedProjectGID,
			"section": resolvedSectionGID,
		}}
	}

	payload, err := runtimeClient(cmd).Post(cmd.Context(), "/tasks", api.RequestBody{
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
	_, err = fmt.Fprintf(cmd.OutOrStdout(), "Created task %s\n", response.Data.GID)
	return err
}

func newTaskListItem(task api.Task, section *api.Section) taskListItem {
	return taskListItem{
		GID:       task.GID,
		Name:      task.Name,
		Completed: task.Completed,
		Assignee:  task.Assignee,
		DueOn:     optionalString(task.DueOn),
		Section:   section,
	}
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func fetchCurrentUser(ctx context.Context) (*api.User, error) {
	query := url.Values{}
	query.Set("opt_fields", userFields)
	payload, err := runtimeFromContext(ctx).client.Get(ctx, "/users/me", query)
	if err != nil {
		return nil, err
	}

	var response api.Response[api.User]
	if err := json.Unmarshal(payload, &response); err != nil {
		return nil, &api.ResponseError{Err: err}
	}
	return &response.Data, nil
}

func fetchTask(ctx context.Context, gid string) (*api.Task, error) {
	return fetchTaskWithFields(ctx, gid, taskViewFields)
}

func fetchTaskWithFields(ctx context.Context, gid, fields string) (*api.Task, error) {
	query := url.Values{}
	query.Set("opt_fields", fields)
	payload, err := runtimeFromContext(ctx).client.Get(ctx, "/tasks/"+gid, query)
	if err != nil {
		return nil, err
	}

	var response api.Response[api.Task]
	if err := json.Unmarshal(payload, &response); err != nil {
		return nil, &api.ResponseError{Err: err}
	}
	return &response.Data, nil
}

func fetchTaskStories(ctx context.Context, gid string) ([]api.Story, error) {
	query := url.Values{}
	query.Set("opt_fields", storyFields)
	stories, err := api.Paginate[api.Story](ctx, runtimeFromContext(ctx).client, "/tasks/"+gid+"/stories", query)
	if err != nil {
		return nil, err
	}

	return stories, nil
}

func fetchSection(ctx context.Context, gid string) (*api.Section, error) {
	query := url.Values{}
	query.Set("opt_fields", sectionFields)
	payload, err := runtimeFromContext(ctx).client.Get(ctx, "/sections/"+gid, query)
	if err != nil {
		return nil, err
	}

	var response api.Response[api.Section]
	if err := json.Unmarshal(payload, &response); err != nil {
		return nil, &api.ResponseError{Err: err}
	}
	return &response.Data, nil
}

func fetchSectionWithProject(ctx context.Context, gid string) (*api.SectionWithProject, error) {
	query := url.Values{}
	query.Set("opt_fields", sectionProjectFields)
	payload, err := runtimeFromContext(ctx).client.Get(ctx, "/sections/"+gid, query)
	if err != nil {
		return nil, err
	}

	var response api.Response[api.SectionWithProject]
	if err := json.Unmarshal(payload, &response); err != nil {
		return nil, &api.ResponseError{Err: err}
	}
	return &response.Data, nil
}

func fetchProject(ctx context.Context, gid string) (*api.Project, error) {
	query := url.Values{}
	query.Set("opt_fields", projectFields)
	payload, err := runtimeFromContext(ctx).client.Get(ctx, "/projects/"+gid, query)
	if err != nil {
		return nil, err
	}

	var response api.Response[api.Project]
	if err := json.Unmarshal(payload, &response); err != nil {
		return nil, &api.ResponseError{Err: err}
	}
	return &response.Data, nil
}

func fetchProjectSections(ctx context.Context, projectGID string) ([]api.Section, error) {
	query := url.Values{}
	query.Set("opt_fields", sectionFields)
	return api.Paginate[api.Section](ctx, runtimeFromContext(ctx).client, "/projects/"+projectGID+"/sections", query)
}

func filterComments(stories []api.Story) []taskComment {
	comments := make([]taskComment, 0)
	for _, story := range stories {
		if story.Subtype != "comment_added" {
			if story.Subtype != "" || strings.TrimSpace(story.Text) == "" {
				continue
			}
		}
		comments = append(comments, taskComment{
			GID:       story.GID,
			CreatedAt: story.CreatedAt,
			CreatedBy: story.CreatedBy,
			Text:      story.Text,
		})
	}
	return comments
}

func compareCommentTime(left, right string) bool {
	leftTime, leftOK := parseTime(left)
	rightTime, rightOK := parseTime(right)
	if leftOK && rightOK {
		return leftTime.After(rightTime)
	}
	if leftOK {
		return true
	}
	if rightOK {
		return false
	}
	return left > right
}

func parseTime(value string) (time.Time, bool) {
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}, false
	}
	return parsed, true
}

func writeTaskViewJSON(cmd *cobra.Command, task *api.Task, comments []taskComment) error {
	output := taskViewOutput{
		GID:         task.GID,
		Name:        task.Name,
		Completed:   task.Completed,
		Assignee:    task.Assignee,
		DueOn:       optionalString(task.DueOn),
		DueAt:       optionalString(task.DueAt),
		Notes:       task.Notes,
		Memberships: task.Memberships,
		Comments:    comments,
	}

	encoder := json.NewEncoder(cmd.OutOrStdout())
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

func writeTaskViewTable(cmd *cobra.Command, task *api.Task, comments []taskComment) error {
	writer := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)

	assignee := "-"
	if task.Assignee != nil && task.Assignee.Name != "" {
		assignee = task.Assignee.Name
	}

	due := "-"
	if task.DueAt != "" {
		due = task.DueAt
	} else if task.DueOn != "" {
		due = task.DueOn
	}

	notes := strings.TrimSpace(task.Notes)
	if notes == "" {
		notes = "-"
	}

	if _, err := fmt.Fprintf(writer, "Name:\t%s\n", task.Name); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(writer, "Assignee:\t%s\n", assignee); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(writer, "Due:\t%s\n", due); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(writer, "Completed:\t%t\n", task.Completed); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(writer, "Notes:\t%s\n", notes); err != nil {
		return err
	}

	if _, err := fmt.Fprintln(writer, ""); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(writer, "Memberships:"); err != nil {
		return err
	}
	for _, membership := range task.Memberships {
		projectName := "-"
		if membership.Project != nil && membership.Project.Name != "" {
			projectName = membership.Project.Name
		}
		sectionName := "-"
		if membership.Section != nil && membership.Section.Name != "" {
			sectionName = membership.Section.Name
		}
		if _, err := fmt.Fprintf(writer, "  %s / %s\n", projectName, sectionName); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprintln(writer, ""); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(writer, "Comments (%d):\n", len(comments)); err != nil {
		return err
	}
	for _, comment := range comments {
		createdAt := comment.CreatedAt
		if parsed, ok := parseTime(comment.CreatedAt); ok {
			createdAt = parsed.Format("2006-01-02 15:04")
		}
		author := "-"
		if comment.CreatedBy != nil && comment.CreatedBy.Name != "" {
			author = comment.CreatedBy.Name
		}
		if _, err := fmt.Fprintf(writer, "  %s  %s: %s\n", createdAt, author, comment.Text); err != nil {
			return err
		}
	}

	return writer.Flush()
}

func writeWriteJSON(cmd *cobra.Command, output writeOutput) error {
	encoder := json.NewEncoder(cmd.OutOrStdout())
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

func splitFields(fields string) []string {
	parts := strings.Split(fields, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

func userRefFromUser(user *api.User) *userRef {
	if user == nil {
		return nil
	}
	return &userRef{
		GID:  user.GID,
		Name: optionalString(user.Name),
	}
}

func projectRefFromProject(project *api.Project) projectRef {
	if project == nil {
		return projectRef{GID: "", Name: nil}
	}
	return projectRef{
		GID:  project.GID,
		Name: optionalString(project.Name),
	}
}

func sectionRefFromMembership(section *api.Section) sectionRef {
	if section == nil {
		return sectionRef{GID: "", Name: nil}
	}
	return sectionRef{
		GID:  section.GID,
		Name: optionalString(section.Name),
	}
}

func sectionRefFromSection(section *api.SectionWithProject) sectionRef {
	if section == nil {
		return sectionRef{GID: "", Name: nil}
	}
	return sectionRef{
		GID:  section.GID,
		Name: optionalString(section.Name),
	}
}

func membershipRefsFrom(memberships []api.Membership) []membershipRef {
	if len(memberships) == 0 {
		return []membershipRef{}
	}

	output := make([]membershipRef, 0, len(memberships))
	for _, membership := range memberships {
		output = append(output, membershipRef{
			Project: projectRefFromProject(membership.Project),
			Section: sectionRefFromMembership(membership.Section),
		})
	}
	return output
}

func membershipRefsWithSection(memberships []api.Membership, projectGID string, section *api.SectionWithProject) []membershipRef {
	updated := make([]api.Membership, len(memberships))
	for i, membership := range memberships {
		updated[i] = membership
		if membership.Project != nil && membership.Project.GID == projectGID {
			updated[i].Section = &api.Section{GID: section.GID, Name: section.Name}
		}
	}
	return membershipRefsFrom(updated)
}

func taskWriteResultFromTask(task *api.Task) *taskWriteResult {
	if task == nil {
		return nil
	}
	return &taskWriteResult{
		GID:         task.GID,
		Name:        task.Name,
		Completed:   task.Completed,
		Assignee:    userRefFromUser(task.Assignee),
		DueOn:       optionalString(task.DueOn),
		DueAt:       optionalString(task.DueAt),
		Memberships: membershipRefsFrom(task.Memberships),
	}
}

func projectOptionsFromMemberships(memberships []api.Membership) []projectOption {
	seen := make(map[string]projectOption)
	for _, membership := range memberships {
		if membership.Project == nil || membership.Project.GID == "" {
			continue
		}
		if _, ok := seen[membership.Project.GID]; ok {
			continue
		}
		seen[membership.Project.GID] = projectOption{
			GID:  membership.Project.GID,
			Name: membership.Project.Name,
		}
	}

	options := make([]projectOption, 0, len(seen))
	for _, option := range seen {
		options = append(options, option)
	}
	sort.Slice(options, func(i, j int) bool {
		leftHasName := options[i].Name != ""
		rightHasName := options[j].Name != ""
		if leftHasName != rightHasName {
			return leftHasName
		}
		if options[i].Name != options[j].Name {
			return options[i].Name < options[j].Name
		}
		return options[i].GID < options[j].GID
	})
	return options
}

func matchProjectOptionsByName(options []projectOption, name string) []projectOption {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil
	}
	matches := make([]projectOption, 0)
	for _, option := range options {
		if strings.EqualFold(option.Name, name) {
			matches = append(matches, option)
		}
	}
	return matches
}

func formatProjectDisambiguation(options []projectOption) string {
	var builder strings.Builder
	builder.WriteString("Task belongs to multiple projects. Use --project or --project-gid. Projects:\n")
	for _, option := range options {
		fmt.Fprintf(&builder, "  %s  %s\n", option.GID, option.Name)
	}
	return strings.TrimRight(builder.String(), "\n")
}

func parseDueDate(input string) (string, string, error) {
	if _, err := time.Parse("2006-01-02", input); err == nil {
		return input, "", nil
	}

	parsed, err := time.Parse(time.RFC3339Nano, input)
	if err != nil {
		return "", "", fmt.Errorf("Invalid date format %q: use YYYY-MM-DD or RFC3339 with timezone", input)
	}

	return "", parsed.UTC().Format("2006-01-02T15:04:05.000Z"), nil
}

func validateGID(value, label string) error {
	if !resolve.IsGID(value) {
		return fmt.Errorf("Invalid %s %q: must be numeric", label, value)
	}
	return nil
}

func readCommentInput(reader io.Reader) (string, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return "", fmt.Errorf("failed to read from stdin: %w", err)
	}
	text := strings.TrimRight(string(data), "\r\n")
	if strings.TrimSpace(text) == "" {
		return "", fmt.Errorf("Comment cannot be empty")
	}
	return text, nil
}
