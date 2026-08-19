// Package cmd implements portfolio commands: list, view, create.
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

var portfolioCmd = &cobra.Command{
	Use:   "portfolio",
	Short: "Manage portfolios",
	Long: `Manage Asana portfolios.

Portfolios are collections of projects used to track and organize related work.
They provide a high-level view across multiple projects.

See also: project, goal`,
}

var portfolioListCmd = &cobra.Command{
	Use:   "list",
	Short: "List portfolios",
	Long: `List portfolios in a workspace.

By default, lists portfolios owned by the current user. Use --owner to see
portfolios owned by others.

See also: portfolio view, portfolio create`,
	Example: `  # List your portfolios
  asana portfolio list -w "My Workspace"

  # List all portfolios (any owner)
  asana portfolio list -w "My Workspace" --owner all

  # Output as JSON
  asana portfolio list -w "My Workspace" --json`,
	Args: cobra.NoArgs,
	RunE: runPortfolioList,
}

var portfolioViewCmd = &cobra.Command{
	Use:   "view <portfolio-gid>",
	Short: "View portfolio details",
	Long: `View detailed information about a portfolio including owner and projects.

See also: portfolio list, portfolio project list`,
	Example: `  # View portfolio details
  asana portfolio view 1234567890123456

  # Output as JSON
  asana portfolio view 1234567890123456 --json`,
	Args: cobra.ExactArgs(1),
	RunE: runPortfolioView,
}

var portfolioCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a portfolio",
	Long: `Create a new portfolio in a workspace.

See also: portfolio list, portfolio project add`,
	Example: `  # Create a portfolio
  asana portfolio create "Q1 Projects" -w "My Workspace"

  # Create with color
  asana portfolio create "Engineering" -w "My Workspace" --color dark-blue

  # Preview without creating
  asana portfolio create "Test" -w "My Workspace" --dry-run`,
	Args: cobra.ExactArgs(1),
	RunE: runPortfolioCreate,
}

var portfolioDeleteCmd = &cobra.Command{
	Use:   "delete <portfolio-gid>",
	Short: "Delete a portfolio",
	Long: `Permanently delete a portfolio.

Projects in the portfolio are not deleted. This action cannot be undone.

See also: portfolio list`,
	Example: `  # Delete a portfolio
  asana portfolio delete 1234567890123456 --confirm

  # Preview deletion
  asana portfolio delete 1234567890123456 --dry-run`,
	Args: cobra.ExactArgs(1),
	RunE: runPortfolioDelete,
}

func init() {
	portfolioListCmd.Flags().StringP("workspace", "w", "", "Workspace name")
	portfolioListCmd.Flags().String("workspace-gid", "", "Workspace GID")
	portfolioListCmd.Flags().String("owner", "me", "Portfolio owner (me, GID, or email)")
	portfolioListCmd.Flags().Int("limit", 0, "Limit number of portfolios in output")

	portfolioCreateCmd.Flags().StringP("workspace", "w", "", "Workspace name")
	portfolioCreateCmd.Flags().String("workspace-gid", "", "Workspace GID")
	portfolioCreateCmd.Flags().String("color", "", "Portfolio color")

	portfolioCmd.AddCommand(portfolioListCmd)
	portfolioCmd.AddCommand(portfolioViewCmd)
	portfolioCmd.AddCommand(portfolioCreateCmd)
	portfolioCmd.AddCommand(portfolioDeleteCmd)
}

type portfolioListItem struct {
	GID   string `json:"gid"`
	Name  string `json:"name"`
	Color string `json:"color,omitempty"`
}

type portfolioViewOutput struct {
	GID       string               `json:"gid"`
	Name      string               `json:"name"`
	Color     string               `json:"color,omitempty"`
	Owner     *userCompactOut      `json:"owner,omitempty"`
	Workspace *workspaceCompactOut `json:"workspace,omitempty"`
	CreatedAt string               `json:"created_at,omitempty"`
}

