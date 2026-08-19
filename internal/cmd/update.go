// Package cmd adapts rungrad's update command to Asana release artifacts.
package cmd

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/vincentsch/asana-cli/internal/version"
	"github.com/vincentsch/rungrad"
	rungradupdate "github.com/vincentsch/rungrad/update"
)

const (
	releaseOwner = "vincentsch"
	releaseRepo  = "asana-cli"
)

var updateHTTPClient = &http.Client{Timeout: 30 * time.Second}

// These build stamps provide an offline release-fetcher seam for the
// subprocess conformance binary. Normal builds leave both values empty and use
// GitHub plus the real installer.
var (
	updateFixtureVersion  string
	updateInstallDisabled string
)

type fixedReleaseFetcher struct {
	version string
}

func (f fixedReleaseFetcher) Latest() (rungradupdate.Release, error) {
	return rungradupdate.Release{Version: f.version}, nil
}

type asanaReleaseFetcher struct {
	delegate rungradupdate.Fetcher
}

func (f asanaReleaseFetcher) Latest() (rungradupdate.Release, error) {
	release, err := f.delegate.Latest()
	if err != nil && strings.Contains(err.Error(), "status 404") {
		// An empty repository is a valid state for a newly installed development
		// build, not an upstream failure requiring user action.
		return rungradupdate.Release{}, nil
	}
	return release, err
}

func newUpdateCommand() *rungrad.Command {
	var fetcher rungradupdate.Fetcher = asanaReleaseFetcher{delegate: rungradupdate.GitHubFetcher{
		Owner: releaseOwner,
		Repo:  releaseRepo,
	}}
	if updateFixtureVersion != "" {
		fetcher = fixedReleaseFetcher{version: updateFixtureVersion}
	}

	apply := installAsanaRelease
	if updateInstallDisabled != "" {
		apply = nil
	}
	config := rungradupdate.CommandConfig{
		CurrentVersion: version.Short(),
		Fetcher:        fetcher,
		Apply:          apply,
		ToolName:       "asana",
	}
	command := rungradupdate.Command(config)
	command.Run = runAsanaUpdate(config)
	command.Short = "Update asana-cli to the latest version"
	command.Long = `Check for and install the latest version of asana-cli.

Downloads the appropriate binary for your OS/architecture from GitHub releases
and verifies it against the release checksums before replacing the current
executable. May require sudo on Unix systems. Automatic replacement is not
supported on Windows. Use --check to report update availability without changing
any files.`
	command.Examples = []string{
		"asana update --check",
		"asana update --check --json",
		"asana update",
	}
	command.Related = []string{"version"}
	command.Args = cobra.NoArgs
	return command
}

func runAsanaUpdate(config rungradupdate.CommandConfig) func(*rungrad.Factory, *cobra.Command, []string) error {
	return func(factory *rungrad.Factory, command *cobra.Command, args []string) error {
		check, _ := command.Flags().GetBool("check")
		release, err := config.Fetcher.Latest()
		if err != nil {
			return rungrad.NewError(rungrad.ExitAPI, "failed to check for updates: "+err.Error())
		}
		result := rungradupdate.Evaluate(config.CurrentVersion, release)
		if check || factory.DryRun() || !result.Available {
			return factory.WriteResult(result, func(writer io.Writer) {
				renderUpdateStatus(writer, result, false)
			})
		}
		if config.Apply == nil {
			return rungrad.NewError(rungrad.ExitAPI, "automatic install is unavailable; download the latest release manually")
		}
		if err := config.Apply(release); err != nil {
			return rungrad.NewError(rungrad.ExitAPI, "failed to install update: "+err.Error())
		}
		installed := result
		installed.Current = result.Latest
		installed.Available = false
		installed.Status = rungradupdate.StatusUpToDate
		return factory.WriteResult(installed, func(writer io.Writer) {
			renderUpdateStatus(writer, result, true)
		})
	}
}

