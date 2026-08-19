// Package cmd implements config management commands.
package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/spf13/cobra"
	"github.com/vincentsch/asana-cli/internal/api"
	"github.com/vincentsch/asana-cli/internal/config"
	"github.com/vincentsch/rungrad"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage configuration",
	Long: `Manage CLI configuration.

Configuration includes default workspace and project, and authentication token.
Config is stored in the platform config directory at asana/config.json.

See also: auth login, workspace list`,
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a config value",
	Long: `Set a configuration value.

Supported keys:
  - workspace: Default workspace (resolves name to GID)
  - project: Default project (resolves name to GID)
  - token: Personal Access Token

Setting token does not require an existing credential. Workspace and project
resolution require ASANA_TOKEN or a token already stored in config.json.

See also: config get, config list, workspace list`,
	Example: `  # Set default workspace
  asana config set workspace "My Workspace"

  # Set default project (requires workspace to resolve)
  asana config set project "My Project" -w "My Workspace"

  # Set token directly
  asana config set token YOUR_TOKEN`,
	Args: cobra.ExactArgs(2),
	RunE: runConfigSet,
}

var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get a config value",
	Long: `Get a configuration value by key.

See also: config set, config list`,
	Example: `  # Get default workspace
  asana config get workspace

  # Get default project
  asana config get project

  # Get stored token (masked)
  asana config get token`,
	Args: cobra.ExactArgs(1),
	RunE: runConfigGet,
}

var configListCmd = &cobra.Command{
	Use:   "list",
	Short: "List config values",
	Long: `List all configuration values.

See also: config set, config get`,
	Example: `  # List all config
  asana config list

  # Output as JSON
  asana config list --json`,
	RunE: runConfigList,
}

func init() {
	configSetCmd.Flags().StringP("workspace", "w", "", "Workspace name (scopes project lookup)")
	configSetCmd.Flags().String("workspace-gid", "", "Workspace GID (scopes project lookup)")

	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configListCmd)
}

type configListOutput struct {
	Workspace string `json:"workspace"`
	Project   string `json:"project"`
	Token     string `json:"token"`
}

type configSetOutput struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	GID   string `json:"gid,omitempty"`
}

func runConfigSet(cmd *cobra.Command, args []string) error {
	key := strings.ToLower(strings.TrimSpace(args[0]))
	value := strings.TrimSpace(args[1])
	if value == "" {
		return fmt.Errorf("value cannot be empty")
	}

	switch key {
	case "token":
		registerRuntimeSecret(cmd, value)
		if runtimeDryRun(cmd) {
			return configMutationPreview(key, value, "")
		}
		if err := config.SetToken(runtimeConfigPath(cmd), value); err != nil {
			return err
		}
		return writeConfigSetOutput(cmd, configSetOutput{
			Key:   "token",
			Value: config.MaskToken(value),
		})
	case "workspace":
		if err := ensureConfigMutationClient(cmd); err != nil {
			return err
		}
		gid, err := resolveWorkspaceWithPrompt(cmd.Context(), runtimeClient(cmd), value, runtimeNoPrompt(cmd) || runtimeOutputJSON(cmd))
		if err != nil {
			return err
		}
		if runtimeDryRun(cmd) {
			return configMutationPreview(key, value, gid)
		}
		if err := config.SetDefault(runtimeConfigPath(cmd), "workspace_gid", gid); err != nil {
			return err
		}
		return writeConfigSetOutput(cmd, configSetOutput{
			Key:   "workspace",
			Value: value,
			GID:   gid,
		})
	case "project":
		if err := ensureConfigMutationClient(cmd); err != nil {
			return err
		}
		workspaceName, _ := cmd.Flags().GetString("workspace")
		workspaceGID, _ := cmd.Flags().GetString("workspace-gid")
		if workspaceName != "" && workspaceGID != "" {
			return fmt.Errorf("use only one of --workspace or --workspace-gid")
		}
		if workspaceName == "" && workspaceGID == "" {
			cfg, err := config.LoadConfig(runtimeConfigPath(cmd))
			if err != nil {
				return err
			}
			workspaceGID = cfg.Defaults.WorkspaceGID
		}
		if workspaceName != "" {
			gid, err := resolveWorkspaceWithPrompt(cmd.Context(), runtimeClient(cmd), workspaceName, runtimeNoPrompt(cmd) || runtimeOutputJSON(cmd))
			if err != nil {
				return err
			}
			workspaceGID = gid
		}
		gid, err := resolveProjectWithPrompt(cmd.Context(), runtimeClient(cmd), workspaceGID, value, runtimeNoPrompt(cmd) || runtimeOutputJSON(cmd))
		if err != nil {
			return err
		}
		if runtimeDryRun(cmd) {
			return configMutationPreview(key, value, gid)
		}
		if err := config.SetDefault(runtimeConfigPath(cmd), "project_gid", gid); err != nil {
			return err
		}
		return writeConfigSetOutput(cmd, configSetOutput{
			Key:   "project",
			Value: value,
			GID:   gid,
		})
	default:
		return fmt.Errorf("unknown config key %q (use workspace, project, token)", key)
	}
}

