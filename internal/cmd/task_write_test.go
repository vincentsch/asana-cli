// Package cmd tests task write operations.
package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/vincentsch/asana-cli/internal/resolve"
	"github.com/vincentsch/rungrad/testutil"
)

func TestTaskMoveWithSectionName(t *testing.T) {
	var addTaskCalled bool
	var taskCalls int

	srv := testutil.MockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/tasks/123" && r.Method == http.MethodGet:
			taskCalls++
			if taskCalls == 1 {
				_, _ = w.Write([]byte(`{"data":{"gid":"123","name":"Test","memberships":[{"project":{"gid":"1001","name":"Alpha"},"section":{"gid":"2001","name":"Backlog"}}]}}`))
				return
			}
			_, _ = w.Write([]byte(`{"data":{"gid":"123","name":"Test","memberships":[{"project":{"gid":"1001","name":"Alpha"},"section":{"gid":"2002","name":"Done"}}]}}`))
		case r.URL.Path == "/projects/1001/sections" && r.Method == http.MethodGet:
			_, _ = w.Write([]byte(`{"data":[{"gid":"2002","name":"Done"}]}`))
		case r.URL.Path == "/sections/2002" && r.Method == http.MethodGet:
			_, _ = w.Write([]byte(`{"data":{"gid":"2002","name":"Done","project":{"gid":"1001","name":"Alpha"}}}`))
		case r.URL.Path == "/sections/2002/addTask" && r.Method == http.MethodPost:
			addTaskCalled = true
			body, _ := io.ReadAll(r.Body)
			if !strings.Contains(string(body), `"task":"123"`) {
				t.Fatalf("unexpected request body: %s", body)
			}
			_, _ = w.Write([]byte(`{"data":{}}`))
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))

	apiClient = newTestClient(srv)
	defer func() { apiClient = nil }()

	cmd, buf := newTaskMoveTestCmd()
	setFlag(t, cmd, "section", "Done")

	if err := runTaskMove(cmd, []string{"123"}); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !addTaskCalled {
		t.Fatalf("expected addTask call")
	}
	if strings.TrimSpace(buf.String()) != `Moved task to section "Done"` {
		t.Fatalf("unexpected output: %s", buf.String())
	}
}

func TestTaskMoveWithSectionGID(t *testing.T) {
	var addTaskCalled bool
	var taskCalls int

	srv := testutil.MockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/tasks/123" && r.Method == http.MethodGet:
			taskCalls++
			if taskCalls == 1 {
				_, _ = w.Write([]byte(`{"data":{"gid":"123","name":"Test","memberships":[{"project":{"gid":"1001","name":"Alpha"},"section":{"gid":"2001","name":"Backlog"}}]}}`))
				return
			}
			_, _ = w.Write([]byte(`{"data":{"gid":"123","name":"Test","memberships":[{"project":{"gid":"1001","name":"Alpha"},"section":{"gid":"2002","name":"Done"}}]}}`))
		case r.URL.Path == "/projects/1001/sections" && r.Method == http.MethodGet:
			t.Fatalf("did not expect section list request")
		case r.URL.Path == "/sections/2002" && r.Method == http.MethodGet:
			_, _ = w.Write([]byte(`{"data":{"gid":"2002","name":"Done","project":{"gid":"1001","name":"Alpha"}}}`))
		case r.URL.Path == "/sections/2002/addTask" && r.Method == http.MethodPost:
			addTaskCalled = true
			_, _ = w.Write([]byte(`{"data":{}}`))
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))

	apiClient = newTestClient(srv)
	defer func() { apiClient = nil }()

	cmd, buf := newTaskMoveTestCmd()
	setFlag(t, cmd, "section-gid", "2002")

	if err := runTaskMove(cmd, []string{"123"}); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !addTaskCalled {
		t.Fatalf("expected addTask call")
	}
	if strings.TrimSpace(buf.String()) != `Moved task to section "Done"` {
		t.Fatalf("unexpected output: %s", buf.String())
	}
}

