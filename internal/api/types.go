// Package api defines shared Asana API response and resource types.
package api

// Response wraps standard Asana API responses.
type Response[T any] struct {
	Data     T         `json:"data"`
	NextPage *NextPage `json:"next_page,omitempty"`
}

// NextPage represents cursor pagination metadata.
type NextPage struct {
	Offset string `json:"offset"`
	Path   string `json:"path"`
	URI    string `json:"uri"`
}

// ErrorResponse represents an error payload from Asana.
type ErrorResponse struct {
	Errors []AsanaError `json:"errors"`
}

// AsanaError is a single error entry returned by the API.
type AsanaError struct {
	Message string `json:"message"`
	Help    string `json:"help,omitempty"`
	Phrase  string `json:"phrase,omitempty"`
}

// Workspace represents the Asana workspace resource.
type Workspace struct {
	GID  string `json:"gid"`
	Name string `json:"name"`
}

// Project represents an Asana project.
type Project struct {
	GID  string `json:"gid"`
	Name string `json:"name"`
}

// Section represents a Kanban column within a project.
type Section struct {
	GID  string `json:"gid"`
	Name string `json:"name"`
}

// SectionWithProject represents a section with its parent project reference.
type SectionWithProject struct {
	GID     string   `json:"gid"`
	Name    string   `json:"name"`
	Project *Project `json:"project,omitempty"`
}

// Team represents an Asana team within a workspace.
type Team struct {
	GID  string `json:"gid"`
	Name string `json:"name"`
}

// User represents an Asana user (assignee, creator).
type User struct {
	GID   string `json:"gid"`
	Name  string `json:"name"`
	Email string `json:"email,omitempty"`
}

// Task represents an Asana task.
type Task struct {
	GID         string       `json:"gid"`
	Name        string       `json:"name"`
	Completed   bool         `json:"completed"`
	Assignee    *User        `json:"assignee,omitempty"`
	DueOn       string       `json:"due_on,omitempty"`
	DueAt       string       `json:"due_at,omitempty"`
	Notes       string       `json:"notes,omitempty"`
	Memberships []Membership `json:"memberships,omitempty"`
}

// Membership represents a task's membership in a project/section.
type Membership struct {
	Project *Project `json:"project,omitempty"`
	Section *Section `json:"section,omitempty"`
}

// Story represents a comment or activity on a task.
type Story struct {
	GID       string `json:"gid"`
	Subtype   string `json:"resource_subtype,omitempty"`
	Text      string `json:"text,omitempty"`
	CreatedAt string `json:"created_at"`
	CreatedBy *User  `json:"created_by,omitempty"`
}

// Tag represents an Asana tag.
type Tag struct {
	GID  string `json:"gid"`
	Name string `json:"name"`
}

// WorkspaceDetail extends Workspace with organization info.
type WorkspaceDetail struct {
	GID            string   `json:"gid"`
	Name           string   `json:"name"`
	IsOrganization bool     `json:"is_organization"`
	EmailDomains   []string `json:"email_domains,omitempty"`
}

// ProjectDetail extends Project with full metadata.
type ProjectDetail struct {
	GID            string `json:"gid"`
	Name           string `json:"name"`
	Archived       bool   `json:"archived"`
	Color          string `json:"color,omitempty"`
	Notes          string `json:"notes,omitempty"`
	DueOn          string `json:"due_on,omitempty"`
	PrivacySetting string `json:"privacy_setting,omitempty"`
	Owner          *User  `json:"owner,omitempty"`
	Team           *Team  `json:"team,omitempty"`
}

// SectionDetail extends Section with timestamps.
type SectionDetail struct {
	GID       string   `json:"gid"`
	Name      string   `json:"name"`
	CreatedAt string   `json:"created_at"`
	Project   *Project `json:"project,omitempty"`
}

// ProjectTaskCounts holds task count metrics.
type ProjectTaskCounts struct {
	NumTasks           int `json:"num_tasks"`
	NumIncompleteTasks int `json:"num_incomplete_tasks"`
	NumCompletedTasks  int `json:"num_completed_tasks"`
}

// Job represents an async job response.
type Job struct {
	GID        string   `json:"gid"`
	Status     string   `json:"status"`
	NewTask    *Task    `json:"new_task,omitempty"`
	NewProject *Project `json:"new_project,omitempty"`
}

// ProjectMembership represents a user/team's membership in a project.
type ProjectMembership struct {
	GID         string `json:"gid"`
	Member      *User  `json:"member,omitempty"`
	AccessLevel string `json:"access_level,omitempty"`
}

// TaskDependency represents a task dependency/dependent relationship.
type TaskDependency struct {
	GID  string `json:"gid"`
	Name string `json:"name,omitempty"`
}

