// Package cmd implements team member commands: list, add, remove.
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
	"github.com/vincentsch/asana-cli/internal/resolve"
)

var teamMemberCmd = &cobra.Command{
	Use:   "member",
	Short: "Manage team members",
	Long: `List, add, or remove members from a team.

See also: team view, user list`,
}

var teamMemberListCmd = &cobra.Command{
	Use:   "list <team-gid>",
	Short: "List team members",
	Long: `List all members of a team with their name, email, and admin status.

See also: team member add, team view`,
	Example: `  # List team members
  asana team member list 1234567890123456

  # Limit output
  asana team member list 1234567890123456 --limit 20

  # Output as JSON
  asana team member list 1234567890123456 --json`,
	Args: cobra.ExactArgs(1),
	RunE: runTeamMemberList,
}

var teamMemberAddCmd = &cobra.Command{
	Use:   "add <team-gid>",
	Short: "Add member to team",
	Long: `Add one or more users to a team by GID or email.

See also: team member list, team member remove`,
	Example: `  # Add a user by GID
  asana team member add 1234567890123456 --user 9876543210987654

  # Add by email (requires workspace for resolution)
  asana team member add 1234567890123456 --user jane@example.com -w "My Organization"

  # Add multiple users
  asana team member add 1234567890123456 --user 111 --user 222

  # Preview without adding
  asana team member add 1234567890123456 --user 9876543210987654 --dry-run`,
	Args: cobra.ExactArgs(1),
	RunE: runTeamMemberAdd,
}

var teamMemberRemoveCmd = &cobra.Command{
	Use:   "remove <team-gid>",
	Short: "Remove member from team",
	Long: `Remove one or more users from a team.

See also: team member list, team member add`,
	Example: `  # Remove a user
  asana team member remove 1234567890123456 --user 9876543210987654 --confirm

  # Remove by email
  asana team member remove 1234567890123456 --user jane@example.com -w "My Organization" --confirm

  # Preview without removing
  asana team member remove 1234567890123456 --user 9876543210987654 --dry-run`,
	Args: cobra.ExactArgs(1),
	RunE: runTeamMemberRemove,
}

func init() {
	teamMemberListCmd.Flags().Int("limit", 0, "Limit number of members in output")

	teamMemberAddCmd.Flags().StringSlice("user", nil, "User(s) to add (GID or email)")
	teamMemberAddCmd.Flags().StringP("workspace", "w", "", "Workspace name (for email resolution)")
	teamMemberAddCmd.Flags().String("workspace-gid", "", "Workspace GID")

	teamMemberRemoveCmd.Flags().StringSlice("user", nil, "User(s) to remove (GID or email)")
	teamMemberRemoveCmd.Flags().StringP("workspace", "w", "", "Workspace name (for email resolution)")
	teamMemberRemoveCmd.Flags().String("workspace-gid", "", "Workspace GID")

	teamMemberCmd.AddCommand(teamMemberListCmd)
	teamMemberCmd.AddCommand(teamMemberAddCmd)
	teamMemberCmd.AddCommand(teamMemberRemoveCmd)
	teamCmd.AddCommand(teamMemberCmd)
}

type teamMemberListItem struct {
	GID     string `json:"gid"`
	Name    string `json:"name"`
	Email   string `json:"email,omitempty"`
	IsAdmin bool   `json:"is_admin"`
}

type teamMemberOutput struct {
	Action string   `json:"action"`
	DryRun bool     `json:"dry_run"`
	Team   gidRef   `json:"team"`
	Users  []string `json:"users"`
}

