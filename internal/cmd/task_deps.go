// Package cmd implements task dependency and dependent commands.
package cmd

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/vincentsch/asana-cli/internal/api"
)

var taskDependencyCmd = &cobra.Command{
	Use:   "dependency",
	Short: "Manage task dependencies (tasks this task depends on)",
	Long: `Manage task dependencies - tasks that must be completed before this task.

Dependencies help establish order and blocking relationships between tasks.
A task with dependencies cannot be started until its dependencies are complete.

See also: task dependent, task view`,
}

var taskDependencyListCmd = &cobra.Command{
	Use:   "list <gid>",
	Short: "List dependencies of a task",
	Long: `List all tasks that this task depends on (blockers).

See also: task dependency add, task dependent list`,
	Example: `  # List dependencies
  asana task dependency list 1234567890123456

  # Output as JSON
  asana task dependency list 1234567890123456 --json`,
	Args: cobra.ExactArgs(1),
	RunE: runTaskDependencyList,
}

var taskDependencyAddCmd = &cobra.Command{
	Use:   "add <gid>",
	Short: "Add dependencies to a task",
	Long: `Add one or more tasks as dependencies (blockers) for this task.

See also: task dependency list, task dependency remove`,
	Example: `  # Add a single dependency
  asana task dependency add 1234567890123456 --depends-on 9876543210987654

  # Add multiple dependencies
  asana task dependency add 1234567890123456 --depends-on 111,222,333

  # Workflow: Create a dependency chain
  asana task dependency add 1234567890123456 --depends-on 9876543210987654
  asana task comment 1234567890123456 "Blocked by design review"`,
	Args: cobra.ExactArgs(1),
	RunE: runTaskDependencyAdd,
}

var taskDependencyRemoveCmd = &cobra.Command{
	Use:   "remove <gid>",
	Short: "Remove dependencies from a task",
	Long: `Remove one or more dependencies from a task.

See also: task dependency list, task dependency add`,
	Example: `  # Remove a dependency
  asana task dependency remove 1234567890123456 --depends-on 9876543210987654 --confirm

  # Workflow: Unblock a task
  asana task dependency list 1234567890123456
  asana task dependency remove 1234567890123456 --depends-on 9876543210987654 --confirm
  asana task comment 1234567890123456 "Unblocked - dependency resolved"`,
	Args: cobra.ExactArgs(1),
	RunE: runTaskDependencyRemove,
}

var taskDependentCmd = &cobra.Command{
	Use:   "dependent",
	Short: "Manage task dependents (tasks that depend on this task)",
	Long: `Manage task dependents - tasks that are blocked by this task.

When this task is complete, its dependents become unblocked.

See also: task dependency, task view`,
}

var taskDependentListCmd = &cobra.Command{
	Use:   "list <gid>",
	Short: "List dependents of a task",
	Long: `List all tasks that depend on this task (are blocked by it).

See also: task dependent add, task dependency list`,
	Example: `  # List dependents
  asana task dependent list 1234567890123456

  # Output as JSON
  asana task dependent list 1234567890123456 --json`,
	Args: cobra.ExactArgs(1),
	RunE: runTaskDependentList,
}

var taskDependentAddCmd = &cobra.Command{
	Use:   "add <gid>",
	Short: "Add dependents to a task",
	Long: `Add tasks that will depend on (be blocked by) this task.

See also: task dependent list, task dependent remove`,
	Example: `  # Add a dependent
  asana task dependent add 1234567890123456 --dependent 9876543210987654

  # Add multiple dependents
  asana task dependent add 1234567890123456 --dependent 111,222,333`,
	Args: cobra.ExactArgs(1),
	RunE: runTaskDependentAdd,
}

var taskDependentRemoveCmd = &cobra.Command{
	Use:   "remove <gid>",
	Short: "Remove dependents from a task",
	Long: `Remove tasks from depending on this task.

See also: task dependent list, task dependent add`,
	Example: `  # Remove a dependent
  asana task dependent remove 1234567890123456 --dependent 9876543210987654 --confirm`,
	Args: cobra.ExactArgs(1),
	RunE: runTaskDependentRemove,
}

