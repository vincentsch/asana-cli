// Package cmd tests command catalog, manifest, help, and generated manual consistency.
package cmd

import (
	"encoding/json"
	"flag"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/vincentsch/asana-cli/internal/config"
	"github.com/vincentsch/asana-cli/internal/manualdocs"
	"github.com/vincentsch/asana-cli/internal/version"
	"github.com/vincentsch/rungrad/manifest"
	"github.com/vincentsch/rungrad/testutil"
)

var updateCommandArtifacts = flag.Bool("update", false, "regenerate command help goldens")

func TestCommandCatalogMatchesVisibleTree(t *testing.T) {
	app := NewApp()
	if err := app.ValidateCatalog(); err != nil {
		t.Fatalf("ValidateCatalog() = %v", err)
	}

	document := testutil.CaptureManifest(t, app)
	module := newAsanaCommandModule(&asanaRuntimeAdapter{})
	if got, want := len(document.Commands)-1, len(module.Catalog()); got != want {
		t.Fatalf("cataloged commands = %d, want %d", got, want)
	}
}

func TestHelpDocsManifestAndCatalogAreConsistent(t *testing.T) {
	testutil.AssertConsistent(t, NewApp)
}

func TestGeneratedCommandManualIsInSync(t *testing.T) {
	result, err := manualdocs.Check(NewApp(), filepath.Join("..", "..", "manual", "commands"))
	if err != nil {
		t.Fatalf("check generated manual: %v", err)
	}
	if !result.OK() {
		t.Fatalf("generated command manual is out of sync; run `go run ./cmd/generate-command-docs`: \n%s", result.String())
	}
}

func TestCommandHelpGoldensAreInSync(t *testing.T) {
	testutil.AssertHelpGoldens(t, NewApp, *updateCommandArtifacts, filepath.Join("testdata", "help"))
}

func TestManifestIdentityMetadataAndDeterminism(t *testing.T) {
	const (
		environmentToken = "manifest-environment-token-sentinel"
		configToken      = "manifest-config-token-sentinel"
		endpoint         = "https://manifest-endpoint-sentinel.invalid"
	)
	configFile := filepath.Join(t.TempDir(), "manifest-config-path-sentinel.json")
	if err := config.SaveConfig(configFile, &config.Config{Token: configToken}); err != nil {
		t.Fatalf("SaveConfig() = %v", err)
	}
	t.Setenv("ASANA_TOKEN", environmentToken)
	t.Setenv("ASANA_API_BASE_URL", endpoint)

	commandResult1 := testutil.Run(NewApp(), "--config", configFile, "__rungrad_manifest")

	exit, first, stderr := commandResult1.Exit, commandResult1.Stdout, commandResult1.Stderr
	if exit != 0 || stderr != "" {
		t.Fatalf("first manifest exit = %d, stderr = %q", exit, stderr)
	}
	commandResult2 := testutil.Run(NewApp(), "--config", configFile, "__rungrad_manifest")
	exit, second, stderr := commandResult2.Exit, commandResult2.Stdout, commandResult2.Stderr
	if exit != 0 || stderr != "" {
		t.Fatalf("second manifest exit = %d, stderr = %q", exit, stderr)
	}
	if first != second {
		t.Fatal("manifest bytes differ between identical runs")
	}
	for _, secret := range []string{environmentToken, configToken, endpoint, configFile} {
		if strings.Contains(first, secret) {
			t.Fatalf("manifest contains runtime value %q", secret)
		}
	}

	document := testutil.CaptureManifest(t, NewApp())
	if document.SchemaVersion != manifest.SchemaVersion {
		t.Fatalf("schema_version = %q, want %q", document.SchemaVersion, manifest.SchemaVersion)
	}
	if document.ToolName != "asana" {
		t.Fatalf("tool_name = %q, want asana", document.ToolName)
	}
	if document.ToolVersion != version.Short() {
		t.Fatalf("tool_version = %q, want %q", document.ToolVersion, version.Short())
	}

	commands := manifestCommandsByPath(document)
	assertManifestCommand(t, commands, "workspace list", true, false, false, true, true)
	assertManifestCommand(t, commands, "task delete", true, true, true, true, true)
	assertManifestCommand(t, commands, "auth login", false, true, false, false, true)
	assertManifestCommand(t, commands, "config set", false, true, false, true, true)
	assertManifestCommand(t, commands, "update", false, true, false, false, true)
	assertManifestCommand(t, commands, "version", false, false, false, false, true)
	assertManifestCommand(t, commands, "completion", false, false, false, false, false)
}

func TestManifestRelatedCommandsResolve(t *testing.T) {
	document := testutil.CaptureManifest(t, NewApp())
	commands := manifestCommandsByPath(document)
	for _, command := range document.Commands {
		path := strings.Join(command.Path, " ")
		for _, related := range command.Related {
			if _, ok := commands[related]; !ok {
				t.Errorf("manifest command %q references unknown related command %q", path, related)
			}
		}
	}
}

