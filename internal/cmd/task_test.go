// Package cmd tests task command behavior.
package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/spf13/cobra"
	"github.com/vincentsch/rungrad/testutil"
)

func TestTaskListBySectionPreservesOrderAndLimit(t *testing.T) {
	srv := testutil.MockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/sections/s1":
			_, _ = w.Write([]byte(`{"data":{"gid":"s1","name":"To-do"}}`))
		case "/projects/p1/sections":
			_, _ = w.Write([]byte(`{"data":[{"gid":"s1","name":"To-do"}]}`))
		case "/tasks":
			if r.URL.Query().Get("section") != "s1" {
				t.Fatalf("unexpected section %q", r.URL.Query().Get("section"))
			}
			_, _ = w.Write([]byte(`{"data":[
				{"gid":"t2","name":"Second","completed":false},
				{"gid":"t1","name":"First","completed":false},
				{"gid":"t3","name":"Third","completed":true}
			]}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))

	apiClient = newTestClient(srv)
	defer func() { apiClient = nil }()

	outputJSON = true
	defer func() { outputJSON = false }()

	cmd, buf := newTaskListTestCmd()
	setFlag(t, cmd, "project-gid", "p1")
	setFlag(t, cmd, "section", "To-do")
	setFlag(t, cmd, "limit", "2")

	if err := runTaskList(cmd, nil); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	var tasks []taskListItem
	if err := json.Unmarshal(buf.Bytes(), &tasks); err != nil {
		t.Fatalf("failed to parse output: %v", err)
	}
	if len(tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(tasks))
	}
	if tasks[0].GID != "t2" || tasks[1].GID != "t1" {
		t.Fatalf("unexpected order: %#v", tasks)
	}
	if tasks[0].Section == nil || tasks[0].Section.GID != "s1" || tasks[0].Section.Name != "To-do" {
		t.Fatalf("expected section details in output")
	}
}

func TestTaskListBySectionGIDFetchesSectionName(t *testing.T) {
	srv := testutil.MockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/sections/s1":
			_, _ = w.Write([]byte(`{"data":{"gid":"s1","name":"Backlog"}}`))
		case "/tasks":
			if r.URL.Query().Get("section") != "s1" {
				t.Fatalf("unexpected section %q", r.URL.Query().Get("section"))
			}
			_, _ = w.Write([]byte(`{"data":[{"gid":"t1","name":"Task","completed":false}]}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))

	apiClient = newTestClient(srv)
	defer func() { apiClient = nil }()

	outputJSON = true
	defer func() { outputJSON = false }()

	cmd, buf := newTaskListTestCmd()
	setFlag(t, cmd, "section-gid", "s1")

	if err := runTaskList(cmd, nil); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	var tasks []taskListItem
	if err := json.Unmarshal(buf.Bytes(), &tasks); err != nil {
		t.Fatalf("failed to parse output: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}
	if tasks[0].Section == nil || tasks[0].Section.Name != "Backlog" {
		t.Fatalf("expected section name from API, got %#v", tasks[0].Section)
	}
}

func TestTaskListByProjectNoSectionBucket(t *testing.T) {
	srv := testutil.MockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/projects/p1/sections":
			_, _ = w.Write([]byte(`{"data":[{"gid":"s1","name":"To-do"},{"gid":"s2","name":"Next"}]}`))
		case "/tasks":
			if section := r.URL.Query().Get("section"); section != "" {
				switch section {
				case "s1":
					_, _ = w.Write([]byte(`{"data":[{"gid":"a1","name":"A1","completed":false},{"gid":"a2","name":"A2","completed":false}]}`))
				case "s2":
					_, _ = w.Write([]byte(`{"data":[{"gid":"b1","name":"B1","completed":false}]}`))
				default:
					t.Fatalf("unexpected section %q", section)
				}
				return
			}
			if project := r.URL.Query().Get("project"); project == "p1" {
				_, _ = w.Write([]byte(`{"data":[
					{"gid":"a1","name":"A1","completed":false},
					{"gid":"a2","name":"A2","completed":false},
					{"gid":"b1","name":"B1","completed":false},
					{"gid":"c1","name":"C1","completed":true}
				]}`))
				return
			}
			t.Fatalf("unexpected tasks query %v", r.URL.RawQuery)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))

	apiClient = newTestClient(srv)
	defer func() { apiClient = nil }()

	outputJSON = true
	defer func() { outputJSON = false }()

	cmd, buf := newTaskListTestCmd()
	setFlag(t, cmd, "project-gid", "p1")

	if err := runTaskList(cmd, nil); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	var tasks []taskListItem
	if err := json.Unmarshal(buf.Bytes(), &tasks); err != nil {
		t.Fatalf("failed to parse output: %v", err)
	}
	if len(tasks) != 4 {
		t.Fatalf("expected 4 tasks, got %d", len(tasks))
	}
	if tasks[0].GID != "a1" || tasks[1].GID != "a2" || tasks[2].GID != "b1" || tasks[3].GID != "c1" {
		t.Fatalf("unexpected order: %#v", tasks)
	}
	if tasks[3].Section != nil {
		t.Fatalf("expected no-section task to have nil section")
	}
}

