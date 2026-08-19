// Package interactive tests prompt helper behavior.
package interactive

import (
	"context"
	"net/http"
	"testing"

	"github.com/vincentsch/asana-cli/internal/api"
	"github.com/vincentsch/rungrad/testutil"
)

func TestIsInteractiveNoPrompt(t *testing.T) {
	if IsInteractive(true) {
		t.Fatalf("expected no-prompt to disable interactivity")
	}
}

func TestSelectWorkspaceEmptyList(t *testing.T) {
	srv := testutil.MockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/workspaces" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))

	client := api.NewClient("token")
	client.SetBaseURL(srv.URL)
	client.SetHTTPClient(srv.Client())

	_, err := SelectWorkspace(context.Background(), client)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if err.Error() != "no workspaces found" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDisambiguateEmpty(t *testing.T) {
	_, err := Disambiguate("Pick one", []string{}, func(value string) string { return value })
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if err.Error() != "no options available" {
		t.Fatalf("unexpected error: %v", err)
	}
}