func TestDestructiveCommandsExposeConfirmationFlag(t *testing.T) {
	document := testutil.CaptureManifest(t, NewApp())
	for _, command := range document.Commands {
		if !command.Destructive {
			continue
		}
		hasConfirm := false
		for _, flag := range command.LocalFlags {
			if flag.Name == "confirm" {
				hasConfirm = true
				break
			}
		}
		if !hasConfirm {
			t.Errorf("destructive command %q has no local --confirm flag", strings.Join(command.Path, " "))
		}
	}
}

func TestDestructiveCommandExamplesAreSafe(t *testing.T) {
	document := testutil.CaptureManifest(t, NewApp())
	for _, command := range document.Commands {
		if !command.Destructive {
			continue
		}
		invocationPrefix := "asana " + strings.Join(command.Path, " ")
		for _, example := range command.Examples {
			if !strings.HasPrefix(example, invocationPrefix) {
				continue
			}
			if !strings.Contains(example, " --confirm") && !strings.Contains(example, " --dry-run") {
				t.Errorf("destructive command %q has unsafe example %q", strings.Join(command.Path, " "), example)
			}
		}
	}
}

func TestManifestEndpointIsHiddenFromPublicDiscovery(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("ASANA_TOKEN", "")
	commandResult3 := testutil.Run(NewApp(), "--help")
	exit, help, stderr := commandResult3.Exit, commandResult3.Stdout, commandResult3.Stderr
	if exit != 0 || stderr != "" {
		t.Fatalf("root help exit = %d, stderr = %q", exit, stderr)
	}
	if strings.Contains(help, "__rungrad_manifest") {
		t.Fatalf("root help exposes hidden manifest command:\n%s", help)
	}

	commandResult4 := testutil.Run(NewApp(), "completion", "bash")

	exit, completion, stderr := commandResult4.Exit, commandResult4.Stdout, commandResult4.Stderr
	if exit != 0 || stderr != "" {
		t.Fatalf("completion exit = %d, stderr = %q", exit, stderr)
	}
	if strings.Contains(completion, "__rungrad_manifest") {
		t.Fatal("shell completion exposes hidden manifest command")
	}
}

func TestZZLocalMutationsHonorDryRun(t *testing.T) {
	configFile := filepath.Join(t.TempDir(), "config.json")
	t.Setenv("ASANA_TOKEN", "")

	for _, test := range []struct {
		name   string
		args   []string
		method string
	}{
		{name: "login", args: []string{"auth", "login", "--dry-run", "--json"}, method: "WRITE"},
		{name: "config", args: []string{"--config", configFile, "config", "set", "token", "dry-run-token-sentinel", "--dry-run", "--json"}, method: "WRITE"},
	} {
		t.Run(test.name, func(t *testing.T) {
			commandResult5 := testutil.Run(NewApp(), test.args...)
			exit, stdout, stderr := commandResult5.Exit, commandResult5.Stdout, commandResult5.Stderr
			if exit != 0 || stderr != "" {
				t.Fatalf("exit = %d, stderr = %q", exit, stderr)
			}
			var preview struct {
				DryRun bool   `json:"dry_run"`
				Method string `json:"method"`
			}
			if err := json.Unmarshal([]byte(stdout), &preview); err != nil {
				t.Fatalf("preview JSON: %v\n%s", err, stdout)
			}
			if !preview.DryRun || preview.Method != test.method {
				t.Fatalf("preview = %#v", preview)
			}
		})
	}

	originalVersion := version.Version
	originalFixture := updateFixtureVersion
	originalDisabled := updateInstallDisabled
	t.Cleanup(func() {
		version.Version = originalVersion
		updateFixtureVersion = originalFixture
		updateInstallDisabled = originalDisabled
	})
	version.Version = "v1.0.0"
	updateFixtureVersion = "v1.1.0"
	updateInstallDisabled = "true"
	updateResult := testutil.Run(NewApp(), "update", "--dry-run", "--json")
	if updateResult.Exit != 0 || updateResult.Stderr != "" {
		t.Fatalf("update dry-run exit = %d, stderr = %q", updateResult.Exit, updateResult.Stderr)
	}
	var updateStatus struct {
		Available bool   `json:"available"`
		Status    string `json:"status"`
	}
	if err := json.Unmarshal([]byte(updateResult.Stdout), &updateStatus); err != nil {
		t.Fatalf("update dry-run JSON: %v\n%s", err, updateResult.Stdout)
	}
	if !updateStatus.Available || updateStatus.Status != "update_available" {
		t.Fatalf("unexpected update dry-run status: %#v", updateStatus)
	}
	if _, err := os.Stat(configFile); !os.IsNotExist(err) {
		t.Fatalf("config dry-run created %s or stat failed: %v", configFile, err)
	}
}

