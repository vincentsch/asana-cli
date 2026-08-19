// Package cmd implements tag commands: list, view, create, update, delete, tasks.
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

var tagCmd = &cobra.Command{
	Use:   "tag",
	Short: "Manage tags",
	Long: `Manage Asana tags.

Tags are labels that can be applied to tasks across projects. Unlike projects,
tags are flat (no hierarchy) and workspace-scoped. Use tags for cross-cutting
categorization like "urgent", "blocked", or "needs-review".

See also: task tag add, task tag remove`,
}

var tagListCmd = &cobra.Command{
	Use:   "list",
	Short: "List tags in a workspace",
	Long: `List all tags in a workspace, sorted alphabetically.

See also: tag view, tag create, tag tasks`,
	Example: `  # List all tags
  asana tag list -w "My Workspace"

  # Limit output
  asana tag list -w "My Workspace" --limit 20

  # Output as JSON
  asana tag list -w "My Workspace" --json`,
	Args: cobra.NoArgs,
	RunE: runTagList,
}

var tagViewCmd = &cobra.Command{
	Use:   "view <tag-gid>",
	Short: "View tag details",
	Long: `View detailed information about a tag including name, color, notes, and workspace.

See also: tag list, tag tasks`,
	Example: `  # View tag details
  asana tag view 1234567890123456

  # Output as JSON
  asana tag view 1234567890123456 --json`,
	Args: cobra.ExactArgs(1),
	RunE: runTagView,
}

var tagCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new tag",
	Long: `Create a new tag in a workspace.

Available colors: dark-pink, dark-green, dark-blue, dark-red, dark-teal,
dark-brown, dark-orange, dark-purple, dark-warm-gray, light-pink, light-green,
light-blue, light-red, light-teal, light-brown, light-orange, light-purple,
light-warm-gray, none.

See also: tag list, task tag add`,
	Example: `  # Create a simple tag
  asana tag create "urgent" -w "My Workspace"

  # Create with color
  asana tag create "blocked" -w "My Workspace" --color dark-red

  # Create with notes
  asana tag create "needs-review" -w "My Workspace" --notes "Requires code review"

  # Preview without creating
  asana tag create "test" -w "My Workspace" --dry-run`,
	Args: cobra.ExactArgs(1),
	RunE: runTagCreate,
}

var tagUpdateCmd = &cobra.Command{
	Use:   "update <tag-gid>",
	Short: "Update a tag",
	Long:  `Update a tag's name, color, or notes.`,
	Example: `  # Rename a tag
  asana tag update 1234567890123456 --name "critical"

  # Change color
  asana tag update 1234567890123456 --color dark-orange

  # Preview changes
  asana tag update 1234567890123456 --name "new-name" --dry-run`,
	Args: cobra.ExactArgs(1),
	RunE: runTagUpdate,
}

var tagDeleteCmd = &cobra.Command{
	Use:   "delete <tag-gid>",
	Short: "Delete a tag",
	Long: `Permanently delete a tag.

This removes the tag from all tasks. This action cannot be undone.`,
	Example: `  # Delete a tag
  asana tag delete 1234567890123456 --confirm

  # Preview deletion
  asana tag delete 1234567890123456 --dry-run`,
	Args: cobra.ExactArgs(1),
	RunE: runTagDelete,
}

var tagTasksCmd = &cobra.Command{
	Use:   "tasks <tag-gid>",
	Short: "List tasks with this tag",
	Long: `List all tasks that have this tag applied.

See also: tag list, task tag add, task list`,
	Example: `  # List tasks with a tag
  asana tag tasks 1234567890123456

  # Limit output
  asana tag tasks 1234567890123456 --limit 10

  # Output as JSON
  asana tag tasks 1234567890123456 --json`,
	Args: cobra.ExactArgs(1),
	RunE: runTagTasks,
}

// Valid tag colors
var validTagColors = []string{
	"dark-pink", "dark-green", "dark-blue", "dark-red", "dark-teal",
	"dark-brown", "dark-orange", "dark-purple", "dark-warm-gray",
	"light-pink", "light-green", "light-blue", "light-red", "light-teal",
	"light-brown", "light-orange", "light-purple", "light-warm-gray",
	"none",
}

