// Package cmd tests team member command behavior.
package cmd

import (
	"bytes"
	"net/http"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/vincentsch/rungrad/testutil"
)

func TestTeamMemberAddDryRunValidatesTeam(t *testing.T) {
	srv := testutil.MockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/teams/123" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"errors":[{"message":"Not found"}]}`))
	}))

	apiClient = newTestClient(srv)
	defer func() { apiClient = nil }()

	cmd, _ := newTeamMemberAddTestCmd()
	if err := cmd.Flags().Set("user", "me"); err != nil {
		t.Fatalf("failed to set user: %v", err)
	}
	if err := cmd.Flags().Set("workspace-gid", "1"); err != nil {
		t.Fatalf("failed to set workspace: %v", err)
	}
	if err := cmd.Flags().Set("dry-run", "true"); err != nil {
		t.Fatalf("failed to set dry-run: %v", err)
	}

	err := runTeamMemberAdd(cmd, []string{"123"})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "API error") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTeamMemberRemoveDryRunValidatesTeam(t *testing.T) {
	srv := testutil.MockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/teams/123" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"errors":[{"message":"Not found"}]}`))
	}))

	apiClient = newTestClient(srv)
	defer func() { apiClient = nil }()

	cmd, _ := newTeamMemberRemoveTestCmd()
	if err := cmd.Flags().Set("user", "me"); err != nil {
		t.Fatalf("failed to set user: %v", err)
	}
	if err := cmd.Flags().Set("workspace-gid", "1"); err != nil {
		t.Fatalf("failed to set workspace: %v", err)
	}
	if err := cmd.Flags().Set("dry-run", "true"); err != nil {
		t.Fatalf("failed to set dry-run: %v", err)
	}

	err := runTeamMemberRemove(cmd, []string{"123"})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "API error") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func newTeamMemberAddTestCmd() (*cobra.Command, *bytes.Buffer) {
	var buf bytes.Buffer
	cmd := &cobra.Command{}
	attachTestRuntime(cmd)
	cmd.SetOut(&buf)
	cmd.Flags().StringSlice("user", nil, "")
	cmd.Flags().String("workspace", "", "")
	cmd.Flags().String("workspace-gid", "", "")
	cmd.Flags().Bool("dry-run", false, "")
	return cmd, &buf
}

func newTeamMemberRemoveTestCmd() (*cobra.Command, *bytes.Buffer) {
	var buf bytes.Buffer
	cmd := &cobra.Command{}
	attachTestRuntime(cmd)
	cmd.SetOut(&buf)
	cmd.Flags().StringSlice("user", nil, "")
	cmd.Flags().String("workspace", "", "")
	cmd.Flags().String("workspace-gid", "", "")
	cmd.Flags().Bool("dry-run", false, "")
	return cmd, &buf
}
