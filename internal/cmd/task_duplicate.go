// Package cmd implements task duplicate command.
package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/vincentsch/asana-cli/internal/api"
)

var taskDuplicateCmd = &cobra.Command{
	Use:   "duplicate <gid>",
	Short: "Duplicate a task",
	Long: `Create a copy of a task with optional fields.

By default, duplicates the task name only. Use --include to copy additional
fields like assignee, dates, notes, and subtasks. The operation is asynchronous;
use --wait to block until complete and return the new task GID.

See also: task create, task view`,
	Example: `  # Simple duplicate
  asana task duplicate 1234567890123456

  # Duplicate with a new name
  asana task duplicate 1234567890123456 --name "Copy of Task"

  # Include all task data
  asana task duplicate 1234567890123456 --include assignee,dates,notes,subtasks

  # Wait for completion and get new task GID
  asana task duplicate 1234567890123456 --wait

  # Workflow: Duplicate and customize
  asana task duplicate 1234567890123456 --name "Sprint 2 version" --wait
  asana task update <new-gid> --assignee me --due 2024-03-01`,
	Args: cobra.ExactArgs(1),
	RunE: runTaskDuplicate,
}

func init() {
	taskDuplicateCmd.Flags().String("name", "", "Name for the duplicated task")
	taskDuplicateCmd.Flags().StringSlice("include", nil, "Fields to include: assignee, attachments, dates, dependencies, followers, notes, parent, projects, subtasks, tags")
	taskDuplicateCmd.Flags().Bool("wait", false, "Wait for job completion")

	taskCmd.AddCommand(taskDuplicateCmd)
}

// duplicateOutput extends writeOutput for duplicate operations
type duplicateOutput struct {
	Action string           `json:"action"`
	DryRun bool             `json:"dry_run"`
	Task   *taskWriteResult `json:"task,omitempty"`
	Job    *jobOutput       `json:"job,omitempty"`
}

type jobOutput struct {
	GID    string `json:"gid"`
	Status string `json:"status"`
}

func runTaskDuplicate(cmd *cobra.Command, args []string) error {
	taskGID := args[0]
	name, _ := cmd.Flags().GetString("name")
	include, _ := cmd.Flags().GetStringSlice("include")
	wait, _ := cmd.Flags().GetBool("wait")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	if err := validateGID(taskGID, "gid"); err != nil {
		return err
	}

	// Validate include values
	validInclude := map[string]bool{
		"assignee":     true,
		"attachments":  true,
		"dates":        true,
		"dependencies": true,
		"followers":    true,
		"notes":        true,
		"parent":       true,
		"projects":     true,
		"subtasks":     true,
		"tags":         true,
	}
	for _, item := range include {
		item = strings.TrimSpace(strings.ToLower(item))
		if !validInclude[item] {
			return fmt.Errorf("invalid --include value %q; valid values: assignee, attachments, dates, dependencies, followers, notes, parent, projects, subtasks, tags", item)
		}
	}

	if dryRun {
		task, err := fetchTaskWithFields(cmd.Context(), taskGID, "gid,name")
		if err != nil {
			return err
		}
		duplicateName := name
		if duplicateName == "" {
			duplicateName = fmt.Sprintf("Copy of %s", task.Name)
		}
		output := duplicateOutput{
			Action: "duplicated",
			DryRun: true,
			Task:   &taskWriteResult{GID: "", Name: duplicateName},
		}
		if runtimeOutputJSON(cmd) {
			encoder := json.NewEncoder(cmd.OutOrStdout())
			encoder.SetIndent("", "  ")
			return encoder.Encode(output)
		}
		_, err = fmt.Fprintf(cmd.OutOrStdout(), "[dry-run] Would duplicate task %q as %q\n", task.Name, duplicateName)
		return err
	}

	// Build request body
	data := make(map[string]any)
	if name != "" {
		data["name"] = name
	}
	if len(include) > 0 {
		// Normalize include values
		normalizedInclude := make([]string, len(include))
		for i, item := range include {
			normalizedInclude[i] = strings.TrimSpace(strings.ToLower(item))
		}
		data["include"] = normalizedInclude
	}

	payload, err := runtimeClient(cmd).Post(cmd.Context(), "/tasks/"+taskGID+"/duplicate", api.RequestBody{
		Data: data,
	})
	if err != nil {
		return err
	}

	var response api.Response[api.Job]
	if err := json.Unmarshal(payload, &response); err != nil {
		return &api.ResponseError{Err: err}
	}

	if !wait {
		output := duplicateOutput{
			Action: "duplicated",
			DryRun: false,
			Job: &jobOutput{
				GID:    response.Data.GID,
				Status: response.Data.Status,
			},
		}
		if runtimeOutputJSON(cmd) {
			encoder := json.NewEncoder(cmd.OutOrStdout())
			encoder.SetIndent("", "  ")
			return encoder.Encode(output)
		}
		_, err = fmt.Fprintf(cmd.OutOrStdout(), "Duplicate job started: %s (status: %s)\n", response.Data.GID, response.Data.Status)
		return err
	}

	// Wait for job completion
	job, err := waitForJob(cmd.Context(), runtimeClient(cmd), response.Data.GID)
	if err != nil {
		return err
	}

	if job.NewTask == nil {
		return fmt.Errorf("job completed but no new task returned")
	}

	output := duplicateOutput{
		Action: "duplicated",
		DryRun: false,
		Task: &taskWriteResult{
			GID:  job.NewTask.GID,
			Name: job.NewTask.Name,
		},
	}
	if runtimeOutputJSON(cmd) {
		encoder := json.NewEncoder(cmd.OutOrStdout())
		encoder.SetIndent("", "  ")
		return encoder.Encode(output)
	}
	_, err = fmt.Fprintf(cmd.OutOrStdout(), "Task duplicated: %s\n", job.NewTask.GID)
	return err
}
