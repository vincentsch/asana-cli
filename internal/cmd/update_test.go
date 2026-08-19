// Package cmd tests the Asana adapter around rungrad's update command.
package cmd

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vincentsch/asana-cli/internal/version"
	"github.com/vincentsch/rungrad"
	"github.com/vincentsch/rungrad/testutil"
	rungradupdate "github.com/vincentsch/rungrad/update"
)

func TestUpdateCheckUsesOfflineFetcherAndNeverInstalls(t *testing.T) {
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

	result := testutil.Run(NewApp(), "update", "--check", "--json")
	if result.Exit != 0 || result.Stderr != "" {
		t.Fatalf("exit = %d, stderr = %q", result.Exit, result.Stderr)
	}
	var status struct {
		Current   string `json:"current"`
		Latest    string `json:"latest"`
		Available bool   `json:"available"`
		Status    string `json:"status"`
	}
	if err := json.Unmarshal([]byte(result.Stdout), &status); err != nil {
		t.Fatalf("update check JSON: %v\n%s", err, result.Stdout)
	}
	if status.Current != "v1.0.0" || status.Latest != "v1.1.0" || !status.Available || status.Status != "update_available" {
		t.Fatalf("unexpected update status: %#v", status)
	}
	human := testutil.Run(NewApp(), "update", "--check")
	if human.Exit != 0 || human.Stderr != "" || !strings.Contains(human.Stdout, releasePageURL("v1.1.0")) {
		t.Fatalf("human update check exit = %d, stdout = %q, stderr = %q", human.Exit, human.Stdout, human.Stderr)
	}

	install := testutil.Run(NewApp(), "update")
	if install.Exit != 2 {
		t.Fatalf("disabled installer exit = %d, stderr = %q", install.Exit, install.Stderr)
	}
}

func TestPlainUpdateInstallsOnlyWhenEvaluationReportsAvailable(t *testing.T) {
	var installed rungradupdate.Release
	config := rungradupdate.CommandConfig{
		CurrentVersion: "v1.0.0",
		Fetcher:        fixedReleaseFetcher{version: "v1.1.0"},
		Apply: func(release rungradupdate.Release) error {
			installed = release
			return nil
		},
		ToolName: "asana",
	}
	newApp := func() *rungrad.App {
		app := rungrad.New(rungrad.AppConfig{Name: "asana", Version: config.CurrentVersion})
		command := rungradupdate.Command(config)
		command.Run = runAsanaUpdate(config)
		app.AddCommand(command)
		return app
	}

	result := testutil.Run(newApp(), "update")
	if result.Exit != 0 || result.Stderr != "" {
		t.Fatalf("plain update exit = %d, stderr = %q", result.Exit, result.Stderr)
	}
	if installed.Version != "v1.1.0" {
		t.Fatalf("installed release = %#v, want v1.1.0", installed)
	}
	if !strings.Contains(result.Stdout, "Updated to version 1.1.0!") {
		t.Fatalf("plain update output = %q", result.Stdout)
	}

	config.CurrentVersion = "dev"
	installed = rungradupdate.Release{}
	development := testutil.Run(newApp(), "update")
	if development.Exit != 0 || development.Stderr != "" || installed.Version != "" {
		t.Fatalf("development update exit = %d, installed = %#v, stderr = %q", development.Exit, installed, development.Stderr)
	}
	if !strings.Contains(development.Stdout, "Development builds are not updated automatically.") {
		t.Fatalf("development update output = %q", development.Stdout)
	}

	check := testutil.Run(newApp(), "update", "--check", "--json")
	if check.Exit != 0 || installed.Version != "" {
		t.Fatalf("development check exit = %d, installed = %#v, stderr = %q", check.Exit, installed, check.Stderr)
	}
	var status struct {
		Available bool   `json:"available"`
		Status    string `json:"status"`
	}
	if err := json.Unmarshal([]byte(check.Stdout), &status); err != nil {
		t.Fatalf("development check JSON: %v\n%s", err, check.Stdout)
	}
	if status.Available || status.Status != "development_build" {
		t.Fatalf("unexpected development check status: %#v", status)
	}
}

type failingReleaseFetcher struct {
	err error
}

func (f failingReleaseFetcher) Latest() (rungradupdate.Release, error) {
	return rungradupdate.Release{}, f.err
}

func TestAsanaReleaseFetcherTreatsMissingReleaseAsEmptyRepository(t *testing.T) {
	t.Parallel()
	fetcher := asanaReleaseFetcher{delegate: failingReleaseFetcher{err: errors.New("github releases: status 404")}}
	release, err := fetcher.Latest()
	if err != nil || release.Version != "" {
		t.Fatalf("Latest() = %#v, %v; want empty release without error", release, err)
	}
}

func TestAsanaAssetName(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		goos   string
		goarch string
		want   string
	}{
		{goos: "linux", goarch: "amd64", want: "asana-linux-amd64"},
		{goos: "darwin", goarch: "arm64", want: "asana-darwin-arm64"},
		{goos: "windows", goarch: "amd64", want: "asana-windows-amd64.exe"},
	} {
		test := test
		t.Run(test.want, func(t *testing.T) {
			t.Parallel()
			if got := asanaAssetName(test.goos, test.goarch); got != test.want {
				t.Fatalf("asanaAssetName(%q, %q) = %q, want %q", test.goos, test.goarch, got, test.want)
			}
		})
	}
}

func TestChecksumForAsset(t *testing.T) {
	t.Parallel()
	checksum, ok := checksumForAsset("abc123  asana-linux-amd64\nfff999  checksums.txt\n", "asana-linux-amd64")
	if !ok || checksum != "abc123" {
		t.Fatalf("checksumForAsset() = %q, %v; want abc123, true", checksum, ok)
	}
}

func TestVerifyDownloadedAssetChecksReleaseChecksum(t *testing.T) {
	t.Parallel()
	assetName := "asana-linux-amd64"
	payload := []byte("binary payload")
	sum := sha256.Sum256(payload)
	expected := hex.EncodeToString(sum[:])
	path := filepath.Join(t.TempDir(), assetName)
	if err := os.WriteFile(path, payload, 0o644); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/checksums.txt" {
			http.NotFound(writer, request)
			return
		}
		_, _ = writer.Write([]byte(expected + "  " + assetName + "\n"))
	}))
	t.Cleanup(server.Close)

	release := rungradupdate.Release{
		Version: "v1.2.3",
		Assets: []rungradupdate.Asset{{
			Name: "checksums.txt",
			URL:  server.URL + "/checksums.txt",
		}},
	}
	if err := verifyDownloadedAsset(release, assetName, path); err != nil {
		t.Fatalf("verifyDownloadedAsset() returned error: %v", err)
	}
}

func TestVerifyDownloadedAssetRejectsChecksumMismatch(t *testing.T) {
	t.Parallel()
	assetName := "asana-linux-amd64"
	path := filepath.Join(t.TempDir(), assetName)
	if err := os.WriteFile(path, []byte("binary payload"), 0o644); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		_, _ = writer.Write([]byte(strings.Repeat("0", 64) + "  " + assetName + "\n"))
	}))
	t.Cleanup(server.Close)

	release := rungradupdate.Release{
		Version: "v1.2.3",
		Assets: []rungradupdate.Asset{{
			Name: "checksums.txt",
			URL:  server.URL,
		}},
	}
	if err := verifyDownloadedAsset(release, assetName, path); err == nil {
		t.Fatal("verifyDownloadedAsset() succeeded for mismatched checksum")
	}
}
