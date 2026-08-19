// Package cmd implements goal commands: list, view, create, update, delete, metric set.
package cmd

import (
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/vincentsch/asana-cli/internal/api"
)

var goalCmd = &cobra.Command{
	Use:   "goal",
	Short: "Manage goals",
	Long: `Manage Asana goals (premium feature).

Goals are high-level objectives that can be tracked with metrics. They can be
assigned to teams, have due dates, and track progress numerically.

See also: portfolio, project`,
}

var goalListCmd = &cobra.Command{
	Use:   "list",
	Short: "List goals",
	Long: `List goals in a workspace, optionally filtered by team or time period.

See also: goal view, goal create`,
	Example: `  # List all goals
  asana goal list -w "My Workspace"

  # List goals for a specific team
  asana goal list -w "My Workspace" --team "Engineering"

  # Limit output
  asana goal list -w "My Workspace" --limit 10

  # Output as JSON
  asana goal list -w "My Workspace" --json`,
	Args: cobra.NoArgs,
	RunE: runGoalList,
}

var goalViewCmd = &cobra.Command{
	Use:   "view <goal-gid>",
	Short: "View goal details",
	Long: `View detailed information about a goal including status, owner, and metric progress.

See also: goal list, goal metric set`,
	Example: `  # View goal details
  asana goal view 1234567890123456

  # Output as JSON
  asana goal view 1234567890123456 --json`,
	Args: cobra.ExactArgs(1),
	RunE: runGoalView,
}

var goalCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a goal",
	Long: `Create a new goal in a workspace (premium feature).

See also: goal list, goal update, goal metric set`,
	Example: `  # Create a simple goal
  asana goal create "Increase revenue 20%" -w "My Workspace"

  # Create with owner and dates
  asana goal create "Launch v2.0" -w "My Workspace" --owner me --due-on 2024-12-31

  # Create for a team
  asana goal create "Improve NPS" -w "My Workspace" --team "Customer Success"

  # Preview without creating
  asana goal create "Test Goal" -w "My Workspace" --dry-run`,
	Args: cobra.ExactArgs(1),
	RunE: runGoalCreate,
}

var goalUpdateCmd = &cobra.Command{
	Use:   "update <goal-gid>",
	Short: "Update a goal",
	Long: `Update a goal's name, status, dates, or notes.

Status values: green, yellow, red, on_track, at_risk, off_track.

See also: goal view, goal metric set`,
	Example: `  # Update goal status
  asana goal update 1234567890123456 --status on_track

  # Update name and notes
  asana goal update 1234567890123456 --name "New Goal Name" --notes "Updated description"

  # Update due date
  asana goal update 1234567890123456 --due-on 2025-01-31

  # Preview changes
  asana goal update 1234567890123456 --status at_risk --dry-run`,
	Args: cobra.ExactArgs(1),
	RunE: runGoalUpdate,
}

var goalDeleteCmd = &cobra.Command{
	Use:   "delete <goal-gid>",
	Short: "Delete a goal",
	Long: `Permanently delete a goal.

This action cannot be undone.

See also: goal list`,
	Example: `  # Delete a goal
  asana goal delete 1234567890123456 --confirm

  # Preview deletion
  asana goal delete 1234567890123456 --dry-run`,
	Args: cobra.ExactArgs(1),
	RunE: runGoalDelete,
}

var goalMetricCmd = &cobra.Command{
	Use:   "metric",
	Short: "Manage goal metrics",
	Long: `Set or update numeric metrics for tracking goal progress.

See also: goal view, goal update`,
}

var goalMetricSetCmd = &cobra.Command{
	Use:   "set <goal-gid>",
	Short: "Set or update goal metric",
	Long: `Set the current value of a goal's metric, or create a new metric.

Use --current-value to update progress. Add --target-value to create or replace
the entire metric definition.

See also: goal view`,
	Example: `  # Update current progress
  asana goal metric set 1234567890123456 --current-value 75

  # Create a percentage metric
  asana goal metric set 1234567890123456 --current-value 0 --target-value 100 --unit percentage

  # Create a metric with precision
  asana goal metric set 1234567890123456 --current-value 0 --target-value 1000000 --unit currency --precision 2

  # Preview without changing
  asana goal metric set 1234567890123456 --current-value 50 --dry-run`,
	Args: cobra.ExactArgs(1),
	RunE: runGoalMetricSet,
}

