// Package cmd implements task comment update and delete subcommands.
package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/vincentsch/asana-cli/internal/api"
)

var taskCommentUpdateCmd = &cobra.Command{
	Use:   "update <story-gid> <message>",
	Short: "Update a story/comment",
	Long: `Update the text of an existing comment.

Use "-" as the message to read from stdin.

See also: task comment, task comment delete`,
	Example: `  # Update a comment
  asana task comment update 1234567890123456 "Updated message"

  # Update from stdin
  echo "New text" | asana task comment update 1234567890123456 -

  # Preview without updating
  asana task comment update 1234567890123456 "Test" --dry-run`,
	Args: cobra.ExactArgs(2),
	RunE: runStoryUpdate,
}

var taskCommentDeleteCmd = &cobra.Command{
	Use:   "delete <story-gid>",
	Short: "Delete a story/comment",
	Long: `Permanently delete a comment.

This action cannot be undone.

See also: task comment, task comment update`,
	Example: `  # Delete a comment
  asana task comment delete 1234567890123456 --confirm

  # Preview without deleting
  asana task comment delete 1234567890123456 --dry-run`,
	Args: cobra.ExactArgs(1),
	RunE: runStoryDelete,
}

func init() {

	taskCommentCmd.AddCommand(taskCommentUpdateCmd)
	taskCommentCmd.AddCommand(taskCommentDeleteCmd)
}

func runStoryUpdate(cmd *cobra.Command, args []string) error {
	storyGID := args[0]
	message := args[1]
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	if err := validateGID(storyGID, "story gid"); err != nil {
		return err
	}

	if message == "-" {
		text, err := readCommentInput(cmd.InOrStdin())
		if err != nil {
			return err
		}
		message = text
	} else if strings.TrimSpace(message) == "" {
		return fmt.Errorf("message cannot be empty")
	}

	if dryRun {
		output := writeOutput{
			Action: "updated",
			DryRun: true,
			Story: &storyResult{
				GID:  storyGID,
				Text: message,
			},
		}
		if runtimeOutputJSON(cmd) {
			return writeWriteJSON(cmd, output)
		}
		_, err := fmt.Fprintf(cmd.OutOrStdout(), "[dry-run] Would update story %s\n", storyGID)
		return err
	}

	payload, err := runtimeClient(cmd).Put(cmd.Context(), "/stories/"+storyGID, api.RequestBody{
		Data: map[string]string{"text": message},
		Options: &api.RequestOptions{
			Fields: splitFields(storyWriteFields),
		},
	})
	if err != nil {
		return err
	}

	var response api.Response[api.Story]
	if err := json.Unmarshal(payload, &response); err != nil {
		return &api.ResponseError{Err: err}
	}

	output := writeOutput{
		Action: "updated",
		DryRun: false,
		Story: &storyResult{
			GID:       response.Data.GID,
			Text:      response.Data.Text,
			CreatedAt: response.Data.CreatedAt,
			CreatedBy: userRefFromUser(response.Data.CreatedBy),
		},
	}
	if runtimeOutputJSON(cmd) {
		return writeWriteJSON(cmd, output)
	}
	_, err = fmt.Fprintln(cmd.OutOrStdout(), "Story updated")
	return err
}

func runStoryDelete(cmd *cobra.Command, args []string) error {
	storyGID := args[0]
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	if err := validateGID(storyGID, "story gid"); err != nil {
		return err
	}

	if dryRun {
		output := writeOutput{
			Action: "deleted",
			DryRun: true,
			Story: &storyResult{
				GID: storyGID,
			},
		}
		if runtimeOutputJSON(cmd) {
			return writeWriteJSON(cmd, output)
		}
		_, err := fmt.Fprintf(cmd.OutOrStdout(), "[dry-run] Would delete story %s\n", storyGID)
		return err
	}

	if err := runtimeClient(cmd).Delete(cmd.Context(), "/stories/"+storyGID); err != nil {
		return err
	}

	output := writeOutput{
		Action: "deleted",
		DryRun: false,
		Story: &storyResult{
			GID: storyGID,
		},
	}
	if runtimeOutputJSON(cmd) {
		return writeWriteJSON(cmd, output)
	}
	_, err := fmt.Fprintln(cmd.OutOrStdout(), "Story deleted")
	return err
}
