// Package cmd provides shared helpers for command implementations.
package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/vincentsch/asana-cli/internal/api"
	"github.com/vincentsch/asana-cli/internal/interactive"
	"github.com/vincentsch/asana-cli/internal/resolve"
)

func resolveWorkspaceWithPrompt(ctx context.Context, client *api.Client, nameOrGID string, noPrompt bool) (string, error) {
	if resolve.IsGID(nameOrGID) {
		return nameOrGID, nil
	}
	gid, err := resolve.Workspace(ctx, client, nameOrGID)
	if err == nil {
		return gid, nil
	}
	var ambiguous *resolve.AmbiguousError
	if errors.As(err, &ambiguous) && interactive.IsInteractive(noPrompt) {
		selected, promptErr := interactive.Disambiguate("Select workspace", ambiguous.Matches, formatMatchLabel)
		if promptErr != nil {
			return "", promptErr
		}
		return selected.GID, nil
	}
	return "", err
}

func resolveProjectWithPrompt(ctx context.Context, client *api.Client, workspaceGID, nameOrGID string, noPrompt bool) (string, error) {
	if resolve.IsGID(nameOrGID) {
		return nameOrGID, nil
	}
	gid, err := resolve.Project(ctx, client, workspaceGID, nameOrGID)
	if err == nil {
		return gid, nil
	}
	var ambiguous *resolve.AmbiguousError
	if errors.As(err, &ambiguous) && interactive.IsInteractive(noPrompt) {
		selected, promptErr := interactive.Disambiguate("Select project", ambiguous.Matches, formatMatchLabel)
		if promptErr != nil {
			return "", promptErr
		}
		return selected.GID, nil
	}
	return "", err
}

func resolveSectionWithPrompt(ctx context.Context, client *api.Client, projectGID, nameOrGID string, noPrompt bool) (string, error) {
	if resolve.IsGID(nameOrGID) {
		return nameOrGID, nil
	}
	gid, err := resolve.Section(ctx, client, projectGID, nameOrGID)
	if err == nil {
		return gid, nil
	}
	var ambiguous *resolve.AmbiguousError
	if errors.As(err, &ambiguous) && interactive.IsInteractive(noPrompt) {
		selected, promptErr := interactive.Disambiguate("Select section", ambiguous.Matches, formatMatchLabel)
		if promptErr != nil {
			return "", promptErr
		}
		return selected.GID, nil
	}
	return "", err
}

func resolveTagWithPrompt(ctx context.Context, client *api.Client, workspaceGID, nameOrGID string, noPrompt bool) (string, error) {
	if resolve.IsGID(nameOrGID) {
		return nameOrGID, nil
	}
	gid, err := resolve.Tag(ctx, client, workspaceGID, nameOrGID)
	if err == nil {
		return gid, nil
	}
	var ambiguous *resolve.AmbiguousError
	if errors.As(err, &ambiguous) && interactive.IsInteractive(noPrompt) {
		selected, promptErr := interactive.Disambiguate("Select tag", ambiguous.Matches, formatMatchLabel)
		if promptErr != nil {
			return "", promptErr
		}
		return selected.GID, nil
	}
	return "", err
}

func resolveTeamWithPrompt(ctx context.Context, client *api.Client, workspaceGID, nameOrGID string, noPrompt bool) (string, error) {
	if resolve.IsGID(nameOrGID) {
		return nameOrGID, nil
	}
	gid, err := resolve.Team(ctx, client, workspaceGID, nameOrGID)
	if err == nil {
		return gid, nil
	}
	var ambiguous *resolve.AmbiguousError
	if errors.As(err, &ambiguous) && interactive.IsInteractive(noPrompt) {
		selected, promptErr := interactive.Disambiguate("Select team", ambiguous.Matches, formatMatchLabel)
		if promptErr != nil {
			return "", promptErr
		}
		return selected.GID, nil
	}
	return "", err
}

func formatMatchLabel(match resolve.Match) string {
	if match.Context != "" {
		return fmt.Sprintf("%s (%s) - %s", match.Name, match.GID, match.Context)
	}
	return fmt.Sprintf("%s (%s)", match.Name, match.GID)
}

func resolveUserIdentifiers(ctx context.Context, workspaceGID string, identifiers []string) ([]string, error) {
	if len(identifiers) == 0 {
		return nil, fmt.Errorf("user list cannot be empty")
	}

	resolved := make([]string, 0, len(identifiers))
	for _, identifier := range identifiers {
		identifier = strings.TrimSpace(identifier)
		if identifier == "" {
			return nil, fmt.Errorf("user identifier cannot be empty")
		}
		if strings.EqualFold(identifier, "me") {
			user, err := fetchCurrentUser(ctx)
			if err != nil {
				return nil, err
			}
			resolved = append(resolved, user.GID)
			continue
		}
		if resolve.IsGID(identifier) {
			resolved = append(resolved, identifier)
			continue
		}
		if !strings.Contains(identifier, "@") {
			return nil, fmt.Errorf("invalid user identifier %q: use me, a GID, or an email address", identifier)
		}
		if workspaceGID == "" {
			return nil, fmt.Errorf("workspace gid is required to resolve user email %q", identifier)
		}
		gid, err := resolve.UserInWorkspace(ctx, runtimeFromContext(ctx).client, workspaceGID, identifier)
		if err != nil {
			return nil, err
		}
		resolved = append(resolved, gid)
	}
	return resolved, nil
}

// requireOrganizationWorkspace validates that the workspace is an organization.
// Team commands require organization workspaces.
func requireOrganizationWorkspace(ctx context.Context, workspaceGID string) error {
	query := url.Values{}
	query.Set("opt_fields", "gid,name,is_organization")

	payload, err := runtimeFromContext(ctx).client.Get(ctx, "/workspaces/"+workspaceGID, query)
	if err != nil {
		return err
	}

	var response api.Response[api.WorkspaceDetail]
	if err := json.Unmarshal(payload, &response); err != nil {
		return &api.ResponseError{Err: err}
	}

	if !response.Data.IsOrganization {
		return fmt.Errorf("teams require an organization workspace; %q is not an organization", response.Data.Name)
	}

	return nil
}

// formatPremiumError provides user-friendly message for 402 Payment Required errors.
func formatPremiumError(err error, feature string) error {
	var apiErr *api.APIError
	if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusPaymentRequired {
		return fmt.Errorf("%s require a premium workspace: %w", feature, err)
	}
	return err
}
