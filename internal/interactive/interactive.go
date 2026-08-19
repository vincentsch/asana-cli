// Package interactive provides TTY detection and prompt helpers.
package interactive

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"sort"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/vincentsch/asana-cli/internal/api"
	"golang.org/x/term"
)

const nameFields = "gid,name"

// IsInteractive returns true if stdin and stdout are TTYs and prompts are enabled.
func IsInteractive(noPrompt bool) bool {
	if noPrompt {
		return false
	}
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return false
	}
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		return false
	}
	return true
}

// SelectWorkspace prompts user to select from available workspaces.
func SelectWorkspace(ctx context.Context, client *api.Client) (*api.Workspace, error) {
	query := url.Values{}
	query.Set("opt_fields", nameFields)
	workspaces, err := api.Paginate[api.Workspace](ctx, client, "/workspaces", query)
	if err != nil {
		return nil, err
	}
	if len(workspaces) == 0 {
		return nil, errors.New("no workspaces found")
	}
	sort.Slice(workspaces, func(i, j int) bool {
		left := strings.ToLower(workspaces[i].Name)
		right := strings.ToLower(workspaces[j].Name)
		if left != right {
			return left < right
		}
		return workspaces[i].GID < workspaces[j].GID
	})

	selected, err := Disambiguate("Select workspace", workspaces, func(item api.Workspace) string {
		return fmt.Sprintf("%s (%s)", item.Name, item.GID)
	})
	if err != nil {
		return nil, err
	}
	return &selected, nil
}

// SelectProject prompts user to select from available projects in workspace.
func SelectProject(ctx context.Context, client *api.Client, workspaceGID string) (*api.Project, error) {
	if strings.TrimSpace(workspaceGID) == "" {
		return nil, errors.New("workspace gid is required to select project")
	}
	query := url.Values{}
	query.Set("workspace", workspaceGID)
	query.Set("opt_fields", nameFields)
	projects, err := api.Paginate[api.Project](ctx, client, "/projects", query)
	if err != nil {
		return nil, err
	}
	if len(projects) == 0 {
		return nil, errors.New("no projects found in workspace")
	}
	sort.Slice(projects, func(i, j int) bool {
		left := strings.ToLower(projects[i].Name)
		right := strings.ToLower(projects[j].Name)
		if left != right {
			return left < right
		}
		return projects[i].GID < projects[j].GID
	})

	selected, err := Disambiguate("Select project", projects, func(item api.Project) string {
		return fmt.Sprintf("%s (%s)", item.Name, item.GID)
	})
	if err != nil {
		return nil, err
	}
	return &selected, nil
}

// SelectSection prompts user to select from available sections in project.
func SelectSection(ctx context.Context, client *api.Client, projectGID string) (*api.Section, error) {
	if strings.TrimSpace(projectGID) == "" {
		return nil, errors.New("project gid is required to select section")
	}
	query := url.Values{}
	query.Set("opt_fields", nameFields)
	sections, err := api.Paginate[api.Section](ctx, client, "/projects/"+projectGID+"/sections", query)
	if err != nil {
		return nil, err
	}
	if len(sections) == 0 {
		return nil, errors.New("no sections found in project")
	}
	sort.Slice(sections, func(i, j int) bool {
		left := strings.ToLower(sections[i].Name)
		right := strings.ToLower(sections[j].Name)
		if left != right {
			return left < right
		}
		return sections[i].GID < sections[j].GID
	})

	selected, err := Disambiguate("Select section", sections, func(item api.Section) string {
		return fmt.Sprintf("%s (%s)", item.Name, item.GID)
	})
	if err != nil {
		return nil, err
	}
	return &selected, nil
}

// Disambiguate prompts user to select from multiple matches.
func Disambiguate[T comparable](title string, matches []T, display func(T) string) (T, error) {
	var zero T
	if len(matches) == 0 {
		return zero, errors.New("no options available")
	}
	options := make([]huh.Option[T], 0, len(matches))
	for _, match := range matches {
		options = append(options, huh.NewOption(display(match), match))
	}

	var selection T
	selectInput := huh.NewSelect[T]().
		Title(title).
		Options(options...).
		Value(&selection)
	form := huh.NewForm(huh.NewGroup(selectInput))
	if err := form.Run(); err != nil {
		return zero, err
	}
	return selection, nil
}

// PromptToken prompts for masked token input.
func PromptToken() (string, error) {
	var token string
	input := huh.NewInput().
		Title("Asana token").
		Prompt("Token: ").
		EchoMode(huh.EchoModePassword).
		Value(&token)
	form := huh.NewForm(huh.NewGroup(input))
	if err := form.Run(); err != nil {
		return "", err
	}
	token = strings.TrimSpace(token)
	if token == "" {
		return "", errors.New("token cannot be empty")
	}
	return token, nil
}
