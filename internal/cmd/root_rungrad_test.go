// Package cmd tests the rungrad root, credential, config, and endpoint adapters.
package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/spf13/cobra"
	"github.com/vincentsch/asana-cli/internal/config"
	"github.com/vincentsch/asana-cli/internal/version"
	"github.com/vincentsch/rungrad"
	"github.com/vincentsch/rungrad/testutil"
)

func TestRungradRootHelpNeedsNoCredential(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("ASANA_TOKEN", "")

	commandResult1 := testutil.Run(NewApp(), "--help")

	exit, stdout, stderr := commandResult1.Exit, commandResult1.Stdout, commandResult1.Stderr
	if exit != 0 {
		t.Fatalf("exit = %d, stderr = %s", exit, stderr)
	}
	for _, family := range []string{"auth", "config", "project", "task", "workspace"} {
		if !strings.Contains(stdout, family) {
			t.Fatalf("root help does not list %q:\n%s", family, stdout)
		}
	}
	if strings.Contains(stdout, "config.yaml") {
		t.Fatalf("root help exposed rungrad's native config path:\n%s", stdout)
	}
}

func TestNewAppBuildsIndependentCommandTrees(t *testing.T) {
	first := NewApp()
	second := NewApp()
	if first == second || first.Root() == second.Root() {
		t.Fatal("NewApp reused application or root command state")
	}

	firstList, _, err := first.Root().Find([]string{"task", "list"})
	if err != nil {
		t.Fatalf("find first task list: %v", err)
	}
	secondList, _, err := second.Root().Find([]string{"task", "list"})
	if err != nil {
		t.Fatalf("find second task list: %v", err)
	}
	if firstList == secondList {
		t.Fatal("NewApp reused a command value")
	}
	if err := firstList.Flags().Set("project", "first-only"); err != nil {
		t.Fatalf("set first project flag: %v", err)
	}
	if got, err := secondList.Flags().GetString("project"); err != nil || got != "" {
		t.Fatalf("second project flag = %q, err = %v; want independent default", got, err)
	}
}

func TestNewAppConcurrentConstruction(t *testing.T) {
	const builders = 32
	start := make(chan struct{})
	roots := make(chan *cobra.Command, builders)
	var group sync.WaitGroup
	group.Add(builders)
	for range builders {
		go func() {
			defer group.Done()
			<-start
			roots <- NewApp().Root()
		}()
	}
	close(start)
	group.Wait()
	close(roots)

	seen := make(map[*cobra.Command]struct{}, builders)
	for root := range roots {
		if root == nil {
			t.Fatal("NewApp returned a nil root")
		}
		if _, exists := seen[root]; exists {
			t.Fatal("concurrent NewApp calls reused a root command")
		}
		seen[root] = struct{}{}
	}
	if len(seen) != builders {
		t.Fatalf("constructed %d independent roots, want %d", len(seen), builders)
	}
}

func TestMutatingCommandsUseOnlyGlobalDryRunFlag(t *testing.T) {
	var walk func(*cobra.Command)
	walk = func(command *cobra.Command) {
		if command.Annotations[rungrad.AnnotationMutates] == "true" {
			if command.LocalNonPersistentFlags().Lookup("dry-run") != nil {
				t.Errorf("%s registers a local --dry-run flag", command.CommandPath())
			}
			if command.InheritedFlags().Lookup("dry-run") == nil {
				t.Errorf("%s does not inherit the global --dry-run flag", command.CommandPath())
			}
		}
		for _, child := range command.Commands() {
			walk(child)
		}
	}
	walk(NewApp().Root())
}

func TestRungradRootReportsMissingTokenWithAuthExit(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("ASANA_TOKEN", "")

	commandResult2 := testutil.Run(NewApp(), "workspace", "list")

	exit, _, stderr := commandResult2.Exit, commandResult2.Stdout, commandResult2.Stderr
	if exit != rungrad.ExitAuth {
		t.Fatalf("exit = %d, want %d; stderr = %s", exit, rungrad.ExitAuth, stderr)
	}
	if !strings.Contains(stderr, config.ErrNoToken.Error()) {
		t.Fatalf("stderr = %q, want missing-token guidance", stderr)
	}
}

func TestRungradBareCommandGroupNeedsNoCredential(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("ASANA_TOKEN", "")

	commandResult3 := testutil.Run(NewApp(), "task")

	exit, stdout, stderr := commandResult3.Exit, commandResult3.Stdout, commandResult3.Stderr
	if exit != 0 {
		t.Fatalf("exit = %d, stderr = %s", exit, stderr)
	}
	if !strings.Contains(stdout, "asana task [command]") {
		t.Fatalf("bare task group did not render help: %s", stdout)
	}
}

func TestRungradVersionAndCompletionNeedNoCredential(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("ASANA_TOKEN", "")

	for _, args := range [][]string{{"version"}, {"completion", "bash"}} {
		commandResult4 := testutil.Run(NewApp(), args...)
		exit, stdout, stderr := commandResult4.Exit, commandResult4.Stdout, commandResult4.Stderr
		if exit != 0 {
			t.Fatalf("%v exit = %d, stderr = %s", args, exit, stderr)
		}
		if stdout == "" {
			t.Fatalf("%v returned empty stdout", args)
		}
	}
}

func TestVersionSupportsMachineOutput(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("ASANA_TOKEN", "")

	commandResult5 := testutil.Run(NewApp(), "version", "--json")

	exit, stdout, stderr := commandResult5.Exit, commandResult5.Stdout, commandResult5.Stderr
	if exit != 0 || stderr != "" {
		t.Fatalf("exit = %d, stderr = %s", exit, stderr)
	}
	var result map[string]string
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("version output is not JSON: %v\n%s", err, stdout)
	}
	if result["output"] == "" {
		t.Fatalf("version output is empty: %#v", result)
	}
}

