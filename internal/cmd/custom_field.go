// Package cmd implements custom-field commands: list, view, create, update, delete.
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

var customFieldCmd = &cobra.Command{
	Use:   "custom-field",
	Short: "Manage custom fields",
	Long: `Manage Asana custom fields (premium feature).

Custom fields add structured data to tasks like priority, effort, or status.
Types include text, number, enum (dropdown), multi_enum, date, people, and reference.

See also: task view, task update`,
}

var customFieldListCmd = &cobra.Command{
	Use:   "list",
	Short: "List custom fields in a workspace",
	Long: `List all custom fields in a workspace (premium feature).

See also: custom-field view, custom-field create`,
	Example: `  # List custom fields
  asana custom-field list -w "My Workspace"

  # Limit output
  asana custom-field list -w "My Workspace" --limit 10

  # Output as JSON
  asana custom-field list -w "My Workspace" --json`,
	Args: cobra.NoArgs,
	RunE: runCustomFieldList,
}

var customFieldViewCmd = &cobra.Command{
	Use:   "view <custom-field-gid>",
	Short: "View custom field details",
	Long: `View details about a custom field including type, options, and description.

See also: custom-field list, custom-field update`,
	Example: `  # View custom field details
  asana custom-field view 1234567890123456

  # Output as JSON
  asana custom-field view 1234567890123456 --json`,
	Args: cobra.ExactArgs(1),
	RunE: runCustomFieldView,
}

var customFieldCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a custom field",
	Long: `Create a new custom field in a workspace (premium feature).

Types: text, enum, multi_enum, number, date, people, reference.
For enum/multi_enum, provide options with --enum-options.

See also: custom-field list, custom-field view`,
	Example: `  # Create a text field
  asana custom-field create "Notes" -w "My Workspace" --type text

  # Create an enum (dropdown) field
  asana custom-field create "Priority" -w "My Workspace" --type enum --enum-options "Low,Medium,High"

  # Create a number field with precision
  asana custom-field create "Story Points" -w "My Workspace" --type number --precision 0

  # Preview without creating
  asana custom-field create "Test" -w "My Workspace" --type text --dry-run`,
	Args: cobra.ExactArgs(1),
	RunE: runCustomFieldCreate,
}

var customFieldUpdateCmd = &cobra.Command{
	Use:   "update <custom-field-gid>",
	Short: "Update a custom field",
	Long: `Update a custom field's name, description, or enabled status.

See also: custom-field view`,
	Example: `  # Rename a custom field
  asana custom-field update 1234567890123456 --name "New Name"

  # Disable a custom field
  asana custom-field update 1234567890123456 --enabled false

  # Preview changes
  asana custom-field update 1234567890123456 --name "Test" --dry-run`,
	Args: cobra.ExactArgs(1),
	RunE: runCustomFieldUpdate,
}

var customFieldDeleteCmd = &cobra.Command{
	Use:   "delete <custom-field-gid>",
	Short: "Delete a custom field",
	Long: `Permanently delete a custom field.

This removes the field from all tasks. This action cannot be undone.

See also: custom-field list`,
	Example: `  # Delete a custom field
  asana custom-field delete 1234567890123456 --confirm

  # Preview deletion
  asana custom-field delete 1234567890123456 --dry-run`,
	Args: cobra.ExactArgs(1),
	RunE: runCustomFieldDelete,
}

// Valid custom field types
var validCustomFieldTypes = []string{
	"text", "enum", "multi_enum", "number", "date", "people", "reference",
}

func init() {
	customFieldListCmd.Flags().StringP("workspace", "w", "", "Workspace name")
	customFieldListCmd.Flags().String("workspace-gid", "", "Workspace GID")
	customFieldListCmd.Flags().Int("limit", 0, "Limit number of custom fields in output")

	customFieldCreateCmd.Flags().StringP("workspace", "w", "", "Workspace name")
	customFieldCreateCmd.Flags().String("workspace-gid", "", "Workspace GID")
	customFieldCreateCmd.Flags().String("type", "", "Field type (text, enum, multi_enum, number, date, people, reference)")
	customFieldCreateCmd.Flags().Int("precision", 0, "Decimal precision for number type")
	customFieldCreateCmd.Flags().StringSlice("enum-options", nil, "Enum option names (required for enum/multi_enum)")
	customFieldCreateCmd.Flags().String("description", "", "Field description")

	customFieldUpdateCmd.Flags().String("name", "", "New field name")
	customFieldUpdateCmd.Flags().String("description", "", "Field description")
	customFieldUpdateCmd.Flags().String("enabled", "", "Enable or disable field (true/false)")

	customFieldCmd.AddCommand(customFieldListCmd)
	customFieldCmd.AddCommand(customFieldViewCmd)
	customFieldCmd.AddCommand(customFieldCreateCmd)
	customFieldCmd.AddCommand(customFieldUpdateCmd)
	customFieldCmd.AddCommand(customFieldDeleteCmd)
}

