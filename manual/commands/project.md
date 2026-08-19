# asana project

Manage projects

## Usage

```
asana project
```

## Related commands

- task
- section
- workspace

## asana project create

Create a new project

### Usage

```
asana project create <name>
```

### Examples

```
asana project create "My Project" -w "My Workspace"
asana project create "Q1 Goals" -w "Company" -t "Engineering" --color dark-blue
asana project create "Sprint 1" -w "Company" -t "Dev" --due-on 2024-12-31 --privacy private_to_team
asana project create "Test" -w "My Workspace" --dry-run
asana project create "New Sprint" -w "My Workspace"
asana section create "To Do" --project-gid <new-gid>
asana section create "In Progress" --project-gid <new-gid>
asana section create "Done" --project-gid <new-gid>
```

### Flags

- `--color` Project color
- `--due-on` Project due date (YYYY-MM-DD)
- `--privacy` Privacy setting: public_to_workspace, private_to_team, or private
- `--team` Team name (required for organizations)
- `--team-gid` Team GID
- `--workspace` Workspace name
- `--workspace-gid` Workspace GID

### Output modes

human, json, plain, jq, template

### Authentication

This command requires a credential.

### Changes state

This command changes state and honors `--dry-run`.

### Metadata

This command can attach request metadata. In advanced-output apps, add `--include-meta` to a supported machine output mode to wrap the result as `{data, meta}`.

### Related commands

- project list
- project view
- section create

## asana project delete

Delete a project

### Usage

```
asana project delete <project-gid>
```

### Examples

```
asana project delete 1234567890123456 --confirm
asana project delete 1234567890123456 --dry-run
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

- project update
- project list

## asana project duplicate

Duplicate a project

### Usage

```
asana project duplicate <project-gid>
```

### Examples

```
asana project duplicate 1234567890123456 --name "Q2 Sprint"
asana project duplicate 1234567890123456 --name "Copy" --include task_notes,task_assignee,task_dates
asana project duplicate 1234567890123456 --name "New Project" -t "Engineering"
asana project duplicate 1234567890123456 --name "Q2" --include task_dates --schedule-start-on 2024-04-01
asana project duplicate 1234567890123456 --name "Copy" --wait
asana project duplicate 1234567890123456 --name "Test" --dry-run
```

### Flags

- `--include` Comma-separated fields to include: members,notes,task_notes,task_assignee,task_subtasks,task_attachments,task_dates,task_dependencies,task_followers,task_tags,task_projects
- `--name` Name for the duplicated project (required)
- `--schedule-due-on` Shift dates based on new due date (YYYY-MM-DD)
- `--schedule-skip-weekends` Skip weekends when shifting dates
- `--schedule-start-on` Shift dates based on new start date (YYYY-MM-DD)
- `--team` Team name for the new project
- `--team-gid` Team GID for the new project
- `--wait` Wait for duplication to complete

### Output modes

human, json, plain, jq, template

### Authentication

This command requires a credential.

### Changes state

This command changes state and honors `--dry-run`.

### Metadata

This command can attach request metadata. In advanced-output apps, add `--include-meta` to a supported machine output mode to wrap the result as `{data, meta}`.

### Related commands

- project create
- project view

## asana project list

List projects in a workspace

### Usage

```
asana project list
```

### Examples

```
asana project list -w "My Workspace"
asana project list -w "My Workspace" -t "Engineering"
asana project list -w "My Workspace" --limit 10
asana project list -w "My Workspace" --json
asana project list -w "My Workspace"
asana task list -p "My Project"
```

### Flags

- `--limit` Limit number of projects in output
- `--team` Team name
- `--team-gid` Team GID
- `--workspace` Workspace name
- `--workspace-gid` Workspace GID

### Output modes

human, json, plain, jq, template

### Authentication

This command requires a credential.

### Metadata

This command can attach request metadata. In advanced-output apps, add `--include-meta` to a supported machine output mode to wrap the result as `{data, meta}`.

### Related commands

- project view
- project create
- task list

## asana project member

Manage project members

### Usage

```
asana project member
```

### Related commands

- project view
- user list

### asana project member add

Add members to a project

#### Usage

```
asana project member add <project-gid>
```

#### Examples

```
asana project member add 1234567890123456 --user 9876543210987654
asana project member add 1234567890123456 --user jane@example.com
asana project member add 1234567890123456 --user 111 --user 222
asana project member add 1234567890123456 --user 9876543210987654 --dry-run
```

#### Flags

- `--user` User(s) to add (me, GID, or email)

#### Output modes

human, json, plain, jq, template

#### Authentication

This command requires a credential.

#### Changes state

This command changes state and honors `--dry-run`.

#### Metadata

This command can attach request metadata. In advanced-output apps, add `--include-meta` to a supported machine output mode to wrap the result as `{data, meta}`.

#### Related commands

- project member list
- project member remove

### asana project member list

List project members

#### Usage

```
asana project member list <project-gid>
```

#### Examples

```
asana project member list 1234567890123456
asana project member list 1234567890123456 --limit 10
asana project member list 1234567890123456 --json
```

#### Flags

- `--limit` Limit number of members in output

#### Output modes

human, json, plain, jq, template

#### Authentication

This command requires a credential.

#### Metadata

This command can attach request metadata. In advanced-output apps, add `--include-meta` to a supported machine output mode to wrap the result as `{data, meta}`.

#### Related commands

- project member add
- user list

### asana project member remove

Remove members from a project

#### Usage

```
asana project member remove <project-gid>
```

#### Examples

```
asana project member remove 1234567890123456 --user 9876543210987654 --confirm
asana project member remove 1234567890123456 --user jane@example.com --confirm
asana project member remove 1234567890123456 --user 9876543210987654 --dry-run
```

#### Flags

- `--confirm` Confirm the destructive action without a prompt
- `--user` User(s) to remove (me, GID, or email)

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

- project member list
- project member add

## asana project update

Update a project

### Usage

```
asana project update <project-gid>
```

### Examples

```
asana project update 1234567890123456 --name "New Name"
asana project update 1234567890123456 --archived true
asana project update 1234567890123456 --color light-green --due-on 2025-01-15
asana project update 1234567890123456 --name "Test" --dry-run
```

### Flags

- `--archived` Archive status: true or false
- `--color` Project color
- `--due-on` Project due date (YYYY-MM-DD)
- `--name` New project name
- `--notes` Project notes/description

### Output modes

human, json, plain, jq, template

### Authentication

This command requires a credential.

### Changes state

This command changes state and honors `--dry-run`.

### Metadata

This command can attach request metadata. In advanced-output apps, add `--include-meta` to a supported machine output mode to wrap the result as `{data, meta}`.

### Related commands

- project view
- project delete

## asana project view

View project details

### Usage

```
asana project view <project-gid>
```

### Examples

```
asana project view 1234567890123456
asana project view 1234567890123456 --json
asana project view 1234567890123456
asana section list --project-gid 1234567890123456
asana task list --project-gid 1234567890123456
```

### Output modes

human, json, plain, jq, template

### Authentication

This command requires a credential.

### Metadata

This command can attach request metadata. In advanced-output apps, add `--include-meta` to a supported machine output mode to wrap the result as `{data, meta}`.

### Related commands

- project list
- section list
- task list