func TestVersionCommandAndFlagPreserveBuildMetadata(t *testing.T) {
	originalVersion := version.Version
	originalCommit := version.Commit
	t.Cleanup(func() {
		version.Version = originalVersion
		version.Commit = originalCommit
	})
	version.Version = "v1.2.3"
	version.Commit = "abcdef1"

	want := "asana v1.2.3 (abcdef1)\n"
	for _, args := range [][]string{{"version"}, {"--version"}} {
		result := testutil.Run(NewApp(), args...)
		if result.Exit != 0 || result.Stderr != "" || result.Stdout != want {
			t.Fatalf("%v exit = %d, stdout = %q, stderr = %q; want stdout %q", args, result.Exit, result.Stdout, result.Stderr, want)
		}
	}
}

func TestRungradConfigSetTokenUsesAsanaJSONDefault(t *testing.T) {
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	t.Setenv("ASANA_TOKEN", "")

	commandResult6 := testutil.Run(NewApp(), "config", "set", "token", "stored-token")

	exit, stdout, stderr := commandResult6.Exit, commandResult6.Stdout, commandResult6.Stderr
	if exit != 0 {
		t.Fatalf("exit = %d, stdout = %s, stderr = %s", exit, stdout, stderr)
	}
	jsonPath := filepath.Join(configHome, "asana", "config.json")
	cfg, err := config.LoadConfig(jsonPath)
	if err != nil {
		t.Fatalf("LoadConfig returned an error: %v", err)
	}
	if cfg.Token != "stored-token" {
		t.Fatalf("stored token = %q, want stored-token", cfg.Token)
	}
	if _, err := os.Stat(filepath.Join(configHome, "asana", "config.yaml")); !os.IsNotExist(err) {
		t.Fatalf("rungrad-native config.yaml was created or stat failed unexpectedly: %v", err)
	}
}

func TestRungradConfigSetWorkspaceStillRequiresCredential(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("ASANA_TOKEN", "")

	result := testutil.Run(NewApp(), "config", "set", "workspace", "Acme")
	if result.Exit != rungrad.ExitAuth {
		t.Fatalf("exit = %d, want %d; stderr = %s", result.Exit, rungrad.ExitAuth, result.Stderr)
	}
	if !strings.Contains(result.Stderr, config.ErrNoToken.Error()) {
		t.Fatalf("stderr = %q, want missing-token guidance", result.Stderr)
	}
}

func TestRungradConfigSetWorkspaceBuildsAuthenticatedClient(t *testing.T) {
	configHome := t.TempDir()
	var authorization string
	server := testutil.MockServer(t, http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		authorization = request.Header.Get("Authorization")
		if request.URL.Path != "/workspaces" {
			t.Fatalf("unexpected request path %q", request.URL.Path)
		}
		_, _ = w.Write([]byte(`{"data":[{"gid":"w1","name":"Acme"}]}`))
	}))
	t.Setenv("XDG_CONFIG_HOME", configHome)
	t.Setenv("ASANA_TOKEN", "workspace-config-token")
	t.Setenv("ASANA_API_BASE_URL", server.URL)

	result := testutil.Run(NewApp(), "config", "set", "workspace", "Acme", "--json")
	if result.Exit != 0 || result.Stderr != "" {
		t.Fatalf("exit = %d, stderr = %q", result.Exit, result.Stderr)
	}
	if authorization != "Bearer workspace-config-token" {
		t.Fatalf("authorization = %q", authorization)
	}
	cfg, err := config.LoadConfig(filepath.Join(configHome, "asana", "config.json"))
	if err != nil {
		t.Fatalf("LoadConfig() = %v", err)
	}
	if cfg.Defaults.WorkspaceGID != "w1" {
		t.Fatalf("workspace gid = %q, want w1", cfg.Defaults.WorkspaceGID)
	}
}

func TestConfigSetRefreshesCredentialOnReusedApp(t *testing.T) {
	const (
		firstToken  = "first-reused-app-token"
		secondToken = "second-reused-app-token"
	)
	configHome := t.TempDir()
	firstRequests := 0
	firstServer := testutil.MockServer(t, http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		firstRequests++
		if authorization := request.Header.Get("Authorization"); authorization != "Bearer "+firstToken {
			t.Fatalf("first authorization = %q", authorization)
		}
		_, _ = w.Write([]byte(`{"data":[{"gid":"w1","name":"First"}]}`))
	}))
	secondRequests := 0
	secondServer := testutil.MockServer(t, http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		secondRequests++
		if authorization := request.Header.Get("Authorization"); authorization != "Bearer "+secondToken {
			t.Fatalf("second authorization = %q", authorization)
		}
		_, _ = w.Write([]byte(`{"data":[{"gid":"w2","name":"` + secondToken + `"}]}`))
	}))

	t.Setenv("XDG_CONFIG_HOME", configHome)
	t.Setenv("ASANA_TOKEN", firstToken)
	t.Setenv("ASANA_API_BASE_URL", firstServer.URL)
	app := NewApp()

	first := testutil.Run(app, "workspace", "list", "--json")
	if first.Exit != 0 || first.Stderr != "" {
		t.Fatalf("first run exit = %d, stderr = %q", first.Exit, first.Stderr)
	}

	t.Setenv("ASANA_TOKEN", secondToken)
	t.Setenv("ASANA_API_BASE_URL", secondServer.URL)
	second := testutil.Run(app, "config", "set", "workspace", secondToken, "--json")
	if second.Exit != 0 || second.Stderr != "" {
		t.Fatalf("second run exit = %d, stderr = %q", second.Exit, second.Stderr)
	}
	if strings.Contains(second.Stdout, secondToken) || !strings.Contains(second.Stdout, "[REDACTED]") {
		t.Fatalf("second run did not redact the current credential: %s", second.Stdout)
	}
	if firstRequests != 1 || secondRequests != 1 {
		t.Fatalf("requests after credential rotation = first:%d second:%d", firstRequests, secondRequests)
	}

	t.Setenv("ASANA_TOKEN", "")
	third := testutil.Run(app, "config", "set", "workspace", "Missing")
	if third.Exit != rungrad.ExitAuth || !strings.Contains(third.Stderr, config.ErrNoToken.Error()) {
		t.Fatalf("third run exit = %d, stderr = %q", third.Exit, third.Stderr)
	}
	if firstRequests != 1 || secondRequests != 1 {
		t.Fatalf("missing credential reused a stale client: first:%d second:%d", firstRequests, secondRequests)
	}
}

