// Package cmd tests tag command behavior.
package cmd

import (
	"bytes"
	"net/http"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/vincentsch/rungrad/testutil"
)

func TestTagTasksTableOutputOrder(t *testing.T) {
	srv := testutil.MockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/tags/123/tasks" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"data":[{"gid":"2","name":"beta","completed":false},{"gid":"1","name":"Alpha","completed":true}]}`))
	}))

	apiClient = newTestClient(srv)
	defer func() { apiClient = nil }()

	outputJSON = false

	cmd, buf := newTagTasksTestCmd()
	if err := runTagTasks(cmd, []string{"123"}); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) < 2 {
		t.Fatalf("expected at least 2 lines, got %q", buf.String())
	}
	header := strings.Fields(lines[0])
	if len(header) < 3 || header[0] != "GID" || header[1] != "NAME" || header[2] != "COMPLETED" {
		t.Fatalf("unexpected header %q", lines[0])
	}
	row := strings.Fields(lines[1])
	if len(row) < 3 || row[0] != "1" || row[1] != "Alpha" || row[2] != "true" {
		t.Fatalf("unexpected row %q", lines[1])
	}
}

func newTagTasksTestCmd() (*cobra.Command, *bytes.Buffer) {
	var buf bytes.Buffer
	cmd := &cobra.Command{}
	attachTestRuntime(cmd)
	cmd.SetOut(&buf)
	cmd.Flags().Int("limit", 0, "")
	return cmd, &buf
}