func init() {
	goalListCmd.Flags().StringP("workspace", "w", "", "Workspace name")
	goalListCmd.Flags().String("workspace-gid", "", "Workspace GID")
	goalListCmd.Flags().String("team", "", "Team name")
	goalListCmd.Flags().String("team-gid", "", "Team GID")
	goalListCmd.Flags().String("time-period", "", "Time period GID")
	goalListCmd.Flags().Int("limit", 0, "Limit number of goals in output")

	goalCreateCmd.Flags().StringP("workspace", "w", "", "Workspace name")
	goalCreateCmd.Flags().String("workspace-gid", "", "Workspace GID")
	goalCreateCmd.Flags().String("team", "", "Team name")
	goalCreateCmd.Flags().String("team-gid", "", "Team GID")
	goalCreateCmd.Flags().String("owner", "", "Owner (me, GID, or email)")
	goalCreateCmd.Flags().String("due-on", "", "Due date (YYYY-MM-DD)")
	goalCreateCmd.Flags().String("start-on", "", "Start date (YYYY-MM-DD)")
	goalCreateCmd.Flags().String("notes", "", "Goal notes")

	goalUpdateCmd.Flags().String("name", "", "New goal name")
	goalUpdateCmd.Flags().String("notes", "", "Goal notes")
	goalUpdateCmd.Flags().String("status", "", "Goal status (green, yellow, red, on_track, at_risk, off_track)")
	goalUpdateCmd.Flags().String("due-on", "", "Due date (YYYY-MM-DD)")
	goalUpdateCmd.Flags().String("start-on", "", "Start date (YYYY-MM-DD)")

	goalMetricSetCmd.Flags().Float64("current-value", 0, "Current metric value (required)")
	goalMetricSetCmd.Flags().Float64("target-value", 0, "Target metric value (creates/replaces metric)")
	goalMetricSetCmd.Flags().Float64("initial-value", 0, "Initial metric value")
	goalMetricSetCmd.Flags().String("unit", "", "Metric unit (none, currency, percentage)")
	goalMetricSetCmd.Flags().Int("precision", 0, "Decimal precision")

	goalMetricCmd.AddCommand(goalMetricSetCmd)

	goalCmd.AddCommand(goalListCmd)
	goalCmd.AddCommand(goalViewCmd)
	goalCmd.AddCommand(goalCreateCmd)
	goalCmd.AddCommand(goalUpdateCmd)
	goalCmd.AddCommand(goalDeleteCmd)
	goalCmd.AddCommand(goalMetricCmd)
}

type goalListItem struct {
	GID    string `json:"gid"`
	Name   string `json:"name"`
	Status string `json:"status,omitempty"`
	Owner  string `json:"owner,omitempty"`
}

type goalViewOutput struct {
	GID       string               `json:"gid"`
	Name      string               `json:"name"`
	Notes     string               `json:"notes,omitempty"`
	DueOn     string               `json:"due_on,omitempty"`
	StartOn   string               `json:"start_on,omitempty"`
	Status    string               `json:"status,omitempty"`
	Owner     *userCompactOut      `json:"owner,omitempty"`
	Metric    *goalMetricOut       `json:"metric,omitempty"`
	Team      *teamCompactOut      `json:"team,omitempty"`
	Workspace *workspaceCompactOut `json:"workspace,omitempty"`
}

type userCompactOut struct {
	GID  string `json:"gid"`
	Name string `json:"name"`
}

type teamCompactOut struct {
	GID  string `json:"gid"`
	Name string `json:"name"`
}

type goalMetricOut struct {
	CurrentNumberValue float64 `json:"current_number_value"`
	TargetNumberValue  float64 `json:"target_number_value"`
	InitialNumberValue float64 `json:"initial_number_value,omitempty"`
	Unit               string  `json:"unit,omitempty"`
	Precision          int     `json:"precision,omitempty"`
}

type goalWriteOutput struct {
	Action string  `json:"action"`
	DryRun bool    `json:"dry_run"`
	Goal   *gidRef `json:"goal,omitempty"`
	Name   string  `json:"name,omitempty"`
}

type goalMetricOutput struct {
	Action       string  `json:"action"`
	DryRun       bool    `json:"dry_run"`
	Goal         *gidRef `json:"goal"`
	CurrentValue float64 `json:"current_value"`
	TargetValue  float64 `json:"target_value,omitempty"`
}

