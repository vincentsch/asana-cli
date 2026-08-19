// Package config adapts Asana's product-owned JSON configuration to rungrad resolution.
package config

import (
	"os"
	"path/filepath"
	"strings"

	rungradconfig "github.com/vincentsch/rungrad/config"
)

// EffectivePath converts rungrad's generated default path into Asana's
// established config.json path. Explicit flag and environment paths pass through.
func EffectivePath(resolvedPath, flagPath string, lookup func(string) (string, bool)) (string, error) {
	if flagPath != "" {
		return resolvedPath, nil
	}
	if lookup == nil {
		lookup = os.LookupEnv
	}
	if envPath, ok := lookup("ASANA_CONFIG"); ok && envPath != "" {
		return resolvedPath, nil
	}

	defaultPath, err := DefaultConfigPath()
	if err != nil {
		return "", err
	}
	if resolvedPath == "" {
		return defaultPath, nil
	}

	nativeDefault := filepath.Join(filepath.Dir(defaultPath), "config.yaml")
	if filepath.Clean(resolvedPath) == filepath.Clean(nativeDefault) {
		return defaultPath, nil
	}
	return resolvedPath, nil
}

// LoadResolutionConfig reads Asana JSON and projects only non-secret defaults
// into rungrad. The token remains exclusively in Asana's credential adapter.
func LoadResolutionConfig(configPath string) (rungradconfig.Config, error) {
	cfg, err := LoadConfig(configPath)
	if err != nil {
		// rungrad resolves config even for version, completion, and login. Defer
		// product config failures to credential loading or the command handler so
		// Commands that do not require credentials remain independent of config.
		return rungradconfig.Config{Version: 1}, nil
	}

	defaults := map[string]string{}
	if cfg.Defaults.WorkspaceGID != "" {
		defaults["workspace_gid"] = cfg.Defaults.WorkspaceGID
	}
	if cfg.Defaults.ProjectGID != "" {
		defaults["project_gid"] = cfg.Defaults.ProjectGID
	}
	return rungradconfig.Config{
		Version:  1,
		Defaults: defaults,
	}, nil
}

// LoadTokenWithLookup resolves a token with the established environment-first
// precedence while allowing rungrad's injected environment lookup in tests.
func LoadTokenWithLookup(configPath string, lookup func(string) (string, bool)) (token, source string, err error) {
	if lookup == nil {
		lookup = os.LookupEnv
	}
	if value, ok := lookup("ASANA_TOKEN"); ok {
		if token := strings.TrimSpace(value); token != "" {
			return token, "env", nil
		}
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		return "", "", err
	}
	token = strings.TrimSpace(cfg.Token)
	if token == "" {
		return "", "", ErrNoToken
	}
	return token, "config", nil
}
