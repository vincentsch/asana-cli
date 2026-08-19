// Package cmd adapts Asana result models to rungrad's output boundary.
package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/vincentsch/asana-cli/internal/api"
	"github.com/vincentsch/rungrad"
	"github.com/vincentsch/rungrad/output"
)

var structuredOutputExceptions = map[string]bool{
	"completion": true,
}

var textResultCommands = map[string]bool{
	"auth login": true,
	"login":      true,
	"version":    true,
}

// commandSupportsStructuredOutput keeps stream-oriented and interactive
// commands off machine transforms until they have a real result model.
func commandSupportsStructuredOutput(cmd *cobra.Command) bool {
	if cmd.Run == nil && cmd.RunE == nil {
		return false
	}
	return !structuredOutputExceptions[commandPath(cmd)]
}

func commandSupportsDryRun(cmd *cobra.Command) bool {
	return mutatingCommandPaths[commandPath(cmd)]
}

func commandPath(cmd *cobra.Command) string {
	parts := make([]string, 0, 4)
	for current := cmd; current != nil && current.Name() != "asana"; current = current.Parent() {
		parts = append(parts, current.Name())
	}
	for left, right := 0, len(parts)-1; left < right; left, right = left+1, right-1 {
		parts[left], parts[right] = parts[right], parts[left]
	}
	return strings.Join(parts, " ")
}

func runCommandWithOutput(factory *rungrad.Factory, cmd *cobra.Command, args []string, run func(*cobra.Command, []string) error) error {
	var captured bytes.Buffer
	cmd.SetOut(&captured)
	defer cmd.SetOut(nil)

	dryRun := factory.DryRun() && commandSupportsDryRun(cmd)
	if !dryRun && destructiveCommandPaths[commandPath(cmd)] {
		confirmed, _ := cmd.Flags().GetBool("confirm")
		if err := factory.ConfirmDestructive(rungrad.ConfirmOptions{
			Action:    commandPath(cmd),
			Target:    destructiveTarget(cmd, args),
			Confirmed: confirmed,
		}); err != nil {
			return err
		}
	}
	err := runWithMutationPreview(factory, dryRun, cmd, args, run)
	if dryRun {
		var preview *api.MutationPreview
		if errors.As(err, &preview) {
			return factory.WritePreview(output.DryRunPreview{
				Method: preview.Method,
				Path:   preview.Path,
				Body:   previewFields(preview.Body),
			})
		}
		if err == nil {
			return fmt.Errorf("dry-run command completed without constructing a mutation request")
		}
	}
	if err != nil {
		return err
	}

	if !commandSupportsStructuredOutput(cmd) {
		return factory.WriteResult(nil, func(w io.Writer) {
			_, _ = w.Write(captured.Bytes())
		})
	}
	if !structuredModeActive(factory, cmd) {
		return factory.WriteResult(nil, func(w io.Writer) {
			_, _ = w.Write(captured.Bytes())
		})
	}

	model, err := decodeLegacyResult(captured.Bytes())
	if err != nil && textResultCommands[commandPath(cmd)] {
		model = map[string]string{"output": strings.TrimSuffix(captured.String(), "\n")}
		err = nil
	}
	if err != nil {
		return fmt.Errorf("encode command result: %w", err)
	}
	return factory.WriteOutput(rungrad.Output{
		Model: model,
		Meta:  outputMeta(runtimeClient(cmd)),
		Human: func(w io.Writer) {
			_, _ = w.Write(captured.Bytes())
		},
		Plain: func(w io.Writer) {
			renderPlainResult(w, commandPath(cmd), model)
		},
	})
}

func destructiveTarget(cmd *cobra.Command, args []string) string {
	parts := append([]string(nil), args...)
	cmd.LocalNonPersistentFlags().Visit(func(flag *pflag.Flag) {
		if flag.Name != "confirm" {
			parts = append(parts, fmt.Sprintf("--%s=%s", flag.Name, flag.Value.String()))
		}
	})
	if len(parts) == 0 {
		return "requested target"
	}
	return strings.Join(parts, " ")
}