func init() {
	taskDependencyListCmd.Flags().Int("limit", 0, "Limit number of dependencies in output")
	taskDependencyAddCmd.Flags().StringSlice("depends-on", nil, "Task GIDs to add as dependencies (required)")

	taskDependencyRemoveCmd.Flags().StringSlice("depends-on", nil, "Task GIDs to remove as dependencies (required)")

	taskDependentListCmd.Flags().Int("limit", 0, "Limit number of dependents in output")
	taskDependentAddCmd.Flags().StringSlice("dependent", nil, "Task GIDs to add as dependents (required)")

	taskDependentRemoveCmd.Flags().StringSlice("dependent", nil, "Task GIDs to remove as dependents (required)")

	taskDependencyCmd.AddCommand(taskDependencyListCmd)
	taskDependencyCmd.AddCommand(taskDependencyAddCmd)
	taskDependencyCmd.AddCommand(taskDependencyRemoveCmd)
	taskCmd.AddCommand(taskDependencyCmd)

	taskDependentCmd.AddCommand(taskDependentListCmd)
	taskDependentCmd.AddCommand(taskDependentAddCmd)
	taskDependentCmd.AddCommand(taskDependentRemoveCmd)
	taskCmd.AddCommand(taskDependentCmd)
}

type dependencyListItem struct {
	GID  string `json:"gid"`
	Name string `json:"name"`
}

type dependencyOutput struct {
	Action       string   `json:"action"`
	DryRun       bool     `json:"dry_run"`
	Task         *gidRef  `json:"task,omitempty"`
	Dependencies []string `json:"dependencies,omitempty"`
	Dependents   []string `json:"dependents,omitempty"`
}

func runTaskDependencyList(cmd *cobra.Command, args []string) error {
	taskGID := args[0]
	limit, _ := cmd.Flags().GetInt("limit")
	if err := validateGID(taskGID, "gid"); err != nil {
		return err
	}
	if cmd.Flags().Changed("limit") && limit < 1 {
		return fmt.Errorf("--limit must be >= 1")
	}

	query := url.Values{}
	query.Set("opt_fields", "gid,name")
	deps, err := api.Paginate[api.TaskDependency](cmd.Context(), runtimeClient(cmd), "/tasks/"+taskGID+"/dependencies", query)
	if err != nil {
		return err
	}

	if runtimeOutputJSON(cmd) {
		items := make([]dependencyListItem, len(deps))
		for i, dep := range deps {
			items[i] = dependencyListItem{GID: dep.GID, Name: dep.Name}
		}
		if limit > 0 && len(items) > limit {
			items = items[:limit]
		}
		encoder := json.NewEncoder(cmd.OutOrStdout())
		encoder.SetIndent("", "  ")
		return encoder.Encode(items)
	}

	if limit > 0 && len(deps) > limit {
		deps = deps[:limit]
	}
	writer := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(writer, "GID\tNAME"); err != nil {
		return err
	}
	for _, dep := range deps {
		if _, err := fmt.Fprintf(writer, "%s\t%s\n", dep.GID, dep.Name); err != nil {
			return err
		}
	}
	return writer.Flush()
}

func runTaskDependencyAdd(cmd *cobra.Command, args []string) error {
	taskGID := args[0]
	dependsOn, _ := cmd.Flags().GetStringSlice("depends-on")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	if err := validateGID(taskGID, "gid"); err != nil {
		return err
	}
	if len(dependsOn) == 0 {
		return fmt.Errorf("--depends-on is required")
	}

	// Validate all GIDs
	dependsOn = normalizeDependencyGIDs(dependsOn)
	for _, gid := range dependsOn {
		if err := validateGID(gid, "depends-on gid"); err != nil {
			return err
		}
	}
	if len(dependsOn) == 0 {
		return fmt.Errorf("--depends-on is required")
	}

	if dryRun {
		// Validate task exists
		if _, err := fetchTaskWithFields(cmd.Context(), taskGID, "gid"); err != nil {
			return err
		}
		output := dependencyOutput{
			Action:       "added",
			DryRun:       true,
			Task:         &gidRef{GID: taskGID},
			Dependencies: dependsOn,
		}
		if runtimeOutputJSON(cmd) {
			encoder := json.NewEncoder(cmd.OutOrStdout())
			encoder.SetIndent("", "  ")
			return encoder.Encode(output)
		}
		_, err := fmt.Fprintf(cmd.OutOrStdout(), "[dry-run] Would add %d dependencies to task %s\n", len(dependsOn), taskGID)
		return err
	}

	_, err := runtimeClient(cmd).Post(cmd.Context(), "/tasks/"+taskGID+"/addDependencies", api.RequestBody{
		Data: map[string][]string{"dependencies": dependsOn},
	})
	if err != nil {
		return err
	}

	output := dependencyOutput{
		Action:       "added",
		DryRun:       false,
		Task:         &gidRef{GID: taskGID},
		Dependencies: dependsOn,
	}
	if runtimeOutputJSON(cmd) {
		encoder := json.NewEncoder(cmd.OutOrStdout())
		encoder.SetIndent("", "  ")
		return encoder.Encode(output)
	}
	_, err = fmt.Fprintf(cmd.OutOrStdout(), "Added %d dependencies\n", len(dependsOn))
	return err
}

