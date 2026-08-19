// Package resolve tests name-to-GID resolution behavior.
package resolve

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vincentsch/asana-cli/internal/api"
	"github.com/vincentsch/rungrad/testutil"
)

func TestWorkspaceGIDPassthrough(t *testing.T) {
	client := api.NewClient("test-token")
	got, err := Workspace(context.Background(), client, "123456")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got != "123456" {
		t.Fatalf("expected gid passthrough, got %q", got)
	}
}

func TestWorkspaceSingleMatch(t *testing.T) {
	srv := testutil.MockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/workspaces" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"data":[{"gid":"1","name":"Acme"}]}`))
	}))

	client := newTestClient(srv)
	got, err := Workspace(context.Background(), client, "Acme")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got != "1" {
		t.Fatalf("expected gid 1, got %q", got)
	}
}

func TestWorkspaceAmbiguous(t *testing.T) {
	srv := testutil.MockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/workspaces" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"data":[{"gid":"1","name":"Acme"},{"gid":"2","name":"Acme"}]}`))
	}))

	client := newTestClient(srv)
	_, err := Workspace(context.Background(), client, "Acme")
	var ambErr *AmbiguousError
	if !errors.As(err, &ambErr) {
		t.Fatalf("expected AmbiguousError, got %T", err)
	}
	if len(ambErr.Matches) != 2 {
		t.Fatalf("expected 2 matches, got %d", len(ambErr.Matches))
	}
}

func TestWorkspaceNotFound(t *testing.T) {
	srv := testutil.MockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/workspaces" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"data":[{"gid":"1","name":"Other"}]}`))
	}))

	client := newTestClient(srv)
	_, err := Workspace(context.Background(), client, "Acme")
	var notFound *NotFoundError
	if !errors.As(err, &notFound) {
		t.Fatalf("expected NotFoundError, got %T", err)
	}
}

func TestWorkspaceCaseInsensitive(t *testing.T) {
	srv := testutil.MockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":[{"gid":"1","name":"ACME"}]}`))
	}))

	client := newTestClient(srv)
	got, err := Workspace(context.Background(), client, "acme")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got != "1" {
		t.Fatalf("expected gid 1, got %q", got)
	}
}

func TestProjectUnscopedAmbiguousIncludesWorkspaceContext(t *testing.T) {
	srv := testutil.MockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/workspaces":
			_, _ = w.Write([]byte(`{"data":[{"gid":"w1","name":"Acme"},{"gid":"w2","name":"Globex"}]}`))
		case "/projects":
			switch r.URL.Query().Get("workspace") {
			case "w1":
				_, _ = w.Write([]byte(`{"data":[{"gid":"p1","name":"Alpha"}]}`))
			case "w2":
				_, _ = w.Write([]byte(`{"data":[{"gid":"p2","name":"Alpha"}]}`))
			default:
				t.Fatalf("unexpected workspace %q", r.URL.Query().Get("workspace"))
			}
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))

	client := newTestClient(srv)
	_, err := Project(context.Background(), client, "", "Alpha")
	var ambErr *AmbiguousError
	if !errors.As(err, &ambErr) {
		t.Fatalf("expected AmbiguousError, got %T", err)
	}
	if len(ambErr.Matches) != 2 {
		t.Fatalf("expected 2 matches, got %d", len(ambErr.Matches))
	}
	for _, match := range ambErr.Matches {
		if match.Context == "" {
			t.Fatalf("expected workspace context on matches")
		}
	}
}

func TestTeamResolutionUsesWorkspaceScopedEndpoint(t *testing.T) {
	srv := testutil.MockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/workspaces/w1/teams" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"data":[{"gid":"t1","name":"Core"}]}`))
	}))

	client := newTestClient(srv)
	got, err := Team(context.Background(), client, "w1", "Core")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got != "t1" {
		t.Fatalf("expected gid t1, got %q", got)
	}
}

func newTestClient(srv *httptest.Server) *api.Client {
	client := api.NewClient("test-token")
	client.SetBaseURL(srv.URL)
	client.SetHTTPClient(srv.Client())
	return client
}
