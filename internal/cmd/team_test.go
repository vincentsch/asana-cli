// Package cmd tests team command behavior.
package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/vincentsch/rungrad/testutil"
)

func TestTeamListRequiresOrganization(t *testing.T) {
	var calls int
	srv := testutil.MockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Path != "/workspaces/w1" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"data":{"gid":"w1","name":"Acme","is_organization":false}}`))
	}))

	apiClient = newTestClient(srv)
	defer func() { apiClient = nil }()

	cmd, _ := newTeamListTestCmd()
	if err := cmd.Flags().Set("workspace-gid", "w1"); err != nil {
		t.Fatalf("failed to set flag: %v", err)
	}

	err := runTeamList(cmd, nil)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "teams require an organization workspace") {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected 1 call, got %d", calls)
	}
}

func TestTeamViewIncludesMembers(t *testing.T) {
	var gotTeam bool
	var gotMembers bool
	srv := testutil.MockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/teams/123":
			gotTeam = true
			_, _ = w.Write([]byte(`{"data":{"gid":"123","name":"Engineering","description":"Eng","visibility":"public","organization":{"gid":"w1","name":"Acme"}}}`))
		case "/teams/123/users":
			gotMembers = true
			_, _ = w.Write([]byte(`{"data":[{"gid":"m2","user":{"gid":"u2","name":"bob","email":"bob@example.com"},"is_admin":false},{"gid":"m1","user":{"gid":"u1","name":"Alice","email":"alice@example.com"},"is_admin":true}]}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))

	apiClient = newTestClient(srv)
	defer func() { apiClient = nil }()

	outputJSON = true
	defer func() { outputJSON = false }()

	cmd, buf := newTeamViewTestCmd()
	if err := runTeamView(cmd, []string{"123"}); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !gotTeam || !gotMembers {
		t.Fatalf("expected both team and members calls, got team=%t members=%t", gotTeam, gotMembers)
	}

	var out teamViewOutput
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("failed to parse output: %v", err)
	}
	if len(out.Members) != 2 {
		t.Fatalf("expected 2 members, got %d", len(out.Members))
	}
	if out.Members[0].GID != "u1" || out.Members[0].Name != "Alice" || !out.Members[0].IsAdmin {
		t.Fatalf("unexpected first member: %#v", out.Members[0])
	}
	if out.Members[1].GID != "u2" || out.Members[1].Name != "bob" {
		t.Fatalf("unexpected second member: %#v", out.Members[1])
	}
}

func newTeamListTestCmd() (*cobra.Command, *bytes.Buffer) {
	var buf bytes.Buffer
	cmd := &cobra.Command{}
	attachTestRuntime(cmd)
	cmd.SetOut(&buf)
	cmd.Flags().String("workspace", "", "")
	cmd.Flags().String("workspace-gid", "", "")
	cmd.Flags().Int("limit", 0, "")
	return cmd, &buf
}

func newTeamViewTestCmd() (*cobra.Command, *bytes.Buffer) {
	var buf bytes.Buffer
	cmd := &cobra.Command{}
	attachTestRuntime(cmd)
	cmd.SetOut(&buf)
	return cmd, &buf
}