func init() {
	tagListCmd.Flags().StringP("workspace", "w", "", "Workspace name")
	tagListCmd.Flags().String("workspace-gid", "", "Workspace GID")
	tagListCmd.Flags().Int("limit", 0, "Limit number of tags in output")

	tagCreateCmd.Flags().StringP("workspace", "w", "", "Workspace name")
	tagCreateCmd.Flags().String("workspace-gid", "", "Workspace GID")
	tagCreateCmd.Flags().String("color", "", "Tag color")
	tagCreateCmd.Flags().String("notes", "", "Tag notes/description")

	tagUpdateCmd.Flags().String("name", "", "New tag name")
	tagUpdateCmd.Flags().String("color", "", "Tag color")
	tagUpdateCmd.Flags().String("notes", "", "Tag notes/description")

	tagTasksCmd.Flags().Int("limit", 0, "Limit number of tasks in output")

	tagCmd.AddCommand(tagListCmd)
	tagCmd.AddCommand(tagViewCmd)
	tagCmd.AddCommand(tagCreateCmd)
	tagCmd.AddCommand(tagUpdateCmd)
	tagCmd.AddCommand(tagDeleteCmd)
	tagCmd.AddCommand(tagTasksCmd)
}

type tagListItem struct {
	GID   string `json:"gid"`
	Name  string `json:"name"`
	Color string `json:"color,omitempty"`
}

type tagViewOutput struct {
	GID       string               `json:"gid"`
	Name      string               `json:"name"`
	Color     string               `json:"color,omitempty"`
	Notes     string               `json:"notes,omitempty"`
	Workspace *workspaceCompactOut `json:"workspace,omitempty"`
	CreatedAt string               `json:"created_at,omitempty"`
}

type tagWriteOutput struct {
	Action string  `json:"action"`
	DryRun bool    `json:"dry_run"`
	Tag    *gidRef `json:"tag,omitempty"`
	Name   string  `json:"name,omitempty"`
	Color  string  `json:"color,omitempty"`
	Notes  string  `json:"notes,omitempty"`
}

type tagTaskItem struct {
	GID       string `json:"gid"`
	Name      string `json:"name"`
	Completed bool   `json:"completed"`
}

func validateTagColor(color string) error {
	if color == "" {
		return nil
	}
	for _, valid := range validTagColors {
		if color == valid {
			return nil
		}
	}
	return fmt.Errorf("invalid color %q: must be one of %v", color, validTagColors)
}

func runTagList(cmd *cobra.Command, args []string) error {
	workspaceName, _ := cmd.Flags().GetString("workspace")
	workspaceGID, _ := cmd.Flags().GetString("workspace-gid")
	limit, _ := cmd.Flags().GetInt("limit")

	if workspaceName != "" && workspaceGID != "" {
		return fmt.Errorf("use only one of --workspace or --workspace-gid")
	}
	if cmd.Flags().Changed("limit") && limit < 1 {
		return fmt.Errorf("--limit must be >= 1")
	}

	resolvedWorkspace, err := resolveWorkspaceFromFlags(cmd, workspaceName, workspaceGID)
	if err != nil {
		return err
	}

	query := url.Values{}
	query.Set("opt_fields", "gid,name,color")

	tags, err := api.Paginate[api.TagDetail](cmd.Context(), runtimeClient(cmd), "/workspaces/"+resolvedWorkspace+"/tags", query)
	if err != nil {
		return err
	}

	// Sort by name (case-insensitive), GID tiebreaker
	sort.Slice(tags, func(i, j int) bool {
		cmp := strings.Compare(strings.ToLower(tags[i].Name), strings.ToLower(tags[j].Name))
		if cmp != 0 {
			return cmp < 0
		}
		return tags[i].GID < tags[j].GID
	})

	// Convert to output format
	output := make([]tagListItem, len(tags))
	for i, t := range tags {
		output[i] = tagListItem{
			GID:   t.GID,
			Name:  t.Name,
			Color: t.Color,
		}
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
		fmt.Fprintln(cmd.OutOrStdout(), "No tags found")
		return nil
	}

	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tCOLOR\tGID")
	for _, t := range output {
		fmt.Fprintf(w, "%s\t%s\t%s\n", t.Name, t.Color, t.GID)
	}
	return w.Flush()
}