func TestTaskMoveRequiresProjectDisambiguation(t *testing.T) {
	srv := testutil.MockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/tasks/123" || r.Method != http.MethodGet {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"data":{"gid":"123","name":"Test","memberships":[{"project":{"gid":"1002","name":"Project Beta"},"section":{"gid":"2001","name":"Done"}},{"project":{"gid":"1001","name":"Project Alpha"},"section":{"gid":"2002","name":"Todo"}}]}}`))
	}))

	apiClient = newTestClient(srv)
	defer func() { apiClient = nil }()

	cmd, _ := newTaskMoveTestCmd()
	setFlag(t, cmd, "section-gid", "2002")

	err := runTaskMove(cmd, []string{"123"})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	expected := "Task belongs to multiple projects. Use --project or --project-gid. Projects:\n  1001  Project Alpha\n  1002  Project Beta"
	if err.Error() != expected {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTaskMoveWithProjectDisambiguation(t *testing.T) {
	var taskCalls int

	srv := testutil.MockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/tasks/123" && r.Method == http.MethodGet:
			taskCalls++
			if taskCalls == 1 {
				_, _ = w.Write([]byte(`{"data":{"gid":"123","name":"Test","memberships":[{"project":{"gid":"1001","name":"Project Alpha"},"section":{"gid":"2001","name":"Todo"}},{"project":{"gid":"1002","name":"Project Beta"},"section":{"gid":"2009","name":"Other"}}]}}`))
				return
			}
			_, _ = w.Write([]byte(`{"data":{"gid":"123","name":"Test","memberships":[{"project":{"gid":"1001","name":"Project Alpha"},"section":{"gid":"2002","name":"Done"}}]}}`))
		case r.URL.Path == "/workspaces" && r.Method == http.MethodGet:
			_, _ = w.Write([]byte(`{"data":[{"gid":"3001","name":"Workspace"}]}`))
		case r.URL.Path == "/projects" && r.Method == http.MethodGet:
			if r.URL.Query().Get("workspace") != "3001" {
				t.Fatalf("unexpected workspace query: %s", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`{"data":[{"gid":"1001","name":"Project Alpha"}]}`))
		case r.URL.Path == "/sections/2002" && r.Method == http.MethodGet:
			_, _ = w.Write([]byte(`{"data":{"gid":"2002","name":"Done","project":{"gid":"1001","name":"Project Alpha"}}}`))
		case r.URL.Path == "/sections/2002/addTask" && r.Method == http.MethodPost:
			_, _ = w.Write([]byte(`{"data":{}}`))
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))

	apiClient = newTestClient(srv)
	defer func() { apiClient = nil }()

	cmd, _ := newTaskMoveTestCmd()
	setFlag(t, cmd, "section-gid", "2002")
	setFlag(t, cmd, "project", "Project Alpha")

	if err := runTaskMove(cmd, []string{"123"}); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestTaskMoveSectionNotInProject(t *testing.T) {
	srv := testutil.MockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/tasks/123" && r.Method == http.MethodGet:
			_, _ = w.Write([]byte(`{"data":{"gid":"123","name":"Test","memberships":[{"project":{"gid":"1001","name":"Alpha"},"section":{"gid":"2001","name":"Todo"}}]}}`))
		case r.URL.Path == "/sections/2002" && r.Method == http.MethodGet:
			_, _ = w.Write([]byte(`{"data":{"gid":"2002","name":"Done","project":{"gid":"1002","name":"Other"}}}`))
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))

	apiClient = newTestClient(srv)
	defer func() { apiClient = nil }()

	cmd, _ := newTaskMoveTestCmd()
	setFlag(t, cmd, "section-gid", "2002")

	err := runTaskMove(cmd, []string{"123"})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	expected := "Section \"Done\" (gid: 2002) does not belong to project \"Alpha\" (gid: 1001)"
	if err.Error() != expected {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTaskMoveInvalidGIDs(t *testing.T) {
	cases := []struct {
		name    string
		args    []string
		flags   map[string]string
		wantErr string
	}{
		{
			name:    "task",
			args:    []string{"abc"},
			flags:   map[string]string{"section-gid": "123"},
			wantErr: "Invalid gid \"abc\": must be numeric",
		},
		{
			name:    "section",
			args:    []string{"123"},
			flags:   map[string]string{"section-gid": "abc"},
			wantErr: "Invalid section gid \"abc\": must be numeric",
		},
		{
			name:    "project",
			args:    []string{"123"},
			flags:   map[string]string{"section-gid": "456", "project-gid": "abc"},
			wantErr: "Invalid project gid \"abc\": must be numeric",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cmd, _ := newTaskMoveTestCmd()
			for key, value := range tc.flags {
				setFlag(t, cmd, key, value)
			}

			err := runTaskMove(cmd, tc.args)
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if err.Error() != tc.wantErr {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestTaskMoveAmbiguousSectionName(t *testing.T) {
	srv := testutil.MockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/tasks/123" && r.Method == http.MethodGet:
			_, _ = w.Write([]byte(`{"data":{"gid":"123","name":"Test","memberships":[{"project":{"gid":"1001","name":"Alpha"},"section":{"gid":"2001","name":"Todo"}}]}}`))
		case r.URL.Path == "/projects/1001/sections" && r.Method == http.MethodGet:
			_, _ = w.Write([]byte(`{"data":[{"gid":"2001","name":"Done"},{"gid":"2002","name":"Done"}]}`))
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))

	apiClient = newTestClient(srv)
	defer func() { apiClient = nil }()

	cmd, _ := newTaskMoveTestCmd()
	setFlag(t, cmd, "section", "Done")

	err := runTaskMove(cmd, []string{"123"})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	var ambiguous *resolve.AmbiguousError
	if !errors.As(err, &ambiguous) {
		t.Fatalf("expected ambiguous error, got %T", err)
	}
}

func TestTaskMoveDryRun(t *testing.T) {
	srv := testutil.MockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/tasks/123" && r.Method == http.MethodGet:
			_, _ = w.Write([]byte(`{"data":{"gid":"123","name":"Test","memberships":[{"project":{"gid":"1001","name":"Alpha"},"section":{"gid":"2001","name":"Todo"}}]}}`))
		case r.URL.Path == "/sections/2002" && r.Method == http.MethodGet:
			_, _ = w.Write([]byte(`{"data":{"gid":"2002","name":"Done","project":{"gid":"1001","name":"Alpha"}}}`))
		case r.Method == http.MethodPost:
			t.Fatalf("unexpected POST in dry-run")
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))

	apiClient = newTestClient(srv)
	defer func() { apiClient = nil }()

	cmd, buf := newTaskMoveTestCmd()
	setFlag(t, cmd, "section-gid", "2002")
	setFlag(t, cmd, "dry-run", "true")

	if err := runTaskMove(cmd, []string{"123"}); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !strings.Contains(buf.String(), "[dry-run]") {
		t.Fatalf("expected dry-run output, got %s", buf.String())
	}
}

func TestTaskMoveJSONOutput(t *testing.T) {
	var taskCalls int

	srv := testutil.MockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/tasks/123" && r.Method == http.MethodGet:
			taskCalls++
			if taskCalls == 1 {
				_, _ = w.Write([]byte(`{"data":{"gid":"123","name":"Test","completed":false,"memberships":[{"project":{"gid":"1001","name":"Alpha"},"section":{"gid":"2001","name":"Todo"}}]}}`))
				return
			}
			_, _ = w.Write([]byte(`{"data":{"gid":"123","name":"Test","completed":false,"memberships":[{"project":{"gid":"1001","name":"Alpha"},"section":{"gid":"2002","name":"Done"}}]}}`))
		case r.URL.Path == "/sections/2002" && r.Method == http.MethodGet:
			_, _ = w.Write([]byte(`{"data":{"gid":"2002","name":"Done","project":{"gid":"1001","name":"Alpha"}}}`))
		case r.URL.Path == "/sections/2002/addTask" && r.Method == http.MethodPost:
			_, _ = w.Write([]byte(`{"data":{}}`))
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))

	apiClient = newTestClient(srv)
	defer func() { apiClient = nil }()

	outputJSON = true
	defer func() { outputJSON = false }()

	cmd, buf := newTaskMoveTestCmd()
	setFlag(t, cmd, "section-gid", "2002")

	if err := runTaskMove(cmd, []string{"123"}); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	output := decodeJSONOutput(t, buf)
	if output["action"] != "moved" {
		t.Fatalf("unexpected action: %#v", output["action"])
	}
	if output["dry_run"] != false {
		t.Fatalf("expected dry_run false")
	}
	task := asMap(t, output["task"])
	if _, ok := task["assignee"]; !ok {
		t.Fatalf("expected assignee key in output")
	}
	memberships := asSlice(t, task["memberships"])
	if len(memberships) != 1 {
		t.Fatalf("expected 1 membership, got %d", len(memberships))
	}
	membership := asMap(t, memberships[0])
	section := asMap(t, membership["section"])
	if _, ok := section["gid"]; !ok {
		t.Fatalf("expected section gid in output")
	}
	if _, ok := section["name"]; !ok {
		t.Fatalf("expected section name in output")
	}
}