// ensureConfigMutationClient supplies credentials for the workspace/project
// variants of `config set`. The command also owns token bootstrap, so it cannot
// be marked uniformly auth-required in rungrad metadata.
func ensureConfigMutationClient(cmd *cobra.Command) error {
	runtime := runtimeFromCommand(cmd)
	if runtime.factory == nil {
		// Direct domain tests may inject a client without constructing an app.
		if runtime.client != nil {
			return nil
		}
		return fmt.Errorf("command runtime is not configured")
	}
	// A rungrad App may execute repeatedly. Never reuse the adapter client from a
	// prior run because the factory's credentials, services, and redactor reset.
	runtime.client = nil
	token, _, err := config.LoadTokenWithLookup(runtime.configPath, runtime.factory.LookupEnv)
	if errors.Is(err, config.ErrNoToken) {
		return &rungrad.Error{Code: rungrad.ExitAuth, Msg: config.ErrNoToken.Error(), Err: config.ErrNoToken}
	}
	if err != nil {
		return err
	}
	runtime.factory.Token = token
	runtime.factory.RegisterSecret(token)
	client := api.NewClient(token)
	client.SetBaseURL(runtimeEndpoint(cmd))
	runtime.client = client
	return nil
}

func configMutationPreview(key, value, gid string) error {
	body := map[string]any{"key": key, "value": value}
	if gid != "" {
		body["gid"] = gid
	}
	return &api.MutationPreview{Method: "WRITE", Path: "config.json", Body: body}
}

func runConfigGet(cmd *cobra.Command, args []string) error {
	key := strings.ToLower(strings.TrimSpace(args[0]))
	cfg, err := config.LoadConfig(runtimeConfigPath(cmd))
	if err != nil {
		return err
	}

	switch key {
	case "workspace":
		value, err := resolveWorkspaceValue(cmd.Context(), cfg.Defaults.WorkspaceGID)
		if err != nil {
			return err
		}
		return writeConfigGetOutput(cmd, key, value)
	case "project":
		value, err := resolveProjectValue(cmd.Context(), cfg.Defaults.ProjectGID)
		if err != nil {
			return err
		}
		return writeConfigGetOutput(cmd, key, value)
	case "token":
		return writeConfigGetOutput(cmd, key, config.MaskToken(cfg.Token))
	default:
		return fmt.Errorf("unknown config key %q (use workspace, project, token)", key)
	}
}

func runConfigList(cmd *cobra.Command, args []string) error {
	cfg, err := config.LoadConfig(runtimeConfigPath(cmd))
	if err != nil {
		return err
	}

	workspaceValue, err := resolveWorkspaceValue(cmd.Context(), cfg.Defaults.WorkspaceGID)
	if err != nil {
		return err
	}
	projectValue, err := resolveProjectValue(cmd.Context(), cfg.Defaults.ProjectGID)
	if err != nil {
		return err
	}
	tokenValue := config.MaskToken(cfg.Token)

	if runtimeOutputJSON(cmd) {
		encoder := json.NewEncoder(cmd.OutOrStdout())
		encoder.SetIndent("", "  ")
		return encoder.Encode(configListOutput{
			Workspace: workspaceValue,
			Project:   projectValue,
			Token:     tokenValue,
		})
	}

	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "workspace: %s\n", workspaceValue); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "project: %s\n", projectValue); err != nil {
		return err
	}
	_, err = fmt.Fprintf(cmd.OutOrStdout(), "token: %s\n", tokenValue)
	return err
}

func writeConfigSetOutput(cmd *cobra.Command, output configSetOutput) error {
	if runtimeOutputJSON(cmd) {
		encoder := json.NewEncoder(cmd.OutOrStdout())
		encoder.SetIndent("", "  ")
		return encoder.Encode(output)
	}

	if output.GID != "" {
		_, err := fmt.Fprintf(cmd.OutOrStdout(), "Set %s to %s (gid: %s)\n", output.Key, output.Value, output.GID)
		return err
	}
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "Set %s to %s\n", output.Key, output.Value)
	return err
}

func writeConfigGetOutput(cmd *cobra.Command, key, value string) error {
	if runtimeOutputJSON(cmd) {
		encoder := json.NewEncoder(cmd.OutOrStdout())
		encoder.SetIndent("", "  ")
		return encoder.Encode(map[string]string{key: value})
	}
	_, err := fmt.Fprintln(cmd.OutOrStdout(), value)
	return err
}

func resolveWorkspaceValue(ctx context.Context, gid string) (string, error) {
	if strings.TrimSpace(gid) == "" {
		return "", nil
	}
	name, err := fetchWorkspaceName(ctx, gid)
	if err != nil {
		return gid, nil
	}
	if name != "" {
		return name, nil
	}
	return gid, nil
}

func resolveProjectValue(ctx context.Context, gid string) (string, error) {
	if strings.TrimSpace(gid) == "" {
		return "", nil
	}
	name, err := fetchProjectName(ctx, gid)
	if err != nil {
		return gid, nil
	}
	if name != "" {
		return name, nil
	}
	return gid, nil
}

func fetchWorkspaceName(ctx context.Context, gid string) (string, error) {
	query := url.Values{}
	query.Set("opt_fields", "gid,name")
	payload, err := runtimeFromContext(ctx).client.Get(ctx, "/workspaces/"+gid, query)
	if err != nil {
		var apiErr *api.APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound {
			return "", nil
		}
		return "", err
	}

	var response api.Response[api.Workspace]
	if err := json.Unmarshal(payload, &response); err != nil {
		return "", &api.ResponseError{Err: err}
	}
	return strings.TrimSpace(response.Data.Name), nil
}

func fetchProjectName(ctx context.Context, gid string) (string, error) {
	query := url.Values{}
	query.Set("opt_fields", "gid,name")
	payload, err := runtimeFromContext(ctx).client.Get(ctx, "/projects/"+gid, query)
	if err != nil {
		var apiErr *api.APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound {
			return "", nil
		}
		return "", err
	}

	var response api.Response[api.Project]
	if err := json.Unmarshal(payload, &response); err != nil {
		return "", &api.ResponseError{Err: err}
	}
	return strings.TrimSpace(response.Data.Name), nil
}
