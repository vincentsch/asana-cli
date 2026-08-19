// Package cmd implements workspace view and user management commands.
package cmd

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/vincentsch/asana-cli/internal/api"
	"github.com/vincentsch/asana-cli/internal/resolve"
)

var workspaceViewCmd = &cobra.Command{
	Use:   "view <workspace-gid>",
	Short: "View workspace details",
	Long: `View detailed information about a workspace.

Shows workspace name, GID, whether it's an organization, and email domains.

See also: workspace list, workspace user list`,
	Example: `  # View workspace by GID
  asana workspace view 1234567890123456

  # Output as JSON
  asana workspace view 1234567890123456 --json`,
	Args: cobra.ExactArgs(1),
	RunE: runWorkspaceView,
}

var workspaceUserCmd = &cobra.Command{
	Use:   "user",
	Short: "Manage workspace users",
	Long: `List, add, or remove users from a workspace.

See also: user list, team member`,
}

var workspaceUserListCmd = &cobra.Command{
	Use:   "list",
	Short: "List users in a workspace",
	Long: `List all users in a workspace with their name, email, and GID.

See also: workspace user add, user list`,
	Example: `  # List users
  asana workspace user list -w "My Workspace"

  # Limit output
  asana workspace user list -w "My Workspace" --limit 20

  # Output as JSON
  asana workspace user list -w "My Workspace" --json`,
	Args: cobra.NoArgs,
	RunE: runWorkspaceUserList,
}

var workspaceUserAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a user to a workspace",
	Long: `Add a user to a workspace by email or GID.

See also: workspace user list, workspace user remove`,
	Example: `  # Add user by email
  asana workspace user add -w "My Workspace" --user jane@example.com

  # Preview without adding
  asana workspace user add -w "My Workspace" --user jane@example.com --dry-run`,
	Args: cobra.NoArgs,
	RunE: runWorkspaceUserAdd,
}

var workspaceUserRemoveCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove a user from a workspace",
	Long: `Remove a user from a workspace by email or GID.

See also: workspace user list, workspace user add`,
	Example: `  # Remove user
  asana workspace user remove -w "My Workspace" --user jane@example.com --confirm

  # Preview without removing
  asana workspace user remove -w "My Workspace" --user jane@example.com --dry-run`,
	Args: cobra.NoArgs,
	RunE: runWorkspaceUserRemove,
}

// workspaceDetailFields are fields requested when viewing a workspace.
const workspaceDetailFields = "gid,name,is_organization,email_domains"

func init() {
	// User commands
	workspaceUserListCmd.Flags().StringP("workspace", "w", "", "Workspace name")
	workspaceUserListCmd.Flags().String("workspace-gid", "", "Workspace GID")
	workspaceUserListCmd.Flags().Int("limit", 0, "Limit number of users in output")

	workspaceUserAddCmd.Flags().StringP("workspace", "w", "", "Workspace name")
	workspaceUserAddCmd.Flags().String("workspace-gid", "", "Workspace GID")
	workspaceUserAddCmd.Flags().String("user", "", "User email or GID")

	workspaceUserRemoveCmd.Flags().StringP("workspace", "w", "", "Workspace name")
	workspaceUserRemoveCmd.Flags().String("workspace-gid", "", "Workspace GID")
	workspaceUserRemoveCmd.Flags().String("user", "", "User email or GID")

	workspaceUserCmd.AddCommand(workspaceUserListCmd)
	workspaceUserCmd.AddCommand(workspaceUserAddCmd)
	workspaceUserCmd.AddCommand(workspaceUserRemoveCmd)

	workspaceCmd.AddCommand(workspaceViewCmd)
	workspaceCmd.AddCommand(workspaceUserCmd)
}

// workspaceOutput is used for JSON output of workspace commands.
type workspaceOutput struct {
	Action    string                 `json:"action,omitempty"`
	DryRun    bool                   `json:"dry_run,omitempty"`
	Workspace *workspaceOutputDetail `json:"workspace,omitempty"`
}