func TestTaskCommentDirectMessage(t *testing.T) {
	srv := testutil.MockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/tasks/123/stories" || r.Method != http.MethodPost {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		data := decodeRequestData(t, r)
		if data["text"] != "Hello" {
			t.Fatalf("unexpected comment text: %#v", data["text"])
		}
		_, _ = w.Write([]byte(`{"data":{"gid":"5001","text":"Hello","created_at":"2025-01-10T14:30:00.000Z","created_by":{"gid":"6001","name":"User"}}}`))
	}))

	apiClient = newTestClient(srv)
	defer func() { apiClient = nil }()

	cmd, buf := newTaskCommentTestCmd()

	if err := runTaskComment(cmd, []string{"123", "Hello"}); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if strings.TrimSpace(buf.String()) != "Comment added" {
		t.Fatalf("unexpected output: %s", buf.String())
	}
}

func TestTaskCommentStdinMessage(t *testing.T) {
	srv := testutil.MockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/tasks/123/stories" || r.Method != http.MethodPost {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		data := decodeRequestData(t, r)
		if data["text"] != "Line 1\nLine 2" {
			t.Fatalf("unexpected comment text: %#v", data["text"])
		}
		_, _ = w.Write([]byte(`{"data":{"gid":"5001","text":"Line 1\nLine 2","created_at":"2025-01-10T14:30:00.000Z","created_by":{"gid":"6001","name":"User"}}}`))
	}))

	apiClient = newTestClient(srv)
	defer func() { apiClient = nil }()

	cmd, _ := newTaskCommentTestCmd()
	cmd.SetIn(bytes.NewBufferString("Line 1\nLine 2\n"))

	if err := runTaskComment(cmd, []string{"123", "-"}); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestTaskCommentEmpty(t *testing.T) {
	cmd, _ := newTaskCommentTestCmd()
	cmd.SetIn(bytes.NewBufferString(""))

	err := runTaskComment(cmd, []string{"123", "-"})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if err.Error() != "Comment cannot be empty" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTaskCommentInvalidGID(t *testing.T) {
	cmd, _ := newTaskCommentTestCmd()
	err := runTaskComment(cmd, []string{"abc", "Hello"})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if err.Error() != "Invalid gid \"abc\": must be numeric" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTaskCommentDryRun(t *testing.T) {
	srv := testutil.MockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/tasks/123" || r.Method != http.MethodGet {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"data":{"gid":"123"}}`))
	}))

	apiClient = newTestClient(srv)
	defer func() { apiClient = nil }()

	cmd, buf := newTaskCommentTestCmd()
	setFlag(t, cmd, "dry-run", "true")

	if err := runTaskComment(cmd, []string{"123", "Hello"}); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !strings.Contains(buf.String(), "[dry-run]") {
		t.Fatalf("expected dry-run output, got %s", buf.String())
	}
}