func TestRungradCredentialPrecedenceAndAPIEndpoint(t *testing.T) {
	const envToken = "environment-secret-token"
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	path := filepath.Join(configHome, "asana", "config.json")
	if err := config.SaveConfig(path, &config.Config{Token: "config-secret-token"}); err != nil {
		t.Fatalf("SaveConfig returned an error: %v", err)
	}

	var authorization string
	server := testutil.MockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authorization = r.Header.Get("Authorization")
		_, _ = w.Write([]byte(`{"data":[{"gid":"w1","name":"Acme"}]}`))
	}))
	t.Setenv("ASANA_TOKEN", envToken)
	t.Setenv("ASANA_API_BASE_URL", server.URL)

	commandResult7 := testutil.Run(NewApp(), "workspace", "list")

	exit, stdout, stderr := commandResult7.Exit, commandResult7.Stdout, commandResult7.Stderr
	if exit != 0 {
		t.Fatalf("exit = %d, stdout = %s, stderr = %s", exit, stdout, stderr)
	}
	if authorization != "Bearer "+envToken {
		t.Fatalf("Authorization = %q, want environment token", authorization)
	}
}

func TestRungradCredentialFallsBackToAsanaConfig(t *testing.T) {
	const storedToken = "stored-secret-token"
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	t.Setenv("ASANA_TOKEN", "")
	if err := config.SaveConfig(filepath.Join(configHome, "asana", "config.json"), &config.Config{Token: storedToken}); err != nil {
		t.Fatalf("SaveConfig returned an error: %v", err)
	}

	var authorization string
	server := testutil.MockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authorization = r.Header.Get("Authorization")
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	t.Setenv("ASANA_API_BASE_URL", server.URL)

	commandResult8 := testutil.Run(NewApp(), "workspace", "list")

	exit, _, stderr := commandResult8.Exit, commandResult8.Stdout, commandResult8.Stderr
	if exit != 0 {
		t.Fatalf("exit = %d, stderr = %s", exit, stderr)
	}
	if authorization != "Bearer "+storedToken {
		t.Fatalf("Authorization = %q, want config token", authorization)
	}
}

func TestRungradNameResolutionErrorsRemainDeterministic(t *testing.T) {
	server := testutil.MockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/workspaces" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(`{"data":[{"gid":"20","name":"Duplicate"},{"gid":"10","name":"Duplicate"},{"gid":"30","name":"Available"}]}`))
	}))
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("ASANA_TOKEN", "resolution-token")
	t.Setenv("ASANA_API_BASE_URL", server.URL)

	commandResult9 := testutil.Run(NewApp(), "project", "list", "--workspace", "Duplicate", "--no-prompt")

	exit, _, stderr := commandResult9.Exit, commandResult9.Stdout, commandResult9.Stderr
	if exit != 1 {
		t.Fatalf("ambiguous exit = %d, stderr = %s", exit, stderr)
	}
	first := strings.Index(stderr, "10  Duplicate")
	second := strings.Index(stderr, "20  Duplicate")
	if first < 0 || second < 0 || first >= second {
		t.Fatalf("ambiguous candidates are not deterministic: %s", stderr)
	}

	commandResult10 := testutil.Run(NewApp(), "project", "list", "--workspace", "Missing", "--no-prompt")

	exit, _, stderr = commandResult10.Exit, commandResult10.Stdout, commandResult10.Stderr
	if exit != rungrad.ExitNotFound || !strings.Contains(stderr, `workspace "Missing" not found. Available: Available, Duplicate`) {
		t.Fatalf("not-found exit = %d, stderr = %s", exit, stderr)
	}
}

func TestRungradExplicitConfigPathsAreNotRemapped(t *testing.T) {
	for _, source := range []string{"flag", "environment"} {
		t.Run(source, func(t *testing.T) {
			const selectedToken = "explicit-path-token"
			configHome := t.TempDir()
			t.Setenv("XDG_CONFIG_HOME", configHome)
			t.Setenv("ASANA_TOKEN", "")
			t.Setenv("ASANA_CONFIG", "")

			configDir := filepath.Join(configHome, "asana")
			if err := config.SaveConfig(filepath.Join(configDir, "config.json"), &config.Config{Token: "default-token"}); err != nil {
				t.Fatalf("SaveConfig default returned an error: %v", err)
			}
			explicitPath := filepath.Join(configDir, "config.yaml")
			if err := config.SaveConfig(explicitPath, &config.Config{Token: selectedToken}); err != nil {
				t.Fatalf("SaveConfig explicit returned an error: %v", err)
			}

			var authorization string
			server := testutil.MockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				authorization = r.Header.Get("Authorization")
				_, _ = w.Write([]byte(`{"data":[]}`))
			}))
			t.Setenv("ASANA_API_BASE_URL", server.URL)

			args := []string{"workspace", "list"}
			if source == "flag" {
				args = append([]string{"--config", explicitPath}, args...)
			} else {
				t.Setenv("ASANA_CONFIG", explicitPath)
			}
			commandResult11 := testutil.Run(NewApp(), args...)
			exit, _, stderr := commandResult11.Exit, commandResult11.Stdout, commandResult11.Stderr
			if exit != 0 {
				t.Fatalf("exit = %d, stderr = %s", exit, stderr)
			}
			if authorization != "Bearer "+selectedToken {
				t.Fatalf("Authorization = %q, want token from explicit %s path", authorization, source)
			}
		})
	}
}