func runWithMutationPreview(factory *rungrad.Factory, dryRun bool, cmd *cobra.Command, args []string, run func(*cobra.Command, []string) error) error {
	if !dryRun {
		return run(cmd, args)
	}
	if localMutationCommand(cmd) {
		// Local configuration and executable changes construct their own previews;
		// unlike API writes, there is no HTTP request for the client to intercept.
		return run(cmd, args)
	}
	client := runtimeClient(cmd)
	if client == nil {
		return fmt.Errorf("dry-run mutation preview requires an API client")
	}
	client.SetMutationPreview(true)
	defer client.SetMutationPreview(false)

	// Legacy dry-run branches perform resource reads that the real mutation path
	// may not need. Run those validations while mutation interception is already
	// active, then run the handler again to capture its constructed request.
	if err := run(cmd, args); err != nil {
		return err
	}
	if factory.Flags == nil {
		return fmt.Errorf("dry-run mutation preview requires global flags")
	}
	originalDryRun := factory.Flags.DryRun
	factory.Flags.DryRun = false
	defer func() { factory.Flags.DryRun = originalDryRun }()
	return run(cmd, args)
}

func localMutationCommand(cmd *cobra.Command) bool {
	switch commandPath(cmd) {
	case "auth login", "config set", "login":
		return true
	default:
		return false
	}
}

func previewFields(body map[string]any) []output.Field {
	keys := make([]string, 0, len(body))
	for key := range body {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	fields := make([]output.Field, 0, len(keys))
	for _, key := range keys {
		fields = append(fields, output.Field{Name: key, Value: previewValue(body[key])})
	}
	return fields
}

func previewValue(value any) string {
	switch value := value.(type) {
	case nil:
		return "null"
	case string:
		return value
	case bool, json.Number, float64, float32, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return fmt.Sprint(value)
	default:
		encoded, err := json.Marshal(value)
		if err != nil {
			return fmt.Sprint(value)
		}
		return string(encoded)
	}
}

func structuredModeActive(factory *rungrad.Factory, cmd *cobra.Command) bool {
	return factory.Flags.JSON || (factory.Flags.DryRun && commandSupportsDryRun(cmd)) || factory.Flags.Plain || factory.Flags.JQ != "" || factory.Flags.Template != ""
}

func decodeLegacyResult(data []byte) (any, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var model any
	if err := decoder.Decode(&model); err != nil {
		return nil, err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return nil, fmt.Errorf("multiple JSON values")
		}
		return nil, err
	}
	return model, nil
}

func outputMeta(client *api.Client) output.Meta {
	if client == nil {
		return output.Meta{}
	}
	meta := client.RequestMetadata()
	out := output.Meta{
		RequestID:  meta.RequestID,
		RequestIDs: meta.RequestIDs,
		Retry:      &output.Retry{Attempts: meta.Attempts, WaitsMS: meta.WaitsMS},
	}
	if meta.Endpoint != "" {
		out.Extra = map[string]any{"endpoint": meta.Endpoint}
	}
	if meta.Paginated {
		hasMore := meta.NextCursor != ""
		out.Pagination = &output.Pagination{NextCursor: meta.NextCursor, HasMore: &hasMore}
	}
	if len(meta.RateLimit) > 0 {
		out.RateLimit = &output.RateLimit{Raw: meta.RateLimit}
	}
	if out.Retry.Attempts == 0 {
		out.Retry = nil
	}
	return out
}

func renderPlainResult(w io.Writer, commandPath string, model any) {
	switch value := model.(type) {
	case []any:
		columns := plainColumns(commandPath, value)
		for _, item := range value {
			row, ok := item.(map[string]any)
			if !ok {
				fmt.Fprintln(w, plainValue(item))
				continue
			}
			writePlainValues(w, row, columns)
		}
	case map[string]any:
		if _, ok := value["action"]; ok {
			renderPlainMutation(w, value)
			return
		}
		renderPlainDetail(w, value)
	default:
		fmt.Fprintln(w, plainValue(value))
	}
}

func plainColumns(commandPath string, rows []any) []string {
	switch commandPath {
	case "attachment list":
		return []string{"name", "host", "size", "gid"}
	case "custom-field list":
		return []string{"name", "resource_subtype", "gid"}
	case "goal list":
		return []string{"name", "status", "owner", "gid"}
	case "portfolio list":
		return []string{"name", "color", "gid"}
	case "portfolio project list", "project list", "section list", "workspace list":
		return []string{"name", "gid"}
	case "project member list":
		return []string{"name", "email", "access_level", "gid"}
	case "tag list":
		return []string{"name", "color", "gid"}
	case "tag tasks", "task list", "task search", "task subtask list":
		return []string{"gid", "name", "completed"}
	case "task dependency list", "task dependent list":
		return []string{"gid", "name"}
	case "team list":
		return []string{"name", "visibility", "gid"}
	case "team member list":
		return []string{"name", "email", "is_admin", "gid"}
	case "user list", "workspace user list":
		return []string{"name", "email", "gid"}
	default:
		for _, item := range rows {
			if row, ok := item.(map[string]any); ok {
				return scalarKeys(row)
			}
		}
		return nil
	}
}