func TestTaskCommentJSONOutput(t *testing.T) {
	srv := testutil.MockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/tasks/123/stories" || r.Method != http.MethodPost {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"data":{"gid":"5001","text":"Hello","created_at":"2025-01-10T14:30:00.000Z","created_by":{"gid":"6001","name":"User"}}}`))
	}))

	apiClient = newTestClient(srv)
	defer func() { apiClient = nil }()

	outputJSON = true
	defer func() { outputJSON = false }()

	cmd, buf := newTaskCommentTestCmd()

	if err := runTaskComment(cmd, []string{"123", "Hello"}); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	output := decodeJSONOutput(t, buf)
	if output["action"] != "commented" {
		t.Fatalf("unexpected action: %#v", output["action"])
	}
	story := asMap(t, output["story"])
	createdBy := asMap(t, story["created_by"])
	if createdBy["gid"] != "6001" {
		t.Fatalf("unexpected created_by: %#v", createdBy)
	}
}

func TestTaskDone(t *testing.T) {
	srv := testutil.MockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/tasks/123" || r.Method != http.MethodPut {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		data := decodeRequestData(t, r)
		if data["completed"] != true {
			t.Fatalf("expected completed true, got %#v", data["completed"])
		}
		_, _ = w.Write([]byte(`{"data":{"gid":"123","name":"Test","completed":true,"memberships":[{"project":{"gid":"1001","name":"Alpha"},"section":{"gid":"2001","name":"Done"}}]}}`))
	}))

	apiClient = newTestClient(srv)
	defer func() { apiClient = nil }()

	cmd, buf := newTaskDoneTestCmd()
	if err := runTaskDone(cmd, []string{"123"}); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if strings.TrimSpace(buf.String()) != "Marked task as completed" {
		t.Fatalf("unexpected output: %s", buf.String())
	}
}

func TestTaskReopen(t *testing.T) {
	srv := testutil.MockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/tasks/123" || r.Method != http.MethodPut {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		data := decodeRequestData(t, r)
		if data["completed"] != false {
			t.Fatalf("expected completed false, got %#v", data["completed"])
		}
		_, _ = w.Write([]byte(`{"data":{"gid":"123","name":"Test","completed":false,"memberships":[{"project":{"gid":"1001","name":"Alpha"},"section":{"gid":"2001","name":"Todo"}}]}}`))
	}))

	apiClient = newTestClient(srv)
	defer func() { apiClient = nil }()

	cmd, buf := newTaskDoneTestCmd()
	if err := runTaskReopen(cmd, []string{"123"}); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if strings.TrimSpace(buf.String()) != "Marked task as incomplete" {
		t.Fatalf("unexpected output: %s", buf.String())
	}
}