func runTagView(cmd *cobra.Command, args []string) error {
	tagGID := args[0]

	if err := validateGID(tagGID, "tag gid"); err != nil {
		return err
	}

	query := url.Values{}
	query.Set("opt_fields", "gid,name,color,notes,workspace.gid,workspace.name,created_at")

	payload, err := runtimeClient(cmd).Get(cmd.Context(), "/tags/"+tagGID, query)
	if err != nil {
		return err
	}

	var response api.Response[api.TagDetail]
	if err := json.Unmarshal(payload, &response); err != nil {
		return &api.ResponseError{Err: err}
	}

	tag := response.Data
	output := tagViewOutput{
		GID:       tag.GID,
		Name:      tag.Name,
		Color:     tag.Color,
		Notes:     tag.Notes,
		CreatedAt: tag.CreatedAt,
	}
	if tag.Workspace != nil {
		output.Workspace = &workspaceCompactOut{
			GID:  tag.Workspace.GID,
			Name: tag.Workspace.Name,
		}
	}

	if runtimeOutputJSON(cmd) {
		encoder := json.NewEncoder(cmd.OutOrStdout())
		encoder.SetIndent("", "  ")
		return encoder.Encode(output)
	}

	// Text output
	fmt.Fprintf(cmd.OutOrStdout(), "GID:       %s\n", tag.GID)
	fmt.Fprintf(cmd.OutOrStdout(), "Name:      %s\n", tag.Name)
	if tag.Color != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "Color:     %s\n", tag.Color)
	}
	if tag.Notes != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "Notes:     %s\n", tag.Notes)
	}
	if tag.Workspace != nil {
		fmt.Fprintf(cmd.OutOrStdout(), "Workspace: %s (%s)\n", tag.Workspace.Name, tag.Workspace.GID)
	}
	if tag.CreatedAt != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "Created:   %s\n", tag.CreatedAt)
	}

	return nil
}

