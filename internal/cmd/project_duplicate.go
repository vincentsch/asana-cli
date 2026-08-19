// Package cmd implements project duplicate command.
package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/vincentsch/asana-cli/internal/api"
)

var projectDuplicateCmd = &cobra.Command{
	Use:   "duplicate <project-gid>",
	Short: "Duplicate a project",
	Long: `Create a copy of a project with optional task data.

Use --include to specify what to copy: members, notes, task details, etc.
Use --schedule-start-on or --schedule-due-on to shift task dates.

See also: project create, project view`,
	Example: `  # Duplicate with a new name
  asana project duplicate 1234567890123456 --name "Q2 Sprint"

  # Include tasks with all details
  asana project duplicate 1234567890123456 --name "Copy" --include task_notes,task_assignee,task_dates

  # Duplicate to a different team
  asana project duplicate 1234567890123456 --name "New Project" -t "Engineering"

  # Shift dates to new start date
  asana project duplicate 1234567890123456 --name "Q2" --include task_dates --schedule-start-on 2024-04-01

  # Wait for completion
  asana project duplicate 1234567890123456 --name "Copy" --wait

  # Preview without duplicating
  asana project duplicate 1234567890123456 --name "Test" --dry-run`,
	Args: cobra.ExactArgs(1),
	RunE: runProjectDuplicate,
}

func init() {
	projectDuplicateCmd.Flags().String("name", "", "Name for the duplicated project (required)")
	projectDuplicateCmd.Flags().StringP("team", "t", "", "Team name for the new project")
	projectDuplicateCmd.Flags().String("team-gid", "", "Team GID for the new project")
	projectDuplicateCmd.Flags().String("include", "", "Comma-separated fields to include: members,notes,task_notes,task_assignee,task_subtasks,task_attachments,task_dates,task_dependencies,task_followers,task_tags,task_projects")
	projectDuplicateCmd.Flags().String("schedule-start-on", "", "Shift dates based on new start date (YYYY-MM-DD)")
	projectDuplicateCmd.Flags().String("schedule-due-on", "", "Shift dates based on new due date (YYYY-MM-DD)")
	projectDuplicateCmd.Flags().Bool("schedule-skip-weekends", false, "Skip weekends when shifting dates")
	projectDuplicateCmd.Flags().Bool("wait", false, "Wait for duplication to complete")

	projectCmd.AddCommand(projectDuplicateCmd)
}

