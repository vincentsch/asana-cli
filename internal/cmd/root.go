// Package cmd builds the rungrad application around Asana's command tree.
package cmd

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"sync"

	"github.com/vincentsch/asana-cli/internal/api"
	"github.com/vincentsch/asana-cli/internal/config"
	"github.com/vincentsch/asana-cli/internal/resolve"
	"github.com/vincentsch/asana-cli/internal/version"
	"github.com/vincentsch/rungrad"
	rungradconfig "github.com/vincentsch/rungrad/config"
)

const apiServiceName = "api"

const rootShort = "CLI for Asana"

const rootLong = `A command-line interface for Asana, inspired by GitHub's gh.

Manage tasks, projects, sections, and workspaces from your terminal.
Supports both human-readable output and JSON for automation.

Quick start:
  asana auth login              # Authenticate with your token
  asana config set workspace "My Workspace"
  asana task list -p "Project"  # List tasks in a project
  asana task create "New task"  # Create a task

Get a Personal Access Token at: https://app.asana.com/0/my-apps`

// appConstructionMu covers rungrad's process-global Cobra setup and Cobra's
// lazy command/flag caches while shared command blueprints are cloned.
var appConstructionMu sync.Mutex

// NewApp builds an independent rungrad application and command tree.
func NewApp() *rungrad.App {
	appConstructionMu.Lock()
	defer appConstructionMu.Unlock()

	runtimeAdapter := &asanaRuntimeAdapter{}
	application := rungrad.New(rungrad.AppConfig{
		Name:           "asana",
		EnvVar:         "ASANA_TOKEN",
		Short:          rootShort,
		Long:           rootLong,
		Version:        version.Short(),
		AdvancedOutput: true,
		Surface: rungrad.SurfaceConfig{
			Completion: rungrad.SurfaceHostOwned,
		},
		Resolution: &rungrad.ResolutionConfig{
			ConfigEnvVar: "ASANA_CONFIG",
			LoadConfig:   runtimeAdapter.LoadResolutionConfig,
			Services: []rungrad.Service{{
				Name:     apiServiceName,
				EnvVar:   "ASANA_API_BASE_URL",
				Default:  api.DefaultBaseURL,
				Validate: validateAPIBaseURL,
			}},
		},
		Auth: runtimeAdapter,
	})
	runtimeAdapter.factory = application.Factory()
	application.Root().SetVersionTemplate(version.String() + "\n")
	application.AddModule(newAsanaCommandModule(runtimeAdapter))
	return application
}

// Execute runs the rungrad-backed CLI and returns its classified exit code.
func Execute() int {
	return NewApp().Run(os.Args[1:], os.Stdout, os.Stderr)
}

// asanaRuntimeAdapter connects product config and credentials to rungrad state.
type asanaRuntimeAdapter struct {
	factory *rungrad.Factory
	client  *api.Client
}

func (a *asanaRuntimeAdapter) LoadResolutionConfig(resolvedPath string) (rungradconfig.Config, error) {
	path, err := config.EffectivePath(resolvedPath, a.configFlagPath(), a.lookupEnv())
	if err != nil {
		return rungradconfig.Config{}, err
	}
	return config.LoadResolutionConfig(path)
}

func (a *asanaRuntimeAdapter) ResolveCredential(ac *rungrad.AuthContext) (rungrad.Credential, error) {
	path, err := config.EffectivePath(ac.ConfigPath, a.configFlagPath(), ac.LookupEnv)
	if err != nil {
		return rungrad.Credential{}, adaptCommandError(err)
	}
	token, source, err := config.LoadTokenWithLookup(path, ac.LookupEnv)
	if errors.Is(err, config.ErrNoToken) {
		return rungrad.Credential{}, &rungrad.Error{
			Code: rungrad.ExitAuth,
			Msg:  config.ErrNoToken.Error(),
			Err:  config.ErrNoToken,
		}
	}
	if err != nil {
		return rungrad.Credential{}, adaptCommandError(err)
	}

	endpoint := api.DefaultBaseURL
	if service, ok := ac.Service(apiServiceName); ok {
		endpoint = service.Value
	}
	client := api.NewClient(token)
	client.SetBaseURL(endpoint)
	a.client = client

	return rungrad.Credential{Token: token, Source: source}, nil
}

func (a *asanaRuntimeAdapter) configFlagPath() string {
	if a == nil || a.factory == nil || a.factory.Flags == nil {
		return ""
	}
	return a.factory.Flags.Config
}

func (a *asanaRuntimeAdapter) lookupEnv() func(string) (string, bool) {
	if a == nil || a.factory == nil {
		return nil
	}
	return a.factory.LookupEnv
}

func validateAPIBaseURL(value string) error {
	parsed, err := url.Parse(value)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return fmt.Errorf("must be an absolute HTTP or HTTPS URL")
	}
	return nil
}

func exitCodeForError(err error) int {
	var apiErr *api.APIError
	if errors.As(err, &apiErr) {
		switch apiErr.StatusCode {
		case 403:
			return rungrad.ExitForbidden
		case 404:
			return rungrad.ExitNotFound
		case 429:
			return rungrad.ExitRateLimited
		default:
			return api.ExitAPIError
		}
	}
	var responseErr *api.ResponseError
	if errors.As(err, &responseErr) {
		return api.ExitAPIError
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		return api.ExitAPIError
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return api.ExitAPIError
	}
	var ambiguousErr *resolve.AmbiguousError
	if errors.As(err, &ambiguousErr) {
		return api.ExitUsageError
	}
	var notFoundErr *resolve.NotFoundError
	if errors.As(err, &notFoundErr) {
		return rungrad.ExitNotFound
	}
	return api.ExitUsageError
}

func adaptCommandError(err error) error {
	if err == nil {
		return nil
	}
	var rungradErr *rungrad.Error
	if errors.As(err, &rungradErr) {
		return err
	}
	var apiErr *api.APIError
	if errors.As(err, &apiErr) && apiErr.IsUnauthorized() {
		return &rungrad.Error{
			Code: api.ExitUsageError,
			Msg:  "Invalid or expired token\nCreate a new token at: https://app.asana.com/0/my-apps",
			Err:  err,
		}
	}
	return &rungrad.Error{Code: exitCodeForError(err), Msg: err.Error(), Err: err}
}