func runTeamMemberList(cmd *cobra.Command, args []string) error {
	teamGID := args[0]
	limit, _ := cmd.Flags().GetInt("limit")

	if err := validateGID(teamGID, "team gid"); err != nil {
		return err
	}
	if cmd.Flags().Changed("limit") && limit < 1 {
		return fmt.Errorf("--limit must be >= 1")
	}

	query := url.Values{}
	query.Set("opt_fields", "gid,user.gid,user.name,user.email,is_admin")

	members, err := api.Paginate[api.TeamMember](cmd.Context(), runtimeClient(cmd), "/teams/"+teamGID+"/users", query)
	if err != nil {
		return err
	}

	// Sort by user name (case-insensitive), user GID tiebreaker
	sort.Slice(members, func(i, j int) bool {
		nameI := ""
		nameJ := ""
		gidI := members[i].GID
		gidJ := members[j].GID
		if members[i].User != nil {
			nameI = members[i].User.Name
			gidI = members[i].User.GID
		}
		if members[j].User != nil {
			nameJ = members[j].User.Name
			gidJ = members[j].User.GID
		}
		cmp := strings.Compare(strings.ToLower(nameI), strings.ToLower(nameJ))
		if cmp != 0 {
			return cmp < 0
		}
		return gidI < gidJ
	})

	// Convert to output format
	output := make([]teamMemberListItem, 0, len(members))
	for _, m := range members {
		item := teamMemberListItem{
			GID:     m.GID,
			IsAdmin: m.IsAdmin,
		}
		if m.User != nil {
			item.GID = m.User.GID
			item.Name = m.User.Name
			item.Email = m.User.Email
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
	fmt.Fprintln(w, "NAME\tEMAIL\tADMIN\tGID")
	for _, m := range output {
		fmt.Fprintf(w, "%s\t%s\t%t\t%s\n", m.Name, m.Email, m.IsAdmin, m.GID)
	}
	return w.Flush()
}

func runTeamMemberAdd(cmd *cobra.Command, args []string) error {
	teamGID := args[0]
	users, _ := cmd.Flags().GetStringSlice("user")
	workspaceName, _ := cmd.Flags().GetString("workspace")
	workspaceGID, _ := cmd.Flags().GetString("workspace-gid")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	if err := validateGID(teamGID, "team gid"); err != nil {
		return err
	}
	if len(users) == 0 {
		return fmt.Errorf("--user is required")
	}
	if workspaceName != "" && workspaceGID != "" {
		return fmt.Errorf("use only one of --workspace or --workspace-gid")
	}

	// Check if we need workspace for email resolution
	needsWorkspace := false
	for _, user := range users {
		user = strings.TrimSpace(user)
		if !resolve.IsGID(user) && !strings.EqualFold(user, "me") && strings.Contains(user, "@") {
			needsWorkspace = true
			break
		}
	}

	// Resolve workspace if needed
	resolvedWorkspace := workspaceGID
	if needsWorkspace {
		gid, err := resolveWorkspaceFromFlags(cmd, workspaceName, workspaceGID)
		if err != nil {
			return fmt.Errorf("workspace is required to resolve user email: %w", err)
		}
		resolvedWorkspace = gid
	} else if workspaceName != "" {
		gid, err := resolveWorkspaceWithPrompt(cmd.Context(), runtimeClient(cmd), workspaceName, runtimeNoPrompt(cmd) || runtimeOutputJSON(cmd))
		if err != nil {
			return err
		}
		resolvedWorkspace = gid
	}

	// Resolve users
	resolvedUsers, err := resolveUserIdentifiers(cmd.Context(), resolvedWorkspace, users)
	if err != nil {
		return err
	}

	if dryRun {
		// Validate team exists
		query := url.Values{}
		query.Set("opt_fields", "gid")
		if _, err := runtimeClient(cmd).Get(cmd.Context(), "/teams/"+teamGID, query); err != nil {
			return err
		}

		output := teamMemberOutput{
			Action: "added",
			DryRun: true,
			Team:   gidRef{GID: teamGID},
			Users:  resolvedUsers,
		}
		if runtimeOutputJSON(cmd) {
			encoder := json.NewEncoder(cmd.OutOrStdout())
			encoder.SetIndent("", "  ")
			return encoder.Encode(output)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "[dry-run] Would add %d member(s) to team %s\n", len(resolvedUsers), teamGID)
		return nil
	}

	// Add each user
	for _, userGID := range resolvedUsers {
		_, err := runtimeClient(cmd).Post(cmd.Context(), "/teams/"+teamGID+"/addUser", api.RequestBody{
			Data: map[string]string{"user": userGID},
		})
		if err != nil {
			return err
		}
	}

	output := teamMemberOutput{
		Action: "added",
		DryRun: false,
		Team:   gidRef{GID: teamGID},
		Users:  resolvedUsers,
	}

	if runtimeOutputJSON(cmd) {
		encoder := json.NewEncoder(cmd.OutOrStdout())
		encoder.SetIndent("", "  ")
		return encoder.Encode(output)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Added %d member(s) to team\n", len(resolvedUsers))
	return nil
}

func runTeamMemberRemove(cmd *cobra.Command, args []string) error {
	teamGID := args[0]
	users, _ := cmd.Flags().GetStringSlice("user")
	workspaceName, _ := cmd.Flags().GetString("workspace")
	workspaceGID, _ := cmd.Flags().GetString("workspace-gid")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	if err := validateGID(teamGID, "team gid"); err != nil {
		return err
	}
	if len(users) == 0 {
		return fmt.Errorf("--user is required")
	}
	if workspaceName != "" && workspaceGID != "" {
		return fmt.Errorf("use only one of --workspace or --workspace-gid")
	}

	// Check if we need workspace for email resolution
	needsWorkspace := false
	for _, user := range users {
		user = strings.TrimSpace(user)
		if !resolve.IsGID(user) && !strings.EqualFold(user, "me") && strings.Contains(user, "@") {
			needsWorkspace = true
			break
		}
	}

	// Resolve workspace if needed
	resolvedWorkspace := workspaceGID
	if needsWorkspace {
		gid, err := resolveWorkspaceFromFlags(cmd, workspaceName, workspaceGID)
		if err != nil {
			return fmt.Errorf("workspace is required to resolve user email: %w", err)
		}
		resolvedWorkspace = gid
	} else if workspaceName != "" {
		gid, err := resolveWorkspaceWithPrompt(cmd.Context(), runtimeClient(cmd), workspaceName, runtimeNoPrompt(cmd) || runtimeOutputJSON(cmd))
		if err != nil {
			return err
		}
		resolvedWorkspace = gid
	}

	// Resolve users
	resolvedUsers, err := resolveUserIdentifiers(cmd.Context(), resolvedWorkspace, users)
	if err != nil {
		return err
	}

	if dryRun {
		// Validate team exists
		query := url.Values{}
		query.Set("opt_fields", "gid")
		if _, err := runtimeClient(cmd).Get(cmd.Context(), "/teams/"+teamGID, query); err != nil {
			return err
		}

		output := teamMemberOutput{
			Action: "removed",
			DryRun: true,
			Team:   gidRef{GID: teamGID},
			Users:  resolvedUsers,
		}
		if runtimeOutputJSON(cmd) {
			encoder := json.NewEncoder(cmd.OutOrStdout())
			encoder.SetIndent("", "  ")
			return encoder.Encode(output)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "[dry-run] Would remove %d member(s) from team %s\n", len(resolvedUsers), teamGID)
		return nil
	}

	// Remove each user
	for _, userGID := range resolvedUsers {
		_, err := runtimeClient(cmd).Post(cmd.Context(), "/teams/"+teamGID+"/removeUser", api.RequestBody{
			Data: map[string]string{"user": userGID},
		})
		if err != nil {
			return err
		}
	}

	output := teamMemberOutput{
		Action: "removed",
		DryRun: false,
		Team:   gidRef{GID: teamGID},
		Users:  resolvedUsers,
	}

	if runtimeOutputJSON(cmd) {
		encoder := json.NewEncoder(cmd.OutOrStdout())
		encoder.SetIndent("", "  ")
		return encoder.Encode(output)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Removed %d member(s) from team\n", len(resolvedUsers))
	return nil
}
