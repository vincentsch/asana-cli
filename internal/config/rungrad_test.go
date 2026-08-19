// Package config tests the rungrad-facing configuration adapter.
package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEffectivePathUsesAsanaJSONOnlyForGeneratedDefault(t *testing.T) {
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	nativePath := filepath.Join(configHome, "asana", "config.yaml")

	got, err := EffectivePath(nativePath, "", func(string) (string, bool) { return "", false })
	if err != nil {
		t.Fatalf("EffectivePath returned an error: %v", err)
	}
	want := filepath.Join(configHome, "asana", "config.json")
	if got != want {
		t.Fatalf("EffectivePath = %q, want %q", got, want)
	}

	got, err = EffectivePath(nativePath, nativePath, func(string) (string, bool) { return "", false })
	if err != nil {
		t.Fatalf("EffectivePath with flag override returned an error: %v", err)
	}
	if got != nativePath {
		t.Fatalf("flag-selected EffectivePath = %q, want %q", got, nativePath)
	}

	got, err = EffectivePath(nativePath, "", func(key string) (string, bool) {
		return nativePath, key == "ASANA_CONFIG"
	})
	if err != nil {
		t.Fatalf("EffectivePath with environment override returned an error: %v", err)
	}
	if got != nativePath {
		t.Fatalf("environment-selected EffectivePath = %q, want %q", got, nativePath)
	}
}

func TestLoadResolutionConfigNormalizesDefaultsWithoutToken(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := SaveConfig(path, &Config{
		Token: "secret-token",
		Defaults: Defaults{
			WorkspaceGID: "workspace-1",
			ProjectGID:   "project-1",
		},
	}); err != nil {
		t.Fatalf("SaveConfig returned an error: %v", err)
	}

	got, err := LoadResolutionConfig(path)
	if err != nil {
		t.Fatalf("LoadResolutionConfig returned an error: %v", err)
	}
	if got.Defaults["workspace_gid"] != "workspace-1" || got.Defaults["project_gid"] != "project-1" {
		t.Fatalf("unexpected normalized defaults: %#v", got.Defaults)
	}
	if got.Services != nil || got.Profiles != nil {
		t.Fatalf("adapter exposed unexpected resolution data: %#v", got)
	}
}

func TestLoadTokenWithLookupUsesInjectedEnvironment(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := SaveConfig(path, &Config{Token: "config-token"}); err != nil {
		t.Fatalf("SaveConfig returned an error: %v", err)
	}

	token, source, err := LoadTokenWithLookup(path, func(key string) (string, bool) {
		return " env-token ", key == "ASANA_TOKEN"
	})
	if err != nil {
		t.Fatalf("LoadTokenWithLookup returned an error: %v", err)
	}
	if token != "env-token" || source != "env" {
		t.Fatalf("got token %q from %q, want env-token from env", token, source)
	}
}

func TestLoadResolutionConfigDefersMalformedJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte("not-json"), 0o600); err != nil {
		t.Fatalf("WriteFile returned an error: %v", err)
	}

	got, err := LoadResolutionConfig(path)
	if err != nil {
		t.Fatalf("LoadResolutionConfig returned an error: %v", err)
	}
	if got.Version != 1 {
		t.Fatalf("normalized version = %d, want 1", got.Version)
	}
}
