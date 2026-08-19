// Package cmd provides shared test helpers for command tests.
package cmd

import (
	"context"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/vincentsch/asana-cli/internal/api"
)

// These values support the older domain-unit helpers that invoke handlers
// directly. Production command execution always supplies an app-scoped runtime.
var (
	apiClient  *api.Client
	outputJSON bool
	noPrompt   bool
	configPath string
)

func newTestClient(srv *httptest.Server) *api.Client {
	client := api.NewClient("test-token")
	client.SetBaseURL(srv.URL)
	client.SetHTTPClient(srv.Client())
	return client
}

func attachTestRuntime(cmd *cobra.Command) {
	runtime := &commandRuntime{
		client:     apiClient,
		configPath: configPath,
		outputJSON: outputJSON,
		noPrompt:   noPrompt,
	}
	cmd.SetContext(context.WithValue(context.Background(), commandRuntimeKey{}, runtime))
}

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "asana-cli-config")
	if err != nil {
		panic(err)
	}
	configPath = filepath.Join(dir, "config.json")
	code := m.Run()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}
