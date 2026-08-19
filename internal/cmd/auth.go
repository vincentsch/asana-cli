// Package cmd implements authentication commands.
package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"

	"github.com/spf13/cobra"
	"github.com/vincentsch/asana-cli/internal/api"
	"github.com/vincentsch/asana-cli/internal/config"
	"github.com/vincentsch/asana-cli/internal/interactive"
)

var newAuthClient = func(token, endpoint string) *api.Client {
	client := api.NewClient(token)
	client.SetBaseURL(endpoint)
	return client
}
var promptToken = interactive.PromptToken

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage authentication",
	Long: `Manage authentication with Asana.

The CLI uses Personal Access Tokens for authentication. Generate a token
at https://app.asana.com/0/my-apps and use 'auth login' to store it.

Alternatively, set the ASANA_TOKEN environment variable.

See also: config, user me`,
}

var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Log in to Asana",
	Long: `Authenticate with Asana using a Personal Access Token.

The token is stored in Asana's config.json file. This command prompts
interactively for the token.

See also: config set, user me`,
	Example: `  # Log in interactively (prompts for token)
  asana auth login`,
	RunE: runAuthLogin,
}

// loginCmd is a top-level shortcut for "auth login"
var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Log in to Asana (shortcut for 'auth login')",
	Long: `Authenticate with Asana using a Personal Access Token.

This is a shortcut for 'asana auth login'.

The token is stored in Asana's config.json file. This command prompts
interactively for the token.

See also: auth, config set, user me`,
	Example: `  # Log in interactively (prompts for token)
  asana login`,
	RunE: runAuthLogin,
}

func init() {
	authCmd.AddCommand(authLoginCmd)
}

func runAuthLogin(cmd *cobra.Command, args []string) error {
	if runtimeDryRun(cmd) {
		return &api.MutationPreview{
			Method: "WRITE",
			Path:   "config.json",
			Body:   map[string]any{"credential": "personal access token"},
		}
	}
	if _, err := fmt.Fprintln(cmd.OutOrStdout(), "Create a personal access token at: https://app.asana.com/0/my-apps"); err != nil {
		return err
	}

	token, err := promptToken()
	if err != nil {
		return err
	}
	registerRuntimeSecret(cmd, token)

	client := newAuthClient(token, runtimeEndpoint(cmd))
	query := url.Values{}
	query.Set("opt_fields", "gid,name,email")
	payload, err := client.Get(cmd.Context(), "/users/me", query)
	if err != nil {
		var apiErr *api.APIError
		if errors.As(err, &apiErr) && apiErr.IsUnauthorized() {
			return fmt.Errorf("Invalid token. Create one at https://app.asana.com/0/my-apps")
		}
		return err
	}

	var response api.Response[api.User]
	if err := json.Unmarshal(payload, &response); err != nil {
		return &api.ResponseError{Err: err}
	}

	if err := config.SetToken(runtimeConfigPath(cmd), token); err != nil {
		return err
	}

	if response.Data.Email != "" {
		_, err = fmt.Fprintf(cmd.OutOrStdout(), "Authenticated as %s (%s)\n", response.Data.Name, response.Data.Email)
		return err
	}
	_, err = fmt.Fprintf(cmd.OutOrStdout(), "Authenticated as %s\n", response.Data.Name)
	return err
}