func TestRungradMalformedConfigKeepsUsageExit(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("ASANA_TOKEN", "")
	t.Setenv("ASANA_CONFIG", "")
	path := filepath.Join(t.TempDir(), "malformed.json")
	if err := os.WriteFile(path, []byte("not-json"), 0o600); err != nil {
		t.Fatalf("WriteFile returned an error: %v", err)
	}

	commandResult12 := testutil.Run(NewApp(), "--config", path, "workspace", "list")

	exit, _, stderr := commandResult12.Exit, commandResult12.Stdout, commandResult12.Stderr
	if exit != 1 {
		t.Fatalf("exit = %d, want 1; stderr = %s", exit, stderr)
	}
	if stderr == "" {
		t.Fatal("malformed config returned empty stderr")
	}
}

func TestRungradUnauthorizedGuidanceAndRedaction(t *testing.T) {
	const token = "raw-secret-token-value"
	server := testutil.MockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = fmt.Fprintf(w, `{"errors":[{"message":%q}]}`, token)
	}))
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("ASANA_TOKEN", token)
	t.Setenv("ASANA_API_BASE_URL", server.URL)

	commandResult13 := testutil.Run(NewApp(), "workspace", "list")

	exit, _, stderr := commandResult13.Exit, commandResult13.Stdout, commandResult13.Stderr
	if exit != 1 {
		t.Fatalf("exit = %d, want 1; stderr = %s", exit, stderr)
	}
	if !strings.Contains(stderr, "Invalid or expired token") || !strings.Contains(stderr, "Create a new token at:") {
		t.Fatalf("stderr lacks unauthorized guidance: %s", stderr)
	}
	if strings.Contains(stderr, token) {
		t.Fatalf("stderr leaked the resolved token: %s", stderr)
	}
}

func TestRungradLoginErrorRedactsPromptedToken(t *testing.T) {
	const token = "newly-prompted-secret-token"
	server := testutil.MockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprintf(w, `{"errors":[{"message":%q}]}`, token)
	}))
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("ASANA_TOKEN", "")
	t.Setenv("ASANA_API_BASE_URL", server.URL)

	previousPrompt := promptToken
	promptToken = func() (string, error) { return token, nil }
	defer func() { promptToken = previousPrompt }()

	commandResult14 := testutil.Run(NewApp(), "auth", "login")

	exit, _, stderr := commandResult14.Exit, commandResult14.Stdout, commandResult14.Stderr
	if exit != 2 {
		t.Fatalf("exit = %d, want 2; stderr = %s", exit, stderr)
	}
	if strings.Contains(stderr, token) {
		t.Fatalf("stderr leaked the prompted token: %s", stderr)
	}
	if !strings.Contains(stderr, "[REDACTED]") {
		t.Fatalf("stderr did not show redaction marker: %s", stderr)
	}
}

func TestRungradJSONErrorRedactsResolvedToken(t *testing.T) {
	const token = "machine-output-secret-token"
	server := testutil.MockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = fmt.Fprintf(w, `{"errors":[{"message":%q}]}`, token)
	}))
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("ASANA_TOKEN", token)
	t.Setenv("ASANA_API_BASE_URL", server.URL)

	commandResult15 := testutil.Run(NewApp(), "workspace", "list", "--json")

	exit, _, stderr := commandResult15.Exit, commandResult15.Stdout, commandResult15.Stderr
	if exit != 2 {
		t.Fatalf("exit = %d, want 2; stderr = %s", exit, stderr)
	}
	if strings.Contains(stderr, token) {
		t.Fatalf("JSON stderr leaked the resolved token: %s", stderr)
	}
	var body map[string]any
	if err := json.Unmarshal([]byte(stderr), &body); err != nil {
		t.Fatalf("stderr is not valid JSON: %v\n%s", err, stderr)
	}
	if !strings.Contains(fmt.Sprint(body["error"]), "[REDACTED]") {
		t.Fatalf("JSON error did not contain a redaction marker: %#v", body)
	}
}