type customFieldListItem struct {
	GID             string `json:"gid"`
	Name            string `json:"name"`
	ResourceSubtype string `json:"resource_subtype,omitempty"`
}

type customFieldViewOutput struct {
	GID             string          `json:"gid"`
	Name            string          `json:"name"`
	ResourceSubtype string          `json:"resource_subtype,omitempty"`
	Description     string          `json:"description,omitempty"`
	Precision       *int            `json:"precision,omitempty"`
	Enabled         bool            `json:"enabled"`
	EnumOptions     []enumOptionOut `json:"enum_options,omitempty"`
}

type enumOptionOut struct {
	GID     string `json:"gid"`
	Name    string `json:"name"`
	Color   string `json:"color,omitempty"`
	Enabled bool   `json:"enabled"`
}

type customFieldWriteOutput struct {
	Action      string  `json:"action"`
	DryRun      bool    `json:"dry_run"`
	CustomField *gidRef `json:"custom_field,omitempty"`
	Name        string  `json:"name,omitempty"`
	Type        string  `json:"type,omitempty"`
}

func runCustomFieldList(cmd *cobra.Command, args []string) error {
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
	query.Set("opt_fields", "gid,name,resource_subtype")

	fields, err := api.Paginate[api.CustomFieldCompact](cmd.Context(), runtimeClient(cmd), "/workspaces/"+resolvedWorkspace+"/custom_fields", query)
	if err != nil {
		return formatPremiumError(err, "custom fields")
	}

	// Sort by name (case-insensitive), GID tiebreaker
	sort.Slice(fields, func(i, j int) bool {
		cmp := strings.Compare(strings.ToLower(fields[i].Name), strings.ToLower(fields[j].Name))
		if cmp != 0 {
			return cmp < 0
		}
		return fields[i].GID < fields[j].GID
	})

	// Convert to output format
	output := make([]customFieldListItem, len(fields))
	for i, f := range fields {
		output[i] = customFieldListItem{
			GID:             f.GID,
			Name:            f.Name,
			ResourceSubtype: f.ResourceSubtype,
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
		fmt.Fprintln(cmd.OutOrStdout(), "No custom fields found")
		return nil
	}

	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tTYPE\tGID")
	for _, f := range output {
		fmt.Fprintf(w, "%s\t%s\t%s\n", f.Name, f.ResourceSubtype, f.GID)
	}
	return w.Flush()
}

func runCustomFieldView(cmd *cobra.Command, args []string) error {
	fieldGID := args[0]

	if err := validateGID(fieldGID, "custom field gid"); err != nil {
		return err
	}

	query := url.Values{}
	query.Set("opt_fields", "gid,name,resource_subtype,description,precision,enabled,enum_options.gid,enum_options.name,enum_options.color,enum_options.enabled")

	payload, err := runtimeClient(cmd).Get(cmd.Context(), "/custom_fields/"+fieldGID, query)
	if err != nil {
		return formatPremiumError(err, "custom fields")
	}

	var response api.Response[api.CustomFieldDetail]
	if err := json.Unmarshal(payload, &response); err != nil {
		return &api.ResponseError{Err: err}
	}

	f := response.Data
	output := customFieldViewOutput{
		GID:             f.GID,
		Name:            f.Name,
		ResourceSubtype: f.ResourceSubtype,
		Description:     f.Description,
		Precision:       f.Precision,
		Enabled:         f.Enabled,
	}
	for _, opt := range f.EnumOptions {
		output.EnumOptions = append(output.EnumOptions, enumOptionOut{
			GID:     opt.GID,
			Name:    opt.Name,
			Color:   opt.Color,
			Enabled: opt.Enabled,
		})
	}

	if runtimeOutputJSON(cmd) {
		encoder := json.NewEncoder(cmd.OutOrStdout())
		encoder.SetIndent("", "  ")
		return encoder.Encode(output)
	}

	// Text output
	fmt.Fprintf(cmd.OutOrStdout(), "GID:         %s\n", f.GID)
	fmt.Fprintf(cmd.OutOrStdout(), "Name:        %s\n", f.Name)
	fmt.Fprintf(cmd.OutOrStdout(), "Type:        %s\n", f.ResourceSubtype)
	if f.Description != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "Description: %s\n", f.Description)
	}
	if f.Precision != nil {
		fmt.Fprintf(cmd.OutOrStdout(), "Precision:   %d\n", *f.Precision)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Enabled:     %t\n", f.Enabled)
	if len(f.EnumOptions) > 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "Enum Options:")
		for _, opt := range f.EnumOptions {
			fmt.Fprintf(cmd.OutOrStdout(), "  - %s (%s)\n", opt.Name, opt.GID)
		}
	}

	return nil
}