type workspaceOutputDetail struct {
	GID            string   `json:"gid"`
	Name           string   `json:"name"`
	IsOrganization bool     `json:"is_organization,omitempty"`
	EmailDomains   []string `json:"email_domains,omitempty"`
}

func runWorkspaceView(cmd *cobra.Command, args []string) error {
	workspaceGID := args[0]

	if err := validateGID(workspaceGID, "workspace gid"); err != nil {
		return err
	}

	query := url.Values{}
	query.Set("opt_fields", workspaceDetailFields)

	payload, err := runtimeClient(cmd).Get(cmd.Context(), "/workspaces/"+workspaceGID, query)
	if err != nil {
		return err
	}

	var response api.Response[api.WorkspaceDetail]
	if err := json.Unmarshal(payload, &response); err != nil {
		return &api.ResponseError{Err: err}
	}

	workspace := response.Data
	output := workspaceOutputDetail{
		GID:            workspace.GID,
		Name:           workspace.Name,
		IsOrganization: workspace.IsOrganization,
		EmailDomains:   workspace.EmailDomains,
	}

	if runtimeOutputJSON(cmd) {
		encoder := json.NewEncoder(cmd.OutOrStdout())
		encoder.SetIndent("", "  ")
		return encoder.Encode(output)
	}

	// Text output
	fmt.Fprintf(cmd.OutOrStdout(), "GID:          %s\n", workspace.GID)
	fmt.Fprintf(cmd.OutOrStdout(), "Name:         %s\n", workspace.Name)
	fmt.Fprintf(cmd.OutOrStdout(), "Organization: %t\n", workspace.IsOrganization)
	if len(workspace.EmailDomains) > 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "Domains:      %v\n", workspace.EmailDomains)
	}

	return nil
}

type workspaceUserListItem struct {
	GID   string `json:"gid"`
	Name  string `json:"name"`
	Email string `json:"email,omitempty"`
}

type workspaceUserOutput struct {
	Action    string `json:"action"`
	DryRun    bool   `json:"dry_run"`
	Workspace gidRef `json:"workspace"`
	User      string `json:"user"`
}

func runWorkspaceUserList(cmd *cobra.Command, args []string) error {
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
	query.Set("opt_fields", "gid,name,email")

	users, err := api.Paginate[api.User](cmd.Context(), runtimeClient(cmd), "/workspaces/"+resolvedWorkspace+"/users", query)
	if err != nil {
		return err
	}

	// Convert to output format
	output := make([]workspaceUserListItem, 0, len(users))
	for _, u := range users {
		output = append(output, workspaceUserListItem{
			GID:   u.GID,
			Name:  u.Name,
			Email: u.Email,
		})
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
		fmt.Fprintln(cmd.OutOrStdout(), "No users found")
		return nil
	}

	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tEMAIL\tGID")
	for _, u := range output {
		fmt.Fprintf(w, "%s\t%s\t%s\n", u.Name, u.Email, u.GID)
	}
	return w.Flush()
}

