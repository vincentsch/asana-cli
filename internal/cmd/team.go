// Package cmd implements team commands: list, view, create.
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

var teamCmd = &cobra.Command{
	Use:   "team",
	Short: "Manage teams",
	Long: `Manage Asana teams.

Teams are groups of users within an organization workspace. They provide access
control for projects and help organize work. Teams are only available in
organization workspaces, not personal workspaces.

See also: workspace list, project list, user list`,
}

var teamListCmd = &cobra.Command{
	Use:   "list",
	Short: "List teams in an organization",
	Long: `List all teams in an organization workspace.

See also: team view, team create, project list`,
	Example: `  # List teams in an organization
  asana team list -w "My Organization"

  # Limit output
  asana team list -w "My Organization" --limit 10

  # Output as JSON
  asana team list -w "My Organization" --json`,
	Args: cobra.NoArgs,
	RunE: runTeamList,
}

var teamViewCmd = &cobra.Command{
	Use:   "view <team-gid>",
	Short: "View team details",
	Long: `View detailed information about a team including members, visibility, and description.

See also: team list, team member list`,
	Example: `  # View team details
  asana team view 1234567890123456

  # Output as JSON
  asana team view 1234567890123456 --json`,
	Args: cobra.ExactArgs(1),
	RunE: runTeamView,
}

var teamCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new team",
	Long: `Create a new team in an organization.

Visibility options:
  - secret: Only members can see the team
  - request_to_join: Anyone in org can request to join
  - public: Anyone in org can join

See also: team list, team member add`,
	Example: `  # Create a team
  asana team create "Engineering" -w "My Organization"

  # Create with visibility and description
  asana team create "Design" -w "My Organization" --visibility public --description "Design team"

  # Preview without creating
  asana team create "Test Team" -w "My Organization" --dry-run`,
	Args: cobra.ExactArgs(1),
	RunE: runTeamCreate,
}

func init() {
	teamListCmd.Flags().StringP("workspace", "w", "", "Organization/workspace name")
	teamListCmd.Flags().String("workspace-gid", "", "Organization/workspace GID")
	teamListCmd.Flags().Int("limit", 0, "Limit number of teams in output")

	teamCreateCmd.Flags().StringP("workspace", "w", "", "Organization name")
	teamCreateCmd.Flags().String("workspace-gid", "", "Organization GID")
	teamCreateCmd.Flags().String("description", "", "Team description")
	teamCreateCmd.Flags().String("visibility", "", "Team visibility (secret, request_to_join, public)")

	teamCmd.AddCommand(teamListCmd)
	teamCmd.AddCommand(teamViewCmd)
	teamCmd.AddCommand(teamCreateCmd)
}

type teamListItem struct {
	GID        string `json:"gid"`
	Name       string `json:"name"`
	Visibility string `json:"visibility,omitempty"`
}

type teamViewOutput struct {
	GID          string               `json:"gid"`
	Name         string               `json:"name"`
	Description  string               `json:"description,omitempty"`
	Visibility   string               `json:"visibility,omitempty"`
	Organization *workspaceCompactOut `json:"organization,omitempty"`
	Members      []teamMemberViewItem `json:"members,omitempty"`
}

type teamMemberViewItem struct {
	GID     string `json:"gid"`
	Name    string `json:"name"`
	Email   string `json:"email,omitempty"`
	IsAdmin bool   `json:"is_admin"`
}

type teamCreateOutput struct {
	Action      string  `json:"action"`
	DryRun      bool    `json:"dry_run"`
	Team        *gidRef `json:"team,omitempty"`
	Name        string  `json:"name"`
	Description string  `json:"description,omitempty"`
	Visibility  string  `json:"visibility,omitempty"`
}