func TestAdvancedListOutputUsesStableModel(t *testing.T) {
	server := testutil.MockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Asana-Request-Id", "req-workspaces")
		w.Header().Set("X-RateLimit-Remaining", "149")
		switch r.URL.Path {
		case "/workspaces", "/projects":
			_, _ = w.Write([]byte(`{"data":[{"gid":"2","name":"Beta"},{"gid":"1","name":"Alpha"}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("ASANA_TOKEN", "advanced-output-token")
	t.Setenv("ASANA_API_BASE_URL", server.URL)

	commandResult16 := testutil.Run(NewApp(), "workspace", "list", "--plain")

	plainExit, plain, plainErr := commandResult16.Exit, commandResult16.Stdout, commandResult16.Stderr
	if plainExit != 0 || plainErr != "" {
		t.Fatalf("plain exit = %d, stderr = %s", plainExit, plainErr)
	}
	if plain != "Alpha\t1\nBeta\t2\n" {
		t.Fatalf("plain output = %q", plain)
	}
	commandResult17 := testutil.Run(NewApp(), "workspace", "list", "--dry-run")
	humanExit, human, humanErr := commandResult17.Exit, commandResult17.Stdout, commandResult17.Stderr
	if humanExit != 0 || humanErr != "" {
		t.Fatalf("read dry-run exit = %d, stderr = %s", humanExit, humanErr)
	}
	if !strings.HasPrefix(human, "NAME") || !strings.Contains(human, "Alpha") || strings.HasPrefix(strings.TrimSpace(human), "[") {
		t.Fatalf("read dry-run changed human output: %q", human)
	}
	commandResult18 := testutil.Run(NewApp(), "project", "list", "--workspace-gid", "10", "--plain")
	projectExit, projects, projectErr := commandResult18.Exit, commandResult18.Stdout, commandResult18.Stderr
	if projectExit != 0 || projectErr != "" || projects != "Alpha\t1\nBeta\t2\n" {
		t.Fatalf("project plain exit = %d, stdout = %q, stderr = %s", projectExit, projects, projectErr)
	}

	commandResult19 := testutil.Run(NewApp(), "workspace", "list", "--jq", ".[].gid")

	jqExit, jqOutput, jqErr := commandResult19.Exit, commandResult19.Stdout, commandResult19.Stderr
	if jqExit != 0 || jqErr != "" {
		t.Fatalf("jq exit = %d, stderr = %s", jqExit, jqErr)
	}
	if jqOutput != "\"1\"\n\"2\"\n" {
		t.Fatalf("jq output = %q", jqOutput)
	}

	commandResult20 := testutil.Run(NewApp(), "workspace", "list", "--include-meta", "--json")

	metaExit, metaJSON, metaErr := commandResult20.Exit, commandResult20.Stdout, commandResult20.Stderr
	if metaExit != 0 || metaErr != "" {
		t.Fatalf("metadata exit = %d, stderr = %s", metaExit, metaErr)
	}
	var envelope struct {
		Data []struct {
			GID string `json:"gid"`
		} `json:"data"`
		Meta struct {
			RequestID  string `json:"request_id"`
			Pagination struct {
				HasMore bool `json:"has_more"`
			} `json:"pagination"`
			RateLimit struct {
				Raw map[string]string `json:"raw"`
			} `json:"rate_limit"`
			Extra map[string]any `json:"extra"`
		} `json:"meta"`
	}
	if err := json.Unmarshal([]byte(metaJSON), &envelope); err != nil {
		t.Fatalf("metadata output is not JSON: %v\n%s", err, metaJSON)
	}
	if len(envelope.Data) != 2 || envelope.Meta.RequestID != "req-workspaces" || envelope.Meta.Pagination.HasMore {
		t.Fatalf("unexpected metadata envelope: %#v", envelope)
	}
	if envelope.Meta.RateLimit.Raw["X-RateLimit-Remaining"] != "149" || envelope.Meta.Extra["endpoint"] != "/workspaces" {
		t.Fatalf("unexpected response metadata: %#v", envelope.Meta)
	}
}

func TestAdvancedDetailOutputUsesStableModel(t *testing.T) {
	server := testutil.MockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/tasks/123":
			_, _ = w.Write([]byte(`{"data":{"gid":"123","name":"Test task","completed":false,"notes":"Notes","memberships":[{"project":{"gid":"10","name":"Alpha"},"section":{"gid":"20","name":"Todo"}}]}}`))
		case "/tasks/123/stories":
			_, _ = w.Write([]byte(`{"data":[]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("ASANA_TOKEN", "detail-output-token")
	t.Setenv("ASANA_API_BASE_URL", server.URL)

	commandResult21 := testutil.Run(NewApp(), "task", "view", "123", "--template", "{{.name}}")

	exit, stdout, stderr := commandResult21.Exit, commandResult21.Stdout, commandResult21.Stderr
	if exit != 0 || stderr != "" || stdout != "Test task\n" {
		t.Fatalf("template exit = %d, stdout = %q, stderr = %s", exit, stdout, stderr)
	}
	commandResult22 := testutil.Run(NewApp(), "task", "view", "123", "--plain")
	exit, stdout, stderr = commandResult22.Exit, commandResult22.Stdout, commandResult22.Stderr
	if exit != 0 || stderr != "" {
		t.Fatalf("plain exit = %d, stderr = %s", exit, stderr)
	}
	if !strings.HasPrefix(stdout, "name\tTest task\ngid\t123\n") {
		t.Fatalf("detail plain output = %q", stdout)
	}
	if !strings.Contains(stdout, "memberships\t10\tAlpha\t20\tTodo\n") {
		t.Fatalf("detail plain output lacks stable child row: %q", stdout)
	}
}

func TestInvalidTransformsFailBeforeAPIRequest(t *testing.T) {
	var requests atomic.Int32
	server := testutil.MockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("ASANA_TOKEN", "transform-validation-token")
	t.Setenv("ASANA_API_BASE_URL", server.URL)

	for _, args := range [][]string{
		{"workspace", "list", "--jq", "["},
		{"workspace", "list", "--template", "{{"},
	} {
		commandResult23 := testutil.Run(NewApp(), args...)
		exit, stdout, stderr := commandResult23.Exit, commandResult23.Stdout, commandResult23.Stderr
		if exit != 1 {
			t.Fatalf("%v exit = %d, stderr = %s", args, exit, stderr)
		}
		if stdout != "" {
			t.Fatalf("%v wrote stdout on validation failure: %q", args, stdout)
		}
	}
	if requests.Load() != 0 {
		t.Fatalf("invalid transforms made %d API request(s)", requests.Load())
	}
}

func TestUnsupportedMetadataCombinationsFailBeforeAPIRequest(t *testing.T) {
	var requests atomic.Int32
	server := testutil.MockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("ASANA_TOKEN", "metadata-validation-token")
	t.Setenv("ASANA_API_BASE_URL", server.URL)

	for _, args := range [][]string{
		{"workspace", "list", "--include-meta"},
		{"workspace", "list", "--include-meta", "--plain"},
		{"task", "create", "Test", "--project-gid", "100", "--include-meta", "--dry-run", "--json"},
	} {
		commandResult24 := testutil.Run(NewApp(), args...)
		exit, stdout, stderr := commandResult24.Exit, commandResult24.Stdout, commandResult24.Stderr
		if exit != 1 || stdout != "" || stderr == "" {
			t.Fatalf("%v exit = %d, stdout = %q, stderr = %q", args, exit, stdout, stderr)
		}
	}
	if requests.Load() != 0 {
		t.Fatalf("unsupported metadata combinations made %d API request(s)", requests.Load())
	}
}

func TestDryRunUsesFrameworkPreviewWithoutMutation(t *testing.T) {
	var mutations atomic.Int32
	server := testutil.MockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/projects/100":
			_, _ = w.Write([]byte(`{"data":{"gid":"100","name":"Alpha"}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/projects/100/sections":
			_, _ = w.Write([]byte(`{"data":[]}`))
		default:
			mutations.Add(1)
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("ASANA_TOKEN", "dry-run-preview-token")
	t.Setenv("ASANA_API_BASE_URL", server.URL)

	commandResult25 := testutil.Run(NewApp(), "task", "create", "Test", "--project-gid", "100", "--dry-run", "--json")

	exit, stdout, stderr := commandResult25.Exit, commandResult25.Stdout, commandResult25.Stderr
	if exit != 0 || stderr != "" {
		t.Fatalf("exit = %d, stderr = %s", exit, stderr)
	}
	var preview struct {
		DryRun bool              `json:"dry_run"`
		Method string            `json:"method"`
		Path   string            `json:"path"`
		Body   map[string]string `json:"body"`
	}
	if err := json.Unmarshal([]byte(stdout), &preview); err != nil {
		t.Fatalf("preview is not JSON: %v\n%s", err, stdout)
	}
	if !preview.DryRun || preview.Method != http.MethodPost || preview.Path != "/tasks" {
		t.Fatalf("unexpected preview: %#v", preview)
	}
	if preview.Body["name"] != "Test" || preview.Body["projects"] != `["100"]` {
		t.Fatalf("preview does not contain the request body: %#v", preview.Body)
	}
	if mutations.Load() != 0 {
		t.Fatalf("dry-run made %d mutation request(s)", mutations.Load())
	}
}

func TestDryRunValidatesTargetBeforePreview(t *testing.T) {
	var reads atomic.Int32
	var mutations atomic.Int32
	server := testutil.MockServer(t, http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if request.Method == http.MethodGet && request.URL.Path == "/tasks/404" {
			reads.Add(1)
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"errors":[{"message":"task does not exist"}]}`))
			return
		}
		mutations.Add(1)
		http.Error(w, "unexpected request", http.StatusInternalServerError)
	}))
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("ASANA_TOKEN", "dry-run-validation-token")
	t.Setenv("ASANA_API_BASE_URL", server.URL)

	result := testutil.Run(NewApp(), "task", "delete", "404", "--dry-run", "--json")
	if result.Exit != rungrad.ExitNotFound {
		t.Fatalf("exit = %d, stdout = %q, stderr = %q; want not-found", result.Exit, result.Stdout, result.Stderr)
	}
	if result.Stdout != "" || !strings.Contains(result.Stderr, "task does not exist") {
		t.Fatalf("stdout = %q, stderr = %q", result.Stdout, result.Stderr)
	}
	if reads.Load() != 1 {
		t.Fatalf("dry-run made %d validation reads, want 1", reads.Load())
	}
	if mutations.Load() != 0 {
		t.Fatalf("dry-run made %d mutation request(s)", mutations.Load())
	}
}