func runGoalList(cmd *cobra.Command, args []string) error {
	workspaceName, _ := cmd.Flags().GetString("workspace")
	workspaceGID, _ := cmd.Flags().GetString("workspace-gid")
	teamName, _ := cmd.Flags().GetString("team")
	teamGID, _ := cmd.Flags().GetString("team-gid")
	timePeriod, _ := cmd.Flags().GetString("time-period")
	limit, _ := cmd.Flags().GetInt("limit")

	if workspaceName != "" && workspaceGID != "" {
		return fmt.Errorf("use only one of --workspace or --workspace-gid")
	}
	if teamName != "" && teamGID != "" {
		return fmt.Errorf("use only one of --team or --team-gid")
	}
	if cmd.Flags().Changed("limit") && limit < 1 {
		return fmt.Errorf("--limit must be >= 1")
	}

	resolvedWorkspace, err := resolveWorkspaceFromFlags(cmd, workspaceName, workspaceGID)
	if err != nil {
		return err
	}

	// Resolve team if specified
	resolvedTeam := teamGID
	if teamName != "" {
		gid, err := resolveTeamWithPrompt(cmd.Context(), runtimeClient(cmd), resolvedWorkspace, teamName, runtimeNoPrompt(cmd) || runtimeOutputJSON(cmd))
		if err != nil {
			return err
		}
		resolvedTeam = gid
	}

	query := url.Values{}
	query.Set("workspace", resolvedWorkspace)
	query.Set("opt_fields", "gid,name,status,owner.gid,owner.name")
	if resolvedTeam != "" {
		query.Set("team", resolvedTeam)
	}
	if timePeriod != "" {
		query.Set("time_periods", timePeriod)
	}

	goals, err := api.Paginate[api.Goal](cmd.Context(), runtimeClient(cmd), "/goals", query)
	if err != nil {
		return formatPremiumError(err, "goals")
	}

	// Sort by name (case-insensitive), GID tiebreaker
	sort.Slice(goals, func(i, j int) bool {
		cmp := strings.Compare(strings.ToLower(goals[i].Name), strings.ToLower(goals[j].Name))
		if cmp != 0 {
			return cmp < 0
		}
		return goals[i].GID < goals[j].GID
	})

	// Convert to output format
	output := make([]goalListItem, len(goals))
	for i, g := range goals {
		item := goalListItem{
			GID:    g.GID,
			Name:   g.Name,
			Status: g.Status,
		}
		if g.Owner != nil {
			item.Owner = g.Owner.Name
		}
		output[i] = item
	}

	if limit > 0 && len(output) > limit {
		output = output[:limit]
	}

	if runtimeOutputJSON(cmd) {
		encoder := json.NewEncoder(cmd.OutOrStdout())
		encoder.SetIndent("", "  ")
		return encoder.Encode(output)
	}

	if len(output) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No goals found")
		return nil
	}

	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tSTATUS\tOWNER\tGID")
	for _, g := range output {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", g.Name, g.Status, g.Owner, g.GID)
	}
	return w.Flush()
}