func runCustomFieldCreate(cmd *cobra.Command, args []string) error {
	name := args[0]
	workspaceName, _ := cmd.Flags().GetString("workspace")
	workspaceGID, _ := cmd.Flags().GetString("workspace-gid")
	fieldType, _ := cmd.Flags().GetString("type")
	precision, _ := cmd.Flags().GetInt("precision")
	enumOptions, _ := cmd.Flags().GetStringSlice("enum-options")
	description, _ := cmd.Flags().GetString("description")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	if workspaceName != "" && workspaceGID != "" {
		return fmt.Errorf("use only one of --workspace or --workspace-gid")
	}

	// Validate type
	if fieldType == "" {
		return fmt.Errorf("--type is required")
	}
	validType := false
	for _, t := range validCustomFieldTypes {
		if fieldType == t {
			validType = true
			break
		}
	}
	if !validType {
		return fmt.Errorf("invalid type %q: must be one of %v", fieldType, validCustomFieldTypes)
	}

	// Validate enum-options
	isEnumType := fieldType == "enum" || fieldType == "multi_enum"
	if isEnumType && len(enumOptions) == 0 {
		return fmt.Errorf("--enum-options is required for %s type", fieldType)
	}
	if !isEnumType && len(enumOptions) > 0 {
		return fmt.Errorf("--enum-options is only valid for enum or multi_enum types")
	}

	// Validate precision
	if cmd.Flags().Changed("precision") && fieldType != "number" {
		return fmt.Errorf("--precision is only valid for number type")
	}

	resolvedWorkspace, err := resolveWorkspaceFromFlags(cmd, workspaceName, workspaceGID)
	if err != nil {
		return err
	}

	if dryRun {
		output := customFieldWriteOutput{
			Action: "created",
			DryRun: true,
			Name:   name,
			Type:   fieldType,
		}
		if runtimeOutputJSON(cmd) {
			encoder := json.NewEncoder(cmd.OutOrStdout())
			encoder.SetIndent("", "  ")
			return encoder.Encode(output)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "[dry-run] Would create %s custom field %q in workspace %s\n", fieldType, name, resolvedWorkspace)
		return nil
	}

	data := map[string]any{
		"workspace":        resolvedWorkspace,
		"name":             name,
		"resource_subtype": fieldType,
	}
	if description != "" {
		data["description"] = description
	}
	if fieldType == "number" && cmd.Flags().Changed("precision") {
		data["precision"] = precision
	}
	if isEnumType {
		opts := make([]map[string]string, len(enumOptions))
		for i, opt := range enumOptions {
			opts[i] = map[string]string{"name": opt}
		}
		data["enum_options"] = opts
	}

	payload, err := runtimeClient(cmd).Post(cmd.Context(), "/custom_fields", api.RequestBody{
		Data: data,
		Options: &api.RequestOptions{
			Fields: []string{"gid", "name"},
		},
	})
	if err != nil {
		return formatPremiumError(err, "custom fields")
	}

	var response api.Response[api.CustomFieldCompact]
	if err := json.Unmarshal(payload, &response); err != nil {
		return &api.ResponseError{Err: err}
	}

	f := response.Data
	output := customFieldWriteOutput{
		Action:      "created",
		DryRun:      false,
		CustomField: &gidRef{GID: f.GID},
		Name:        f.Name,
		Type:        fieldType,
	}

	if runtimeOutputJSON(cmd) {
		encoder := json.NewEncoder(cmd.OutOrStdout())
		encoder.SetIndent("", "  ")
		return encoder.Encode(output)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Created custom field: %s (%s)\n", f.Name, f.GID)
	return nil
}

func runCustomFieldUpdate(cmd *cobra.Command, args []string) error {
	fieldGID := args[0]
	name, _ := cmd.Flags().GetString("name")
	description, _ := cmd.Flags().GetString("description")
	enabled, _ := cmd.Flags().GetString("enabled")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	if err := validateGID(fieldGID, "custom field gid"); err != nil {
		return err
	}

	if !cmd.Flags().Changed("name") && !cmd.Flags().Changed("description") && !cmd.Flags().Changed("enabled") {
		return fmt.Errorf("at least one of --name, --description, or --enabled is required")
	}

	// Validate enabled flag
	if cmd.Flags().Changed("enabled") && enabled != "true" && enabled != "false" {
		return fmt.Errorf("--enabled must be 'true' or 'false'")
	}

	if dryRun {
		// Validate field exists
		query := url.Values{}
		query.Set("opt_fields", "gid")
		if _, err := runtimeClient(cmd).Get(cmd.Context(), "/custom_fields/"+fieldGID, query); err != nil {
			return formatPremiumError(err, "custom fields")
		}

		output := customFieldWriteOutput{
			Action:      "updated",
			DryRun:      true,
			CustomField: &gidRef{GID: fieldGID},
			Name:        name,
		}
		if runtimeOutputJSON(cmd) {
			encoder := json.NewEncoder(cmd.OutOrStdout())
			encoder.SetIndent("", "  ")
			return encoder.Encode(output)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "[dry-run] Would update custom field %s\n", fieldGID)
		return nil
	}

	data := map[string]any{}
	if cmd.Flags().Changed("name") {
		data["name"] = name
	}
	if cmd.Flags().Changed("description") {
		data["description"] = description
	}
	if cmd.Flags().Changed("enabled") {
		data["enabled"] = enabled == "true"
	}

	payload, err := runtimeClient(cmd).Put(cmd.Context(), "/custom_fields/"+fieldGID, api.RequestBody{
		Data: data,
		Options: &api.RequestOptions{
			Fields: []string{"gid", "name"},
		},
	})
	if err != nil {
		return formatPremiumError(err, "custom fields")
	}

	var response api.Response[api.CustomFieldCompact]
	if err := json.Unmarshal(payload, &response); err != nil {
		return &api.ResponseError{Err: err}
	}

	f := response.Data
	output := customFieldWriteOutput{
		Action:      "updated",
		DryRun:      false,
		CustomField: &gidRef{GID: f.GID},
		Name:        f.Name,
	}

	if runtimeOutputJSON(cmd) {
		encoder := json.NewEncoder(cmd.OutOrStdout())
		encoder.SetIndent("", "  ")
		return encoder.Encode(output)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Updated custom field: %s (%s)\n", f.Name, f.GID)
	return nil
}

func runCustomFieldDelete(cmd *cobra.Command, args []string) error {
	fieldGID := args[0]
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	if err := validateGID(fieldGID, "custom field gid"); err != nil {
		return err
	}

	if dryRun {
		// Validate field exists
		query := url.Values{}
		query.Set("opt_fields", "gid,name")
		payload, err := runtimeClient(cmd).Get(cmd.Context(), "/custom_fields/"+fieldGID, query)
		if err != nil {
			return formatPremiumError(err, "custom fields")
		}

		var response api.Response[api.CustomFieldCompact]
		if err := json.Unmarshal(payload, &response); err != nil {
			return &api.ResponseError{Err: err}
		}

		output := customFieldWriteOutput{
			Action:      "deleted",
			DryRun:      true,
			CustomField: &gidRef{GID: fieldGID},
			Name:        response.Data.Name,
		}
		if runtimeOutputJSON(cmd) {
			encoder := json.NewEncoder(cmd.OutOrStdout())
			encoder.SetIndent("", "  ")
			return encoder.Encode(output)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "[dry-run] Would delete custom field %s (%s)\n", response.Data.Name, fieldGID)
		return nil
	}

	if err := runtimeClient(cmd).Delete(cmd.Context(), "/custom_fields/"+fieldGID); err != nil {
		return formatPremiumError(err, "custom fields")
	}

	output := customFieldWriteOutput{
		Action:      "deleted",
		DryRun:      false,
		CustomField: &gidRef{GID: fieldGID},
	}

	if runtimeOutputJSON(cmd) {
		encoder := json.NewEncoder(cmd.OutOrStdout())
		encoder.SetIndent("", "  ")
		return encoder.Encode(output)
	}

	fmt.Fprintln(cmd.OutOrStdout(), "Custom field deleted")
	return nil
}
