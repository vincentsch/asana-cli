// Package version provides build-time version information for the CLI.
// The Version and Commit variables are set via ldflags during build.
package version

import (
	"runtime/debug"
	"strings"
)

// Version is the semantic version (e.g., "v1.0.0"). Set via ldflags.
var Version = "dev"

// Commit is the short git commit hash (e.g., "abc1234"). Set via ldflags.
var Commit = "dev"

func init() {
	// Fall back to Go's embedded build metadata when ldflags are not set (e.g., go install).
	if Version != "dev" && Commit != "dev" {
		return
	}

	info, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}

	if Version == "dev" && info.Main.Version != "" && info.Main.Version != "(devel)" {
		Version = info.Main.Version
	}

	if Commit == "dev" {
		if revision := commitFromSettings(info.Settings); revision != "" {
			Commit = revision
		} else if revision := commitFromVersion(info.Main.Version); revision != "" {
			Commit = revision
		}
	}
}

// String returns the full version string for display (e.g., "asana v1.0.0 (abc1234)").
func String() string {
	if Commit == "dev" && Version != "dev" {
		return "asana " + Version
	}
	return "asana " + Version + " (" + Commit + ")"
}

// Short returns just the version for Cobra's version flag.
func Short() string {
	return Version
}

// UserAgent returns the User-Agent string for HTTP requests.
func UserAgent() string {
	return "asana-cli/" + Version
}

func commitFromSettings(settings []debug.BuildSetting) string {
	for _, setting := range settings {
		if setting.Key == "vcs.revision" && setting.Value != "" {
			return shortCommit(setting.Value)
		}
	}
	return ""
}

func commitFromVersion(version string) string {
	if version == "" || version == "(devel)" {
		return ""
	}
	// Pseudo-versions encode the VCS revision as the last hyphen segment.
	base := strings.SplitN(version, "+", 2)[0]
	parts := strings.Split(base, "-")
	if len(parts) < 3 {
		return ""
	}
	commit := parts[len(parts)-1]
	if len(commit) < 7 || !isHex(commit) {
		return ""
	}
	return shortCommit(commit)
}

func shortCommit(commit string) string {
	if len(commit) > 7 {
		return commit[:7]
	}
	return commit
}

func isHex(value string) bool {
	for _, r := range value {
		switch {
		case r >= '0' && r <= '9':
		case r >= 'a' && r <= 'f':
		case r >= 'A' && r <= 'F':
		default:
			return false
		}
	}
	return true
}
