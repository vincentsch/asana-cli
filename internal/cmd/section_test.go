// Package cmd tests section command behavior.
package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/vincentsch/asana-cli/internal/api"
	"github.com/vincentsch/rungrad/testutil"
)

func TestSectionListRequiresProject(t *testing.T) {
	cmd, _ := newSectionListTestCmd()
	err := runSectionList(cmd, nil)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if err.Error() != "either --project or --project-gid is required" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSectionListJSONSortingAndPagination(t *testing.T) {
	var calls int
	srv := testutil.MockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/projects/p1/sections" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
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

	cmd, buf := newSectionListTestCmd()
	if err := cmd.Flags().Set("project-gid", "p1"); err != nil {
		t.Fatalf("failed to set flag: %v", err)
	}

	if err := runSectionList(cmd, nil); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if calls != 2 {
		t.Fatalf("expected 2 calls, got %d", calls)
	}

	var sections []api.Section
	if err := json.Unmarshal(buf.Bytes(), &sections); err != nil {
		t.Fatalf("failed to parse output: %v", err)
	}
	if len(sections) != 3 {
		t.Fatalf("expected 3 sections, got %d", len(sections))
	}
	if sections[0].GID != "1" || sections[1].GID != "3" || sections[2].GID != "2" {
		t.Fatalf("unexpected sort order: %#v", sections)
	}
}

func TestSectionListTableOutput(t *testing.T) {
	srv := testutil.MockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":[{"gid":"1","name":"To-do"}]}`))
	}))

	client := newTestClient(srv)
	apiClient = client
	defer func() { apiClient = nil }()

	outputJSON = false

	cmd, buf := newSectionListTestCmd()
	if err := cmd.Flags().Set("project-gid", "p1"); err != nil {
		t.Fatalf("failed to set flag: %v", err)
	}

	if err := runSectionList(cmd, nil); err != nil {
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
	if len(row) < 2 || row[0] != "To-do" || row[1] != "1" {
		t.Fatalf("unexpected row %q", lines[1])
	}
}

func newSectionListTestCmd() (*cobra.Command, *bytes.Buffer) {
	var buf bytes.Buffer
	cmd := &cobra.Command{}
	attachTestRuntime(cmd)
	cmd.SetOut(&buf)
	cmd.Flags().String("project", "", "")
	cmd.Flags().String("project-gid", "", "")
	cmd.Flags().String("workspace", "", "")
	cmd.Flags().String("workspace-gid", "", "")
	return cmd, &buf
}