func TestDryRunPreviewRedactsRegisteredCredential(t *testing.T) {
	const token = "dry-run-body-secret-token"
	var mutations atomic.Int32
	server := testutil.MockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/projects/100" {
			_, _ = w.Write([]byte(`{"data":{"gid":"100","name":"Alpha"}}`))
			return
		}
		if r.Method == http.MethodGet && r.URL.Path == "/projects/100/sections" {
			_, _ = w.Write([]byte(`{"data":[]}`))
			return
		}
		mutations.Add(1)
		http.NotFound(w, r)
	}))
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("ASANA_TOKEN", token)
	t.Setenv("ASANA_API_BASE_URL", server.URL)

	commandResult26 := testutil.Run(NewApp(), "task", "create", token, "--project-gid", "100", "--dry-run", "--json")

	exit, stdout, stderr := commandResult26.Exit, commandResult26.Stdout, commandResult26.Stderr
	if exit != 0 || stderr != "" {
		t.Fatalf("exit = %d, stderr = %s", exit, stderr)
	}
	if strings.Contains(stdout, token) || !strings.Contains(stdout, "[REDACTED]") {
		t.Fatalf("dry-run preview was not safely redacted: %s", stdout)
	}
	if mutations.Load() != 0 {
		t.Fatalf("dry-run made %d mutation request(s)", mutations.Load())
	}
}