func TestTaskCompletionInvalidGID(t *testing.T) {
	cmd, _ := newTaskDoneTestCmd()
	err := runTaskDone(cmd, []string{"abc"})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if err.Error() != "Invalid gid \"abc\": must be numeric" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTaskCompletionDryRun(t *testing.T) {
	srv := testutil.MockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/tasks/123" || r.Method != http.MethodGet {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"data":{"gid":"123","name":"Test","completed":false,"memberships":[{"project":{"gid":"1001","name":"Alpha"},"section":{"gid":"2001","name":"Todo"}}]}}`))
	}))

	apiClient = newTestClient(srv)
	defer func() { apiClient = nil }()

	cmd, buf := newTaskDoneTestCmd()
	setFlag(t, cmd, "dry-run", "true")

	if err := runTaskDone(cmd, []string{"123"}); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !strings.Contains(buf.String(), "[dry-run]") {
		t.Fatalf("expected dry-run output, got %s", buf.String())
	}
}

func TestTaskCompletionJSONOutput(t *testing.T) {
	srv := testutil.MockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/tasks/123" || r.Method != http.MethodPut {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"data":{"gid":"123","name":"Test","completed":true,"memberships":[{"project":{"gid":"1001","name":"Alpha"},"section":null}]}}`))
	}))

	apiClient = newTestClient(srv)
	defer func() { apiClient = nil }()

	outputJSON = true
	defer func() { outputJSON = false }()

	cmd, buf := newTaskDoneTestCmd()
	if err := runTaskDone(cmd, []string{"123"}); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	output := decodeJSONOutput(t, buf)
	if output["action"] != "completed" {
		t.Fatalf("unexpected action: %#v", output["action"])
	}
	task := asMap(t, output["task"])
	if _, ok := task["assignee"]; !ok {
		t.Fatalf("expected assignee key in output")
	}
	memberships := asSlice(t, task["memberships"])
	membership := asMap(t, memberships[0])
	section := asMap(t, membership["section"])
	if _, ok := section["gid"]; !ok {
		t.Fatalf("expected section gid")
	}
	if _, ok := section["name"]; !ok {
		t.Fatalf("expected section name")
	}
}

func TestTaskCreateBasicNoSections(t *testing.T) {
	srv := testutil.MockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/projects/1001/sections" && r.Method == http.MethodGet:
			_, _ = w.Write([]byte(`{"data":[]}`))
		case r.URL.Path == "/tasks" && r.Method == http.MethodPost:
			data := decodeRequestData(t, r)
			if data["name"] != "New Task" {
				t.Fatalf("unexpected name: %#v", data["name"])
			}
			if _, ok := data["memberships"]; ok {
				t.Fatalf("did not expect memberships in request")
			}
			_, _ = w.Write([]byte(`{"data":{"gid":"4001","name":"New Task","completed":false,"memberships":[{"project":{"gid":"1001","name":"Alpha"},"section":null}]}}`))
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))

	apiClient = newTestClient(srv)
	defer func() { apiClient = nil }()

	cmd, buf := newTaskCreateTestCmd()
	setFlag(t, cmd, "project-gid", "1001")

	if err := runTaskCreate(cmd, []string{"New Task"}); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if strings.TrimSpace(buf.String()) != "Created task 4001" {
		t.Fatalf("unexpected output: %s", buf.String())
	}
}

