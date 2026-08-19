# asana task

Manage tasks

## Usage

```
asana task
```

## Related commands

- project
- section
- user

## asana task comment

Manage task comments

### Usage

```
asana task comment <task-gid> <message>
```

### Examples

```
asana task comment 1234567890123456 "Looking into this now"
printf 'Line 1\nLine 2\n' | asana task comment 1234567890123456 -
asana task view 1234567890123456
asana task view 1234567890123456
asana task comment 1234567890123456 "Fixed in commit abc123"
asana task done 1234567890123456
```

### Output modes

human, json, plain, jq, template

### Authentication

This command requires a credential.

### Changes state

This command changes state and honors `--dry-run`.

### Metadata

This command can attach request metadata. In advanced-output apps, add `--include-meta` to a supported machine output mode to wrap the result as `{data, meta}`.

### Related commands

- task view
- task update

### asana task comment delete

Delete a story/comment

#### Usage

```
asana task comment delete <story-gid>
```

#### Examples

```
asana task comment delete 1234567890123456 --confirm
asana task comment delete 1234567890123456 --dry-run
```

#### Flags

- `--confirm` Confirm the destructive action without a prompt

#### Output modes

human, json, plain, jq, template

#### Authentication

This command requires a credential.

#### Changes state

This command changes state and honors `--dry-run`.

#### Destructive

This command performs a destructive action and asks for confirmation before acting. Preview it first with `--dry-run`; outside a dry run it proceeds only after explicit confirmation, and in non-interactive mode (`--json`, `--no-prompt`, or no terminal) it requires a confirmation flag instead of blocking.

#### Metadata

This command can attach request metadata. In advanced-output apps, add `--include-meta` to a supported machine output mode to wrap the result as `{data, meta}`.

#### Related commands

- task comment
- task comment update

### asana task comment update

Update a story/comment

#### Usage

```
asana task comment update <story-gid> <message>
```

#### Examples

```
asana task comment update 1234567890123456 "Updated message"
echo "New text" | asana task comment update 1234567890123456 -
asana task comment update 1234567890123456 "Test" --dry-run
```

#### Output modes

human, json, plain, jq, template

#### Authentication

This command requires a credential.

#### Changes state

This command changes state and honors `--dry-run`.

#### Metadata

This command can attach request metadata. In advanced-output apps, add `--include-meta` to a supported machine output mode to wrap the result as `{data, meta}`.

#### Related commands

- task comment
- task comment delete

## asana task create

Create a new task

### Usage

```
asana task create <title>
```

### Examples

```
asana task create "Fix login bug" -p "My Project"
asana task create "Review PR" -p "Sprint" --assignee me --due 2024-12-31
asana task create "New feature" -p "My Project" -s "To Do"
asana task create "Test task" -p "My Project" --dry-run
asana task create "Urgent fix" -p "Sprint" --json | jq -r '.task.gid'
asana task view <gid-from-above>
```

### Flags

- `--assignee` Assignee (me or GID)
- `--due` Due date (YYYY-MM-DD or RFC3339 with timezone)
- `--project` Project name
- `--project-gid` Project GID
- `--section` Section name
- `--section-gid` Section GID
- `--workspace` Workspace name (scopes project lookup)
- `--workspace-gid` Workspace GID (scopes project lookup)

### Output modes

human, json, plain, jq, template

### Authentication

This command requires a credential.

### Changes state

This command changes state and honors `--dry-run`.

### Metadata

This command can attach request metadata. In advanced-output apps, add `--include-meta` to a supported machine output mode to wrap the result as `{data, meta}`.

### Related commands

- task list
- task view
- task update
- section list

## asana task delete

Delete a task

### Usage

```
asana task delete <gid>
```

### Examples

```
asana task delete 1234567890123456 --confirm
asana task delete 1234567890123456 --dry-run
```

### Flags

- `--confirm` Confirm the destructive action without a prompt

### Output modes

human, json, plain, jq, template

### Authentication

This command requires a credential.

### Changes state

This command changes state and honors `--dry-run`.

### Destructive

This command performs a destructive action and asks for confirmation before acting. Preview it first with `--dry-run`; outside a dry run it proceeds only after explicit confirmation, and in non-interactive mode (`--json`, `--no-prompt`, or no terminal) it requires a confirmation flag instead of blocking.

### Metadata

