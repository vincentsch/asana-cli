// Package cmd tests completion command behavior.
package cmd

import (
	"strings"
	"testing"

	"github.com/vincentsch/rungrad/testutil"
)

func TestCompletionBash(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("ASANA_TOKEN", "")
	commandResult1 := testutil.Run(NewApp(), "completion", "bash")
	exit, stdout, stderr := commandResult1.Exit, commandResult1.Stdout, commandResult1.Stderr
	if exit != 0 {
		t.Fatalf("exit = %d, stderr = %s", exit, stderr)
	}
	if !strings.Contains(stdout, "asana") {
		t.Fatalf("expected output to contain command name, got %s", stdout)
	}
}

func TestCompletionInvalidShell(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("ASANA_TOKEN", "")
	commandResult2 := testutil.Run(NewApp(), "completion", "nope")
	exit, _, stderr := commandResult2.Exit, commandResult2.Stdout, commandResult2.Stderr
	if exit == 0 || stderr == "" {
		t.Fatalf("exit = %d, stderr = %q", exit, stderr)
	}
}