func TestTaskCreateWithSectionName(t *testing.T) {
	srv := testutil.MockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/projects/1001/sections" && r.Method == http.MethodGet:
			_, _ = w.Write([]byte(`{"data":[{"gid":"2001","name":"Done"}]}`))
		case r.URL.Path == "/sections/2001" && r.Method == http.MethodGet:
			_, _ = w.Write([]byte(`{"data":{"gid":"2001","name":"Done","project":{"gid":"1001","name":"Alpha"}}}`))
		case r.URL.Path == "/tasks" && r.Method == http.MethodPost:
			data := decodeRequestData(t, r)
			memberships := asSlice(t, data["memberships"])
			membership := asMap(t, memberships[0])
			if membership["section"] != "2001" {
				t.Fatalf("unexpected membership: %#v", membership)
			}
			_, _ = w.Write([]byte(`{"data":{"gid":"4001","name":"New Task","completed":false,"memberships":[{"project":{"gid":"1001","name":"Alpha"},"section":{"gid":"2001","name":"Done"}}]}}`))
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))

	apiClient = newTestClient(srv)
	defer func() { apiClient = nil }()

	cmd, _ := newTaskCreateTestCmd()
	setFlag(t, cmd, "project-gid", "1001")
	setFlag(t, cmd, "section", "Done")

	if err := runTaskCreate(cmd, []string{"New Task"}); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestTaskCreateDefaultSection(t *testing.T) {
	srv := testutil.MockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/projects/1001/sections" && r.Method == http.MethodGet:
			_, _ = w.Write([]byte(`{"data":[{"gid":"2001","name":"One"},{"gid":"2002","name":"Two"}]}`))
		case r.URL.Path == "/tasks" && r.Method == http.MethodPost:
			data := decodeRequestData(t, r)
			memberships := asSlice(t, data["memberships"])
			membership := asMap(t, memberships[0])
			if membership["section"] != "2001" {
				t.Fatalf("expected first section, got %#v", membership["section"])
			}
			_, _ = w.Write([]byte(`{"data":{"gid":"4001","name":"New Task","completed":false,"memberships":[{"project":{"gid":"1001","name":"Alpha"},"section":{"gid":"2001","name":"One"}}]}}`))
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))

	apiClient = newTestClient(srv)
	defer func() { apiClient = nil }()

	cmd, _ := newTaskCreateTestCmd()
	setFlag(t, cmd, "project-gid", "1001")

	if err := runTaskCreate(cmd, []string{"New Task"}); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestTaskCreateAssigneeMe(t *testing.T) {
	srv := testutil.MockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/users/me" && r.Method == http.MethodGet:
			_, _ = w.Write([]byte(`{"data":{"gid":"me","name":"Me"}}`))
		case r.URL.Path == "/projects/1001/sections" && r.Method == http.MethodGet:
			_, _ = w.Write([]byte(`{"data":[]}`))
		case r.URL.Path == "/tasks" && r.Method == http.MethodPost:
			data := decodeRequestData(t, r)
			if data["assignee"] != "me" {
				t.Fatalf("expected assignee me, got %#v", data["assignee"])
			}
			_, _ = w.Write([]byte(`{"data":{"gid":"4001","name":"New Task","completed":false,"assignee":{"gid":"me","name":"Me"},"memberships":[{"project":{"gid":"1001","name":"Alpha"},"section":null}]}}`))
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))

	apiClient = newTestClient(srv)
	defer func() { apiClient = nil }()

	cmd, _ := newTaskCreateTestCmd()
	setFlag(t, cmd, "project-gid", "1001")
	setFlag(t, cmd, "assignee", "me")

	if err := runTaskCreate(cmd, []string{"New Task"}); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestTaskCreateAssigneeGID(t *testing.T) {
	srv := testutil.MockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/users/me":
			t.Fatalf("did not expect /users/me request")
		case r.URL.Path == "/projects/1001/sections" && r.Method == http.MethodGet:
			_, _ = w.Write([]byte(`{"data":[]}`))
		case r.URL.Path == "/tasks" && r.Method == http.MethodPost:
			data := decodeRequestData(t, r)
			if data["assignee"] != "123" {
				t.Fatalf("expected assignee 123, got %#v", data["assignee"])
			}
			_, _ = w.Write([]byte(`{"data":{"gid":"4001","name":"New Task","completed":false,"assignee":{"gid":"123","name":"Other"},"memberships":[{"project":{"gid":"1001","name":"Alpha"},"section":null}]}}`))
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))

	apiClient = newTestClient(srv)
	defer func() { apiClient = nil }()

	cmd, _ := newTaskCreateTestCmd()
	setFlag(t, cmd, "project-gid", "1001")
	setFlag(t, cmd, "assignee", "123")

	if err := runTaskCreate(cmd, []string{"New Task"}); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestTaskCreateDueOn(t *testing.T) {
	srv := testutil.MockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/projects/1001/sections" && r.Method == http.MethodGet:
			_, _ = w.Write([]byte(`{"data":[]}`))
		case r.URL.Path == "/tasks" && r.Method == http.MethodPost:
			data := decodeRequestData(t, r)
			if data["due_on"] != "2025-01-15" {
				t.Fatalf("expected due_on, got %#v", data["due_on"])
			}
			if _, ok := data["due_at"]; ok {
				t.Fatalf("did not expect due_at")
			}
			_, _ = w.Write([]byte(`{"data":{"gid":"4001","name":"New Task","completed":false,"memberships":[{"project":{"gid":"1001","name":"Alpha"},"section":null}]}}`))
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))

	apiClient = newTestClient(srv)
	defer func() { apiClient = nil }()

	cmd, _ := newTaskCreateTestCmd()
	setFlag(t, cmd, "project-gid", "1001")
	setFlag(t, cmd, "due", "2025-01-15")

	if err := runTaskCreate(cmd, []string{"New Task"}); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestTaskCreateDueAt(t *testing.T) {
	srv := testutil.MockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/projects/1001/sections" && r.Method == http.MethodGet:
			_, _ = w.Write([]byte(`{"data":[]}`))
		case r.URL.Path == "/tasks" && r.Method == http.MethodPost:
			data := decodeRequestData(t, r)
			if data["due_at"] != "2025-01-15T13:30:00.000Z" {
				t.Fatalf("expected due_at converted to UTC, got %#v", data["due_at"])
			}
			if _, ok := data["due_on"]; ok {
				t.Fatalf("did not expect due_on")
			}
			_, _ = w.Write([]byte(`{"data":{"gid":"4001","name":"New Task","completed":false,"memberships":[{"project":{"gid":"1001","name":"Alpha"},"section":null}]}}`))
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))

	apiClient = newTestClient(srv)
	defer func() { apiClient = nil }()

	cmd, _ := newTaskCreateTestCmd()
	setFlag(t, cmd, "project-gid", "1001")
	setFlag(t, cmd, "due", "2025-01-15T14:30:00+01:00")

	if err := runTaskCreate(cmd, []string{"New Task"}); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestParseDueDateRejectsLocalTime(t *testing.T) {
	_, _, err := parseDueDate("2025-01-15T14:30:00")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestTaskCreateMissingProject(t *testing.T) {
	cmd, _ := newTaskCreateTestCmd()
	if err := runTaskCreate(cmd, []string{"New Task"}); err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestTaskCreateAmbiguousProject(t *testing.T) {
	srv := testutil.MockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/workspaces":
			_, _ = w.Write([]byte(`{"data":[{"gid":"3001","name":"One"},{"gid":"3002","name":"Two"}]}`))
		case r.URL.Path == "/projects":
			if r.URL.Query().Get("workspace") == "3001" {
				_, _ = w.Write([]byte(`{"data":[{"gid":"1001","name":"Alpha"}]}`))
				return
			}
			_, _ = w.Write([]byte(`{"data":[{"gid":"1002","name":"Alpha"}]}`))
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))

	apiClient = newTestClient(srv)
	defer func() { apiClient = nil }()

	cmd, _ := newTaskCreateTestCmd()
	setFlag(t, cmd, "project", "Alpha")

	err := runTaskCreate(cmd, []string{"New Task"})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	var ambiguous *resolve.AmbiguousError
	if !errors.As(err, &ambiguous) {
		t.Fatalf("expected ambiguous error, got %T", err)
	}
}