func TestZZDestructiveCommandsRequireConfirmation(t *testing.T) {
	var validations atomic.Int32
	var mutations atomic.Int32
	server := testutil.MockServer(t, http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		switch {
		case request.Method == http.MethodGet && request.URL.Path == "/tasks/123":
			validations.Add(1)
			_, _ = w.Write([]byte(`{"data":{"gid":"123","name":"Example"}}`))
		case request.Method == http.MethodDelete && request.URL.Path == "/tasks/123":
			mutations.Add(1)
			_, _ = w.Write([]byte(`{"data":{}}`))
		default:
			t.Fatalf("unexpected request %s %s", request.Method, request.URL.Path)
		}
	}))
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("ASANA_TOKEN", "confirmation-test-token")
	t.Setenv("ASANA_API_BASE_URL", server.URL)

	commandResult6 := testutil.Run(NewApp(), "task", "delete", "123", "--no-prompt")

	exit, stdout, stderr := commandResult6.Exit, commandResult6.Stdout, commandResult6.Stderr
	if exit == 0 {
		t.Fatalf("unconfirmed delete exit = %d, want non-zero", exit)
	}
	if stdout != "" || !strings.Contains(stderr, "destructive action requires --confirm") {
		t.Fatalf("unconfirmed delete stdout = %q, stderr = %q", stdout, stderr)
	}
	if got := mutations.Load(); got != 0 {
		t.Fatalf("unconfirmed delete made %d mutation request(s)", got)
	}

	commandResult7 := testutil.Run(NewApp(), "task", "delete", "123", "--confirm", "--no-prompt")

	exit, stdout, stderr = commandResult7.Exit, commandResult7.Stdout, commandResult7.Stderr
	if exit != 0 || stderr != "" {
		t.Fatalf("confirmed delete exit = %d, stderr = %q", exit, stderr)
	}
	if !strings.Contains(stdout, "Task deleted") {
		t.Fatalf("confirmed delete stdout = %q", stdout)
	}
	if got := mutations.Load(); got != 1 {
		t.Fatalf("confirmed delete made %d mutation request(s), want 1", got)
	}

	commandResult8 := testutil.Run(NewApp(), "task", "delete", "123", "--dry-run", "--json", "--no-prompt")

	exit, stdout, stderr = commandResult8.Exit, commandResult8.Stdout, commandResult8.Stderr
	if exit != 0 || stderr != "" {
		t.Fatalf("dry-run delete exit = %d, stderr = %q", exit, stderr)
	}
	var preview struct {
		DryRun bool   `json:"dry_run"`
		Method string `json:"method"`
		Path   string `json:"path"`
	}
	if err := json.Unmarshal([]byte(stdout), &preview); err != nil {
		t.Fatalf("dry-run preview JSON: %v\n%s", err, stdout)
	}
	if !preview.DryRun || preview.Method != http.MethodDelete || preview.Path != "/tasks/123" {
		t.Fatalf("dry-run preview = %#v", preview)
	}
	if got := mutations.Load(); got != 1 {
		t.Fatalf("dry-run delete changed mutation count to %d", got)
	}
	if got := validations.Load(); got != 1 {
		t.Fatalf("dry-run delete made %d validation reads, want 1", got)
	}
}

func manifestCommandsByPath(document manifest.Manifest) map[string]manifest.Command {
	commands := make(map[string]manifest.Command, len(document.Commands))
	for _, command := range document.Commands {
		commands[strings.Join(command.Path, " ")] = command
	}
	return commands
}

func assertManifestCommand(t *testing.T, commands map[string]manifest.Command, path string, auth, mutates, destructive, meta, output bool) {
	t.Helper()
	command, ok := commands[path]
	if !ok {
		t.Fatalf("manifest is missing command %q", path)
	}
	if command.RequiresAuth != auth || command.Mutates != mutates || command.Destructive != destructive || command.SupportsMeta != meta {
		t.Fatalf("manifest metadata for %q = auth:%t mutates:%t destructive:%t meta:%t", path, command.RequiresAuth, command.Mutates, command.Destructive, command.SupportsMeta)
	}
	if command.SupportsDryRun != mutates {
		t.Fatalf("supports_dry_run for %q = %t, want %t", path, command.SupportsDryRun, mutates)
	}
	if command.RequiresConfirmation != destructive {
		t.Fatalf("requires_confirmation for %q = %t, want %t", path, command.RequiresConfirmation, destructive)
	}
	if (len(command.OutputModes) > 0) != output {
		t.Fatalf("output_modes for %q = %v", path, command.OutputModes)
	}
}
