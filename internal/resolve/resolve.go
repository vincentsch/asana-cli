// Package resolve handles name-to-GID lookups with disambiguation support.
package resolve

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strings"

	"github.com/vincentsch/asana-cli/internal/api"
	rungradresolve "github.com/vincentsch/rungrad/resolve"
)

const fieldsGIDName = "gid,name"

// AmbiguousError indicates multiple resources matched a name.
type AmbiguousError struct {
	ResourceType string
	Name         string
	Matches      []Match
}

// Match holds a single resource's identifying info.
type Match struct {
	GID     string
	Name    string
	Context string
}

func (e *AmbiguousError) Error() string {
	if e == nil {
		return ""
	}

	var builder strings.Builder
	fmt.Fprintf(&builder, "Ambiguous %s name %q. Candidates (use --%s-gid to specify):\n", e.ResourceType, e.Name, e.ResourceType)
	for _, match := range e.Matches {
		if match.Context != "" {
			fmt.Fprintf(&builder, "  %s  %s (%s)\n", match.GID, match.Name, match.Context)
			continue
		}
		fmt.Fprintf(&builder, "  %s  %s\n", match.GID, match.Name)
	}
	return strings.TrimRight(builder.String(), "\n")
}

// NotFoundError indicates no resource matched a name.
type NotFoundError struct {
	ResourceType string
	Name         string
	Available    []string
}

func (e *NotFoundError) Error() string {
	if e == nil {
		return ""
	}
	base := fmt.Sprintf("%s %q not found", e.ResourceType, e.Name)
	if len(e.Available) == 0 {
		return base
	}
	return fmt.Sprintf("%s. Available: %s", base, strings.Join(e.Available, ", "))
}

// IsGID returns true if the string looks like an Asana GID (all digits).
func IsGID(s string) bool {
	return rungradresolve.IsNumericID(s)
}

// Workspace resolves a workspace name or GID to a GID.
func Workspace(ctx context.Context, client *api.Client, nameOrGID string) (string, error) {
	if IsGID(nameOrGID) {
		return nameOrGID, nil
	}

	query := url.Values{}
	query.Set("opt_fields", fieldsGIDName)
	workspaces, err := api.Paginate[api.Workspace](ctx, client, "/workspaces", query)
	if err != nil {
		return "", err
	}

	matches := make([]Match, 0)
	for _, workspace := range workspaces {
		if strings.EqualFold(workspace.Name, nameOrGID) {
			matches = append(matches, Match{GID: workspace.GID, Name: workspace.Name})
		}
	}

	available := make([]string, 0, len(workspaces))
	for _, workspace := range workspaces {
		available = append(available, workspace.Name)
	}

	return resolveMatches("workspace", nameOrGID, matches, available)
}

// Project resolves a project name or GID to a GID.
func Project(ctx context.Context, client *api.Client, workspaceGID, nameOrGID string) (string, error) {
	if IsGID(nameOrGID) {
		return nameOrGID, nil
	}
	if workspaceGID != "" {
		query := url.Values{}
		query.Set("workspace", workspaceGID)
		query.Set("opt_fields", fieldsGIDName)
		projects, err := api.Paginate[api.Project](ctx, client, "/projects", query)
		if err != nil {
			return "", err
		}

		matches := make([]Match, 0)
		available := make([]string, 0, len(projects))
		for _, project := range projects {
			available = append(available, project.Name)
			if strings.EqualFold(project.Name, nameOrGID) {
				matches = append(matches, Match{GID: project.GID, Name: project.Name})
			}
		}
		return resolveMatches("project", nameOrGID, matches, available)
	}

	query := url.Values{}
	query.Set("opt_fields", fieldsGIDName)
	workspaces, err := api.Paginate[api.Workspace](ctx, client, "/workspaces", query)
	if err != nil {
		return "", err
	}

	matches := make([]Match, 0)
	available := make([]string, 0)
	for _, workspace := range workspaces {
		projectQuery := url.Values{}
		projectQuery.Set("workspace", workspace.GID)
		projectQuery.Set("opt_fields", fieldsGIDName)
		projects, err := api.Paginate[api.Project](ctx, client, "/projects", projectQuery)
		if err != nil {
			return "", err
		}

		for _, project := range projects {
			available = append(available, project.Name)
			if strings.EqualFold(project.Name, nameOrGID) {
				matches = append(matches, Match{
					GID:     project.GID,
					Name:    project.Name,
					Context: fmt.Sprintf("Workspace: %s", workspace.Name),
				})
			}
		}
	}

	return resolveMatches("project", nameOrGID, matches, available)
}

// Section resolves a section name or GID to a GID within a project.
func Section(ctx context.Context, client *api.Client, projectGID, nameOrGID string) (string, error) {
	if IsGID(nameOrGID) {
		return nameOrGID, nil
	}
	if projectGID == "" {
		return "", fmt.Errorf("project gid is required to resolve section %q", nameOrGID)
	}

	query := url.Values{}
	query.Set("opt_fields", fieldsGIDName)
	sections, err := api.Paginate[api.Section](ctx, client, "/projects/"+projectGID+"/sections", query)
	if err != nil {
		return "", err
	}

	matches := make([]Match, 0)
	available := make([]string, 0, len(sections))
	for _, section := range sections {
		available = append(available, section.Name)
		if strings.EqualFold(section.Name, nameOrGID) {
			matches = append(matches, Match{GID: section.GID, Name: section.Name})
		}
	}

	return resolveMatches("section", nameOrGID, matches, available)
}