// UserDetail extends User with workspaces list.
type UserDetail struct {
	GID        string      `json:"gid"`
	Name       string      `json:"name"`
	Email      string      `json:"email,omitempty"`
	Workspaces []Workspace `json:"workspaces,omitempty"`
}

// TeamDetail extends Team with full metadata.
type TeamDetail struct {
	GID          string     `json:"gid"`
	Name         string     `json:"name"`
	Description  string     `json:"description,omitempty"`
	Visibility   string     `json:"visibility,omitempty"`
	Organization *Workspace `json:"organization,omitempty"`
}

// TeamMember represents a team membership with user info.
type TeamMember struct {
	GID             string `json:"gid"`
	User            *User  `json:"user,omitempty"`
	IsGuest         bool   `json:"is_guest,omitempty"`
	IsLimitedAccess bool   `json:"is_limited_access,omitempty"`
	IsAdmin         bool   `json:"is_admin,omitempty"`
}

// TagDetail extends Tag with full metadata.
type TagDetail struct {
	GID       string     `json:"gid"`
	Name      string     `json:"name"`
	Color     string     `json:"color,omitempty"`
	Notes     string     `json:"notes,omitempty"`
	Workspace *Workspace `json:"workspace,omitempty"`
	CreatedAt string     `json:"created_at,omitempty"`
}

// Attachment represents a file or URL attached to a task.
type Attachment struct {
	GID             string `json:"gid"`
	Name            string `json:"name"`
	ResourceSubtype string `json:"resource_subtype,omitempty"`
	Host            string `json:"host,omitempty"`
	DownloadURL     string `json:"download_url,omitempty"`
	PermanentURL    string `json:"permanent_url,omitempty"`
	ViewURL         string `json:"view_url,omitempty"`
	Size            int64  `json:"size,omitempty"`
	CreatedAt       string `json:"created_at,omitempty"`
}

// CustomFieldDetail represents a custom field definition.
type CustomFieldDetail struct {
	GID             string       `json:"gid"`
	Name            string       `json:"name"`
	ResourceSubtype string       `json:"resource_subtype,omitempty"`
	Description     string       `json:"description,omitempty"`
	Precision       *int         `json:"precision,omitempty"`
	EnumOptions     []EnumOption `json:"enum_options,omitempty"`
	Enabled         bool         `json:"enabled"`
}

// CustomFieldCompact represents a custom field for listing.
type CustomFieldCompact struct {
	GID             string `json:"gid"`
	Name            string `json:"name"`
	ResourceSubtype string `json:"resource_subtype,omitempty"`
}

// EnumOption represents an option for enum/multi_enum custom fields.
type EnumOption struct {
	GID     string `json:"gid"`
	Name    string `json:"name"`
	Color   string `json:"color,omitempty"`
	Enabled bool   `json:"enabled"`
}

// Goal represents an Asana goal.
type Goal struct {
	GID       string      `json:"gid"`
	Name      string      `json:"name"`
	Notes     string      `json:"notes,omitempty"`
	DueOn     string      `json:"due_on,omitempty"`
	StartOn   string      `json:"start_on,omitempty"`
	Status    string      `json:"status,omitempty"`
	Owner     *User       `json:"owner,omitempty"`
	Metric    *GoalMetric `json:"metric,omitempty"`
	Team      *Team       `json:"team,omitempty"`
	Workspace *Workspace  `json:"workspace,omitempty"`
}

// GoalMetric represents a goal's progress metric.
type GoalMetric struct {
	GID                 string  `json:"gid,omitempty"`
	ResourceSubtype     string  `json:"resource_subtype,omitempty"`
	Unit                string  `json:"unit,omitempty"`
	Precision           int     `json:"precision,omitempty"`
	CurrentNumberValue  float64 `json:"current_number_value,omitempty"`
	TargetNumberValue   float64 `json:"target_number_value,omitempty"`
	InitialNumberValue  float64 `json:"initial_number_value,omitempty"`
	CurrentDisplayValue string  `json:"current_display_value,omitempty"`
}

// Portfolio represents an Asana portfolio.
type Portfolio struct {
	GID       string     `json:"gid"`
	Name      string     `json:"name"`
	Color     string     `json:"color,omitempty"`
	Owner     *User      `json:"owner,omitempty"`
	Workspace *Workspace `json:"workspace,omitempty"`
	CreatedAt string     `json:"created_at,omitempty"`
}

// PortfolioItem represents an item in a portfolio.
type PortfolioItem struct {
	GID          string `json:"gid"`
	Name         string `json:"name"`
	ResourceType string `json:"resource_type,omitempty"`
}