func runProjectDuplicate(cmd *cobra.Command, args []string) error {
	projectGID := args[0]
	name, _ := cmd.Flags().GetString("name")
	teamName, _ := cmd.Flags().GetString("team")
	teamGID, _ := cmd.Flags().GetString("team-gid")
	include, _ := cmd.Flags().GetString("include")
	scheduleStartOn, _ := cmd.Flags().GetString("schedule-start-on")
	scheduleDueOn, _ := cmd.Flags().GetString("schedule-due-on")
	scheduleSkipWeekends, _ := cmd.Flags().GetBool("schedule-skip-weekends")
	wait, _ := cmd.Flags().GetBool("wait")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	if err := validateGID(projectGID, "project gid"); err != nil {
		return err
	}

	if teamName != "" && teamGID != "" {
		return fmt.Errorf("use only one of --team or --team-gid")
	}
	if name == "" {
		return fmt.Errorf("--name is required")
	}
	if scheduleStartOn != "" && scheduleDueOn != "" {
		return fmt.Errorf("use only one of --schedule-start-on or --schedule-due-on")
	}

	// Resolve team if provided - need workspace from source project
	var resolvedTeam string
	if teamGID != "" {
		resolvedTeam = teamGID
	}
	if teamName != "" {
		project, err := fetchProjectForWorkspace(cmd.Context(), projectGID)
		if err != nil {
			return err
		}
		if project.WorkspaceGID == "" {
			return fmt.Errorf("workspace gid is required to resolve team name")
		}
		gid, err := resolveTeamWithPrompt(cmd.Context(), runtimeClient(cmd), project.WorkspaceGID, teamName, runtimeNoPrompt(cmd) || runtimeOutputJSON(cmd))
		if err != nil {
			return err
		}
		resolvedTeam = gid
	}

	if scheduleStartOn != "" {
		if _, err := parseDateOnly(scheduleStartOn); err != nil {
			return fmt.Errorf("--schedule-start-on: %w", err)
		}
	}
	if scheduleDueOn != "" {
		if _, err := parseDateOnly(scheduleDueOn); err != nil {
			return fmt.Errorf("--schedule-due-on: %w", err)
		}
	}

	includeValues := []string{}
	if include != "" {
		parts := strings.Split(include, ",")
		for _, part := range parts {
			part = strings.TrimSpace(strings.ToLower(part))
			if part == "" {
				continue
			}
			includeValues = append(includeValues, part)
		}
	}
	if len(includeValues) > 0 {
		validInclude := map[string]bool{
			"members":           true,
			"notes":             true,
			"task_notes":        true,
			"task_assignee":     true,
			"task_subtasks":     true,
			"task_attachments":  true,
			"task_dates":        true,
			"task_dependencies": true,
			"task_followers":    true,
			"task_tags":         true,
			"task_projects":     true,
		}
		for _, item := range includeValues {
			if !validInclude[item] {
				return fmt.Errorf("invalid --include value %q; valid values: members, notes, task_notes, task_assignee, task_subtasks, task_attachments, task_dates, task_dependencies, task_followers, task_tags, task_projects", item)
			}
		}
	}

	if scheduleStartOn != "" || scheduleDueOn != "" || scheduleSkipWeekends {
		if !containsInclude(includeValues, "task_dates") {
			return fmt.Errorf("--schedule-start-on/--schedule-due-on/--schedule-skip-weekends require task_dates in --include")
		}
	}
	if scheduleSkipWeekends && scheduleStartOn == "" && scheduleDueOn == "" {
		return fmt.Errorf("--schedule-skip-weekends requires --schedule-start-on or --schedule-due-on")
	}

	if dryRun {
		output := projectDuplicateOutput{
			Action: "duplicated",
			DryRun: true,
			Project: &projectOutputDetail{
				Name: name,
			},
		}
		if runtimeOutputJSON(cmd) {
			encoder := json.NewEncoder(cmd.OutOrStdout())
			encoder.SetIndent("", "  ")
			return encoder.Encode(output)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "[dry-run] Would duplicate project %s as %q\n", projectGID, name)
		return nil
	}

	// Build request
	data := map[string]interface{}{
		"name": name,
	}
	if resolvedTeam != "" {
		data["team"] = resolvedTeam
	}
	if len(includeValues) > 0 {
		data["include"] = includeValues
	}

	// Schedule duplication
	if scheduleStartOn != "" || scheduleDueOn != "" || scheduleSkipWeekends {
		schedule := map[string]interface{}{
			"should_skip_weekends": scheduleSkipWeekends,
		}
		if scheduleStartOn != "" {
			schedule["start_on"] = scheduleStartOn
		}
		if scheduleDueOn != "" {
			schedule["due_on"] = scheduleDueOn
		}
		data["schedule_dates"] = schedule
	}

	payload, err := runtimeClient(cmd).Post(cmd.Context(), "/projects/"+projectGID+"/duplicate", api.RequestBody{
		Data: data,
	})
	if err != nil {
		return err
	}

	var jobResp api.Response[api.Job]
	if err := json.Unmarshal(payload, &jobResp); err != nil {
		return &api.ResponseError{Err: err}
	}

	job := jobResp.Data

	if !wait {
		// Return job info immediately
		output := projectDuplicateOutput{
			Action: "duplicated",
			DryRun: false,
			Job: &jobOutput{
				GID:    job.GID,
				Status: job.Status,
			},
		}
		if runtimeOutputJSON(cmd) {
			encoder := json.NewEncoder(cmd.OutOrStdout())
			encoder.SetIndent("", "  ")
			return encoder.Encode(output)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Duplication started: job %s (status: %s)\n", job.GID, job.Status)
		return nil
	}

	// Wait for job completion
	completedJob, err := waitForJob(cmd.Context(), runtimeClient(cmd), job.GID)
	if err != nil {
		return err
	}

	if completedJob.NewProject == nil {
		return fmt.Errorf("job completed but no new project returned")
	}

	output := projectDuplicateOutput{
		Action: "duplicated",
		DryRun: false,
		Project: &projectOutputDetail{
			GID:  completedJob.NewProject.GID,
			Name: completedJob.NewProject.Name,
		},
	}
	if runtimeOutputJSON(cmd) {
		encoder := json.NewEncoder(cmd.OutOrStdout())
		encoder.SetIndent("", "  ")
		return encoder.Encode(output)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Duplicated project: %s (%s)\n", completedJob.NewProject.Name, completedJob.NewProject.GID)
	return nil
}

type projectDuplicateOutput struct {
	Action  string               `json:"action"`
	DryRun  bool                 `json:"dry_run"`
	Project *projectOutputDetail `json:"project,omitempty"`
	Job     *jobOutput           `json:"job,omitempty"`
}

func containsInclude(values []string, value string) bool {
	for _, item := range values {
		if item == value {
			return true
		}
	}
	return false
}
