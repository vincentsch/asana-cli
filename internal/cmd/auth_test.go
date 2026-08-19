// Package cmd tests auth command behavior.
package cmd

import (
	"bytes"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/vincentsch/asana-cli/internal/api"
	"github.com/vincentsch/asana-cli/internal/config"
	"github.com/vincentsch/rungrad/testutil"
)

func TestAuthLoginSuccess(t *testing.T) {
	srv := testutil.MockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/users/me" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"data":{"gid":"u1","name":"Ava","email":"ava@example.com"}}`))
	}))

	prevNewAuthClient := newAuthClient
	newAuthClient = func(token, endpoint string) *api.Client {
		client := api.NewClient(token)
		client.SetBaseURL(srv.URL)
		client.SetHTTPClient(srv.Client())
		return client
	}
	defer func() { newAuthClient = prevNewAuthClient }()

	prevPrompt := promptToken
	promptToken = func() (string, error) { return "test-token", nil }
	defer func() { promptToken = prevPrompt }()

	prevConfigPath := configPath
	configPath = filepath.Join(t.TempDir(), "config.json")
	defer func() { configPath = prevConfigPath }()

	cmd := &cobra.Command{}
	attachTestRuntime(cmd)
	var buf bytes.Buffer
	cmd.SetOut(&buf)

	if err := runAuthLogin(cmd, nil); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !strings.Contains(buf.String(), "Authenticated as Ava (ava@example.com)") {
		t.Fatalf("unexpected output: %s", buf.String())
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	if cfg.Token != "test-token" {
		t.Fatalf("expected token to be saved, got %q", cfg.Token)
	}
}

func TestAuthLoginInvalidToken(t *testing.T) {
	srv := testutil.MockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/users/me" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"errors":[{"message":"unauthorized"}]}`))
	}))

	prevNewAuthClient := newAuthClient
	newAuthClient = func(token, endpoint string) *api.Client {
		client := api.NewClient(token)
		client.SetBaseURL(srv.URL)
		client.SetHTTPClient(srv.Client())
		return client
	}
	defer func() { newAuthClient = prevNewAuthClient }()

	prevPrompt := promptToken
	promptToken = func() (string, error) { return "bad-token", nil }
	defer func() { promptToken = prevPrompt }()

	cmd := &cobra.Command{}
	attachTestRuntime(cmd)
	cmd.SetOut(&bytes.Buffer{})

	err := runAuthLogin(cmd, nil)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if err.Error() != "Invalid token. Create one at https://app.asana.com/0/my-apps" {
		t.Fatalf("unexpected error: %v", err)
	}
}