func runWorkspaceUserAdd(cmd *cobra.Command, args []string) error {
	workspaceName, _ := cmd.Flags().GetString("workspace")
	workspaceGID, _ := cmd.Flags().GetString("workspace-gid")
	userValue, _ := cmd.Flags().GetString("user")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	if workspaceName != "" && workspaceGID != "" {
		return fmt.Errorf("use only one of --workspace or --workspace-gid")
	}
	if strings.TrimSpace(userValue) == "" {
		return fmt.Errorf("--user is required")
	}
	if strings.EqualFold(strings.TrimSpace(userValue), "me") {
		return fmt.Errorf("--user must be an email or GID (me is not supported)")
	}

	resolvedWorkspace, err := resolveWorkspaceFromFlags(cmd, workspaceName, workspaceGID)
	if err != nil {
		return err
	}

	userValue = strings.TrimSpace(userValue)
	if resolve.IsGID(userValue) {
		if err := validateGID(userValue, "user gid"); err != nil {
			return err
		}
	} else if !strings.Contains(userValue, "@") {
		return fmt.Errorf("--user must be an email or GID")
	}

	if dryRun {
		output := workspaceUserOutput{
			Action:    "added",
			DryRun:    true,
			Workspace: gidRef{GID: resolvedWorkspace},
			User:      userValue,
		}
		if runtimeOutputJSON(cmd) {
			encoder := json.NewEncoder(cmd.OutOrStdout())
			encoder.SetIndent("", "  ")
			return encoder.Encode(output)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "[dry-run] Would add user %s to workspace %s\n", userValue, resolvedWorkspace)
		return nil
	}

	// Asana uses addUser endpoint
	payload, err := runtimeClient(cmd).Post(cmd.Context(), "/workspaces/"+resolvedWorkspace+"/addUser", api.RequestBody{
		Data: map[string]string{
			"user": userValue,
		},
		Options: &api.RequestOptions{
			Fields: []string{"gid", "name", "email"},
		},
	})
	if err != nil {
		return err
	}

	var response api.Response[api.User]
	if err := json.Unmarshal(payload, &response); err != nil {
		return &api.ResponseError{Err: err}
	}

	user := response.Data
	output := workspaceUserOutput{
		Action:    "added",
		DryRun:    false,
		Workspace: gidRef{GID: resolvedWorkspace},
		User:      userValue,
	}
	if runtimeOutputJSON(cmd) {
		encoder := json.NewEncoder(cmd.OutOrStdout())
		encoder.SetIndent("", "  ")
		return encoder.Encode(output)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Added user: %s (%s)\n", user.Name, user.GID)
	return nil
}

func runWorkspaceUserRemove(cmd *cobra.Command, args []string) error {
	workspaceName, _ := cmd.Flags().GetString("workspace")
	workspaceGID, _ := cmd.Flags().GetString("workspace-gid")
	userValue, _ := cmd.Flags().GetString("user")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	if workspaceName != "" && workspaceGID != "" {
		return fmt.Errorf("use only one of --workspace or --workspace-gid")
	}
	if strings.TrimSpace(userValue) == "" {
		return fmt.Errorf("--user is required")
	}
	if strings.EqualFold(strings.TrimSpace(userValue), "me") {
		return fmt.Errorf("--user must be an email or GID (me is not supported)")
	}

	resolvedWorkspace, err := resolveWorkspaceFromFlags(cmd, workspaceName, workspaceGID)
	if err != nil {
		return err
	}

	userValue = strings.TrimSpace(userValue)
	if resolve.IsGID(userValue) {
		if err := validateGID(userValue, "user gid"); err != nil {
			return err
		}
	} else if !strings.Contains(userValue, "@") {
		return fmt.Errorf("--user must be an email or GID")
	}

	if dryRun {
		output := workspaceUserOutput{
			Action:    "removed",
			DryRun:    true,
			Workspace: gidRef{GID: resolvedWorkspace},
			User:      userValue,
		}
		if runtimeOutputJSON(cmd) {
			encoder := json.NewEncoder(cmd.OutOrStdout())
			encoder.SetIndent("", "  ")
			return encoder.Encode(output)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "[dry-run] Would remove user %s from workspace %s\n", userValue, resolvedWorkspace)
		return nil
	}

	// Asana uses removeUser endpoint
	_, err = runtimeClient(cmd).Post(cmd.Context(), "/workspaces/"+resolvedWorkspace+"/removeUser", api.RequestBody{
		Data: map[string]string{
			"user": userValue,
		},
	})
	if err != nil {
		return err
	}

	output := workspaceUserOutput{
		Action:    "removed",
		DryRun:    false,
		Workspace: gidRef{GID: resolvedWorkspace},
		User:      userValue,
	}
	if runtimeOutputJSON(cmd) {
		encoder := json.NewEncoder(cmd.OutOrStdout())
		encoder.SetIndent("", "  ")
		return encoder.Encode(output)
	}

	fmt.Fprintln(cmd.OutOrStdout(), "User removed from workspace")
	return nil
}
