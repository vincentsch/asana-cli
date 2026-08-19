// Package cmd tests user command behavior.
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

func TestUserListRequiresWorkspace(t *testing.T) {
	cmd, _ := newUserListTestCmd()
	if err := runUserList(cmd, nil); err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestUserListJSONIncludesWorkspacesAndSorts(t *testing.T) {
	srv := testutil.MockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/workspaces/w1/users" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		fields := r.URL.Query().Get("opt_fields")
		if !strings.Contains(fields, "workspaces.gid") || !strings.Contains(fields, "workspaces.name") {
			t.Fatalf("expected workspace fields, got %q", fields)
		}
		_, _ = w.Write([]byte(`{"data":[{"gid":"2","name":"beta","email":"b@example.com","workspaces":[{"gid":"w1","name":"Acme"}]},{"gid":"1","name":"Alpha","email":"a@example.com","workspaces":[{"gid":"w2","name":"Globex"}]}]}`))
	}))

	apiClient = newTestClient(srv)
	defer func() { apiClient = nil }()

	outputJSON = true
	defer func() { outputJSON = false }()

	cmd, buf := newUserListTestCmd()
	if err := cmd.Flags().Set("workspace-gid", "w1"); err != nil {
		t.Fatalf("failed to set flag: %v", err)
	}

	if err := runUserList(cmd, nil); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	var users []userListItem
	if err := json.Unmarshal(buf.Bytes(), &users); err != nil {
		t.Fatalf("failed to parse output: %v", err)
	}
	if len(users) != 2 {
		t.Fatalf("expected 2 users, got %d", len(users))
	}
	if users[0].GID != "1" || users[1].GID != "2" {
		t.Fatalf("unexpected sort order: %#v", users)
	}
	if len(users[0].Workspaces) != 1 || users[0].Workspaces[0].GID != "w2" {
		t.Fatalf("expected workspaces for first user, got %#v", users[0].Workspaces)
	}
}

func newUserListTestCmd() (*cobra.Command, *bytes.Buffer) {
	var buf bytes.Buffer
	cmd := &cobra.Command{}
	attachTestRuntime(cmd)
	cmd.SetOut(&buf)
	cmd.Flags().String("workspace", "", "")
	cmd.Flags().String("workspace-gid", "", "")
	cmd.Flags().Int("limit", 0, "")
	return cmd, &buf
}
