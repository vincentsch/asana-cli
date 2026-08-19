// Package cmd tests config command behavior.
package cmd

import (
	"bytes"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/vincentsch/asana-cli/internal/config"
	"github.com/vincentsch/rungrad/testutil"
)

func TestConfigSetToken(t *testing.T) {
	prevConfigPath := configPath
	configPath = filepath.Join(t.TempDir(), "config.json")
	defer func() { configPath = prevConfigPath }()

	cmd, buf := newConfigSetTestCmd()
	if err := runConfigSet(cmd, []string{"token", "token"}); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if strings.TrimSpace(buf.String()) != "Set token to *****" {
		t.Fatalf("unexpected output: %s", buf.String())
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	if cfg.Token != "token" {
		t.Fatalf("expected token saved, got %q", cfg.Token)
	}
}

func TestConfigSetWorkspaceByName(t *testing.T) {
	srv := testutil.MockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/workspaces" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"data":[{"gid":"w1","name":"Acme"}]}`))
	}))

	apiClient = newTestClient(srv)
	defer func() { apiClient = nil }()

	prevConfigPath := configPath
	configPath = filepath.Join(t.TempDir(), "config.json")
	defer func() { configPath = prevConfigPath }()

	cmd, _ := newConfigSetTestCmd()
	if err := runConfigSet(cmd, []string{"workspace", "Acme"}); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	if cfg.Defaults.WorkspaceGID != "w1" {
		t.Fatalf("expected workspace gid w1, got %q", cfg.Defaults.WorkspaceGID)
	}
}

func TestConfigGetWorkspaceResolvesName(t *testing.T) {
	srv := testutil.MockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/workspaces/w1" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"data":{"gid":"w1","name":"Acme"}}`))
	}))

	apiClient = newTestClient(srv)
	defer func() { apiClient = nil }()

	prevConfigPath := configPath
	configPath = filepath.Join(t.TempDir(), "config.json")
	defer func() { configPath = prevConfigPath }()
	if err := config.SaveConfig(configPath, &config.Config{
		Token: "token",
		Defaults: config.Defaults{
			WorkspaceGID: "w1",
		},
	}); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	cmd, buf := newConfigGetTestCmd()
	if err := runConfigGet(cmd, []string{"workspace"}); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if strings.TrimSpace(buf.String()) != "Acme" {
		t.Fatalf("unexpected output: %s", buf.String())
	}
}

func TestConfigGetWorkspaceFallsBackOnLookupError(t *testing.T) {
	srv := testutil.MockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"errors":[{"message":"server error"}]}`))
	}))

	apiClient = newTestClient(srv)
	defer func() { apiClient = nil }()

	prevConfigPath := configPath
	configPath = filepath.Join(t.TempDir(), "config.json")
	defer func() { configPath = prevConfigPath }()
	if err := config.SaveConfig(configPath, &config.Config{
		Token: "token",
		Defaults: config.Defaults{
			WorkspaceGID: "w1",
		},
	}); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	cmd, buf := newConfigGetTestCmd()
	if err := runConfigGet(cmd, []string{"workspace"}); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if strings.TrimSpace(buf.String()) != "w1" {
		t.Fatalf("expected fallback gid, got %s", buf.String())
	}
}

func TestConfigListMasksToken(t *testing.T) {
	prevConfigPath := configPath
	configPath = filepath.Join(t.TempDir(), "config.json")
	defer func() { configPath = prevConfigPath }()
	if err := config.SaveConfig(configPath, &config.Config{
		Token: "12345678901",
		Defaults: config.Defaults{
			WorkspaceGID: "",
			ProjectGID:   "",
		},
	}); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	cmd, buf := newConfigListTestCmd()
	if err := runConfigList(cmd, nil); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !strings.Contains(buf.String(), "token: 1234...8901") {
		t.Fatalf("expected masked token, got %s", buf.String())
	}
}

func newConfigSetTestCmd() (*cobra.Command, *bytes.Buffer) {
	cmd := &cobra.Command{}
	attachTestRuntime(cmd)
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.Flags().String("workspace", "", "")
	cmd.Flags().String("workspace-gid", "", "")
	return cmd, &buf
}

func newConfigGetTestCmd() (*cobra.Command, *bytes.Buffer) {
	cmd := &cobra.Command{}
	attachTestRuntime(cmd)
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	return cmd, &buf
}

func newConfigListTestCmd() (*cobra.Command, *bytes.Buffer) {
	cmd := &cobra.Command{}
	attachTestRuntime(cmd)
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	return cmd, &buf
}
