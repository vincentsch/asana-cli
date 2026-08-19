// Package cmd implements user commands: list, view, me.
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

var userCmd = &cobra.Command{
	Use:   "user",
	Short: "Manage users",
	Long: `Manage Asana users.

Users are people who have access to a workspace. Use these commands to list
workspace members, view user details, or check the currently authenticated user.

See also: task follower add, team member list`,
}

var userListCmd = &cobra.Command{
	Use:   "list",
	Short: "List users in a workspace",
	Long: `List all users in a workspace with their name, email, and GID.

See also: user view, task follower add`,
	Example: `  # List users in a workspace
  asana user list -w "My Workspace"

  # Limit output
  asana user list -w "My Workspace" --limit 50

  # Output as JSON
  asana user list -w "My Workspace" --json`,
	Args: cobra.NoArgs,
	RunE: runUserList,
}

var userViewCmd = &cobra.Command{
	Use:   "view <user-gid>",
	Short: "View user details",
	Long: `View detailed information about a user.

You can use "me" as a shortcut for the current user.`,
	Example: `  # View user by GID
  asana user view 1234567890123456

  # View current user
  asana user view me

  # Output as JSON
  asana user view 1234567890123456 --json`,
	Args: cobra.ExactArgs(1),
	RunE: runUserView,
}

var userMeCmd = &cobra.Command{
	Use:   "me",
	Short: "Show current authenticated user",
	Long: `Show details about the currently authenticated user including workspaces.

See also: auth login, config set`,
	Example: `  # Show current user
  asana user me

  # Output as JSON
  asana user me --json`,
	Args: cobra.NoArgs,
	RunE: runUserMe,
}

func init() {
	userListCmd.Flags().StringP("workspace", "w", "", "Workspace name")
	userListCmd.Flags().String("workspace-gid", "", "Workspace GID")
	userListCmd.Flags().Int("limit", 0, "Limit number of users in output")

	userCmd.AddCommand(userListCmd)
	userCmd.AddCommand(userViewCmd)
	userCmd.AddCommand(userMeCmd)
}

type userListItem struct {
	GID        string                `json:"gid"`
	Name       string                `json:"name"`
	Email      string                `json:"email,omitempty"`
	Workspaces []workspaceCompactOut `json:"workspaces,omitempty"`
}

type userViewOutput struct {
	GID        string                `json:"gid"`
	Name       string                `json:"name"`
	Email      string                `json:"email,omitempty"`
	Workspaces []workspaceCompactOut `json:"workspaces,omitempty"`
}

type workspaceCompactOut struct {
	GID  string `json:"gid"`
	Name string `json:"name"`
}

func runUserList(cmd *cobra.Command, args []string) error {
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
	query.Set("opt_fields", "gid,name,email,workspaces.gid,workspaces.name")

	users, err := api.Paginate[api.UserDetail](cmd.Context(), runtimeClient(cmd), "/workspaces/"+resolvedWorkspace+"/users", query)
	if err != nil {
		return err
	}

	// Sort by name (case-insensitive), GID tiebreaker
	sort.Slice(users, func(i, j int) bool {
		cmp := strings.Compare(strings.ToLower(users[i].Name), strings.ToLower(users[j].Name))
		if cmp != 0 {
			return cmp < 0
		}
		return users[i].GID < users[j].GID
	})

	// Convert to output format
	output := make([]userListItem, len(users))
	for i, u := range users {
		item := userListItem{
			GID:   u.GID,
			Name:  u.Name,
			Email: u.Email,
		}
		for _, ws := range u.Workspaces {
			item.Workspaces = append(item.Workspaces, workspaceCompactOut{
				GID:  ws.GID,
				Name: ws.Name,
			})
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

func runUserView(cmd *cobra.Command, args []string) error {
	userGID := args[0]

	// Allow "me" as a special identifier
	if strings.EqualFold(userGID, "me") {
		return runUserMe(cmd, nil)
	}

	if err := validateGID(userGID, "user gid"); err != nil {
		return err
	}

	query := url.Values{}
	query.Set("opt_fields", "gid,name,email,workspaces.gid,workspaces.name")

	payload, err := runtimeClient(cmd).Get(cmd.Context(), "/users/"+userGID, query)
	if err != nil {
		return err
	}

	var response api.Response[api.UserDetail]
	if err := json.Unmarshal(payload, &response); err != nil {
		return &api.ResponseError{Err: err}
	}

	user := response.Data
	output := userViewOutput{
		GID:   user.GID,
		Name:  user.Name,
		Email: user.Email,
	}
	for _, ws := range user.Workspaces {
		output.Workspaces = append(output.Workspaces, workspaceCompactOut{
			GID:  ws.GID,
			Name: ws.Name,
		})
	}

	if runtimeOutputJSON(cmd) {
		encoder := json.NewEncoder(cmd.OutOrStdout())
		encoder.SetIndent("", "  ")
		return encoder.Encode(output)
	}

	// Text output
	fmt.Fprintf(cmd.OutOrStdout(), "GID:   %s\n", user.GID)
	fmt.Fprintf(cmd.OutOrStdout(), "Name:  %s\n", user.Name)
	if user.Email != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "Email: %s\n", user.Email)
	}
	if len(user.Workspaces) > 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "Workspaces:")
		for _, ws := range user.Workspaces {
			fmt.Fprintf(cmd.OutOrStdout(), "  - %s (%s)\n", ws.Name, ws.GID)
		}
	}

	return nil
}

func runUserMe(cmd *cobra.Command, args []string) error {
	query := url.Values{}
	query.Set("opt_fields", "gid,name,email,workspaces.gid,workspaces.name")

	payload, err := runtimeClient(cmd).Get(cmd.Context(), "/users/me", query)
	if err != nil {
		return err
	}

	var response api.Response[api.UserDetail]
	if err := json.Unmarshal(payload, &response); err != nil {
		return &api.ResponseError{Err: err}
	}

	user := response.Data
	output := userViewOutput{
		GID:   user.GID,
		Name:  user.Name,
		Email: user.Email,
	}
	for _, ws := range user.Workspaces {
		output.Workspaces = append(output.Workspaces, workspaceCompactOut{
			GID:  ws.GID,
			Name: ws.Name,
		})
	}

	if runtimeOutputJSON(cmd) {
		encoder := json.NewEncoder(cmd.OutOrStdout())
		encoder.SetIndent("", "  ")
		return encoder.Encode(output)
	}

	// Text output
	fmt.Fprintf(cmd.OutOrStdout(), "GID:   %s\n", user.GID)
	fmt.Fprintf(cmd.OutOrStdout(), "Name:  %s\n", user.Name)
	if user.Email != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "Email: %s\n", user.Email)
	}
	if len(user.Workspaces) > 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "Workspaces:")
		for _, ws := range user.Workspaces {
			fmt.Fprintf(cmd.OutOrStdout(), "  - %s (%s)\n", ws.Name, ws.GID)
		}
	}

	return nil
}