func TestTaskCreateAmbiguousSection(t *testing.T) {
	srv := testutil.MockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/projects/1001/sections" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"data":[{"gid":"2001","name":"Done"},{"gid":"2002","name":"Done"}]}`))
	}))

	apiClient = newTestClient(srv)
	defer func() { apiClient = nil }()

	cmd, _ := newTaskCreateTestCmd()
	setFlag(t, cmd, "project-gid", "1001")
	setFlag(t, cmd, "section", "Done")

	err := runTaskCreate(cmd, []string{"New Task"})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	var ambiguous *resolve.AmbiguousError
	if !errors.As(err, &ambiguous) {
		t.Fatalf("expected ambiguous error, got %T", err)
	}
}

func TestTaskCreateInvalidGIDs(t *testing.T) {
	cases := []struct {
		name       string
		flags      map[string]string
		wantErr    string
		withServer bool
	}{
		{
			name:    "project",
			flags:   map[string]string{"project-gid": "abc"},
			wantErr: "Invalid project gid \"abc\": must be numeric",
		},
		{
			name:    "section",
			flags:   map[string]string{"project-gid": "1001", "section-gid": "abc"},
			wantErr: "Invalid section gid \"abc\": must be numeric",
		},
		{
			name:       "assignee",
			flags:      map[string]string{"project-gid": "1001", "assignee": "bad"},
			wantErr:    "invalid --assignee value \"bad\" (use me or a GID)",
			withServer: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.withServer {
				srv := testutil.MockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path != "/projects/1001/sections" {
						t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
					}
					_, _ = w.Write([]byte(`{"data":[]}`))
				}))
				apiClient = newTestClient(srv)
				defer func() { apiClient = nil }()
			}

			cmd, _ := newTaskCreateTestCmd()
			for key, value := range tc.flags {
				setFlag(t, cmd, key, value)
			}

			err := runTaskCreate(cmd, []string{"New Task"})
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if err.Error() != tc.wantErr {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestTaskCreateDryRun(t *testing.T) {
	srv := testutil.MockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/projects/1001" && r.Method == http.MethodGet:
			_, _ = w.Write([]byte(`{"data":{"gid":"1001","name":"Alpha"}}`))
		case r.URL.Path == "/projects/1001/sections" && r.Method == http.MethodGet:
			_, _ = w.Write([]byte(`{"data":[]}`))
		case r.Method == http.MethodPost:
			t.Fatalf("unexpected POST in dry-run")
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))

	apiClient = newTestClient(srv)
	defer func() { apiClient = nil }()

	cmd, buf := newTaskCreateTestCmd()
	setFlag(t, cmd, "project-gid", "1001")
	setFlag(t, cmd, "dry-run", "true")

	if err := runTaskCreate(cmd, []string{"New Task"}); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !strings.Contains(buf.String(), "[dry-run]") {
		t.Fatalf("expected dry-run output, got %s", buf.String())
	}
}

