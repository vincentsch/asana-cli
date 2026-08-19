// Package cmd implements project membership commands.
package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/vincentsch/asana-cli/internal/api"
	"github.com/vincentsch/asana-cli/internal/resolve"
)

var projectMemberCmd = &cobra.Command{
	Use:   "member",
	Short: "Manage project members",
	Long: `Manage project members.

Project members have access to view and edit the project. Use these commands
to list, add, or remove members.

See also: project view, user list`,
}

var projectMemberListCmd = &cobra.Command{
	Use:   "list <project-gid>",
	Short: "List project members",
	Long: `List all members of a project with their access level.

See also: project member add, user list`,
	Example: `  # List project members
  asana project member list 1234567890123456

  # Limit output
  asana project member list 1234567890123456 --limit 10

  # Output as JSON
  asana project member list 1234567890123456 --json`,
	Args: cobra.ExactArgs(1),
	RunE: runProjectMemberList,
}

var projectMemberAddCmd = &cobra.Command{
	Use:   "add <project-gid>",
	Short: "Add members to a project",
	Long: `Add one or more users to a project by GID or email.

See also: project member list, project member remove`,
	Example: `  # Add a user by GID
  asana project member add 1234567890123456 --user 9876543210987654

  # Add by email
  asana project member add 1234567890123456 --user jane@example.com

  # Add multiple users
  asana project member add 1234567890123456 --user 111 --user 222

  # Preview without adding
  asana project member add 1234567890123456 --user 9876543210987654 --dry-run`,
	Args: cobra.ExactArgs(1),
	RunE: runProjectMemberAdd,
}

var projectMemberRemoveCmd = &cobra.Command{
	Use:   "remove <project-gid>",
	Short: "Remove members from a project",
	Long: `Remove one or more users from a project.

See also: project member list, project member add`,
	Example: `  # Remove a user
  asana project member remove 1234567890123456 --user 9876543210987654 --confirm

  # Remove by email
  asana project member remove 1234567890123456 --user jane@example.com --confirm

  # Preview without removing
  asana project member remove 1234567890123456 --user 9876543210987654 --dry-run`,
	Args: cobra.ExactArgs(1),
	RunE: runProjectMemberRemove,
}

func init() {
	// Add flags
	projectMemberListCmd.Flags().Int("limit", 0, "Limit number of members in output")
	projectMemberAddCmd.Flags().StringSlice("user", nil, "User(s) to add (me, GID, or email)")
	projectMemberRemoveCmd.Flags().StringSlice("user", nil, "User(s) to remove (me, GID, or email)")

	projectMemberCmd.AddCommand(projectMemberListCmd)
	projectMemberCmd.AddCommand(projectMemberAddCmd)
	projectMemberCmd.AddCommand(projectMemberRemoveCmd)
	projectCmd.AddCommand(projectMemberCmd)
}

type memberListItem struct {
	GID         string `json:"gid"`
	Name        string `json:"name,omitempty"`
	Email       string `json:"email,omitempty"`
	AccessLevel string `json:"access_level,omitempty"`
}

type projectMemberOutput struct {
	Action  string   `json:"action"`
	DryRun  bool     `json:"dry_run"`
	Project gidRef   `json:"project"`
	Users   []string `json:"users"`
}

func runProjectMemberList(cmd *cobra.Command, args []string) error {
	projectGID := args[0]
	limit, _ := cmd.Flags().GetInt("limit")

	if err := validateGID(projectGID, "project gid"); err != nil {
		return err
	}
	if cmd.Flags().Changed("limit") && limit < 1 {
		return fmt.Errorf("--limit must be >= 1")
	}

	query := url.Values{}
	query.Set("opt_fields", "gid,member.gid,member.name,member.email,access_level")

	memberships, err := api.Paginate[api.ProjectMembership](cmd.Context(), runtimeClient(cmd), "/projects/"+projectGID+"/project_memberships", query)
	if err != nil {
		return err
	}

	// Convert to output format
	output := make([]memberListItem, 0, len(memberships))
	for _, m := range memberships {
		item := memberListItem{
			AccessLevel: m.AccessLevel,
		}
		if m.Member != nil {
			item.GID = m.Member.GID
			item.Name = m.Member.Name
			item.Email = m.Member.Email
		}
		output = append(output, item)
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
		fmt.Fprintln(cmd.OutOrStdout(), "No members found")
		return nil
	}

	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tEMAIL\tACCESS\tGID")
	for _, m := range output {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", m.Name, m.Email, m.AccessLevel, m.GID)
	}
	return w.Flush()
}