This command can attach request metadata. In advanced-output apps, add `--include-meta` to a supported machine output mode to wrap the result as `{data, meta}`.

### Related commands

- task done
- task view

## asana task dependency

Manage task dependencies (tasks this task depends on)

### Usage

```
asana task dependency
```

### Related commands

- task dependent
- task view

### asana task dependency add

Add dependencies to a task

#### Usage

```
asana task dependency add <gid>
```

#### Examples

```
asana task dependency add 1234567890123456 --depends-on 9876543210987654
asana task dependency add 1234567890123456 --depends-on 111,222,333
asana task dependency add 1234567890123456 --depends-on 9876543210987654
asana task comment 1234567890123456 "Blocked by design review"
```

#### Flags

- `--depends-on` Task GIDs to add as dependencies (required)

#### Output modes

human, json, plain, jq, template

#### Authentication

This command requires a credential.

#### Changes state

This command changes state and honors `--dry-run`.

#### Metadata

This command can attach request metadata. In advanced-output apps, add `--include-meta` to a supported machine output mode to wrap the result as `{data, meta}`.

#### Related commands

- task dependency list
- task dependency remove

### asana task dependency list

List dependencies of a task

#### Usage

```
asana task dependency list <gid>
```

#### Examples

```
asana task dependency list 1234567890123456
asana task dependency list 1234567890123456 --json
```

#### Flags

- `--limit` Limit number of dependencies in output

#### Output modes

human, json, plain, jq, template

#### Authentication

This command requires a credential.

#### Metadata

This command can attach request metadata. In advanced-output apps, add `--include-meta` to a supported machine output mode to wrap the result as `{data, meta}`.

#### Related commands

- task dependency add
- task dependent list

### asana task dependency remove

Remove dependencies from a task

#### Usage

```
asana task dependency remove <gid>
```

#### Examples

```
asana task dependency remove 1234567890123456 --depends-on 9876543210987654 --confirm
asana task dependency list 1234567890123456
asana task dependency remove 1234567890123456 --depends-on 9876543210987654 --confirm
asana task comment 1234567890123456 "Unblocked - dependency resolved"
```

#### Flags

- `--confirm` Confirm the destructive action without a prompt
- `--depends-on` Task GIDs to remove as dependencies (required)

#### Output modes

human, json, plain, jq, template

#### Authentication

This command requires a credential.

#### Changes state

This command changes state and honors `--dry-run`.

#### Destructive

This command performs a destructive action and asks for confirmation before acting. Preview it first with `--dry-run`; outside a dry run it proceeds only after explicit confirmation, and in non-interactive mode (`--json`, `--no-prompt`, or no terminal) it requires a confirmation flag instead of blocking.

#### Metadata

This command can attach request metadata. In advanced-output apps, add `--include-meta` to a supported machine output mode to wrap the result as `{data, meta}`.

#### Related commands

- task dependency list
- task dependency add

## asana task dependent

Manage task dependents (tasks that depend on this task)

### Usage

```
asana task dependent
```

### Related commands

- task dependency
- task view

### asana task dependent add

Add dependents to a task

#### Usage

```
asana task dependent add <gid>
```

#### Examples

```
asana task dependent add 1234567890123456 --dependent 9876543210987654
asana task dependent add 1234567890123456 --dependent 111,222,333
```

#### Flags

- `--dependent` Task GIDs to add as dependents (required)

#### Output modes

human, json, plain, jq, template

#### Authentication

This command requires a credential.

#### Changes state

This command changes state and honors `--dry-run`.

#### Metadata

This command can attach request metadata. In advanced-output apps, add `--include-meta` to a supported machine output mode to wrap the result as `{data, meta}`.

#### Related commands

- task dependent list
- task dependent remove

### asana task dependent list

List dependents of a task

#### Usage

```
asana task dependent list <gid>
```

#### Examples

```
asana task dependent list 1234567890123456
asana task dependent list 1234567890123456 --json
```

#### Flags

- `--limit` Limit number of dependents in output

#### Output modes

human, json, plain, jq, template

#### Authentication

This command requires a credential.

#### Metadata

This command can attach request metadata. In advanced-output apps, add `--include-meta` to a supported machine output mode to wrap the result as `{data, meta}`.

#### Related commands

- task dependent add
- task dependency list

### asana task dependent remove

Remove dependents from a task

#### Usage

```
asana task dependent remove <gid>
```