func TestTaskCreateJSONOutput(t *testing.T) {
	srv := testutil.MockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/projects/1001" && r.Method == http.MethodGet:
			_, _ = w.Write([]byte(`{"data":{"gid":"1001","name":"Alpha"}}`))
		case r.URL.Path == "/projects/1001/sections" && r.Method == http.MethodGet:
			_, _ = w.Write([]byte(`{"data":[]}`))
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))

	apiClient = newTestClient(srv)
	defer func() { apiClient = nil }()

	outputJSON = true
	defer func() { outputJSON = false }()

	cmd, buf := newTaskCreateTestCmd()
	setFlag(t, cmd, "project-gid", "1001")
	setFlag(t, cmd, "dry-run", "true")

	if err := runTaskCreate(cmd, []string{"New Task"}); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	output := decodeJSONOutput(t, buf)
	if output["action"] != "created" {
		t.Fatalf("unexpected action: %#v", output["action"])
	}
	task := asMap(t, output["task"])
	if _, ok := task["assignee"]; !ok {
		t.Fatalf("expected assignee key in output")
	}
	memberships := asSlice(t, task["memberships"])
	membership := asMap(t, memberships[0])
	section := asMap(t, membership["section"])
	if _, ok := section["gid"]; !ok {
		t.Fatalf("expected section gid in output")
	}
	if _, ok := section["name"]; !ok {
		t.Fatalf("expected section name in output")
	}
}

func decodeJSONOutput(t *testing.T, buf *bytes.Buffer) map[string]any {
	t.Helper()
	var output map[string]any
	if err := json.Unmarshal(buf.Bytes(), &output); err != nil {
		t.Fatalf("failed to decode output: %v", err)
	}
	return output
}

func decodeRequestData(t *testing.T, r *http.Request) map[string]any {
	t.Helper()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatalf("failed to read request body: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("failed to decode request body: %v", err)
	}
	data, ok := payload["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected data object, got %#v", payload["data"])
	}
	return data
}

func asMap(t *testing.T, value any) map[string]any {
	t.Helper()
	output, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("expected map, got %#v", value)
	}
	return output
}

func asSlice(t *testing.T, value any) []any {
	t.Helper()
	output, ok := value.([]any)
	if !ok {
		t.Fatalf("expected slice, got %#v", value)
	}
	return output
}

func newTaskMoveTestCmd() (*cobra.Command, *bytes.Buffer) {
	var buf bytes.Buffer
	cmd := &cobra.Command{}
	attachTestRuntime(cmd)
	cmd.SetOut(&buf)
	cmd.Flags().String("section", "", "")
	cmd.Flags().String("section-gid", "", "")
	cmd.Flags().String("project", "", "")
	cmd.Flags().String("project-gid", "", "")
	cmd.Flags().Bool("dry-run", false, "")
	return cmd, &buf
}

func newTaskCommentTestCmd() (*cobra.Command, *bytes.Buffer) {
	var buf bytes.Buffer
	cmd := &cobra.Command{}
	attachTestRuntime(cmd)
	cmd.SetOut(&buf)
	cmd.SetIn(bytes.NewBufferString(""))
	cmd.Flags().Bool("dry-run", false, "")
	return cmd, &buf
}

func newTaskDoneTestCmd() (*cobra.Command, *bytes.Buffer) {
	var buf bytes.Buffer
	cmd := &cobra.Command{}
	attachTestRuntime(cmd)
	cmd.SetOut(&buf)
	cmd.Flags().Bool("dry-run", false, "")
	return cmd, &buf
}

func newTaskCreateTestCmd() (*cobra.Command, *bytes.Buffer) {
	var buf bytes.Buffer
	cmd := &cobra.Command{}
	attachTestRuntime(cmd)
	cmd.SetOut(&buf)
	cmd.Flags().String("project", "", "")
	cmd.Flags().String("project-gid", "", "")
	cmd.Flags().String("section", "", "")
	cmd.Flags().String("section-gid", "", "")
	cmd.Flags().String("assignee", "", "")
	cmd.Flags().String("due", "", "")
	cmd.Flags().Bool("dry-run", false, "")
	return cmd, &buf
}