func TestDryRunPreviewUsesConstructedGoalMetricRequest(t *testing.T) {
	var mutations atomic.Int32
	server := testutil.MockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			mutations.Add(1)
		}
		if r.Method == http.MethodGet && r.URL.Path == "/goals/123" {
			_, _ = w.Write([]byte(`{"data":{"gid":"123"}}`))
			return
		}
		http.NotFound(w, r)
	}))
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("ASANA_TOKEN", "goal-preview-token")
	t.Setenv("ASANA_API_BASE_URL", server.URL)

	commandResult27 := testutil.Run(NewApp(), "goal", "metric", "set", "123", "--current-value", "50", "--target-value", "100", "--unit", "percentage", "--precision", "2", "--dry-run", "--json")

	exit, stdout, stderr := commandResult27.Exit, commandResult27.Stdout, commandResult27.Stderr
	if exit != 0 || stderr != "" {
		t.Fatalf("exit = %d, stderr = %s", exit, stderr)
	}
	var preview struct {
		Method string            `json:"method"`
		Path   string            `json:"path"`
		Body   map[string]string `json:"body"`
	}
	if err := json.Unmarshal([]byte(stdout), &preview); err != nil {
		t.Fatalf("preview is not JSON: %v\n%s", err, stdout)
	}
	if preview.Method != http.MethodPost || preview.Path != "/goals/123/setMetric" {
		t.Fatalf("unexpected goal metric route: %#v", preview)
	}
	if preview.Body["current_number_value"] != "50" || preview.Body["target_number_value"] != "100" || preview.Body["unit"] != "percentage" || preview.Body["precision"] != "2" {
		t.Fatalf("unexpected goal metric body: %#v", preview.Body)
	}
	if mutations.Load() != 0 {
		t.Fatalf("dry-run made %d mutation request(s)", mutations.Load())
	}

	commandResult28 := testutil.Run(NewApp(), "goal", "metric", "set", "123", "--current-value", "25", "--dry-run", "--json")

	exit, stdout, stderr = commandResult28.Exit, commandResult28.Stdout, commandResult28.Stderr
	if exit != 0 || stderr != "" {
		t.Fatalf("current-value exit = %d, stderr = %s", exit, stderr)
	}
	preview = struct {
		Method string            `json:"method"`
		Path   string            `json:"path"`
		Body   map[string]string `json:"body"`
	}{}
	if err := json.Unmarshal([]byte(stdout), &preview); err != nil {
		t.Fatalf("current-value preview is not JSON: %v\n%s", err, stdout)
	}
	if preview.Method != http.MethodPost || preview.Path != "/goals/123/setMetricCurrentValue" {
		t.Fatalf("unexpected current-value route: %#v", preview)
	}
	if len(preview.Body) != 1 || preview.Body["current_number_value"] != "25" {
		t.Fatalf("unexpected current-value body: %#v", preview.Body)
	}
	if mutations.Load() != 0 {
		t.Fatalf("current-value dry-run made %d mutation request(s)", mutations.Load())
	}
}

func TestDependencyPlainOutputUsesCommandColumns(t *testing.T) {
	server := testutil.MockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/tasks/123/dependencies" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(`{"data":[{"gid":"456","name":"Dependency"}]}`))
	}))
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("ASANA_TOKEN", "dependency-plain-token")
	t.Setenv("ASANA_API_BASE_URL", server.URL)

	commandResult29 := testutil.Run(NewApp(), "task", "dependency", "list", "123", "--plain")

	exit, stdout, stderr := commandResult29.Exit, commandResult29.Stdout, commandResult29.Stderr
	if exit != 0 || stderr != "" {
		t.Fatalf("exit = %d, stderr = %s", exit, stderr)
	}
	if stdout != "456\tDependency\n" {
		t.Fatalf("plain output = %q", stdout)
	}
}

func TestAttachmentDryRunCapturesMultipartRequestWithoutReadingFile(t *testing.T) {
	var validations atomic.Int32
	var mutations atomic.Int32
	server := testutil.MockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/tasks/123" {
			validations.Add(1)
			_, _ = w.Write([]byte(`{"data":{"gid":"123"}}`))
			return
		}
		mutations.Add(1)
		http.Error(w, "unexpected request", http.StatusInternalServerError)
	}))
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("ASANA_TOKEN", "attachment-preview-token")
	t.Setenv("ASANA_API_BASE_URL", server.URL)

	missingFile := filepath.Join(t.TempDir(), "report.pdf")
	commandResult30 := testutil.Run(NewApp(), "attachment", "upload", missingFile, "--task", "123", "--dry-run", "--json")
	exit, stdout, stderr := commandResult30.Exit, commandResult30.Stdout, commandResult30.Stderr
	if exit != 0 || stderr != "" {
		t.Fatalf("exit = %d, stderr = %s", exit, stderr)
	}
	var preview struct {
		Method string            `json:"method"`
		Path   string            `json:"path"`
		Body   map[string]string `json:"body"`
	}
	if err := json.Unmarshal([]byte(stdout), &preview); err != nil {
		t.Fatalf("preview is not JSON: %v\n%s", err, stdout)
	}
	if preview.Method != http.MethodPost || preview.Path != "/attachments" {
		t.Fatalf("unexpected attachment route: %#v", preview)
	}
	if preview.Body["parent"] != "123" || preview.Body["file"] != "report.pdf" {
		t.Fatalf("unexpected attachment body: %#v", preview.Body)
	}
	if validations.Load() != 1 {
		t.Fatalf("attachment dry-run made %d validation reads, want 1", validations.Load())
	}
	if mutations.Load() != 0 {
		t.Fatalf("attachment dry-run made %d mutation request(s)", mutations.Load())
	}
}

