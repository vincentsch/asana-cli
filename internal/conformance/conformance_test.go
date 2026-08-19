// Package conformance tests the built Asana CLI against rungrad's specification.
package conformance_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync/atomic"
	"testing"

	"github.com/vincentsch/rungrad/conformance"
	"github.com/vincentsch/rungrad/testutil"
)

func TestAsanaConformance(t *testing.T) {
	var mutationRequests atomic.Int32
	server := testutil.MockServer(t, http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet {
			mutationRequests.Add(1)
			http.Error(w, "mock fixtures are read-only", http.StatusInternalServerError)
			return
		}
		switch request.URL.Path {
		case "/workspaces":
			w.Header().Set("X-Asana-Request-Id", "named-metadata-fixture")
			fmt.Fprint(w, `{"data":[{"gid":"3","name":"Duplicate"},{"gid":"1","name":"Alpha"},{"gid":"2","name":"Duplicate"}]}`)
		case "/users/me":
			fmt.Fprint(w, `{"data":{"gid":"7","name":"Mock User","email":"mock@example.invalid","workspaces":[]}}`)
		case "/tasks/100", "/tasks/600":
			fmt.Fprintf(w, `{"data":{"gid":%q,"name":"Validated fixture"}}`, filepath.Base(request.URL.Path))
		case "/tasks/404":
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, `{"errors":[{"message":"named not-found fixture"}]}`)
		case "/tasks/500":
			fmt.Fprint(w, `{"data":`)
		case "/tasks/403":
			w.WriteHeader(http.StatusForbidden)
			fmt.Fprint(w, `{"errors":[{"message":"named forbidden fixture"}]}`)
		case "/tasks/429":
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			fmt.Fprint(w, `{"errors":[{"message":"named rate-limit fixture"}]}`)
		default:
			http.NotFound(w, request)
		}
	}))

	buildDir := t.TempDir()
	buildScoredBinary(t, buildDir)
	proxy := buildProxy(t, buildDir, server.URL)
	assertMetadataFixture(t, proxy)
	rules, err := conformance.DefaultRuleset()
	if err != nil {
		t.Fatal(err)
	}
	runner, err := conformance.NewRunner(conformance.Target{
		Path:         proxy,
		Read:         []string{"workspace", "list"},
		Mutate:       []string{"task", "done", "100"},
		Auth:         []string{"user", "me"},
		Ambiguous:    []string{"project", "list", "--workspace", "Duplicate"},
		NotFound:     []string{"task", "view", "404"},
		APIError:     []string{"task", "view", "500"},
		Forbidden:    []string{"task", "view", "403"},
		RateLimited:  []string{"task", "view", "429"},
		Destructive:  []string{"task", "delete", "600"},
		HasUpdate:    true,
		Secret:       []string{"user", "me"},
		SecretEnv:    "ASANA_TOKEN",
		ManifestMode: conformance.ManifestModeRequired,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = runner.Close() })
	if err := runner.DiscoverManifest(); err != nil {
		t.Fatalf("discover manifest: %v", err)
	}

	score := runner.Score(rules)
	t.Log("\n" + score.Report())
	for _, rule := range score.Rules {
		if rule.Result == conformance.ResultNotApplicable {
			t.Errorf("probe %s was not applicable: %s", rule.ID, rule.Reason)
		}
		if rule.Result == conformance.ResultFail {
			t.Errorf("probe %s failed: %s", rule.ID, rule.Reason)
		}
	}
	if score.Manifest.Status != conformance.ManifestPresent {
		t.Errorf("manifest status = %q, want %q", score.Manifest.Status, conformance.ManifestPresent)
	}
	if score.Overall != 1 {
		t.Errorf("overall score = %.2f, want 1.00", score.Overall)
	}
	if mutationRequests.Load() != 0 {
		t.Errorf("conformance made %d mutation request(s)", mutationRequests.Load())
	}
}

func assertMetadataFixture(t *testing.T, proxy string) {
	t.Helper()
	configHome := t.TempDir()
	command := exec.Command(proxy, "workspace", "list", "--include-meta", "--json")
	command.Env = []string{
		"PATH=" + os.Getenv("PATH"),
		"HOME=" + configHome,
		"XDG_CONFIG_HOME=" + configHome,
		"TMPDIR=" + os.TempDir(),
		"TEMP=" + os.TempDir(),
		"TMP=" + os.TempDir(),
	}
	if runtime.GOOS == "windows" {
		command.Env = append(command.Env,
			"APPDATA="+configHome,
			"LOCALAPPDATA="+configHome,
			"SystemRoot="+os.Getenv("SystemRoot"),
			"SYSTEMROOT="+os.Getenv("SYSTEMROOT"),
		)
	}
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		t.Fatalf("metadata fixture: %v\nstdout=%s\nstderr=%s", err, stdout.String(), stderr.String())
	}
	var envelope struct {
		Meta struct {
			RequestID string         `json:"request_id"`
			Extra     map[string]any `json:"extra"`
		} `json:"meta"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatalf("metadata fixture JSON: %v\n%s", err, stdout.String())
	}
	if envelope.Meta.RequestID != "named-metadata-fixture" || envelope.Meta.Extra["endpoint"] != "/workspaces" {
		t.Fatalf("unexpected metadata fixture: %#v", envelope.Meta)
	}
}

func buildScoredBinary(t *testing.T, dir string) {
	t.Helper()
	name := "asana-target"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	path := filepath.Join(dir, name)
	ldflags := "-X github.com/vincentsch/asana-cli/internal/version.Version=v1.0.0 " +
		"-X github.com/vincentsch/asana-cli/internal/version.Commit=conform " +
		"-X github.com/vincentsch/asana-cli/internal/cmd.updateFixtureVersion=v1.1.0 " +
		"-X github.com/vincentsch/asana-cli/internal/cmd.updateInstallDisabled=true"
	build(t, path, ldflags, "./cmd/asana")
}

func buildProxy(t *testing.T, dir, endpoint string) string {
	t.Helper()
	name := "asana-conformance"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	path := filepath.Join(dir, name)
	ldflags := "-X main.apiBaseURL=" + endpoint
	build(t, path, ldflags, "./internal/conformanceproxy")
	return path
}

func build(t *testing.T, output, ldflags, pkg string) {
	t.Helper()
	command := exec.Command("go", "build", "-ldflags", ldflags, "-o", output, pkg)
	command.Dir = filepath.Join("..", "..")
	command.Env = append(os.Environ(), "GOPROXY=off")
	if result, err := command.CombinedOutput(); err != nil {
		t.Fatalf("build %s: %v\n%s", pkg, err, result)
	}
}
