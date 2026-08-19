// Package cmd implements attachment commands: list, view, upload, delete.
package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/vincentsch/asana-cli/internal/api"
)

var attachmentCmd = &cobra.Command{
	Use:   "attachment",
	Short: "Manage task attachments",
	Long: `Manage task attachments.

Attachments can be files uploaded directly or links to external URLs.
Use these commands to list, view, upload, or delete attachments on tasks.

See also: task view`,
}

var attachmentListCmd = &cobra.Command{
	Use:   "list",
	Short: "List attachments on a task",
	Long: `List all attachments on a task.

See also: attachment view, attachment upload`,
	Example: `  # List attachments on a task
  asana attachment list --task 1234567890123456

  # Limit output
  asana attachment list --task 1234567890123456 --limit 10

  # Output as JSON
  asana attachment list --task 1234567890123456 --json`,
	Args: cobra.NoArgs,
	RunE: runAttachmentList,
}

var attachmentViewCmd = &cobra.Command{
	Use:   "view <attachment-gid>",
	Short: "View attachment details",
	Long: `View details about an attachment including download URL.

See also: attachment list, attachment delete`,
	Example: `  # View attachment details
  asana attachment view 1234567890123456

  # Output as JSON
  asana attachment view 1234567890123456 --json`,
	Args: cobra.ExactArgs(1),
	RunE: runAttachmentView,
}

var attachmentUploadCmd = &cobra.Command{
	Use:   "upload [file]",
	Short: "Upload file or URL attachment to task",
	Long: `Upload a file or attach an external URL to a task.

For file uploads, provide the file path as an argument.
For URL attachments, use --url and --name flags.

See also: attachment list, task view`,
	Example: `  # Upload a file
  asana attachment upload /path/to/file.pdf --task 1234567890123456

  # Upload with custom display name
  asana attachment upload /path/to/file.pdf --task 1234567890123456 --name "Report Q4"

  # Attach an external URL
  asana attachment upload --task 1234567890123456 --url "https://example.com/doc" --name "Design Doc"

  # Preview without uploading
  asana attachment upload /path/to/file.pdf --task 1234567890123456 --dry-run`,
	Args: cobra.MaximumNArgs(1),
	RunE: runAttachmentUpload,
}

var attachmentDeleteCmd = &cobra.Command{
	Use:   "delete <attachment-gid>",
	Short: "Delete an attachment",
	Long: `Permanently delete an attachment.

This action cannot be undone.

See also: attachment list, attachment view`,
	Example: `  # Delete an attachment
  asana attachment delete 1234567890123456 --confirm

  # Preview deletion
  asana attachment delete 1234567890123456 --dry-run`,
	Args: cobra.ExactArgs(1),
	RunE: runAttachmentDelete,
}

func init() {
	attachmentListCmd.Flags().String("task", "", "Task GID")
	attachmentListCmd.Flags().String("task-gid", "", "Task GID")
	attachmentListCmd.Flags().Int("limit", 0, "Limit number of attachments in output")

	attachmentUploadCmd.Flags().String("task", "", "Task GID")
	attachmentUploadCmd.Flags().String("task-gid", "", "Task GID")
	attachmentUploadCmd.Flags().String("name", "", "Display name (required for URL, optional for file)")
	attachmentUploadCmd.Flags().String("url", "", "External URL to attach")
	attachmentUploadCmd.Flags().Bool("connect-to-app", false, "Connect URL attachment to app")

	attachmentCmd.AddCommand(attachmentListCmd)
	attachmentCmd.AddCommand(attachmentViewCmd)
	attachmentCmd.AddCommand(attachmentUploadCmd)
	attachmentCmd.AddCommand(attachmentDeleteCmd)
}

type attachmentListItem struct {
	GID             string `json:"gid"`
	Name            string `json:"name"`
	ResourceSubtype string `json:"resource_subtype,omitempty"`
	Host            string `json:"host,omitempty"`
	Size            int64  `json:"size,omitempty"`
}

type attachmentViewOutput struct {
	GID             string  `json:"gid"`
	Name            string  `json:"name"`
	ResourceSubtype string  `json:"resource_subtype,omitempty"`
	Host            string  `json:"host,omitempty"`
	Size            int64   `json:"size,omitempty"`
	DownloadURL     *string `json:"download_url"`
	PermanentURL    *string `json:"permanent_url"`
	ViewURL         *string `json:"view_url"`
	CreatedAt       string  `json:"created_at,omitempty"`
}

type attachmentWriteOutput struct {
	Action     string  `json:"action"`
	DryRun     bool    `json:"dry_run"`
	Attachment *gidRef `json:"attachment,omitempty"`
	Task       *gidRef `json:"task,omitempty"`
	Name       string  `json:"name,omitempty"`
	URL        string  `json:"url,omitempty"`
	File       string  `json:"file,omitempty"`
}