#### Examples

```
asana task dependent remove 1234567890123456 --dependent 9876543210987654 --confirm
```

#### Flags

- `--confirm` Confirm the destructive action without a prompt
- `--dependent` Task GIDs to remove as dependents (required)

#### Output modes

human, json, plain, jq, template

#### Authentication

This command requires a credential.

#### Changes state

This command changes state and honors `--dry-run`.

#### Destructive

This command performs a destructive action and asks for confirmation before acting. Preview it first with `--dry-run`; outside a dry run it proceeds only after explicit confirmation, and in non-interactive mode (`--json`, `--no-prompt`, or no terminal) it requires a confirmation flag instead of blocking.

#### Metadata

This command can attach request metadata. In advanced-output apps, add `--include-meta` to a supported machine output mode to wrap the result as `{data, meta}`.

#### Related commands

- task dependent list
- task dependent add

## asana task done

Mark task as complete

### Usage

```
asana task done <gid>
```

### Examples

```
asana task done 1234567890123456
asana task done 1234567890123456 --dry-run
asana task comment 1234567890123456 "Completed - deployed to staging"
asana task done 1234567890123456
```

### Output modes

human, json, plain, jq, template

### Authentication

This command requires a credential.

### Changes state

This command changes state and honors `--dry-run`.

### Metadata

This command can attach request metadata. In advanced-output apps, add `--include-meta` to a supported machine output mode to wrap the result as `{data, meta}`.

### Related commands

- task reopen
- task move
- task update

## asana task duplicate

Duplicate a task

### Usage

```
asana task duplicate <gid>
```

### Examples

```
asana task duplicate 1234567890123456
asana task duplicate 1234567890123456 --name "Copy of Task"
asana task duplicate 1234567890123456 --include assignee,dates,notes,subtasks
asana task duplicate 1234567890123456 --wait
asana task duplicate 1234567890123456 --name "Sprint 2 version" --wait
asana task update <new-gid> --assignee me --due 2024-03-01
```

### Flags

- `--include` Fields to include: assignee, attachments, dates, dependencies, followers, notes, parent, projects, subtasks, tags
- `--name` Name for the duplicated task
- `--wait` Wait for job completion

### Output modes

human, json, plain, jq, template

### Authentication

This command requires a credential.

### Changes state

This command changes state and honors `--dry-run`.

### Metadata

This command can attach request metadata. In advanced-output apps, add `--include-meta` to a supported machine output mode to wrap the result as `{data, meta}`.

### Related commands

- task create
- task view

## asana task follower

Manage task followers

### Usage

```
asana task follower
```

### Related commands

- user list
- task view

### asana task follower add

Add followers to task

#### Usage

```
asana task follower add <gid>
```

#### Examples

```
asana task follower add 1234567890123456 --user me
asana task follower add 1234567890123456 --user jane@example.com -w "My Workspace"
asana task follower add 1234567890123456 --user me --user 9876543210987654
asana task create "New feature" -p "Sprint"
asana task follower add <new-gid> --user me --user dev@example.com
```

#### Flags

- `--user` User(s) to add (me, GID, or email)
- `--workspace` Workspace name (for email resolution)
- `--workspace-gid` Workspace GID

#### Output modes

human, json, plain, jq, template

#### Authentication

This command requires a credential.

#### Changes state

This command changes state and honors `--dry-run`.

#### Metadata

This command can attach request metadata. In advanced-output apps, add `--include-meta` to a supported machine output mode to wrap the result as `{data, meta}`.

#### Related commands

- task follower remove
- user list

### asana task follower remove

Remove followers from task

#### Usage

```
asana task follower remove <gid>
```

#### Examples

```
asana task follower remove 1234567890123456 --user 9876543210987654 --confirm
asana task follower remove 1234567890123456 --user me --dry-run
```

#### Flags

- `--confirm` Confirm the destructive action without a prompt
- `--user` User(s) to remove (me, GID, or email)
- `--workspace` Workspace name (for email resolution)
- `--workspace-gid` Workspace GID

#### Output modes

human, json, plain, jq, template

#### Authentication

This command requires a credential.

#### Changes state

This command changes state and honors `--dry-run`.

#### Destructive

This command performs a destructive action and asks for confirmation before acting. Preview it first with `--dry-run`; outside a dry run it proceeds only after explicit confirmation, and in non-interactive mode (`--json`, `--no-prompt`, or no terminal) it requires a confirmation flag instead of blocking.

