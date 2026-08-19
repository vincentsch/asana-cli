// Package cmd tests attachment command behavior.
package cmd

import (
	"bytes"
	"encoding/json"
	"mime"
	"net/http"
	"testing"

	"github.com/spf13/cobra"
	"github.com/vincentsch/rungrad/testutil"
)

func TestAttachmentListRequiresTask(t *testing.T) {
	cmd, _ := newAttachmentListTestCmd()
	if err := runAttachmentList(cmd, nil); err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestAttachmentListJSONOrder(t *testing.T) {
	srv := testutil.MockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/attachments" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if r.URL.Query().Get("parent") != "123" {
			t.Fatalf("expected parent 123, got %q", r.URL.Query().Get("parent"))
		}
		_, _ = w.Write([]byte(`{"data":[{"gid":"a1","name":"first","host":"asana","size":1},{"gid":"a2","name":"second","host":"asana","size":2}]}`))
	}))

	apiClient = newTestClient(srv)
	defer func() { apiClient = nil }()

	outputJSON = true
	defer func() { outputJSON = false }()

	cmd, buf := newAttachmentListTestCmd()
	if err := cmd.Flags().Set("task-gid", "123"); err != nil {
		t.Fatalf("failed to set flag: %v", err)
	}

	if err := runAttachmentList(cmd, nil); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	var items []attachmentListItem
	if err := json.Unmarshal(buf.Bytes(), &items); err != nil {
		t.Fatalf("failed to parse output: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	if items[0].GID != "a1" || items[1].GID != "a2" {
		t.Fatalf("unexpected order: %#v", items)
	}
}

func TestAttachmentUploadURLMultipart(t *testing.T) {
	var sawRequest bool
	srv := testutil.MockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawRequest = true
		if r.URL.Path != "/attachments" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if err != nil {
			t.Fatalf("failed to parse content type: %v", err)
		}
		if mediaType != "multipart/form-data" {
			t.Fatalf("expected multipart/form-data, got %q", mediaType)
		}
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			t.Fatalf("failed to parse multipart form: %v", err)
		}
		values := r.MultipartForm.Value
		if firstValue(values["parent"]) != "123" {
			t.Fatalf("expected parent 123, got %q", firstValue(values["parent"]))
		}
		if firstValue(values["resource_subtype"]) != "external" {
			t.Fatalf("expected resource_subtype external, got %q", firstValue(values["resource_subtype"]))
		}
		if firstValue(values["url"]) != "https://example.com" {
			t.Fatalf("expected url, got %q", firstValue(values["url"]))
		}
		if firstValue(values["name"]) != "Example" {
			t.Fatalf("expected name, got %q", firstValue(values["name"]))
		}
		if firstValue(values["connect_to_app"]) != "true" {
			t.Fatalf("expected connect_to_app true, got %q", firstValue(values["connect_to_app"]))
		}
		if len(r.MultipartForm.File) != 0 {
			t.Fatalf("did not expect file upload")
		}

		_, _ = w.Write([]byte(`{"data":{"gid":"a1","name":"Example"}}`))
	}))

	apiClient = newTestClient(srv)
	defer func() { apiClient = nil }()

	outputJSON = true
	defer func() { outputJSON = false }()

	cmd, buf := newAttachmentUploadTestCmd()
	if err := cmd.Flags().Set("task-gid", "123"); err != nil {
		t.Fatalf("failed to set flag: %v", err)
	}
	if err := cmd.Flags().Set("url", "https://example.com"); err != nil {
		t.Fatalf("failed to set flag: %v", err)
	}
	if err := cmd.Flags().Set("name", "Example"); err != nil {
		t.Fatalf("failed to set flag: %v", err)
	}
	if err := cmd.Flags().Set("connect-to-app", "true"); err != nil {
		t.Fatalf("failed to set flag: %v", err)
	}

	if err := runAttachmentUpload(cmd, nil); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !sawRequest {
		t.Fatalf("expected upload request")
	}

	if buf.Len() == 0 {
		t.Fatalf("expected output, got empty")
	}
}

func newAttachmentListTestCmd() (*cobra.Command, *bytes.Buffer) {
	var buf bytes.Buffer
	cmd := &cobra.Command{}
	attachTestRuntime(cmd)
	cmd.SetOut(&buf)
	cmd.Flags().String("task", "", "")
	cmd.Flags().String("task-gid", "", "")
	cmd.Flags().Int("limit", 0, "")
	return cmd, &buf
}

func newAttachmentUploadTestCmd() (*cobra.Command, *bytes.Buffer) {
	var buf bytes.Buffer
	cmd := &cobra.Command{}
	attachTestRuntime(cmd)
	cmd.SetOut(&buf)
	cmd.Flags().String("task", "", "")
	cmd.Flags().String("task-gid", "", "")
	cmd.Flags().String("name", "", "")
	cmd.Flags().String("url", "", "")
	cmd.Flags().Bool("connect-to-app", false, "")
	cmd.Flags().Bool("dry-run", false, "")
	return cmd, &buf
}

func firstValue(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
}