func renderPlainMutation(w io.Writer, model map[string]any) {
	action := plainValue(model["action"])
	dryRun := plainValue(model["dry_run"])
	gid, name := nestedIdentity(model)
	values := []string{action, gid, name}
	if _, ok := model["dry_run"]; ok {
		values = append(values, dryRun)
	}
	fmt.Fprintln(w, strings.Join(values, "\t"))
}

func nestedIdentity(model map[string]any) (string, string) {
	if gid := plainValue(model["gid"]); gid != "" {
		return gid, plainValue(model["name"])
	}
	keys := make([]string, 0, len(model))
	for key := range model {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if child, ok := model[key].(map[string]any); ok {
			if gid := plainValue(child["gid"]); gid != "" {
				return gid, plainValue(child["name"])
			}
		}
	}
	return "", plainValue(model["name"])
}

func renderPlainDetail(w io.Writer, model map[string]any) {
	keys := detailScalarKeys(model)
	for _, key := range keys {
		fmt.Fprintf(w, "%s\t%s\n", key, plainValue(model[key]))
	}
	complexKeys := make([]string, 0)
	for key, value := range model {
		switch value.(type) {
		case []any, map[string]any:
			complexKeys = append(complexKeys, key)
		}
	}
	sort.Strings(complexKeys)
	for _, key := range complexKeys {
		switch value := model[key].(type) {
		case []any:
			for _, child := range value {
				if row, ok := child.(map[string]any); ok {
					fmt.Fprintf(w, "%s\t", key)
					writePlainValues(w, row, flattenedScalarKeys(row))
				} else {
					fmt.Fprintf(w, "%s\t%s\n", key, plainValue(child))
				}
			}
		case map[string]any:
			for _, childKey := range scalarKeys(value) {
				fmt.Fprintf(w, "%s.%s\t%s\n", key, childKey, plainValue(value[childKey]))
			}
		}
	}
}

func flattenedScalarKeys(model map[string]any) []string {
	keys := make([]string, 0)
	var collect func(string, map[string]any)
	collect = func(prefix string, object map[string]any) {
		for key, value := range object {
			path := key
			if prefix != "" {
				path = prefix + "." + key
			}
			switch child := value.(type) {
			case map[string]any:
				collect(path, child)
			case nil, string, bool, json.Number, float64:
				keys = append(keys, path)
			}
		}
	}
	collect("", model)
	sort.Strings(keys)
	return keys
}

func detailScalarKeys(model map[string]any) []string {
	preferred := []string{"name", "gid", "completed", "email", "due_on", "due_at", "created_at", "modified_at"}
	keys := scalarKeys(model)
	seen := make(map[string]bool, len(keys))
	ordered := make([]string, 0, len(keys))
	for _, key := range preferred {
		for _, candidate := range keys {
			if candidate == key {
				ordered = append(ordered, key)
				seen[key] = true
				break
			}
		}
	}
	for _, key := range keys {
		if !seen[key] {
			ordered = append(ordered, key)
		}
	}
	return ordered
}

func scalarKeys(model map[string]any) []string {
	keys := make([]string, 0, len(model))
	for key, value := range model {
		switch value.(type) {
		case nil, string, bool, json.Number, float64:
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	return keys
}

func writePlainValues(w io.Writer, row map[string]any, columns []string) {
	values := make([]string, 0, len(columns))
	for _, column := range columns {
		value, ok := nestedValue(row, column)
		if !ok {
			values = append(values, "")
			continue
		}
		values = append(values, plainValue(value))
	}
	fmt.Fprintln(w, strings.Join(values, "\t"))
}

func nestedValue(model map[string]any, path string) (any, bool) {
	var current any = model
	for _, part := range strings.Split(path, ".") {
		object, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		current, ok = object[part]
		if !ok {
			return nil, false
		}
	}
	return current, true
}

func plainValue(value any) string {
	switch value := value.(type) {
	case nil:
		return ""
	case string:
		return escapePlainField(value)
	case bool:
		return fmt.Sprintf("%t", value)
	case json.Number:
		return value.String()
	case float64:
		return fmt.Sprintf("%v", value)
	default:
		encoded, _ := json.Marshal(value)
		return string(encoded)
	}
}

func escapePlainField(value string) string {
	replacer := strings.NewReplacer(
		"\\", "\\\\",
		"\t", "\\t",
		"\r", "\\r",
		"\n", "\\n",
	)
	return replacer.Replace(value)
}