#### Metadata

This command can attach request metadata. In advanced-output apps, add `--include-meta` to a supported machine output mode to wrap the result as `{data, meta}`.

#### Related commands

- task follower add
- task view

## asana task list

List tasks in a project or section

### Usage

```
asana task list
```

### Examples

```
asana task list -p "My Project"
asana task list -p "My Project" -s "In Progress"
asana task list -p "My Project" --assignee me
asana task list -p "My Project" --json
asana task list -p "My Project" --limit 10
asana task list -p "Sprint" --assignee me
asana task view 1234567890123456
asana task move 1234567890123456 -s "Done"
```

### Flags

- `--assignee` Filter by assignee (any, me, or GID)
- `--limit` Limit number of tasks in output
- `--project` Project name
- `--project-gid` Project GID
- `--section` Section name
- `--section-gid` Section GID
- `--workspace` Workspace name (scopes project lookup)
- `--workspace-gid` Workspace GID (scopes project lookup)

### Output modes

human, json, plain, jq, template

### Authentication

This command requires a credential.

### Metadata

This command can attach request metadata. In advanced-output apps, add `--include-meta` to a supported machine output mode to wrap the result as `{data, meta}`.

### Related commands

- task view
- task create
- section list

## asana task move

Move task to a section

### Usage

```
asana task move <gid>
```

### Examples

```
asana task move 1234567890123456 -s "In Progress"
asana task move 1234567890123456 --section-gid 9876543210987654
asana task move 1234567890123456 -s "Done" --dry-run
asana section list -p "Sprint"           # See available sections
asana task move 1234567890123456 -s "Review"
```

### Flags

- `--project` Project name (required if task has multiple projects)
- `--project-gid` Project GID
- `--section` Target section name
- `--section-gid` Target section GID

### Output modes

human, json, plain, jq, template

### Authentication

This command requires a credential.

### Changes state

This command changes state and honors `--dry-run`.

### Metadata

This command can attach request metadata. In advanced-output apps, add `--include-meta` to a supported machine output mode to wrap the result as `{data, meta}`.

### Related commands

- task done
- task update
- section list

## asana task parent

Manage task parent

### Usage

```
asana task parent
```

### Related commands

- task subtask
- task view

### asana task parent set

Set or change task parent

#### Usage

```
asana task parent set <gid>
```

#### Examples

```
asana task parent set 1234567890123456 --parent 9876543210987654
asana task parent set 1234567890123456 --parent 9876543210987654 --insert-after 5555555555555555
asana task parent set 1234567890123456 --parent 9876543210987654 --dry-run
```

#### Flags

- `--insert-after` Insert after this sibling task GID
- `--insert-before` Insert before this sibling task GID
- `--parent` New parent task GID (required)

#### Output modes

human, json, plain, jq, template

#### Authentication

This command requires a credential.

#### Changes state

This command changes state and honors `--dry-run`.

#### Metadata

This command can attach request metadata. In advanced-output apps, add `--include-meta` to a supported machine output mode to wrap the result as `{data, meta}`.

#### Related commands

- task subtask create
- task subtask list

## asana task project

Manage task project membership

### Usage

```
asana task project
```

### Related commands

- task move
- project list

### asana task project add

Add task to a project

#### Usage

```
asana task project add <gid>
```

#### Examples

```
asana task project add 1234567890123456 -p "My Project"
asana task project add 1234567890123456 -p "My Project" -s "In Progress"
asana task project add 1234567890123456 -p "Sprint 1"
asana task project add 1234567890123456 -p "Q1 Goals"
```

#### Flags

- `--insert-after` Insert after this task GID
- `--insert-before` Insert before this task GID
- `--project` Project name
- `--project-gid` Project GID
- `--section` Section name within project
- `--section-gid` Section GID
- `--workspace` Workspace name (for project/section name resolution)
- `--workspace-gid` Workspace GID

#### Output modes

human, json, plain, jq, template

#### Authentication

This command requires a credential.

#### Changes state

This command changes state and honors `--dry-run`.

#### Metadata

This command can attach request metadata. In advanced-output apps, add `--include-meta` to a supported machine output mode to wrap the result as `{data, meta}`.

#### Related commands

- task project remove
- task move
- project list

### asana task project remove

Remove task from a project