// Team resolves a team name or GID to a GID within a workspace.
func Team(ctx context.Context, client *api.Client, workspaceGID, nameOrGID string) (string, error) {
	if IsGID(nameOrGID) {
		return nameOrGID, nil
	}
	if workspaceGID == "" {
		return "", fmt.Errorf("workspace gid is required to resolve team %q", nameOrGID)
	}

	query := url.Values{}
	query.Set("opt_fields", fieldsGIDName)
	teams, err := api.Paginate[api.Team](ctx, client, "/workspaces/"+workspaceGID+"/teams", query)
	if err != nil {
		return "", err
	}

	matches := make([]Match, 0)
	for _, team := range teams {
		if strings.EqualFold(team.Name, nameOrGID) {
			matches = append(matches, Match{GID: team.GID, Name: team.Name})
		}
	}

	return resolveMatches("team", nameOrGID, matches, nil)
}

// Tag resolves a tag name or GID to a GID within a workspace.
func Tag(ctx context.Context, client *api.Client, workspaceGID, nameOrGID string) (string, error) {
	if IsGID(nameOrGID) {
		return nameOrGID, nil
	}
	if workspaceGID == "" {
		return "", fmt.Errorf("workspace gid is required to resolve tag %q", nameOrGID)
	}

	query := url.Values{}
	query.Set("opt_fields", fieldsGIDName)
	tags, err := api.Paginate[api.Tag](ctx, client, "/workspaces/"+workspaceGID+"/tags", query)
	if err != nil {
		return "", err
	}

	matches := make([]Match, 0)
	available := make([]string, 0, len(tags))
	for _, tag := range tags {
		available = append(available, tag.Name)
		if strings.EqualFold(tag.Name, nameOrGID) {
			matches = append(matches, Match{GID: tag.GID, Name: tag.Name})
		}
	}

	return resolveMatches("tag", nameOrGID, matches, available)
}

// UserInWorkspace resolves a user identifier (me, GID, or email) to a GID.
// For "me", returns "me" (API handles it). For email, fetches workspace users.
func UserInWorkspace(ctx context.Context, client *api.Client, workspaceGID, identifier string) (string, error) {
	identifier = strings.TrimSpace(identifier)
	if identifier == "" {
		return "", fmt.Errorf("user identifier cannot be empty")
	}

	if strings.EqualFold(identifier, "me") {
		return "me", nil
	}

	if IsGID(identifier) {
		return identifier, nil
	}

	// Must be an email - require workspace scope
	if !strings.Contains(identifier, "@") {
		return "", fmt.Errorf("invalid user identifier %q: use me, a GID, or an email address", identifier)
	}

	if workspaceGID == "" {
		return "", fmt.Errorf("workspace gid is required to resolve user email %q", identifier)
	}

	query := url.Values{}
	query.Set("opt_fields", "gid,email")
	users, err := api.Paginate[api.User](ctx, client, "/workspaces/"+workspaceGID+"/users", query)
	if err != nil {
		return "", err
	}

	for _, user := range users {
		if strings.EqualFold(user.Email, identifier) {
			return user.GID, nil
		}
	}

	return "", &NotFoundError{
		ResourceType: "user",
		Name:         identifier,
		Available:    nil,
	}
}

func resolveMatches(resourceType, name string, matches []Match, available []string) (string, error) {
	resolved, err := rungradresolve.Resolve(name, func(string) ([]rungradresolve.Match, error) {
		converted := make([]rungradresolve.Match, len(matches))
		for i, match := range matches {
			converted[i] = rungradresolve.Match{ID: match.GID, Name: match.Name, Context: match.Context}
		}
		return converted, nil
	}, rungradresolve.Options{ResourceType: resourceType})
	if err == nil {
		return resolved, nil
	}

	var notFound *rungradresolve.NotFoundError
	if errors.As(err, &notFound) {
		return "", &NotFoundError{
			ResourceType: resourceType,
			Name:         name,
			Available:    normalizeAvailable(available),
		}
	}
	var ambiguous *rungradresolve.AmbiguousError
	if !errors.As(err, &ambiguous) {
		return "", err
	}

	sortMatches(matches)
	return "", &AmbiguousError{
		ResourceType: resourceType,
		Name:         name,
		Matches:      matches,
	}
}

func sortMatches(matches []Match) {
	sort.Slice(matches, func(i, j int) bool {
		leftName := strings.ToLower(matches[i].Name)
		rightName := strings.ToLower(matches[j].Name)
		if leftName != rightName {
			return leftName < rightName
		}
		leftContext := strings.ToLower(matches[i].Context)
		rightContext := strings.ToLower(matches[j].Context)
		if leftContext != rightContext {
			return leftContext < rightContext
		}
		return matches[i].GID < matches[j].GID
	})
}

func normalizeAvailable(names []string) []string {
	if len(names) == 0 {
		return nil
	}
	seen := make(map[string]string)
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		key := strings.ToLower(name)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = name
	}
	unique := make([]string, 0, len(seen))
	for _, name := range seen {
		unique = append(unique, name)
	}
	sort.Slice(unique, func(i, j int) bool {
		left := strings.ToLower(unique[i])
		right := strings.ToLower(unique[j])
		if left != right {
			return left < right
		}
		return unique[i] < unique[j]
	})
	return unique
}