type portfolioWriteOutput struct {
	Action    string  `json:"action"`
	DryRun    bool    `json:"dry_run"`
	Portfolio *gidRef `json:"portfolio,omitempty"`
	Name      string  `json:"name,omitempty"`
}

func runPortfolioList(cmd *cobra.Command, args []string) error {
	workspaceName, _ := cmd.Flags().GetString("workspace")
	workspaceGID, _ := cmd.Flags().GetString("workspace-gid")
	owner, _ := cmd.Flags().GetString("owner")
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

	// Resolve owner - "me" means current user's GID
	resolvedOwner := owner
	if strings.EqualFold(owner, "me") {
		user, err := fetchCurrentUser(cmd.Context())
		if err != nil {
			return err
		}
		resolvedOwner = user.GID
	}

	query := url.Values{}
	query.Set("workspace", resolvedWorkspace)
	query.Set("owner", resolvedOwner)
	query.Set("opt_fields", "gid,name,color")

	portfolios, err := api.Paginate[api.Portfolio](cmd.Context(), runtimeClient(cmd), "/portfolios", query)
	if err != nil {
		return err
	}

	// Sort by name (case-insensitive), GID tiebreaker
	sort.Slice(portfolios, func(i, j int) bool {
		cmp := strings.Compare(strings.ToLower(portfolios[i].Name), strings.ToLower(portfolios[j].Name))
		if cmp != 0 {
			return cmp < 0
		}
		return portfolios[i].GID < portfolios[j].GID
	})

	// Convert to output format
	output := make([]portfolioListItem, len(portfolios))
	for i, p := range portfolios {
		output[i] = portfolioListItem{
			GID:   p.GID,
			Name:  p.Name,
			Color: p.Color,
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
		fmt.Fprintln(cmd.OutOrStdout(), "No portfolios found")
		return nil
	}

	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tCOLOR\tGID")
	for _, p := range output {
		fmt.Fprintf(w, "%s\t%s\t%s\n", p.Name, p.Color, p.GID)
	}
	return w.Flush()
}

func runPortfolioView(cmd *cobra.Command, args []string) error {
	portfolioGID := args[0]

	if err := validateGID(portfolioGID, "portfolio gid"); err != nil {
		return err
	}

	query := url.Values{}
	query.Set("opt_fields", "gid,name,color,owner.gid,owner.name,workspace.gid,workspace.name,created_at")

	payload, err := runtimeClient(cmd).Get(cmd.Context(), "/portfolios/"+portfolioGID, query)
	if err != nil {
		return err
	}

	var response api.Response[api.Portfolio]
	if err := json.Unmarshal(payload, &response); err != nil {
		return &api.ResponseError{Err: err}
	}

	p := response.Data
	output := portfolioViewOutput{
		GID:       p.GID,
		Name:      p.Name,
		Color:     p.Color,
		CreatedAt: p.CreatedAt,
	}
	if p.Owner != nil {
		output.Owner = &userCompactOut{GID: p.Owner.GID, Name: p.Owner.Name}
	}
	if p.Workspace != nil {
		output.Workspace = &workspaceCompactOut{GID: p.Workspace.GID, Name: p.Workspace.Name}
	}

	if runtimeOutputJSON(cmd) {
		encoder := json.NewEncoder(cmd.OutOrStdout())
		encoder.SetIndent("", "  ")
		return encoder.Encode(output)
	}

	// Text output
	fmt.Fprintf(cmd.OutOrStdout(), "GID:       %s\n", p.GID)
	fmt.Fprintf(cmd.OutOrStdout(), "Name:      %s\n", p.Name)
	if p.Color != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "Color:     %s\n", p.Color)
	}
	if p.Owner != nil {
		fmt.Fprintf(cmd.OutOrStdout(), "Owner:     %s (%s)\n", p.Owner.Name, p.Owner.GID)
	}
	if p.Workspace != nil {
		fmt.Fprintf(cmd.OutOrStdout(), "Workspace: %s (%s)\n", p.Workspace.Name, p.Workspace.GID)
	}
	if p.CreatedAt != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "Created:   %s\n", p.CreatedAt)
	}

	return nil
}

