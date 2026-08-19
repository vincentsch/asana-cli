// Package config loads configuration and resolves authentication tokens.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Defaults struct {
	WorkspaceGID string `json:"workspace_gid"`
	ProjectGID   string `json:"project_gid"`
}

type Config struct {
	Version  int      `json:"version"`
	Token    string   `json:"token"`
	Defaults Defaults `json:"defaults"`
}

// ErrNoToken reports that neither the environment nor the Asana config supplied a token.
var ErrNoToken = errors.New("ASANA_TOKEN not set and no token in config")

// DefaultConfigPath returns the OS-specific default config path.
func DefaultConfigPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(configDir, "asana", "config.json"), nil
}

// LoadConfig loads the config file, returning an empty config if it does not exist.
func LoadConfig(configPath string) (*Config, error) {
	path, err := resolveConfigPath(configPath)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &Config{}, nil
		}
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// LoadToken resolves the Asana token from environment variables or config.
func LoadToken(configPath string) (string, error) {
	token, _, err := LoadTokenWithLookup(configPath, os.LookupEnv)
	return token, err
}

// SaveConfig writes the config to disk with 0600 permissions using atomic write.
func SaveConfig(configPath string, cfg *Config) error {
	if cfg == nil {
		cfg = &Config{}
	}
	if cfg.Version == 0 {
		cfg.Version = 1
	}

	path, err := resolveConfigPath(configPath)
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}

	payload, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	tmpFile, err := os.CreateTemp(dir, "config-*.json")
	if err != nil {
		return err
	}
	tmpName := tmpFile.Name()
	cleanup := func() {
		_ = os.Remove(tmpName)
	}
	if err := tmpFile.Chmod(0o600); err != nil {
		_ = tmpFile.Close()
		cleanup()
		return err
	}
	if _, err := tmpFile.Write(payload); err != nil {
		_ = tmpFile.Close()
		cleanup()
		return err
	}
	if err := tmpFile.Close(); err != nil {
		cleanup()
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		cleanup()
		return err
	}
	if err := os.Chmod(path, 0o600); err != nil {
		return err
	}
	return nil
}

// SetToken updates just the token in config.
func SetToken(configPath, token string) error {
	cfg, err := LoadConfig(configPath)
	if err != nil {
		return err
	}
	cfg.Token = strings.TrimSpace(token)
	return SaveConfig(configPath, cfg)
}

// SetDefault updates a single default value in config.
func SetDefault(configPath, key, value string) error {
	cfg, err := LoadConfig(configPath)
	if err != nil {
		return err
	}
	switch key {
	case "workspace_gid":
		cfg.Defaults.WorkspaceGID = value
	case "project_gid":
		cfg.Defaults.ProjectGID = value
	default:
		return fmt.Errorf("unknown default key %q", key)
	}
	return SaveConfig(configPath, cfg)
}

// MaskToken returns a masked version of token showing only prefix/suffix.
func MaskToken(token string) string {
	token = strings.TrimSpace(token)
	if token == "" {
		return ""
	}
	if len(token) <= 10 {
		return strings.Repeat("*", len(token))
	}
	return token[:4] + "..." + token[len(token)-4:]
}

func resolveConfigPath(configPath string) (string, error) {
	if configPath != "" {
		return configPath, nil
	}

	return DefaultConfigPath()
}
