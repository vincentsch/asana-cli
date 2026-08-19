// Package version tests build metadata parsing helpers.
package version

import (
	"runtime/debug"
	"testing"
)

func TestCommitFromVersion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		version string
		want    string
	}{
		{
			name:    "empty",
			version: "",
			want:    "",
		},
		{
			name:    "devel",
			version: "(devel)",
			want:    "",
		},
		{
			name:    "tag",
			version: "v1.2.3",
			want:    "",
		},
		{
			name:    "pseudo",
			version: "v0.0.0-20240102123456-abcdef123456",
			want:    "abcdef1",
		},
		{
			name:    "pseudo-with-incompatible",
			version: "v1.2.3-0.20240102123456-ABCDEF123456+incompatible",
			want:    "ABCDEF1",
		},
		{
			name:    "prerelease",
			version: "v1.2.3-rc.1",
			want:    "",
		},
		{
			name:    "non-hex-suffix",
			version: "v0.0.0-20240102123456-nothex",
			want:    "",
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if got := commitFromVersion(test.version); got != test.want {
				t.Fatalf("commitFromVersion(%q) = %q, want %q", test.version, got, test.want)
			}
		})
	}
}

func TestCommitFromSettings(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		settings []debug.BuildSetting
		want     string
	}{
		{
			name: "missing",
			settings: []debug.BuildSetting{
				{Key: "vcs.time", Value: "2024-01-01T00:00:00Z"},
			},
			want: "",
		},
		{
			name: "empty",
			settings: []debug.BuildSetting{
				{Key: "vcs.revision", Value: ""},
			},
			want: "",
		},
		{
			name: "revision",
			settings: []debug.BuildSetting{
				{Key: "vcs.revision", Value: "abcdef123456"},
			},
			want: "abcdef1",
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if got := commitFromSettings(test.settings); got != test.want {
				t.Fatalf("commitFromSettings(%v) = %q, want %q", test.settings, got, test.want)
			}
		})
	}
}

func TestStringOmitsMissingCommit(t *testing.T) {
	originalVersion := Version
	originalCommit := Commit
	t.Cleanup(func() {
		Version = originalVersion
		Commit = originalCommit
	})

	Version = "v1.2.3"
	Commit = "dev"

	if got := String(); got != "asana v1.2.3" {
		t.Fatalf("String() = %q, want %q", got, "asana v1.2.3")
	}
}
