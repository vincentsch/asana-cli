// Package cmd carries app-scoped Asana dependencies into command handlers.
package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/vincentsch/asana-cli/internal/api"
	"github.com/vincentsch/asana-cli/internal/config"
	"github.com/vincentsch/rungrad"
)

type commandRuntimeKey struct{}

type commandRuntime struct {
	factory    *rungrad.Factory
	client     *api.Client
	configPath string
	endpoint   string
	outputJSON bool
	noPrompt   bool
}

func bindCommandRuntime(cmd *cobra.Command, factory *rungrad.Factory, adapter *asanaRuntimeAdapter) error {
	if factory == nil {
		return fmt.Errorf("command runtime is not configured")
	}
	configPath, err := config.EffectivePath(factory.ConfigPath(), factory.Flags.Config, factory.LookupEnv)
	if err != nil {
		return err
	}
	runtime := &commandRuntime{
		factory:    factory,
		client:     adapter.client,
		configPath: configPath,
		endpoint:   apiEndpoint(factory),
		outputJSON: structuredModeActive(factory, cmd),
		noPrompt:   factory.Flags != nil && factory.Flags.NoPrompt,
	}
	cmd.SetContext(context.WithValue(cmd.Context(), commandRuntimeKey{}, runtime))
	return nil
}

func runtimeFromContext(ctx context.Context) *commandRuntime {
	if ctx == nil {
		return &commandRuntime{}
	}
	runtime, _ := ctx.Value(commandRuntimeKey{}).(*commandRuntime)
	if runtime == nil {
		return &commandRuntime{}
	}
	return runtime
}

func runtimeFromCommand(cmd *cobra.Command) *commandRuntime {
	if cmd == nil {
		return &commandRuntime{}
	}
	return runtimeFromContext(cmd.Context())
}

func runtimeClient(cmd *cobra.Command) *api.Client { return runtimeFromCommand(cmd).client }

func runtimeOutputJSON(cmd *cobra.Command) bool { return runtimeFromCommand(cmd).outputJSON }

func runtimeNoPrompt(cmd *cobra.Command) bool { return runtimeFromCommand(cmd).noPrompt }

func runtimeConfigPath(cmd *cobra.Command) string { return runtimeFromCommand(cmd).configPath }

func runtimeEndpoint(cmd *cobra.Command) string {
	endpoint := runtimeFromCommand(cmd).endpoint
	if endpoint == "" {
		return api.DefaultBaseURL
	}
	return endpoint
}

func runtimeDryRun(cmd *cobra.Command) bool {
	factory := runtimeFromCommand(cmd).factory
	return factory != nil && factory.DryRun()
}

func registerRuntimeSecret(cmd *cobra.Command, value string) {
	if factory := runtimeFromCommand(cmd).factory; factory != nil {
		factory.RegisterSecret(value)
	}
}

func apiEndpoint(factory *rungrad.Factory) string {
	if service, ok := factory.Service(apiServiceName); ok && service.Value != "" {
		return service.Value
	}
	return api.DefaultBaseURL
}