func runTaskDependencyRemove(cmd *cobra.Command, args []string) error {
	taskGID := args[0]
	dependsOn, _ := cmd.Flags().GetStringSlice("depends-on")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	if err := validateGID(taskGID, "gid"); err != nil {
		return err
	}
	if len(dependsOn) == 0 {
		return fmt.Errorf("--depends-on is required")
	}

	// Validate all GIDs
	dependsOn = normalizeDependencyGIDs(dependsOn)
	for _, gid := range dependsOn {
		if err := validateGID(gid, "depends-on gid"); err != nil {
			return err
		}
	}
	if len(dependsOn) == 0 {
		return fmt.Errorf("--depends-on is required")
	}

	if dryRun {
		// Validate task exists
		if _, err := fetchTaskWithFields(cmd.Context(), taskGID, "gid"); err != nil {
			return err
		}
		output := dependencyOutput{
			Action:       "removed",
			DryRun:       true,
			Task:         &gidRef{GID: taskGID},
			Dependencies: dependsOn,
		}
		if runtimeOutputJSON(cmd) {
			encoder := json.NewEncoder(cmd.OutOrStdout())
			encoder.SetIndent("", "  ")
			return encoder.Encode(output)
		}
		_, err := fmt.Fprintf(cmd.OutOrStdout(), "[dry-run] Would remove %d dependencies from task %s\n", len(dependsOn), taskGID)
		return err
	}

	_, err := runtimeClient(cmd).Post(cmd.Context(), "/tasks/"+taskGID+"/removeDependencies", api.RequestBody{
		Data: map[string][]string{"dependencies": dependsOn},
	})
	if err != nil {
		return err
	}

	output := dependencyOutput{
		Action:       "removed",
		DryRun:       false,
		Task:         &gidRef{GID: taskGID},
		Dependencies: dependsOn,
	}
	if runtimeOutputJSON(cmd) {
		encoder := json.NewEncoder(cmd.OutOrStdout())
		encoder.SetIndent("", "  ")
		return encoder.Encode(output)
	}
	_, err = fmt.Fprintf(cmd.OutOrStdout(), "Removed %d dependencies\n", len(dependsOn))
	return err
}

func runTaskDependentList(cmd *cobra.Command, args []string) error {
	taskGID := args[0]
	limit, _ := cmd.Flags().GetInt("limit")
	if err := validateGID(taskGID, "gid"); err != nil {
		return err
	}
	if cmd.Flags().Changed("limit") && limit < 1 {
		return fmt.Errorf("--limit must be >= 1")
	}

	query := url.Values{}
	query.Set("opt_fields", "gid,name")
	deps, err := api.Paginate[api.TaskDependency](cmd.Context(), runtimeClient(cmd), "/tasks/"+taskGID+"/dependents", query)
	if err != nil {
		return err
	}

	if runtimeOutputJSON(cmd) {
		items := make([]dependencyListItem, len(deps))
		for i, dep := range deps {
			items[i] = dependencyListItem{GID: dep.GID, Name: dep.Name}
		}
		if limit > 0 && len(items) > limit {
			items = items[:limit]
		}
		encoder := json.NewEncoder(cmd.OutOrStdout())
		encoder.SetIndent("", "  ")
		return encoder.Encode(items)
	}

	if limit > 0 && len(deps) > limit {
		deps = deps[:limit]
	}
	writer := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(writer, "GID\tNAME"); err != nil {
		return err
	}
	for _, dep := range deps {
		if _, err := fmt.Fprintf(writer, "%s\t%s\n", dep.GID, dep.Name); err != nil {
			return err
		}
	}
	return writer.Flush()
}

