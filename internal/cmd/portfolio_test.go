// Package cmd tests portfolio command behavior.
package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/spf13/cobra"
	"github.com/vincentsch/rungrad/testutil"
)

func TestPortfolioListNoOwnerFilter(t *testing.T) {
	srv := testutil.MockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/portfolios" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if r.URL.Query().Get("workspace") != "w1" {
			t.Fatalf("expected workspace w1, got %q", r.URL.Query().Get("workspace"))
		}
		if r.URL.Query().Get("owner") != "" {
			t.Fatalf("did not expect owner query param")
		}
		_, _ = w.Write([]byte(`{"data":[{"gid":"2","name":"beta","color":"dark-red"},{"gid":"1","name":"Alpha","color":"dark-blue"}]}`))
	}))

	apiClient = newTestClient(srv)
	defer func() { apiClient = nil }()

	outputJSON = true
	defer func() { outputJSON = false }()

	cmd, buf := newPortfolioListTestCmd()
	if err := cmd.Flags().Set("workspace-gid", "w1"); err != nil {
		t.Fatalf("failed to set flag: %v", err)
	}

	if err := runPortfolioList(cmd, nil); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	var items []portfolioListItem
	if err := json.Unmarshal(buf.Bytes(), &items); err != nil {
		t.Fatalf("failed to parse output: %v", err)
	}
	if len(items) != 2 || items[0].GID != "1" {
		t.Fatalf("unexpected order: %#v", items)
	}
}

func TestPortfolioDeleteDryRun(t *testing.T) {
	srv := testutil.MockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/portfolios/123" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"data":{"gid":"123","name":"Launch"}}`))
	}))

	apiClient = newTestClient(srv)
	defer func() { apiClient = nil }()

	outputJSON = true
	defer func() { outputJSON = false }()

	cmd, buf := newPortfolioDeleteTestCmd()
	if err := cmd.Flags().Set("dry-run", "true"); err != nil {
		t.Fatalf("failed to set flag: %v", err)
	}

	if err := runPortfolioDelete(cmd, []string{"123"}); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	var out portfolioWriteOutput
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("failed to parse output: %v", err)
	}
	if out.Action != "deleted" || !out.DryRun || out.Portfolio == nil || out.Portfolio.GID != "123" {
		t.Fatalf("unexpected output: %#v", out)
	}
	if out.Name != "Launch" {
		t.Fatalf("expected name Launch, got %q", out.Name)
	}
}

func newPortfolioListTestCmd() (*cobra.Command, *bytes.Buffer) {
	var buf bytes.Buffer
	cmd := &cobra.Command{}
	attachTestRuntime(cmd)
	cmd.SetOut(&buf)
	cmd.Flags().String("workspace", "", "")
	cmd.Flags().String("workspace-gid", "", "")
	cmd.Flags().Int("limit", 0, "")
	return cmd, &buf
}

func newPortfolioDeleteTestCmd() (*cobra.Command, *bytes.Buffer) {
	var buf bytes.Buffer
	cmd := &cobra.Command{}
	attachTestRuntime(cmd)
	cmd.SetOut(&buf)
	cmd.Flags().Bool("dry-run", false, "")
	return cmd, &buf
}