func renderUpdateStatus(writer io.Writer, result rungradupdate.Result, installed bool) {
	fmt.Fprintf(writer, "Current version: %s\n", result.Current)
	if result.Latest == "" {
		fmt.Fprintln(writer, "No releases found yet")
		return
	}
	displayLatest := strings.TrimPrefix(result.Latest, "v")
	fmt.Fprintf(writer, "Latest version:  %s\n", displayLatest)
	if installed {
		fmt.Fprintf(writer, "\nUpdating...\n\nUpdated to version %s!\n", displayLatest)
		return
	}
	switch result.Status {
	case rungradupdate.StatusUpToDate:
		fmt.Fprintln(writer, "\nYou're already on the latest version!")
	case rungradupdate.StatusUpdateAvailable:
		fmt.Fprintf(writer, "\nUpdate available: %s\n", releasePageURL(result.Latest))
	case rungradupdate.StatusDevelopmentBuild:
		fmt.Fprintln(writer, "\nDevelopment builds are not updated automatically.")
	case rungradupdate.StatusNewerThanLatest:
		fmt.Fprintln(writer, "\nThis build is newer than the latest release.")
	case rungradupdate.StatusUnknownLatest:
		fmt.Fprintln(writer, "\nThe latest release version could not be evaluated.")
	}
}

func releasePageURL(releaseVersion string) string {
	return fmt.Sprintf("https://github.com/%s/%s/releases/tag/%s", releaseOwner, releaseRepo, releaseVersion)
}

func installAsanaRelease(release rungradupdate.Release) error {
	if runtime.GOOS == "windows" {
		return fmt.Errorf("automatic replacement is not supported on windows; download the latest release manually")
	}
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("get executable path: %w", err)
	}

	assetName := asanaAssetName(runtime.GOOS, runtime.GOARCH)
	tmpFile := execPath + ".new"
	downloadURL := releaseAssetURL(release, assetName)
	curlCmd := exec.Command("curl", "-fsSL", "-o", tmpFile, downloadURL)
	curlCmd.Stdout = os.Stdout
	curlCmd.Stderr = os.Stderr
	if err := curlCmd.Run(); err != nil {
		_ = os.Remove(tmpFile)
		return fmt.Errorf("download update: %w", err)
	}
	if err := verifyDownloadedAsset(release, assetName, tmpFile); err != nil {
		_ = os.Remove(tmpFile)
		return err
	}

	if err := os.Chmod(tmpFile, 0o755); err != nil {
		_ = os.Remove(tmpFile)
		return fmt.Errorf("set update permissions: %w", err)
	}
	if err := os.Rename(tmpFile, execPath); err != nil {
		if runtime.GOOS == "windows" {
			_ = os.Remove(tmpFile)
			return fmt.Errorf("replace binary: %w", err)
		}
		mvCmd := exec.Command("sudo", "mv", tmpFile, execPath)
		mvCmd.Stdin = os.Stdin
		mvCmd.Stdout = os.Stdout
		mvCmd.Stderr = os.Stderr
		if sudoErr := mvCmd.Run(); sudoErr != nil {
			_ = os.Remove(tmpFile)
			return fmt.Errorf("replace binary: %w", sudoErr)
		}
	}
	return nil
}

func verifyDownloadedAsset(release rungradupdate.Release, assetName, path string) error {
	checksumURL := releaseAssetURL(release, "checksums.txt")
	if checksumURL == "" {
		return fmt.Errorf("checksum asset not found")
	}
	response, err := updateHTTPClient.Get(checksumURL)
	if err != nil {
		return fmt.Errorf("download checksums: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("download checksums: status %d", response.StatusCode)
	}
	content, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return fmt.Errorf("read checksums: %w", err)
	}
	expected, ok := checksumForAsset(string(content), assetName)
	if !ok {
		return fmt.Errorf("checksum for %s not found", assetName)
	}
	actual, err := fileSHA256(path)
	if err != nil {
		return err
	}
	if actual != expected {
		return fmt.Errorf("checksum verification failed for %s", assetName)
	}
	return nil
}

func releaseAssetURL(release rungradupdate.Release, assetName string) string {
	for _, asset := range release.Assets {
		if asset.Name == assetName {
			return asset.URL
		}
	}
	if release.Version == "" {
		return ""
	}
	return fmt.Sprintf("https://github.com/%s/%s/releases/download/%s/%s", releaseOwner, releaseRepo, release.Version, assetName)
}

func checksumForAsset(content, assetName string) (string, bool) {
	for _, line := range strings.Split(content, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		name := strings.TrimPrefix(fields[len(fields)-1], "*")
		if name == assetName {
			return strings.ToLower(fields[0]), true
		}
	}
	return "", false
}

func fileSHA256(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open downloaded asset: %w", err)
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", fmt.Errorf("hash downloaded asset: %w", err)
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func asanaAssetName(goos, goarch string) string {
	name := fmt.Sprintf("asana-%s-%s", goos, goarch)
	if goos == "windows" {
		name += ".exe"
	}
	return name
}
