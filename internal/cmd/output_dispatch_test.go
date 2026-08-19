// Package cmd tests shared plain-output and preview dispatch behavior.
package cmd

import (
	"bytes"
	"testing"
)

func TestPlainListLayoutsMatchHumanColumns(t *testing.T) {
	tests := []struct {
		name        string
		commandPath string
		model       []any
		want        string
	}{
		{
			name:        "task dependencies",
			commandPath: "task dependency list",
			model:       []any{map[string]any{"gid": "456", "name": "Dependency"}},
			want:        "456\tDependency\n",
		},
		{
			name:        "tasks",
			commandPath: "task list",
			model:       []any{map[string]any{"gid": "123", "name": "Task", "completed": false, "due_on": "2026-08-09"}},
			want:        "123\tTask\tfalse\n",
		},
		{
			name:        "project members",
			commandPath: "project member list",
			model:       []any{map[string]any{"gid": "7", "name": "Ada", "email": "ada@example.com", "access_level": "editor"}},
			want:        "Ada\tada@example.com\teditor\t7\n",
		},
		{
			name:        "team members",
			commandPath: "team member list",
			model:       []any{map[string]any{"gid": "8", "name": "Grace", "email": "grace@example.com", "is_admin": true}},
			want:        "Grace\tgrace@example.com\ttrue\t8\n",
		},
		{
			name:        "attachments",
			commandPath: "attachment list",
			model:       []any{map[string]any{"gid": "9", "name": "Spec", "host": "asana", "size": 42.0}},
			want:        "Spec\tasana\t42\t9\n",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var stdout bytes.Buffer
			renderPlainResult(&stdout, test.commandPath, test.model)
			if stdout.String() != test.want {
				t.Fatalf("plain output = %q, want %q", stdout.String(), test.want)
			}
		})
	}
}

func TestPlainOutputEscapesRecordSeparators(t *testing.T) {
	model := []any{map[string]any{
		"gid":  "1",
		"name": "Alpha\nInjected\tValue\\Path\rTail",
	}}

	var stdout bytes.Buffer
	renderPlainResult(&stdout, "workspace list", model)
	if got, want := stdout.String(), "Alpha\\nInjected\\tValue\\\\Path\\rTail\t1\n"; got != want {
		t.Fatalf("plain output = %q, want %q", got, want)
	}
}