func runGoalView(cmd *cobra.Command, args []string) error {
	goalGID := args[0]

	if err := validateGID(goalGID, "goal gid"); err != nil {
		return err
	}

	query := url.Values{}
	query.Set("opt_fields", "gid,name,notes,due_on,start_on,status,owner.gid,owner.name,metric,team.gid,team.name,workspace.gid,workspace.name")

	payload, err := runtimeClient(cmd).Get(cmd.Context(), "/goals/"+goalGID, query)
	if err != nil {
		return formatPremiumError(err, "goals")
	}

	var response api.Response[api.Goal]
	if err := json.Unmarshal(payload, &response); err != nil {
		return &api.ResponseError{Err: err}
	}

	g := response.Data
	output := goalViewOutput{
		GID:     g.GID,
		Name:    g.Name,
		Notes:   g.Notes,
		DueOn:   g.DueOn,
		StartOn: g.StartOn,
		Status:  g.Status,
	}
	if g.Owner != nil {
		output.Owner = &userCompactOut{GID: g.Owner.GID, Name: g.Owner.Name}
	}
	if g.Metric != nil {
		output.Metric = &goalMetricOut{
			CurrentNumberValue: g.Metric.CurrentNumberValue,
			TargetNumberValue:  g.Metric.TargetNumberValue,
			InitialNumberValue: g.Metric.InitialNumberValue,
			Unit:               g.Metric.Unit,
			Precision:          g.Metric.Precision,
		}
	}
	if g.Team != nil {
		output.Team = &teamCompactOut{GID: g.Team.GID, Name: g.Team.Name}
	}
	if g.Workspace != nil {
		output.Workspace = &workspaceCompactOut{GID: g.Workspace.GID, Name: g.Workspace.Name}
	}

	if runtimeOutputJSON(cmd) {
		encoder := json.NewEncoder(cmd.OutOrStdout())
		encoder.SetIndent("", "  ")
		return encoder.Encode(output)
	}

	// Text output
	fmt.Fprintf(cmd.OutOrStdout(), "GID:     %s\n", g.GID)
	fmt.Fprintf(cmd.OutOrStdout(), "Name:    %s\n", g.Name)
	if g.Status != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "Status:  %s\n", g.Status)
	}
	if g.Owner != nil {
		fmt.Fprintf(cmd.OutOrStdout(), "Owner:   %s (%s)\n", g.Owner.Name, g.Owner.GID)
	}
	if g.DueOn != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "Due:     %s\n", g.DueOn)
	}
	if g.StartOn != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "Start:   %s\n", g.StartOn)
	}
	if g.Notes != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "Notes:   %s\n", g.Notes)
	}
	if g.Metric != nil {
		fmt.Fprintf(cmd.OutOrStdout(), "Metric:  %.2f / %.2f\n", g.Metric.CurrentNumberValue, g.Metric.TargetNumberValue)
	}
	if g.Team != nil {
		fmt.Fprintf(cmd.OutOrStdout(), "Team:    %s (%s)\n", g.Team.Name, g.Team.GID)
	}

	return nil
}