#### Usage

```
asana task project remove <gid>
```

#### Examples

```
asana task project remove 1234567890123456 -p "My Project" --confirm
asana task project remove 1234567890123456 -p "My Project" --dry-run
```

#### Flags

- `--confirm` Confirm the destructive action without a prompt
- `--project` Project name
- `--project-gid` Project GID
- `--workspace` Workspace name (for project name resolution)
- `--workspace-gid` Workspace GID

#### Output modes

human, json, plain, jq, template

#### Authentication

This command requires a credential.

#### Changes state

This command changes state and honors `--dry-run`.

#### Destructive

This command performs a destructive action and asks for confirmation before acting. Preview it first with `--dry-run`; outside a dry run it proceeds only after explicit confirmation, and in non-interactive mode (`--json`, `--no-prompt`, or no terminal) it requires a confirmation flag instead of blocking.

#### Metadata

This command can attach request metadata. In advanced-output apps, add `--include-meta` to a supported machine output mode to wrap the result as `{data, meta}`.

#### Related commands

- task project add
- task view

## asana task reopen

Mark task as incomplete

### Usage

```
asana task reopen <gid>
```

### Examples

```
asana task reopen 1234567890123456
asana task reopen 1234567890123456 --dry-run
asana task reopen 1234567890123456
asana task move 1234567890123456 -s "In Progress"
```

### Output modes

human, json, plain, jq, template

### Authentication

This command requires a credential.

### Changes state

This command changes state and honors `--dry-run`.

### Metadata

This command can attach request metadata. In advanced-output apps, add `--include-meta` to a supported machine output mode to wrap the result as `{data, meta}`.

### Related commands

- task done
- task move
- task update

## asana task search

Search tasks in a workspace (premium feature)

### Usage

```
asana task search
```

### Examples

```
asana task search -w "My Workspace" --text "login bug"
asana task search -w "My Workspace" --assignee me --completed false
asana task search -w "My Workspace" --due-before 2024-12-31
asana task search -w "My Workspace" -p "Sprint 1"
asana task search -w "My Workspace" --assignee me --text "urgent" --limit 10
asana task search -w "My Workspace" --assignee me --due-before 2024-01-01
asana task update <gid> --due 2024-02-01
```

### Flags

- `--assignee` Filter by assignee (me, GID, or email)
- `--completed` Filter by completion status (true or false)
- `--due-after` Tasks due after date (YYYY-MM-DD)
- `--due-before` Tasks due before date (YYYY-MM-DD)
- `--limit` Maximum results (capped at 100 by API)
- `--project` Filter by project name
- `--project-gid` Filter by project GID
- `--text` Search text
- `--workspace` Workspace name (required)
- `--workspace-gid` Workspace GID

### Output modes

human, json, plain, jq, template

### Authentication

This command requires a credential.

### Metadata

This command can attach request metadata. In advanced-output apps, add `--include-meta` to a supported machine output mode to wrap the result as `{data, meta}`.

### Related commands

- task list
- task view

## asana task subtask

Manage subtasks

### Usage

```
asana task subtask
```

### Related commands

- task view
- task parent

### asana task subtask create

Create a subtask

#### Usage

```
asana task subtask create <parent-gid> <title>
```

#### Examples

```
asana task subtask create 1234567890123456 "Review code"
asana task subtask create 1234567890123456 "Write tests" --assignee me --due 2024-12-31
asana task subtask create 1234567890123456 "New subtask" --dry-run
asana task subtask create 1234567890123456 "Design" --assignee me
asana task subtask create 1234567890123456 "Implement" --assignee me
asana task subtask create 1234567890123456 "Test" --assignee me
```

#### Flags

- `--assignee` Assignee (me, GID, or email)
- `--due` Due date (YYYY-MM-DD or RFC3339 with timezone)

#### Output modes

human, json, plain, jq, template

#### Authentication

This command requires a credential.

#### Changes state

This command changes state and honors `--dry-run`.

#### Metadata

This command can attach request metadata. In advanced-output apps, add `--include-meta` to a supported machine output mode to wrap the result as `{data, meta}`.

#### Related commands

- task subtask list
- task create

### asana task subtask list

List subtasks of a task

#### Usage

```
asana task subtask list <parent-gid>
```

#### Examples

```
asana task subtask list 1234567890123456
asana task subtask list 1234567890123456 --limit 5
asana task subtask list 1234567890123456 --json
```