func resolveAttachmentTaskGID(cmd *cobra.Command) (string, error) {
	task, _ := cmd.Flags().GetString("task")
	taskGID, _ := cmd.Flags().GetString("task-gid")
	if task != "" && taskGID != "" {
		return "", fmt.Errorf("use only one of --task or --task-gid")
	}
	resolved := taskGID
	if resolved == "" {
		resolved = task
	}
	if resolved == "" {
		return "", fmt.Errorf("either --task or --task-gid is required")
	}
	if err := validateGID(resolved, "task gid"); err != nil {
		return "", err
	}
	return resolved, nil
}

func runAttachmentList(cmd *cobra.Command, args []string) error {
	limit, _ := cmd.Flags().GetInt("limit")

	taskGID, err := resolveAttachmentTaskGID(cmd)
	if err != nil {
		return err
	}
	if cmd.Flags().Changed("limit") && limit < 1 {
		return fmt.Errorf("--limit must be >= 1")
	}

	query := url.Values{}
	query.Set("parent", taskGID)
	query.Set("opt_fields", "gid,name,resource_subtype,host,size")

	attachments, err := api.Paginate[api.Attachment](cmd.Context(), runtimeClient(cmd), "/attachments", query)
	if err != nil {
		return err
	}

	// Convert to output format (preserve API order - creation order is meaningful)
	output := make([]attachmentListItem, len(attachments))
	for i, a := range attachments {
		output[i] = attachmentListItem{
			GID:             a.GID,
			Name:            a.Name,
			ResourceSubtype: a.ResourceSubtype,
			Host:            a.Host,
			Size:            a.Size,
		}
	}

	if limit > 0 && len(output) > limit {
		output = output[:limit]
	}

	if runtimeOutputJSON(cmd) {
		encoder := json.NewEncoder(cmd.OutOrStdout())
		encoder.SetIndent("", "  ")
		return encoder.Encode(output)
	}

	if len(output) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No attachments found")
		return nil
	}

	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tHOST\tSIZE\tGID")
	for _, a := range output {
		sizeStr := ""
		if a.Size > 0 {
			sizeStr = fmt.Sprintf("%d", a.Size)
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", a.Name, a.Host, sizeStr, a.GID)
	}
	return w.Flush()
}

func runAttachmentView(cmd *cobra.Command, args []string) error {
	attachmentGID := args[0]

	if err := validateGID(attachmentGID, "attachment gid"); err != nil {
		return err
	}

	query := url.Values{}
	query.Set("opt_fields", "gid,name,resource_subtype,host,size,download_url,permanent_url,view_url,created_at")

	payload, err := runtimeClient(cmd).Get(cmd.Context(), "/attachments/"+attachmentGID, query)
	if err != nil {
		return err
	}

	var response api.Response[api.Attachment]
	if err := json.Unmarshal(payload, &response); err != nil {
		return &api.ResponseError{Err: err}
	}

	a := response.Data
	output := attachmentViewOutput{
		GID:             a.GID,
		Name:            a.Name,
		ResourceSubtype: a.ResourceSubtype,
		Host:            a.Host,
		Size:            a.Size,
		CreatedAt:       a.CreatedAt,
	}
	// Set nullable fields
	if a.DownloadURL != "" {
		output.DownloadURL = &a.DownloadURL
	}
	if a.PermanentURL != "" {
		output.PermanentURL = &a.PermanentURL
	}
	if a.ViewURL != "" {
		output.ViewURL = &a.ViewURL
	}

	if runtimeOutputJSON(cmd) {
		encoder := json.NewEncoder(cmd.OutOrStdout())
		encoder.SetIndent("", "  ")
		return encoder.Encode(output)
	}

	// Text output
	fmt.Fprintf(cmd.OutOrStdout(), "GID:       %s\n", a.GID)
	fmt.Fprintf(cmd.OutOrStdout(), "Name:      %s\n", a.Name)
	if a.Host != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "Host:      %s\n", a.Host)
	}
	if a.Size > 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "Size:      %d\n", a.Size)
	}
	if a.DownloadURL != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "Download:  %s\n", a.DownloadURL)
	}
	if a.PermanentURL != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "Permalink: %s\n", a.PermanentURL)
	}
	if a.ViewURL != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "View URL:  %s\n", a.ViewURL)
	}
	if a.CreatedAt != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "Created:   %s\n", a.CreatedAt)
	}

	return nil
}