func runTagCreate(cmd *cobra.Command, args []string) error {
	name := args[0]
	workspaceName, _ := cmd.Flags().GetString("workspace")
	workspaceGID, _ := cmd.Flags().GetString("workspace-gid")
	color, _ := cmd.Flags().GetString("color")
	notes, _ := cmd.Flags().GetString("notes")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	if workspaceName != "" && workspaceGID != "" {
		return fmt.Errorf("use only one of --workspace or --workspace-gid")
	}

	if err := validateTagColor(color); err != nil {
		return err
	}

	resolvedWorkspace, err := resolveWorkspaceFromFlags(cmd, workspaceName, workspaceGID)
	if err != nil {
		return err
	}

	if dryRun {
		output := tagWriteOutput{
			Action: "created",
			DryRun: true,
			Name:   name,
			Color:  color,
			Notes:  notes,
		}
		if runtimeOutputJSON(cmd) {
			encoder := json.NewEncoder(cmd.OutOrStdout())
			encoder.SetIndent("", "  ")
			return encoder.Encode(output)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "[dry-run] Would create tag %q in workspace %s\n", name, resolvedWorkspace)
		return nil
	}

	data := map[string]any{"name": name}
	if color != "" {
		data["color"] = color
	}
	if notes != "" {
		data["notes"] = notes
	}

	payload, err := runtimeClient(cmd).Post(cmd.Context(), "/workspaces/"+resolvedWorkspace+"/tags", api.RequestBody{
		Data: data,
		Options: &api.RequestOptions{
			Fields: []string{"gid", "name"},
		},
	})
	if err != nil {
		return err
	}

	var response api.Response[api.Tag]
	if err := json.Unmarshal(payload, &response); err != nil {
		return &api.ResponseError{Err: err}
	}

	tag := response.Data
	output := tagWriteOutput{
		Action: "created",
		DryRun: false,
		Tag:    &gidRef{GID: tag.GID},
		Name:   tag.Name,
		Color:  color,
		Notes:  notes,
	}

	if runtimeOutputJSON(cmd) {
		encoder := json.NewEncoder(cmd.OutOrStdout())
		encoder.SetIndent("", "  ")
		return encoder.Encode(output)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Created tag: %s (%s)\n", tag.Name, tag.GID)
	return nil
}

func runTagUpdate(cmd *cobra.Command, args []string) error {
	tagGID := args[0]
	name, _ := cmd.Flags().GetString("name")
	color, _ := cmd.Flags().GetString("color")
	notes, _ := cmd.Flags().GetString("notes")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	if err := validateGID(tagGID, "tag gid"); err != nil {
		return err
	}

	if !cmd.Flags().Changed("name") && !cmd.Flags().Changed("color") && !cmd.Flags().Changed("notes") {
		return fmt.Errorf("at least one of --name, --color, or --notes is required")
	}

	if err := validateTagColor(color); err != nil {
		return err
	}

	if dryRun {
		// Validate tag exists
		query := url.Values{}
		query.Set("opt_fields", "gid")
		if _, err := runtimeClient(cmd).Get(cmd.Context(), "/tags/"+tagGID, query); err != nil {
			return err
		}

		output := tagWriteOutput{
			Action: "updated",
			DryRun: true,
			Tag:    &gidRef{GID: tagGID},
			Name:   name,
			Color:  color,
			Notes:  notes,
		}
		if runtimeOutputJSON(cmd) {
			encoder := json.NewEncoder(cmd.OutOrStdout())
			encoder.SetIndent("", "  ")
			return encoder.Encode(output)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "[dry-run] Would update tag %s\n", tagGID)
		return nil
	}

	data := map[string]any{}
	if cmd.Flags().Changed("name") {
		data["name"] = name
	}
	if cmd.Flags().Changed("color") {
		data["color"] = color
	}
	if cmd.Flags().Changed("notes") {
		data["notes"] = notes
	}

	payload, err := runtimeClient(cmd).Put(cmd.Context(), "/tags/"+tagGID, api.RequestBody{
		Data: data,
		Options: &api.RequestOptions{
			Fields: []string{"gid", "name"},
		},
	})
	if err != nil {
		return err
	}

	var response api.Response[api.Tag]
	if err := json.Unmarshal(payload, &response); err != nil {
		return &api.ResponseError{Err: err}
	}

	tag := response.Data
	output := tagWriteOutput{
		Action: "updated",
		DryRun: false,
		Tag:    &gidRef{GID: tag.GID},
		Name:   tag.Name,
	}

	if runtimeOutputJSON(cmd) {
		encoder := json.NewEncoder(cmd.OutOrStdout())
		encoder.SetIndent("", "  ")
		return encoder.Encode(output)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Updated tag: %s (%s)\n", tag.Name, tag.GID)
	return nil
}

func runTagDelete(cmd *cobra.Command, args []string) error {
	tagGID := args[0]
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	if err := validateGID(tagGID, "tag gid"); err != nil {
		return err
	}

	if dryRun {
		// Validate tag exists
		query := url.Values{}
		query.Set("opt_fields", "gid,name")
		payload, err := runtimeClient(cmd).Get(cmd.Context(), "/tags/"+tagGID, query)
		if err != nil {
			return err
		}

		var response api.Response[api.Tag]
		if err := json.Unmarshal(payload, &response); err != nil {
			return &api.ResponseError{Err: err}
		}

		output := tagWriteOutput{
			Action: "deleted",
			DryRun: true,
			Tag:    &gidRef{GID: tagGID},
			Name:   response.Data.Name,
		}
		if runtimeOutputJSON(cmd) {
			encoder := json.NewEncoder(cmd.OutOrStdout())
			encoder.SetIndent("", "  ")
			return encoder.Encode(output)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "[dry-run] Would delete tag %s (%s)\n", response.Data.Name, tagGID)
		return nil
	}

	if err := runtimeClient(cmd).Delete(cmd.Context(), "/tags/"+tagGID); err != nil {
		return err
	}

	output := tagWriteOutput{
		Action: "deleted",
		DryRun: false,
		Tag:    &gidRef{GID: tagGID},
	}

	if runtimeOutputJSON(cmd) {
		encoder := json.NewEncoder(cmd.OutOrStdout())
		encoder.SetIndent("", "  ")
		return encoder.Encode(output)
	}

	fmt.Fprintln(cmd.OutOrStdout(), "Tag deleted")
	return nil
}

func runTagTasks(cmd *cobra.Command, args []string) error {
	tagGID := args[0]
	limit, _ := cmd.Flags().GetInt("limit")

	if err := validateGID(tagGID, "tag gid"); err != nil {
		return err
	}
	if cmd.Flags().Changed("limit") && limit < 1 {
		return fmt.Errorf("--limit must be >= 1")
	}

	query := url.Values{}
	query.Set("opt_fields", "gid,name,completed")

	tasks, err := api.Paginate[api.Task](cmd.Context(), runtimeClient(cmd), "/tags/"+tagGID+"/tasks", query)
	if err != nil {
		return err
	}

	// Sort by name (case-insensitive), GID tiebreaker
	sort.Slice(tasks, func(i, j int) bool {
		cmp := strings.Compare(strings.ToLower(tasks[i].Name), strings.ToLower(tasks[j].Name))
		if cmp != 0 {
			return cmp < 0
		}
		return tasks[i].GID < tasks[j].GID
	})

	// Convert to output format
	output := make([]tagTaskItem, len(tasks))
	for i, t := range tasks {
		output[i] = tagTaskItem{
			GID:       t.GID,
			Name:      t.Name,
			Completed: t.Completed,
		}
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
		fmt.Fprintln(cmd.OutOrStdout(), "No tasks found")
		return nil
	}

	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "GID\tNAME\tCOMPLETED")
	for _, t := range output {
		fmt.Fprintf(w, "%s\t%s\t%t\n", t.GID, t.Name, t.Completed)
	}
	return w.Flush()
}
