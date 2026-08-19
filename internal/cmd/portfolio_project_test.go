// Package cmd tests portfolio project command behavior.
package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/spf13/cobra"
	"github.com/vincentsch/rungrad/testutil"
)

func TestPortfolioProjectListFiltersProjectsAndSorts(t *testing.T) {
	srv := testutil.MockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/portfolios/123/items" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"data":[{"gid":"i1","name":"Zeta","resource_type":"project"},{"gid":"i2","name":"Alpha","resource_type":"project"},{"gid":"i3","name":"Other","resource_type":"portfolio"}]}`))
	}))

	apiClient = newTestClient(srv)
	defer func() { apiClient = nil }()

	outputJSON = true
	defer func() { outputJSON = false }()

	cmd, buf := newPortfolioProjectListTestCmd()
	if err := runPortfolioProjectList(cmd, []string{"123"}); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	var items []portfolioProjectListItem
	if err := json.Unmarshal(buf.Bytes(), &items); err != nil {
		t.Fatalf("failed to parse output: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	if items[0].GID != "i2" || items[1].GID != "i1" {
		t.Fatalf("unexpected order: %#v", items)
	}
}

func newPortfolioProjectListTestCmd() (*cobra.Command, *bytes.Buffer) {
	var buf bytes.Buffer
	cmd := &cobra.Command{}
	attachTestRuntime(cmd)
	cmd.SetOut(&buf)
	cmd.Flags().Int("limit", 0, "")
	return cmd, &buf
}