func runTeamList(cmd *cobra.Command, args []string) error {
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

	// Validate that workspace is an organization
	if err := requireOrganizationWorkspace(cmd.Context(), resolvedWorkspace); err != nil {
		return err
	}

	query := url.Values{}
	query.Set("opt_fields", "gid,name,visibility")

	teams, err := api.Paginate[api.TeamDetail](cmd.Context(), runtimeClient(cmd), "/workspaces/"+resolvedWorkspace+"/teams", query)
	if err != nil {
		return err
	}

	// Sort by name (case-insensitive), GID tiebreaker
	sort.Slice(teams, func(i, j int) bool {
		cmp := strings.Compare(strings.ToLower(teams[i].Name), strings.ToLower(teams[j].Name))
		if cmp != 0 {
			return cmp < 0
		}
		return teams[i].GID < teams[j].GID
	})

	// Convert to output format
	output := make([]teamListItem, len(teams))
	for i, t := range teams {
		output[i] = teamListItem{
			GID:        t.GID,
			Name:       t.Name,
			Visibility: t.Visibility,
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
		fmt.Fprintln(cmd.OutOrStdout(), "No teams found")
		return nil
	}

	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tVISIBILITY\tGID")
	for _, t := range output {
		fmt.Fprintf(w, "%s\t%s\t%s\n", t.Name, t.Visibility, t.GID)
	}
	return w.Flush()
}

func runTeamView(cmd *cobra.Command, args []string) error {
	teamGID := args[0]

	if err := validateGID(teamGID, "team gid"); err != nil {
		return err
	}

	query := url.Values{}
	query.Set("opt_fields", "gid,name,description,visibility,organization.gid,organization.name")

	payload, err := runtimeClient(cmd).Get(cmd.Context(), "/teams/"+teamGID, query)
	if err != nil {
		return err
	}

	var response api.Response[api.TeamDetail]
	if err := json.Unmarshal(payload, &response); err != nil {
		return &api.ResponseError{Err: err}
	}

	memberQuery := url.Values{}
	memberQuery.Set("opt_fields", "gid,user.gid,user.name,user.email,is_admin")

	members, err := api.Paginate[api.TeamMember](cmd.Context(), runtimeClient(cmd), "/teams/"+teamGID+"/users", memberQuery)
	if err != nil {
		return err
	}

	sort.Slice(members, func(i, j int) bool {
		nameI := ""
		nameJ := ""
		if members[i].User != nil {
			nameI = members[i].User.Name
		}
		if members[j].User != nil {
			nameJ = members[j].User.Name
		}
		cmp := strings.Compare(strings.ToLower(nameI), strings.ToLower(nameJ))
		if cmp != 0 {
			return cmp < 0
		}
		gidI := members[i].GID
		gidJ := members[j].GID
		if members[i].User != nil {
			gidI = members[i].User.GID
		}
		if members[j].User != nil {
			gidJ = members[j].User.GID
		}
		return gidI < gidJ
	})

	team := response.Data
	output := teamViewOutput{
		GID:         team.GID,
		Name:        team.Name,
		Description: team.Description,
		Visibility:  team.Visibility,
	}
	if team.Organization != nil {
		output.Organization = &workspaceCompactOut{
			GID:  team.Organization.GID,
			Name: team.Organization.Name,
		}
	}
	for _, member := range members {
		item := teamMemberViewItem{
			GID:     member.GID,
			IsAdmin: member.IsAdmin,
		}
		if member.User != nil {
			item.GID = member.User.GID
			item.Name = member.User.Name
			item.Email = member.User.Email
		}
		output.Members = append(output.Members, item)
	}

	if runtimeOutputJSON(cmd) {
		encoder := json.NewEncoder(cmd.OutOrStdout())
		encoder.SetIndent("", "  ")
		return encoder.Encode(output)
	}

	// Text output
	fmt.Fprintf(cmd.OutOrStdout(), "GID:          %s\n", team.GID)
	fmt.Fprintf(cmd.OutOrStdout(), "Name:         %s\n", team.Name)
	if team.Description != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "Description:  %s\n", team.Description)
	}
	if team.Visibility != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "Visibility:   %s\n", team.Visibility)
	}
	if team.Organization != nil {
		fmt.Fprintf(cmd.OutOrStdout(), "Organization: %s (%s)\n", team.Organization.Name, team.Organization.GID)
	}
	if len(output.Members) > 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "Members:")
		for _, member := range output.Members {
			line := fmt.Sprintf("  - %s", member.Name)
			if member.Email != "" {
				line = fmt.Sprintf("%s (%s)", line, member.Email)
			}
			if member.IsAdmin {
				line = fmt.Sprintf("%s [admin]", line)
			}
			fmt.Fprintln(cmd.OutOrStdout(), line)
		}
	}

	return nil
}

func runTeamCreate(cmd *cobra.Command, args []string) error {
	name := args[0]
	workspaceName, _ := cmd.Flags().GetString("workspace")
	workspaceGID, _ := cmd.Flags().GetString("workspace-gid")
	description, _ := cmd.Flags().GetString("description")
	visibility, _ := cmd.Flags().GetString("visibility")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	if workspaceName != "" && workspaceGID != "" {
		return fmt.Errorf("use only one of --workspace or --workspace-gid")
	}

	// Validate visibility
	if visibility != "" {
		validVisibilities := []string{"secret", "request_to_join", "public"}
		valid := false
		for _, v := range validVisibilities {
			if visibility == v {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("invalid visibility %q: must be one of %v", visibility, validVisibilities)
		}
	}

	resolvedWorkspace, err := resolveWorkspaceFromFlags(cmd, workspaceName, workspaceGID)
	if err != nil {
		return err
	}

	// Validate that workspace is an organization
	if err := requireOrganizationWorkspace(cmd.Context(), resolvedWorkspace); err != nil {
		return err
	}

	if dryRun {
		output := teamCreateOutput{
			Action:      "created",
			DryRun:      true,
			Name:        name,
			Description: description,
			Visibility:  visibility,
		}
		if runtimeOutputJSON(cmd) {
			encoder := json.NewEncoder(cmd.OutOrStdout())
			encoder.SetIndent("", "  ")
			return encoder.Encode(output)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "[dry-run] Would create team %q in organization %s\n", name, resolvedWorkspace)
		return nil
	}

	data := map[string]any{
		"organization": resolvedWorkspace,
		"name":         name,
	}
	if description != "" {
		data["description"] = description
	}
	if visibility != "" {
		data["visibility"] = visibility
	}

	payload, err := runtimeClient(cmd).Post(cmd.Context(), "/teams", api.RequestBody{
		Data: data,
		Options: &api.RequestOptions{
			Fields: []string{"gid", "name"},
		},
	})
	if err != nil {
		return err
	}

	var response api.Response[api.Team]
	if err := json.Unmarshal(payload, &response); err != nil {
		return &api.ResponseError{Err: err}
	}

	team := response.Data
	output := teamCreateOutput{
		Action:      "created",
		DryRun:      false,
		Team:        &gidRef{GID: team.GID},
		Name:        team.Name,
		Description: description,
		Visibility:  visibility,
	}

	if runtimeOutputJSON(cmd) {
		encoder := json.NewEncoder(cmd.OutOrStdout())
		encoder.SetIndent("", "  ")
		return encoder.Encode(output)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Created team: %s (%s)\n", team.Name, team.GID)
	return nil
}