func runProjectMemberAdd(cmd *cobra.Command, args []string) error {
	projectGID := args[0]
	users, _ := cmd.Flags().GetStringSlice("user")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	if err := validateGID(projectGID, "project gid"); err != nil {
		return err
	}
	if len(users) == 0 {
		return fmt.Errorf("--user is required")
	}

	workspaceGID, err := inferWorkspaceForUserResolution(cmd.Context(), projectGID, users)
	if err != nil {
		return err
	}
	resolvedUsers, err := resolveUserIdentifiers(cmd.Context(), workspaceGID, users)
	if err != nil {
		return err
	}

	if dryRun {
		output := projectMemberOutput{
			Action:  "added",
			DryRun:  true,
			Project: gidRef{GID: projectGID},
			Users:   resolvedUsers,
		}
		if runtimeOutputJSON(cmd) {
			encoder := json.NewEncoder(cmd.OutOrStdout())
			encoder.SetIndent("", "  ")
			return encoder.Encode(output)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "[dry-run] Would add %d member(s) to project %s\n", len(resolvedUsers), projectGID)
		return nil
	}

	// Asana API uses addMembers endpoint
	_, err = runtimeClient(cmd).Post(cmd.Context(), "/projects/"+projectGID+"/addMembers", api.RequestBody{
		Data: map[string]interface{}{
			"members": resolvedUsers,
		},
	})
	if err != nil {
		return err
	}

	output := projectMemberOutput{
		Action:  "added",
		DryRun:  false,
		Project: gidRef{GID: projectGID},
		Users:   resolvedUsers,
	}
	if runtimeOutputJSON(cmd) {
		encoder := json.NewEncoder(cmd.OutOrStdout())
		encoder.SetIndent("", "  ")
		return encoder.Encode(output)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Added %d member(s) to project\n", len(resolvedUsers))
	return nil
}

func runProjectMemberRemove(cmd *cobra.Command, args []string) error {
	projectGID := args[0]
	users, _ := cmd.Flags().GetStringSlice("user")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	if err := validateGID(projectGID, "project gid"); err != nil {
		return err
	}
	if len(users) == 0 {
		return fmt.Errorf("--user is required")
	}

	workspaceGID, err := inferWorkspaceForUserResolution(cmd.Context(), projectGID, users)
	if err != nil {
		return err
	}
	resolvedUsers, err := resolveUserIdentifiers(cmd.Context(), workspaceGID, users)
	if err != nil {
		return err
	}

	if dryRun {
		output := projectMemberOutput{
			Action:  "removed",
			DryRun:  true,
			Project: gidRef{GID: projectGID},
			Users:   resolvedUsers,
		}
		if runtimeOutputJSON(cmd) {
			encoder := json.NewEncoder(cmd.OutOrStdout())
			encoder.SetIndent("", "  ")
			return encoder.Encode(output)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "[dry-run] Would remove %d member(s) from project %s\n", len(resolvedUsers), projectGID)
		return nil
	}

	// Asana API uses removeMembers endpoint
	_, err = runtimeClient(cmd).Post(cmd.Context(), "/projects/"+projectGID+"/removeMembers", api.RequestBody{
		Data: map[string]interface{}{
			"members": resolvedUsers,
		},
	})
	if err != nil {
		return err
	}

	output := projectMemberOutput{
		Action:  "removed",
		DryRun:  false,
		Project: gidRef{GID: projectGID},
		Users:   resolvedUsers,
	}
	if runtimeOutputJSON(cmd) {
		encoder := json.NewEncoder(cmd.OutOrStdout())
		encoder.SetIndent("", "  ")
		return encoder.Encode(output)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Removed %d member(s) from project\n", len(resolvedUsers))
	return nil
}

func inferWorkspaceForUserResolution(ctx context.Context, projectGID string, users []string) (string, error) {
	needsWorkspace := false
	for _, user := range users {
		user = strings.TrimSpace(user)
		if !resolve.IsGID(user) && !strings.EqualFold(user, "me") && strings.Contains(user, "@") {
			needsWorkspace = true
			break
		}
	}
	if !needsWorkspace {
		return "", nil
	}

	project, err := fetchProjectForWorkspace(ctx, projectGID)
	if err != nil {
		return "", err
	}
	if project.WorkspaceGID == "" {
		return "", fmt.Errorf("workspace gid is required to resolve user email")
	}
	return project.WorkspaceGID, nil
}
