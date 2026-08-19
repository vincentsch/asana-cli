// Package config tests config loading and token resolution.
package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadTokenEnvOverridesConfig(t *testing.T) {
	configDir := t.TempDir()
	configPath := filepath.Join(configDir, "config.json")
	writeConfig(t, configPath, "config-token")

	t.Setenv("ASANA_TOKEN", "env-token")

	token, err := LoadToken(configPath)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if token != "env-token" {
		t.Fatalf("expected env token, got %q", token)
	}
}

func TestLoadTokenFromConfig(t *testing.T) {
	configDir := t.TempDir()
	configPath := filepath.Join(configDir, "config.json")
	writeConfig(t, configPath, "config-token")

	t.Setenv("ASANA_TOKEN", "")
	if err := os.Unsetenv("ASANA_TOKEN"); err != nil {
		t.Fatalf("expected env to be unset: %v", err)
	}

	token, err := LoadToken(configPath)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if token != "config-token" {
		t.Fatalf("expected config token, got %q", token)
	}
}

func TestLoadTokenMissing(t *testing.T) {
	configDir := t.TempDir()
	configPath := filepath.Join(configDir, "missing.json")

	if err := os.Unsetenv("ASANA_TOKEN"); err != nil {
		t.Fatalf("expected env to be unset: %v", err)
	}

	_, err := LoadToken(configPath)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if err.Error() != ErrNoToken.Error() {
		t.Fatalf("expected %q, got %q", ErrNoToken.Error(), err.Error())
	}
}

func TestLoadTokenIgnoresWhitespaceEnv(t *testing.T) {
	configDir := t.TempDir()
	configPath := filepath.Join(configDir, "config.json")
	writeConfig(t, configPath, "config-token")

	t.Setenv("ASANA_TOKEN", "   \t\n")

	token, err := LoadToken(configPath)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if token != "config-token" {
		t.Fatalf("expected config token, got %q", token)
	}
}

func TestDefaultConfigPathUsesXDG(t *testing.T) {
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)

	path, err := DefaultConfigPath()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expected := filepath.Join(configHome, "asana", "config.json")
	if path != expected {
		t.Fatalf("expected %q, got %q", expected, path)
	}
}

func TestLoadTokenUsesCustomPath(t *testing.T) {
	configHome := t.TempDir()
	defaultPath := filepath.Join(configHome, "asana", "config.json")
	customPath := filepath.Join(configHome, "custom.json")

	t.Setenv("XDG_CONFIG_HOME", configHome)
	if err := os.Unsetenv("ASANA_TOKEN"); err != nil {
		t.Fatalf("expected env to be unset: %v", err)
	}

	writeConfig(t, defaultPath, "default-token")
	writeConfig(t, customPath, "custom-token")

	token, err := LoadToken(customPath)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if token != "custom-token" {
		t.Fatalf("expected custom token, got %q", token)
	}
}

func TestSaveConfigWritesVersionAndPermissions(t *testing.T) {
	configDir := t.TempDir()
	configPath := filepath.Join(configDir, "config.json")

	cfg := &Config{
		Token: "token",
		Defaults: Defaults{
			WorkspaceGID: "w1",
			ProjectGID:   "p1",
		},
	}
	if err := SaveConfig(configPath, cfg); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config: %v", err)
	}
	if !strings.Contains(string(data), `"version": 1`) {
		t.Fatalf("expected version to be written, got %s", data)
	}

	info, err := os.Stat(configPath)
	if err != nil {
		t.Fatalf("failed to stat config: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("expected permissions 0600, got %o", info.Mode().Perm())
	}
}

func TestMaskToken(t *testing.T) {
	if got := MaskToken("short"); got != "*****" {
		t.Fatalf("expected masked short token, got %q", got)
	}
	if got := MaskToken("12345678901"); got != "1234...8901" {
		t.Fatalf("expected masked token, got %q", got)
	}
}

func writeConfig(t *testing.T, path, token string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}

	payload := []byte(`{"version":1,"token":"` + token + `","defaults":{"workspace_gid":"","project_gid":""}}`)
	if err := os.WriteFile(path, payload, 0o644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}
}