func runPortfolioCreate(cmd *cobra.Command, args []string) error {
	name := args[0]
	workspaceName, _ := cmd.Flags().GetString("workspace")
	workspaceGID, _ := cmd.Flags().GetString("workspace-gid")
	color, _ := cmd.Flags().GetString("color")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	if workspaceName != "" && workspaceGID != "" {
		return fmt.Errorf("use only one of --workspace or --workspace-gid")
	}

	resolvedWorkspace, err := resolveWorkspaceFromFlags(cmd, workspaceName, workspaceGID)
	if err != nil {
		return err
	}

	if dryRun {
		output := portfolioWriteOutput{
			Action: "created",
			DryRun: true,
			Name:   name,
		}
		if runtimeOutputJSON(cmd) {
			encoder := json.NewEncoder(cmd.OutOrStdout())
			encoder.SetIndent("", "  ")
			return encoder.Encode(output)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "[dry-run] Would create portfolio %q in workspace %s\n", name, resolvedWorkspace)
		return nil
	}

	data := map[string]any{
		"workspace": resolvedWorkspace,
		"name":      name,
	}
	if color != "" {
		data["color"] = color
	}

	payload, err := runtimeClient(cmd).Post(cmd.Context(), "/portfolios", api.RequestBody{
		Data: data,
		Options: &api.RequestOptions{
			Fields: []string{"gid", "name"},
		},
	})
	if err != nil {
		return err
	}

	var response api.Response[api.Portfolio]
	if err := json.Unmarshal(payload, &response); err != nil {
		return &api.ResponseError{Err: err}
	}

	p := response.Data
	output := portfolioWriteOutput{
		Action:    "created",
		DryRun:    false,
		Portfolio: &gidRef{GID: p.GID},
		Name:      p.Name,
	}

	if runtimeOutputJSON(cmd) {
		encoder := json.NewEncoder(cmd.OutOrStdout())
		encoder.SetIndent("", "  ")
		return encoder.Encode(output)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Created portfolio: %s (%s)\n", p.Name, p.GID)
	return nil
}

func runPortfolioDelete(cmd *cobra.Command, args []string) error {
	portfolioGID := args[0]
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	if err := validateGID(portfolioGID, "portfolio gid"); err != nil {
		return err
	}

	if dryRun {
		query := url.Values{}
		query.Set("opt_fields", "gid,name")
		payload, err := runtimeClient(cmd).Get(cmd.Context(), "/portfolios/"+portfolioGID, query)
		if err != nil {
			return err
		}

		var response api.Response[api.Portfolio]
		if err := json.Unmarshal(payload, &response); err != nil {
			return &api.ResponseError{Err: err}
		}

		output := portfolioWriteOutput{
			Action:    "deleted",
			DryRun:    true,
			Portfolio: &gidRef{GID: portfolioGID},
			Name:      response.Data.Name,
		}
		if runtimeOutputJSON(cmd) {
			encoder := json.NewEncoder(cmd.OutOrStdout())
			encoder.SetIndent("", "  ")
			return encoder.Encode(output)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "[dry-run] Would delete portfolio %s (%s)\n", response.Data.Name, portfolioGID)
		return nil
	}

	if err := runtimeClient(cmd).Delete(cmd.Context(), "/portfolios/"+portfolioGID); err != nil {
		return err
	}

	output := portfolioWriteOutput{
		Action:    "deleted",
		DryRun:    false,
		Portfolio: &gidRef{GID: portfolioGID},
	}
	if runtimeOutputJSON(cmd) {
		encoder := json.NewEncoder(cmd.OutOrStdout())
		encoder.SetIndent("", "  ")
		return encoder.Encode(output)
	}

	fmt.Fprintln(cmd.OutOrStdout(), "Portfolio deleted")
	return nil
}
