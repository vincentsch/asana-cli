// Package cmd tests client-side checks on task titles.
package cmd

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
)

// A blank title is refused before any request is made. These tests set no
// API client, so a request would fail with a different error.
func TestTaskCreateRejectsBlankTitle(t *testing.T) {
	for _, title := range []string{"", "   "} {
		cmd, _ := newTaskCreateTestCmd()
		setFlag(t, cmd, "project-gid", "1001")
		err := runTaskCreate(cmd, []string{title})
		if err == nil || err.Error() != "task title cannot be empty" {
			t.Fatalf("runTaskCreate(%q) error = %v, want a blank-title error", title, err)
		}
	}
}

func TestTaskUpdateRejectsBlankName(t *testing.T) {
	cmd := &cobra.Command{}
	attachTestRuntime(cmd)
	cmd.SetOut(&bytes.Buffer{})
	cmd.Flags().String("name", "", "")
	cmd.Flags().String("notes", "", "")
	cmd.Flags().String("due", "", "")
	cmd.Flags().String("assignee", "", "")
	cmd.Flags().Bool("clear-due", false, "")
	cmd.Flags().Bool("clear-assignee", false, "")
	cmd.Flags().Bool("clear-notes", false, "")
	cmd.Flags().Bool("dry-run", false, "")
	setFlag(t, cmd, "name", "  ")

	err := runTaskUpdate(cmd, []string{"1234567890"})
	if err == nil || err.Error() != "--name cannot be empty" {
		t.Fatalf("runTaskUpdate with blank --name error = %v, want --name cannot be empty", err)
	}
}
