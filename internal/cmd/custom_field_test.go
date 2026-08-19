// Package cmd tests custom field command behavior.
package cmd

import (
	"bytes"
	"net/http"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/vincentsch/rungrad/testutil"
)

func TestCustomFieldCreateAllowsReferenceType(t *testing.T) {
	cmd, _ := newCustomFieldCreateTestCmd()
	if err := cmd.Flags().Set("workspace-gid", "w1"); err != nil {
		t.Fatalf("failed to set flag: %v", err)
	}
	if err := cmd.Flags().Set("type", "reference"); err != nil {
		t.Fatalf("failed to set flag: %v", err)
	}
	if err := cmd.Flags().Set("dry-run", "true"); err != nil {
		t.Fatalf("failed to set flag: %v", err)
	}

	if err := runCustomFieldCreate(cmd, []string{"Ref"}); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestCustomFieldListPremiumError(t *testing.T) {
	srv := testutil.MockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusPaymentRequired)
		_, _ = w.Write([]byte(`{"errors":[{"message":"Payment required"}]}`))
	}))

	apiClient = newTestClient(srv)
	defer func() { apiClient = nil }()

	cmd, _ := newCustomFieldListTestCmd()
	if err := cmd.Flags().Set("workspace-gid", "w1"); err != nil {
		t.Fatalf("failed to set flag: %v", err)
	}

	err := runCustomFieldList(cmd, nil)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "custom fields require a premium workspace") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func newCustomFieldListTestCmd() (*cobra.Command, *bytes.Buffer) {
	var buf bytes.Buffer
	cmd := &cobra.Command{}
	attachTestRuntime(cmd)
	cmd.SetOut(&buf)
	cmd.Flags().String("workspace", "", "")
	cmd.Flags().String("workspace-gid", "", "")
	cmd.Flags().Int("limit", 0, "")
	return cmd, &buf
}

func newCustomFieldCreateTestCmd() (*cobra.Command, *bytes.Buffer) {
	var buf bytes.Buffer
	cmd := &cobra.Command{}
	attachTestRuntime(cmd)
	cmd.SetOut(&buf)
	cmd.Flags().String("workspace", "", "")
	cmd.Flags().String("workspace-gid", "", "")
	cmd.Flags().String("type", "", "")
	cmd.Flags().Int("precision", 0, "")
	cmd.Flags().StringSlice("enum-options", nil, "")
	cmd.Flags().String("description", "", "")
	cmd.Flags().Bool("dry-run", false, "")
	return cmd, &buf
}