func TestDryRunPreviewsConstructedMembershipAndDuplicateRequests(t *testing.T) {
	var validations atomic.Int32
	var mutations atomic.Int32
	server := testutil.MockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/tasks/123" {
			validations.Add(1)
			_, _ = w.Write([]byte(`{"data":{"gid":"123"}}`))
			return
		}
		mutations.Add(1)
		http.Error(w, "unexpected request", http.StatusInternalServerError)
	}))
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("ASANA_TOKEN", "constructed-preview-token")
	t.Setenv("ASANA_API_BASE_URL", server.URL)

	tests := []struct {
		name     string
		args     []string
		wantPath string
		wantBody map[string]string
	}{
		{
			name:     "task project membership",
			args:     []string{"task", "project", "add", "123", "--project-gid", "456", "--section-gid", "789", "--insert-after", "555", "--dry-run", "--json"},
			wantPath: "/tasks/123/addProject",
			wantBody: map[string]string{
				"project":      "456",
				"section":      "789",
				"insert_after": "555",
			},
		},
		{
			name:     "project duplication",
			args:     []string{"project", "duplicate", "123", "--name", "Copy", "--include", "task_dates", "--schedule-start-on", "2026-08-10", "--schedule-skip-weekends", "--dry-run", "--json"},
			wantPath: "/projects/123/duplicate",
			wantBody: map[string]string{
				"name":           "Copy",
				"include":        `["task_dates"]`,
				"schedule_dates": `{"should_skip_weekends":true,"start_on":"2026-08-10"}`,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			commandResult31 := testutil.Run(NewApp(), test.args...)
			exit, stdout, stderr := commandResult31.Exit, commandResult31.Stdout, commandResult31.Stderr
			if exit != 0 || stderr != "" {
				t.Fatalf("exit = %d, stderr = %s", exit, stderr)
			}
			var preview struct {
				Method string            `json:"method"`
				Path   string            `json:"path"`
				Body   map[string]string `json:"body"`
			}
			if err := json.Unmarshal([]byte(stdout), &preview); err != nil {
				t.Fatalf("preview is not JSON: %v\n%s", err, stdout)
			}
			if preview.Method != http.MethodPost || preview.Path != test.wantPath {
				t.Fatalf("unexpected route: %#v", preview)
			}
			if len(preview.Body) != len(test.wantBody) {
				t.Fatalf("body = %#v, want %#v", preview.Body, test.wantBody)
			}
			for key, want := range test.wantBody {
				if preview.Body[key] != want {
					t.Fatalf("body[%q] = %q, want %q; body = %#v", key, preview.Body[key], want, preview.Body)
				}
			}
		})
	}

	if validations.Load() != 1 {
		t.Fatalf("dry-runs made %d validation reads, want 1", validations.Load())
	}
	if mutations.Load() != 0 {
		t.Fatalf("dry-runs made %d mutation request(s)", mutations.Load())
	}
}

func TestWriteCommandPlainOutputUsesMutationFields(t *testing.T) {
	var mutations atomic.Int32
	server := testutil.MockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/projects/100":
			_, _ = w.Write([]byte(`{"data":{"gid":"100","name":"Alpha"}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/projects/100/sections":
			_, _ = w.Write([]byte(`{"data":[]}`))
		case r.Method == http.MethodPost && r.URL.Path == "/tasks":
			mutations.Add(1)
			_, _ = w.Write([]byte(`{"data":{"gid":"200","name":"Test","completed":false,"memberships":[{"project":{"gid":"100","name":"Alpha"},"section":null}]}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("ASANA_TOKEN", "plain-mutation-token")
	t.Setenv("ASANA_API_BASE_URL", server.URL)

	commandResult32 := testutil.Run(NewApp(), "task", "create", "Test", "--project-gid", "100", "--plain")

	exit, stdout, stderr := commandResult32.Exit, commandResult32.Stdout, commandResult32.Stderr
	if exit != 0 || stderr != "" {
		t.Fatalf("exit = %d, stderr = %s", exit, stderr)
	}
	if stdout != "created\t200\tTest\tfalse\n" {
		t.Fatalf("mutation plain output = %q", stdout)
	}
	if mutations.Load() != 1 {
		t.Fatalf("write command made %d mutation request(s)", mutations.Load())
	}
}

func TestSuccessfulOutputRedactsRegisteredCredential(t *testing.T) {
	const token = "successful-output-secret-token"
	server := testutil.MockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Asana-Request-Id", token)
		_, _ = fmt.Fprintf(w, `{"data":[{"gid":"1","name":%q}]}`, token)
	}))
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("ASANA_TOKEN", token)
	t.Setenv("ASANA_API_BASE_URL", server.URL)

	for _, args := range [][]string{
		{"workspace", "list", "--json"},
		{"workspace", "list", "--plain"},
		{"workspace", "list", "--template", `{{(index . 0).name}}`},
		{"workspace", "list", "--include-meta", "--json"},
	} {
		commandResult33 := testutil.Run(NewApp(), args...)
		exit, stdout, stderr := commandResult33.Exit, commandResult33.Stdout, commandResult33.Stderr
		if exit != 0 || stderr != "" {
			t.Fatalf("%v exit = %d, stderr = %s", args, exit, stderr)
		}
		if strings.Contains(stdout, token) || !strings.Contains(stdout, "[REDACTED]") {
			t.Fatalf("%v output was not safely redacted: %s", args, stdout)
		}
	}
}