#### Flags

- `--limit` Limit number of subtasks in output

#### Output modes

human, json, plain, jq, template

#### Authentication

This command requires a credential.

#### Metadata

This command can attach request metadata. In advanced-output apps, add `--include-meta` to a supported machine output mode to wrap the result as `{data, meta}`.

#### Related commands

- task view
- task subtask create

## asana task tag

Manage task tags

### Usage

```
asana task tag
```

### Related commands

- tag list
- tag tasks

### asana task tag add

Add tag to task

#### Usage

```
asana task tag add <gid>
```

#### Examples

```
asana task tag add 1234567890123456 --tag "urgent" -w "My Workspace"
asana task tag add 1234567890123456 --tag-gid 9876543210987654
asana task tag add 1234567890123456 --tag "urgent" -w "My Workspace"
asana task tag add 1234567890123456 --tag "bug" -w "My Workspace"
```

#### Flags

- `--tag` Tag name
- `--tag-gid` Tag GID
- `--workspace` Workspace name (for tag name resolution)
- `--workspace-gid` Workspace GID

#### Output modes

human, json, plain, jq, template

#### Authentication

This command requires a credential.

#### Changes state

This command changes state and honors `--dry-run`.

#### Metadata

This command can attach request metadata. In advanced-output apps, add `--include-meta` to a supported machine output mode to wrap the result as `{data, meta}`.

#### Related commands

- task tag remove
- tag list
- tag tasks

### asana task tag remove

Remove tag from task

#### Usage

```
asana task tag remove <gid>
```

#### Examples

```
asana task tag remove 1234567890123456 --tag "urgent" -w "My Workspace" --confirm
asana task tag remove 1234567890123456 --tag "urgent" -w "My Workspace" --dry-run
```

#### Flags

- `--confirm` Confirm the destructive action without a prompt
- `--tag` Tag name
- `--tag-gid` Tag GID
- `--workspace` Workspace name (for tag name resolution)
- `--workspace-gid` Workspace GID

#### Output modes

human, json, plain, jq, template

#### Authentication

This command requires a credential.

#### Changes state

This command changes state and honors `--dry-run`.

#### Destructive

This command performs a destructive action and asks for confirmation before acting. Preview it first with `--dry-run`; outside a dry run it proceeds only after explicit confirmation, and in non-interactive mode (`--json`, `--no-prompt`, or no terminal) it requires a confirmation flag instead of blocking.

#### Metadata

This command can attach request metadata. In advanced-output apps, add `--include-meta` to a supported machine output mode to wrap the result as `{data, meta}`.

#### Related commands

- task tag add
- tag list

## asana task update

Update task fields

### Usage

```
asana task update <gid>
```

### Examples

```
asana task update 1234567890123456 --name "Updated title"
asana task update 1234567890123456 --due 2024-12-31 --assignee me
asana task update 1234567890123456 --clear-due
asana task update 1234567890123456 --notes "New description"
asana task update 1234567890123456 --name "New name" --dry-run
asana task update 1234567890123456 --assignee me --due 2024-12-31
asana task comment 1234567890123456 "Taking over this task"
```

### Flags

- `--assignee` Assignee (me, GID, or email)
- `--clear-assignee` Clear assignee
- `--clear-due` Clear due date
- `--clear-notes` Clear notes
- `--due` Due date (YYYY-MM-DD or RFC3339 with timezone)
- `--name` New task name
- `--notes` New task notes

### Output modes

human, json, plain, jq, template

### Authentication

This command requires a credential.

### Changes state

This command changes state and honors `--dry-run`.

### Metadata

This command can attach request metadata. In advanced-output apps, add `--include-meta` to a supported machine output mode to wrap the result as `{data, meta}`.

### Related commands

- task view
- task move
- task done

## asana task view

View task details

### Usage

```
asana task view <gid>
```

### Examples

```
asana task view 1234567890123456
asana task view 1234567890123456 --comments-limit 20
asana task view 1234567890123456 --json
```

### Flags

- `--comments-limit` Limit number of comments to show

### Output modes

human, json, plain, jq, template

### Authentication

This command requires a credential.

### Metadata

This command can attach request metadata. In advanced-output apps, add `--include-meta` to a supported machine output mode to wrap the result as `{data, meta}`.

### Related commands

- task list
- task update
- task comment