func runAttachmentUpload(cmd *cobra.Command, args []string) error {
	name, _ := cmd.Flags().GetString("name")
	urlArg, _ := cmd.Flags().GetString("url")
	connectToApp, _ := cmd.Flags().GetBool("connect-to-app")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	taskGID, err := resolveAttachmentTaskGID(cmd)
	if err != nil {
		return err
	}

	var filePath string
	if len(args) > 0 {
		filePath = args[0]
	}

	// Validate mutual exclusivity
	if filePath != "" && urlArg != "" {
		return fmt.Errorf("cannot specify both file and --url")
	}
	if filePath == "" && urlArg == "" {
		return fmt.Errorf("either file path or --url is required")
	}

	// URL attachment requires name
	if urlArg != "" && name == "" {
		return fmt.Errorf("--name is required for URL attachments")
	}

	// connect-to-app only valid for URL attachments
	if connectToApp && urlArg == "" {
		return fmt.Errorf("--connect-to-app is only valid for URL attachments")
	}

	if dryRun {
		// Validate task exists
		if _, err := fetchTaskWithFields(cmd.Context(), taskGID, "gid"); err != nil {
			return err
		}

		output := attachmentWriteOutput{
			Action: "uploaded",
			DryRun: true,
			Task:   &gidRef{GID: taskGID},
			Name:   name,
		}
		if urlArg != "" {
			output.URL = urlArg
		} else {
			output.File = filePath
			if name == "" {
				output.Name = filepath.Base(filePath)
			}
		}

		if runtimeOutputJSON(cmd) {
			encoder := json.NewEncoder(cmd.OutOrStdout())
			encoder.SetIndent("", "  ")
			return encoder.Encode(output)
		}

		if urlArg != "" {
			fmt.Fprintf(cmd.OutOrStdout(), "[dry-run] Would attach URL %q to task %s\n", urlArg, taskGID)
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "[dry-run] Would upload file %q to task %s\n", filePath, taskGID)
		}
		return nil
	}

	fields := map[string]string{
		"parent": taskGID,
	}

	var file *api.FileUpload

	if urlArg != "" {
		// URL attachment
		fields["resource_subtype"] = "external"
		fields["url"] = urlArg
		fields["name"] = name
		if connectToApp {
			fields["connect_to_app"] = "true"
		}
	} else {
		// File upload
		fileName := filepath.Base(filePath)
		if name != "" {
			fields["name"] = name
		}

		var content io.Reader
		if !runtimeClient(cmd).MutationPreviewEnabled() {
			fileData, err := os.ReadFile(filePath)
			if err != nil {
				return fmt.Errorf("failed to read file: %w", err)
			}
			content = bytes.NewReader(fileData)
		}

		file = &api.FileUpload{
			FieldName: "file",
			FileName:  fileName,
			Content:   content,
		}
	}

	payload, err := runtimeClient(cmd).PostMultipart(cmd.Context(), "/attachments", fields, file)
	if err != nil {
		return err
	}

	var response api.Response[api.Attachment]
	if err := json.Unmarshal(payload, &response); err != nil {
		return &api.ResponseError{Err: err}
	}

	a := response.Data
	output := attachmentWriteOutput{
		Action:     "uploaded",
		DryRun:     false,
		Attachment: &gidRef{GID: a.GID},
		Task:       &gidRef{GID: taskGID},
		Name:       a.Name,
	}
	if urlArg != "" {
		output.URL = urlArg
	} else {
		output.File = filePath
	}

	if runtimeOutputJSON(cmd) {
		encoder := json.NewEncoder(cmd.OutOrStdout())
		encoder.SetIndent("", "  ")
		return encoder.Encode(output)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Uploaded attachment: %s (%s)\n", a.Name, a.GID)
	return nil
}

func runAttachmentDelete(cmd *cobra.Command, args []string) error {
	attachmentGID := args[0]
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	if err := validateGID(attachmentGID, "attachment gid"); err != nil {
		return err
	}

	if dryRun {
		// Validate attachment exists
		query := url.Values{}
		query.Set("opt_fields", "gid,name")
		payload, err := runtimeClient(cmd).Get(cmd.Context(), "/attachments/"+attachmentGID, query)
		if err != nil {
			return err
		}

		var response api.Response[api.Attachment]
		if err := json.Unmarshal(payload, &response); err != nil {
			return &api.ResponseError{Err: err}
		}

		output := attachmentWriteOutput{
			Action:     "deleted",
			DryRun:     true,
			Attachment: &gidRef{GID: attachmentGID},
			Name:       response.Data.Name,
		}
		if runtimeOutputJSON(cmd) {
			encoder := json.NewEncoder(cmd.OutOrStdout())
			encoder.SetIndent("", "  ")
			return encoder.Encode(output)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "[dry-run] Would delete attachment %s (%s)\n", response.Data.Name, attachmentGID)
		return nil
	}

	if err := runtimeClient(cmd).Delete(cmd.Context(), "/attachments/"+attachmentGID); err != nil {
		return err
	}

	output := attachmentWriteOutput{
		Action:     "deleted",
		DryRun:     false,
		Attachment: &gidRef{GID: attachmentGID},
	}

	if runtimeOutputJSON(cmd) {
		encoder := json.NewEncoder(cmd.OutOrStdout())
		encoder.SetIndent("", "  ")
		return encoder.Encode(output)
	}

	fmt.Fprintln(cmd.OutOrStdout(), "Attachment deleted")
	return nil
}