func runGoalCreate(cmd *cobra.Command, args []string) error {
	name := args[0]
	workspaceName, _ := cmd.Flags().GetString("workspace")
	workspaceGID, _ := cmd.Flags().GetString("workspace-gid")
	teamName, _ := cmd.Flags().GetString("team")
	teamGID, _ := cmd.Flags().GetString("team-gid")
	owner, _ := cmd.Flags().GetString("owner")
	dueOn, _ := cmd.Flags().GetString("due-on")
	startOn, _ := cmd.Flags().GetString("start-on")
	notes, _ := cmd.Flags().GetString("notes")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	if workspaceName != "" && workspaceGID != "" {
		return fmt.Errorf("use only one of --workspace or --workspace-gid")
	}
	if teamName != "" && teamGID != "" {
		return fmt.Errorf("use only one of --team or --team-gid")
	}

	resolvedWorkspace, err := resolveWorkspaceFromFlags(cmd, workspaceName, workspaceGID)
	if err != nil {
		return err
	}

	// Resolve team if specified
	resolvedTeam := teamGID
	if teamName != "" {
		gid, err := resolveTeamWithPrompt(cmd.Context(), runtimeClient(cmd), resolvedWorkspace, teamName, runtimeNoPrompt(cmd) || runtimeOutputJSON(cmd))
		if err != nil {
			return err
		}
		resolvedTeam = gid
	}

	// Resolve owner
	var resolvedOwner string
	if owner != "" {
		users, err := resolveUserIdentifiers(cmd.Context(), resolvedWorkspace, []string{owner})
		if err != nil {
			return err
		}
		resolvedOwner = users[0]
	}

	if dryRun {
		output := goalWriteOutput{
			Action: "created",
			DryRun: true,
			Name:   name,
		}
		if runtimeOutputJSON(cmd) {
			encoder := json.NewEncoder(cmd.OutOrStdout())
			encoder.SetIndent("", "  ")
			return encoder.Encode(output)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "[dry-run] Would create goal %q in workspace %s\n", name, resolvedWorkspace)
		return nil
	}

	data := map[string]any{
		"workspace": resolvedWorkspace,
		"name":      name,
	}
	if resolvedTeam != "" {
		data["team"] = resolvedTeam
	}
	if resolvedOwner != "" {
		data["owner"] = resolvedOwner
	}
	if dueOn != "" {
		data["due_on"] = dueOn
	}
	if startOn != "" {
		data["start_on"] = startOn
	}
	if notes != "" {
		data["notes"] = notes
	}

	payload, err := runtimeClient(cmd).Post(cmd.Context(), "/goals", api.RequestBody{
		Data: data,
		Options: &api.RequestOptions{
			Fields: []string{"gid", "name"},
		},
	})
	if err != nil {
		return formatPremiumError(err, "goals")
	}

	var response api.Response[api.Goal]
	if err := json.Unmarshal(payload, &response); err != nil {
		return &api.ResponseError{Err: err}
	}

	g := response.Data
	output := goalWriteOutput{
		Action: "created",
		DryRun: false,
		Goal:   &gidRef{GID: g.GID},
		Name:   g.Name,
	}

	if runtimeOutputJSON(cmd) {
		encoder := json.NewEncoder(cmd.OutOrStdout())
		encoder.SetIndent("", "  ")
		return encoder.Encode(output)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Created goal: %s (%s)\n", g.Name, g.GID)
	return nil
}

func runGoalUpdate(cmd *cobra.Command, args []string) error {
	goalGID := args[0]
	name, _ := cmd.Flags().GetString("name")
	notes, _ := cmd.Flags().GetString("notes")
	status, _ := cmd.Flags().GetString("status")
	dueOn, _ := cmd.Flags().GetString("due-on")
	startOn, _ := cmd.Flags().GetString("start-on")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	if err := validateGID(goalGID, "goal gid"); err != nil {
		return err
	}

	if !cmd.Flags().Changed("name") && !cmd.Flags().Changed("notes") && !cmd.Flags().Changed("status") && !cmd.Flags().Changed("due-on") && !cmd.Flags().Changed("start-on") {
		return fmt.Errorf("at least one of --name, --notes, --status, --due-on, or --start-on is required")
	}

	if dryRun {
		// Validate goal exists
		query := url.Values{}
		query.Set("opt_fields", "gid")
		if _, err := runtimeClient(cmd).Get(cmd.Context(), "/goals/"+goalGID, query); err != nil {
			return formatPremiumError(err, "goals")
		}

		output := goalWriteOutput{
			Action: "updated",
			DryRun: true,
			Goal:   &gidRef{GID: goalGID},
			Name:   name,
		}
		if runtimeOutputJSON(cmd) {
			encoder := json.NewEncoder(cmd.OutOrStdout())
			encoder.SetIndent("", "  ")
			return encoder.Encode(output)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "[dry-run] Would update goal %s\n", goalGID)
		return nil
	}

	data := map[string]any{}
	if cmd.Flags().Changed("name") {
		data["name"] = name
	}
	if cmd.Flags().Changed("notes") {
		data["notes"] = notes
	}
	if cmd.Flags().Changed("status") {
		data["status"] = status
	}
	if cmd.Flags().Changed("due-on") {
		data["due_on"] = dueOn
	}
	if cmd.Flags().Changed("start-on") {
		data["start_on"] = startOn
	}

	payload, err := runtimeClient(cmd).Put(cmd.Context(), "/goals/"+goalGID, api.RequestBody{
		Data: data,
		Options: &api.RequestOptions{
			Fields: []string{"gid", "name"},
		},
	})
	if err != nil {
		return formatPremiumError(err, "goals")
	}

	var response api.Response[api.Goal]
	if err := json.Unmarshal(payload, &response); err != nil {
		return &api.ResponseError{Err: err}
	}

	g := response.Data
	output := goalWriteOutput{
		Action: "updated",
		DryRun: false,
		Goal:   &gidRef{GID: g.GID},
		Name:   g.Name,
	}

	if runtimeOutputJSON(cmd) {
		encoder := json.NewEncoder(cmd.OutOrStdout())
		encoder.SetIndent("", "  ")
		return encoder.Encode(output)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Updated goal: %s (%s)\n", g.Name, g.GID)
	return nil
}

func runGoalDelete(cmd *cobra.Command, args []string) error {
	goalGID := args[0]
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	if err := validateGID(goalGID, "goal gid"); err != nil {
		return err
	}

	if dryRun {
		// Validate goal exists
		query := url.Values{}
		query.Set("opt_fields", "gid,name")
		payload, err := runtimeClient(cmd).Get(cmd.Context(), "/goals/"+goalGID, query)
		if err != nil {
			return formatPremiumError(err, "goals")
		}

		var response api.Response[api.Goal]
		if err := json.Unmarshal(payload, &response); err != nil {
			return &api.ResponseError{Err: err}
		}

		output := goalWriteOutput{
			Action: "deleted",
			DryRun: true,
			Goal:   &gidRef{GID: goalGID},
			Name:   response.Data.Name,
		}
		if runtimeOutputJSON(cmd) {
			encoder := json.NewEncoder(cmd.OutOrStdout())
			encoder.SetIndent("", "  ")
			return encoder.Encode(output)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "[dry-run] Would delete goal %s (%s)\n", response.Data.Name, goalGID)
		return nil
	}

	if err := runtimeClient(cmd).Delete(cmd.Context(), "/goals/"+goalGID); err != nil {
		return formatPremiumError(err, "goals")
	}

	output := goalWriteOutput{
		Action: "deleted",
		DryRun: false,
		Goal:   &gidRef{GID: goalGID},
	}

	if runtimeOutputJSON(cmd) {
		encoder := json.NewEncoder(cmd.OutOrStdout())
		encoder.SetIndent("", "  ")
		return encoder.Encode(output)
	}

	fmt.Fprintln(cmd.OutOrStdout(), "Goal deleted")
	return nil
}

func runGoalMetricSet(cmd *cobra.Command, args []string) error {
	goalGID := args[0]
	currentValue, _ := cmd.Flags().GetFloat64("current-value")
	targetValue, _ := cmd.Flags().GetFloat64("target-value")
	initialValue, _ := cmd.Flags().GetFloat64("initial-value")
	unit, _ := cmd.Flags().GetString("unit")
	precision, _ := cmd.Flags().GetInt("precision")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	if err := validateGID(goalGID, "goal gid"); err != nil {
		return err
	}

	if !cmd.Flags().Changed("current-value") {
		return fmt.Errorf("--current-value is required")
	}

	// Determine which endpoint to use
	useSetMetric := cmd.Flags().Changed("target-value")
	if !useSetMetric {
		if cmd.Flags().Changed("initial-value") || cmd.Flags().Changed("precision") || unit != "" {
			return fmt.Errorf("--unit, --precision, and --initial-value require --target-value")
		}
	}

	if dryRun {
		// Validate goal exists
		query := url.Values{}
		query.Set("opt_fields", "gid")
		if _, err := runtimeClient(cmd).Get(cmd.Context(), "/goals/"+goalGID, query); err != nil {
			return formatPremiumError(err, "goals")
		}

		output := goalMetricOutput{
			Action:       "set_metric",
			DryRun:       true,
			Goal:         &gidRef{GID: goalGID},
			CurrentValue: currentValue,
		}
		if useSetMetric {
			output.TargetValue = targetValue
		}
		if runtimeOutputJSON(cmd) {
			encoder := json.NewEncoder(cmd.OutOrStdout())
			encoder.SetIndent("", "  ")
			return encoder.Encode(output)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "[dry-run] Would set metric on goal %s (current: %.2f)\n", goalGID, currentValue)
		return nil
	}

	var err error
	if useSetMetric {
		// Create or replace the entire metric
		data := map[string]any{
			"current_number_value": currentValue,
			"target_number_value":  targetValue,
		}
		if cmd.Flags().Changed("initial-value") {
			data["initial_number_value"] = initialValue
		}
		if unit != "" {
			data["unit"] = unit
		}
		if cmd.Flags().Changed("precision") {
			data["precision"] = precision
		}

		_, err = runtimeClient(cmd).Post(cmd.Context(), "/goals/"+goalGID+"/setMetric", api.RequestBody{
			Data: data,
		})
	} else {
		// Just update the current value
		_, err = runtimeClient(cmd).Post(cmd.Context(), "/goals/"+goalGID+"/setMetricCurrentValue", api.RequestBody{
			Data: map[string]any{
				"current_number_value": currentValue,
			},
		})
	}

	if err != nil {
		return formatPremiumError(err, "goals")
	}

	output := goalMetricOutput{
		Action:       "set_metric",
		DryRun:       false,
		Goal:         &gidRef{GID: goalGID},
		CurrentValue: currentValue,
	}
	if useSetMetric {
		output.TargetValue = targetValue
	}

	if runtimeOutputJSON(cmd) {
		encoder := json.NewEncoder(cmd.OutOrStdout())
		encoder.SetIndent("", "  ")
		return encoder.Encode(output)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Goal metric updated (current: %.2f)\n", currentValue)
	return nil
}