func TestTaskListAssigneeMe(t *testing.T) {
	srv := testutil.MockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/sections/s1":
			_, _ = w.Write([]byte(`{"data":{"gid":"s1","name":"To-do"}}`))
		case "/users/me":
			_, _ = w.Write([]byte(`{"data":{"gid":"me","name":"Me"}}`))
		case "/tasks":
			_, _ = w.Write([]byte(`{"data":[
				{"gid":"t1","name":"Mine","completed":false,"assignee":{"gid":"me","name":"Me"}},
				{"gid":"t2","name":"Other","completed":false,"assignee":{"gid":"someone","name":"Other"}}
			]}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))

	apiClient = newTestClient(srv)
	defer func() { apiClient = nil }()

	outputJSON = true
	defer func() { outputJSON = false }()

	cmd, buf := newTaskListTestCmd()
	setFlag(t, cmd, "section-gid", "s1")
	setFlag(t, cmd, "assignee", "me")

	if err := runTaskList(cmd, nil); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	var tasks []taskListItem
	if err := json.Unmarshal(buf.Bytes(), &tasks); err != nil {
		t.Fatalf("failed to parse output: %v", err)
	}
	if len(tasks) != 1 || tasks[0].GID != "t1" {
		t.Fatalf("unexpected filter result: %#v", tasks)
	}
}

func TestTaskListInvalidAssignee(t *testing.T) {
	cmd, _ := newTaskListTestCmd()
	setFlag(t, cmd, "section-gid", "s1")
	setFlag(t, cmd, "assignee", "bob")

	if err := runTaskList(cmd, nil); err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestTaskListSectionRequiresProject(t *testing.T) {
	cmd, _ := newTaskListTestCmd()
	setFlag(t, cmd, "section", "To-do")

	err := runTaskList(cmd, nil)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if err.Error() != "--section requires --project or --project-gid" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTaskListPagination(t *testing.T) {
	srv := testutil.MockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/sections/s1":
			_, _ = w.Write([]byte(`{"data":{"gid":"s1","name":"To-do"}}`))
		case "/tasks":
			switch r.URL.Query().Get("offset") {
			case "":
				_, _ = w.Write([]byte(`{"data":[{"gid":"t1","name":"First","completed":false}],"next_page":{"offset":"next"}}`))
			case "next":
				_, _ = w.Write([]byte(`{"data":[{"gid":"t2","name":"Second","completed":false}]}`))
			default:
				t.Fatalf("unexpected offset %q", r.URL.Query().Get("offset"))
			}
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))

	apiClient = newTestClient(srv)
	defer func() { apiClient = nil }()

	outputJSON = true
	defer func() { outputJSON = false }()

	cmd, buf := newTaskListTestCmd()
	setFlag(t, cmd, "section-gid", "s1")

	if err := runTaskList(cmd, nil); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	var tasks []taskListItem
	if err := json.Unmarshal(buf.Bytes(), &tasks); err != nil {
		t.Fatalf("failed to parse output: %v", err)
	}
	if len(tasks) != 2 || tasks[0].GID != "t1" || tasks[1].GID != "t2" {
		t.Fatalf("unexpected pagination result: %#v", tasks)
	}
}

func TestTaskViewCommentsLimitAndOrder(t *testing.T) {
	var storyCalls int
	srv := testutil.MockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/tasks/123":
			_, _ = w.Write([]byte(`{"data":{
				"gid":"123",
				"name":"Fix login",
				"completed":false,
				"assignee":{"gid":"me","name":"Me"},
				"due_on":"2025-01-15",
				"notes":"Note",
				"memberships":[{"project":{"gid":"p1","name":"Proj"},"section":{"gid":"s1","name":"To-do"}}]
			}}`))
		case "/tasks/123/stories":
			switch r.URL.Query().Get("offset") {
			case "":
				storyCalls++
				_, _ = w.Write([]byte(`{"data":[
					{"gid":"c1","resource_subtype":"comment_added","text":"Latest","created_at":"2025-01-10T14:30:00Z","created_by":{"gid":"me","name":"Me"}},
					{"gid":"s1","resource_subtype":"section_changed","text":"ignored","created_at":"2025-01-08T10:00:00Z"},
					{"gid":"c2","resource_subtype":"comment_added","text":"Older","created_at":"2025-01-09T09:15:00Z","created_by":{"gid":"u2","name":"Other"}}
				],"next_page":{"offset":"next"}}`))
			case "next":
				storyCalls++
				_, _ = w.Write([]byte(`{"data":[
					{"gid":"c3","text":"Fallback","created_at":"2025-01-11T08:00:00Z","created_by":{"gid":"u3","name":"Third"}}
				]}`))
			default:
				t.Fatalf("unexpected offset %q", r.URL.Query().Get("offset"))
			}
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))

	apiClient = newTestClient(srv)
	defer func() { apiClient = nil }()

	outputJSON = true
	defer func() { outputJSON = false }()

	cmd, buf := newTaskViewTestCmd()
	setFlag(t, cmd, "comments-limit", "2")

	if err := runTaskView(cmd, []string{"123"}); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	var output taskViewOutput
	if err := json.Unmarshal(buf.Bytes(), &output); err != nil {
		t.Fatalf("failed to parse output: %v", err)
	}
	if output.GID != "123" || output.DueOn == nil || *output.DueOn != "2025-01-15" {
		t.Fatalf("unexpected task payload: %#v", output)
	}
	if len(output.Comments) != 2 {
		t.Fatalf("expected 2 comments, got %d", len(output.Comments))
	}
	if output.Comments[0].GID != "c3" || output.Comments[1].GID != "c1" {
		t.Fatalf("unexpected comment order: %#v", output.Comments)
	}
	if storyCalls != 2 {
		t.Fatalf("expected 2 story calls, got %d", storyCalls)
	}
}

func newTaskListTestCmd() (*cobra.Command, *bytes.Buffer) {
	var buf bytes.Buffer
	cmd := &cobra.Command{}
	attachTestRuntime(cmd)
	cmd.SetOut(&buf)
	cmd.Flags().String("project", "", "")
	cmd.Flags().String("project-gid", "", "")
	cmd.Flags().String("section", "", "")
	cmd.Flags().String("section-gid", "", "")
	cmd.Flags().String("workspace", "", "")
	cmd.Flags().String("workspace-gid", "", "")
	cmd.Flags().String("assignee", "any", "")
	cmd.Flags().Int("limit", 0, "")
	return cmd, &buf
}

func newTaskViewTestCmd() (*cobra.Command, *bytes.Buffer) {
	var buf bytes.Buffer
	cmd := &cobra.Command{}
	attachTestRuntime(cmd)
	cmd.SetOut(&buf)
	cmd.Flags().Int("comments-limit", 5, "")
	return cmd, &buf
}

func setFlag(t *testing.T, cmd *cobra.Command, name, value string) {
	t.Helper()
	if err := cmd.Flags().Set(name, value); err != nil {
		t.Fatalf("failed to set flag %s: %v", name, err)
	}
}
