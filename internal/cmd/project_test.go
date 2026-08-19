// Package cmd tests project command behavior.
package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/vincentsch/asana-cli/internal/api"
	"github.com/vincentsch/rungrad/testutil"
)

func TestProjectListRequiresWorkspace(t *testing.T) {
	cmd, _ := newProjectListTestCmd()
	err := runProjectList(cmd, nil)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if err.Error() != "either --workspace or --workspace-gid is required" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestProjectListJSONSortingAndPagination(t *testing.T) {
	var calls int
	var seenWorkspace []string

	srv := testutil.MockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/projects" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		seenWorkspace = append(seenWorkspace, r.URL.Query().Get("workspace"))
		switch r.URL.Query().Get("offset") {
		case "":
			calls++
			_, _ = w.Write([]byte(`{"data":[{"gid":"2","name":"beta"},{"gid":"1","name":"Alpha"}],"next_page":{"offset":"next"}}`))
		case "next":
			calls++
			_, _ = w.Write([]byte(`{"data":[{"gid":"3","name":"alpha"}]}`))
		default:
			t.Fatalf("unexpected offset %q", r.URL.Query().Get("offset"))
		}
	}))

	client := newTestClient(srv)
	apiClient = client
	defer func() { apiClient = nil }()

	outputJSON = true
	defer func() { outputJSON = false }()

	cmd, buf := newProjectListTestCmd()
	if err := cmd.Flags().Set("workspace-gid", "w1"); err != nil {
		t.Fatalf("failed to set flag: %v", err)
	}

	if err := runProjectList(cmd, nil); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if calls != 2 {
		t.Fatalf("expected 2 calls, got %d", calls)
	}
	for _, seen := range seenWorkspace {
		if seen != "w1" {
			t.Fatalf("expected workspace w1, got %q", seen)
		}
	}

	var projects []api.Project
	if err := json.Unmarshal(buf.Bytes(), &projects); err != nil {
		t.Fatalf("failed to parse output: %v", err)
	}
	if len(projects) != 3 {
		t.Fatalf("expected 3 projects, got %d", len(projects))
	}
	if projects[0].GID != "1" || projects[1].GID != "3" || projects[2].GID != "2" {
		t.Fatalf("unexpected sort order: %#v", projects)
	}
}

func TestProjectListTableOutput(t *testing.T) {
	srv := testutil.MockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":[{"gid":"1","name":"Alpha"}]}`))
	}))

	client := newTestClient(srv)
	apiClient = client
	defer func() { apiClient = nil }()

	outputJSON = false

	cmd, buf := newProjectListTestCmd()
	if err := cmd.Flags().Set("workspace-gid", "w1"); err != nil {
		t.Fatalf("failed to set flag: %v", err)
	}

	if err := runProjectList(cmd, nil); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) < 2 {
		t.Fatalf("expected at least 2 lines, got %q", buf.String())
	}
	header := strings.Fields(lines[0])
	if len(header) < 2 || header[0] != "NAME" || header[1] != "GID" {
		t.Fatalf("unexpected header %q", lines[0])
	}
	row := strings.Fields(lines[1])
	if len(row) < 2 || row[0] != "Alpha" || row[1] != "1" {
		t.Fatalf("unexpected row %q", lines[1])
	}
}

func TestProjectListUsesTeamFilter(t *testing.T) {
	var sawTeamQuery bool
	var sawWorkspaceQuery bool

	srv := testutil.MockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/workspaces/w1/teams":
			_, _ = w.Write([]byte(`{"data":[{"gid":"t1","name":"Core"}]}`))
		case "/projects":
			if r.URL.Query().Get("team") == "t1" {
				sawTeamQuery = true
			}
			if r.URL.Query().Get("workspace") != "" {
				sawWorkspaceQuery = true
			}
			_, _ = w.Write([]byte(`{"data":[{"gid":"1","name":"Alpha"}]}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))

	client := newTestClient(srv)
	apiClient = client
	defer func() { apiClient = nil }()

	outputJSON = true
	defer func() { outputJSON = false }()

	cmd, _ := newProjectListTestCmd()
	if err := cmd.Flags().Set("workspace-gid", "w1"); err != nil {
		t.Fatalf("failed to set flag: %v", err)
	}
	if err := cmd.Flags().Set("team", "Core"); err != nil {
		t.Fatalf("failed to set flag: %v", err)
	}

	if err := runProjectList(cmd, nil); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !sawTeamQuery {
		t.Fatalf("expected team query param")
	}
	if sawWorkspaceQuery {
		t.Fatalf("did not expect workspace query param when team filter is used")
	}
}

func TestProjectListIgnoresInvalidConfigWhenWorkspaceProvided(t *testing.T) {
	srv := testutil.MockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/projects" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if r.URL.Query().Get("workspace") != "w1" {
			t.Fatalf("expected workspace w1, got %q", r.URL.Query().Get("workspace"))
		}
		_, _ = w.Write([]byte(`{"data":[{"gid":"1","name":"Alpha"}]}`))
	}))

	apiClient = newTestClient(srv)
	defer func() { apiClient = nil }()

	configDir := t.TempDir()
	invalidPath := filepath.Join(configDir, "config.json")
	if err := os.WriteFile(invalidPath, []byte("{invalid"), 0o644); err != nil {
		t.Fatalf("failed to write invalid config: %v", err)
	}
	prevConfigPath := configPath
	configPath = invalidPath
	defer func() { configPath = prevConfigPath }()

	cmd, _ := newProjectListTestCmd()
	if err := cmd.Flags().Set("workspace-gid", "w1"); err != nil {
		t.Fatalf("failed to set flag: %v", err)
	}

	if err := runProjectList(cmd, nil); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func newProjectListTestCmd() (*cobra.Command, *bytes.Buffer) {
	var buf bytes.Buffer
	cmd := &cobra.Command{}
	attachTestRuntime(cmd)
	cmd.SetOut(&buf)
	cmd.Flags().String("workspace", "", "")
	cmd.Flags().String("workspace-gid", "", "")
	cmd.Flags().String("team", "", "")
	cmd.Flags().String("team-gid", "", "")
	return cmd, &buf
}