func runTaskDependentAdd(cmd *cobra.Command, args []string) error {
	taskGID := args[0]
	dependents, _ := cmd.Flags().GetStringSlice("dependent")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	if err := validateGID(taskGID, "gid"); err != nil {
		return err
	}
	if len(dependents) == 0 {
		return fmt.Errorf("--dependent is required")
	}

	// Validate all GIDs
	dependents = normalizeDependencyGIDs(dependents)
	for _, gid := range dependents {
		if err := validateGID(gid, "dependent gid"); err != nil {
			return err
		}
	}
	if len(dependents) == 0 {
		return fmt.Errorf("--dependent is required")
	}

	if dryRun {
		// Validate task exists
		if _, err := fetchTaskWithFields(cmd.Context(), taskGID, "gid"); err != nil {
			return err
		}
		output := dependencyOutput{
			Action:     "added",
			DryRun:     true,
			Task:       &gidRef{GID: taskGID},
			Dependents: dependents,
		}
		if runtimeOutputJSON(cmd) {
			encoder := json.NewEncoder(cmd.OutOrStdout())
			encoder.SetIndent("", "  ")
			return encoder.Encode(output)
		}
		_, err := fmt.Fprintf(cmd.OutOrStdout(), "[dry-run] Would add %d dependents to task %s\n", len(dependents), taskGID)
		return err
	}

	_, err := runtimeClient(cmd).Post(cmd.Context(), "/tasks/"+taskGID+"/addDependents", api.RequestBody{
		Data: map[string][]string{"dependents": dependents},
	})
	if err != nil {
		return err
	}

	output := dependencyOutput{
		Action:     "added",
		DryRun:     false,
		Task:       &gidRef{GID: taskGID},
		Dependents: dependents,
	}
	if runtimeOutputJSON(cmd) {
		encoder := json.NewEncoder(cmd.OutOrStdout())
		encoder.SetIndent("", "  ")
		return encoder.Encode(output)
	}
	_, err = fmt.Fprintf(cmd.OutOrStdout(), "Added %d dependents\n", len(dependents))
	return err
}

func runTaskDependentRemove(cmd *cobra.Command, args []string) error {
	taskGID := args[0]
	dependents, _ := cmd.Flags().GetStringSlice("dependent")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	if err := validateGID(taskGID, "gid"); err != nil {
		return err
	}
	if len(dependents) == 0 {
		return fmt.Errorf("--dependent is required")
	}

	// Validate all GIDs
	dependents = normalizeDependencyGIDs(dependents)
	for _, gid := range dependents {
		if err := validateGID(gid, "dependent gid"); err != nil {
			return err
		}
	}
	if len(dependents) == 0 {
		return fmt.Errorf("--dependent is required")
	}

	if dryRun {
		// Validate task exists
		if _, err := fetchTaskWithFields(cmd.Context(), taskGID, "gid"); err != nil {
			return err
		}
		output := dependencyOutput{
			Action:     "removed",
			DryRun:     true,
			Task:       &gidRef{GID: taskGID},
			Dependents: dependents,
		}
		if runtimeOutputJSON(cmd) {
			encoder := json.NewEncoder(cmd.OutOrStdout())
			encoder.SetIndent("", "  ")
			return encoder.Encode(output)
		}
		_, err := fmt.Fprintf(cmd.OutOrStdout(), "[dry-run] Would remove %d dependents from task %s\n", len(dependents), taskGID)
		return err
	}

	_, err := runtimeClient(cmd).Post(cmd.Context(), "/tasks/"+taskGID+"/removeDependents", api.RequestBody{
		Data: map[string][]string{"dependents": dependents},
	})
	if err != nil {
		return err
	}

	output := dependencyOutput{
		Action:     "removed",
		DryRun:     false,
		Task:       &gidRef{GID: taskGID},
		Dependents: dependents,
	}
	if runtimeOutputJSON(cmd) {
		encoder := json.NewEncoder(cmd.OutOrStdout())
		encoder.SetIndent("", "  ")
		return encoder.Encode(output)
	}
	_, err = fmt.Fprintf(cmd.OutOrStdout(), "Removed %d dependents\n", len(dependents))
	return err
}

func normalizeDependencyGIDs(values []string) []string {
	clean := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		clean = append(clean, value)
	}
	return clean
}
