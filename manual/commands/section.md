# asana section

Manage sections

## Usage

```
asana section
```

## Related commands

- project
- task list
- task move

## asana section create

Create a new section

### Usage

```
asana section create <name>
```

### Examples

```
asana section create "In Progress" -p "My Project"
asana section create "Review" -p "My Project" --insert-after "In Progress"
asana section create "Test" -p "My Project" --dry-run
asana section create "Backlog" -p "My Project"
asana section create "In Progress" -p "My Project"
asana section create "Review" -p "My Project"
asana section create "Done" -p "My Project"
```

### Flags

- `--insert-after` Insert after this section (name or GID)
- `--insert-before` Insert before this section (name or GID)
- `--project` Project name
- `--project-gid` Project GID
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

- section list
- section delete
- task move

## asana section delete

Delete a section

### Usage

```
asana section delete <section-gid>
```

### Examples

```
asana section delete 1234567890123456 --confirm
asana section delete 1234567890123456 --dry-run
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

- section list
- section create

## asana section list

List sections in a project

### Usage

```
asana section list
```

### Examples

```
asana section list -p "My Project"
asana section list -p "My Project" --limit 5
asana section list -p "My Project" --json
asana section list -p "My Project"
asana task list -p "My Project" -s "In Progress"
```

### Flags

- `--limit` Limit number of sections in output
- `--project` Project name
- `--project-gid` Project GID
- `--workspace` Workspace name (scopes project lookup)
- `--workspace-gid` Workspace GID (scopes project lookup)

### Output modes

human, json, plain, jq, template

### Authentication

This command requires a credential.

### Metadata

This command can attach request metadata. In advanced-output apps, add `--include-meta` to a supported machine output mode to wrap the result as `{data, meta}`.

### Related commands

- section view
- section create
- task list

## asana section move

Move a section within a project

### Usage

```
asana section move <section-gid>
```

### Examples

```
asana section move 1234567890123456 -p "My Project" --before "Done"
asana section move 1234567890123456 -p "My Project" --after "To Do"
asana section move 1234567890123456 -p "My Project" --before "Done" --dry-run
```

### Flags

- `--after` Move after this section (name or GID)
- `--before` Move before this section (name or GID)
- `--project` Target project name
- `--project-gid` Target project GID
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

- section list
- task move

## asana section update

Update a section

### Usage

```
asana section update <section-gid>
```

### Examples

```
asana section update 1234567890123456 --name "Done"
asana section update 1234567890123456 --name "Done" --dry-run
```

### Flags

- `--name` New section name

### Output modes

human, json, plain, jq, template

### Authentication

This command requires a credential.

### Changes state

This command changes state and honors `--dry-run`.

### Metadata

This command can attach request metadata. In advanced-output apps, add `--include-meta` to a supported machine output mode to wrap the result as `{data, meta}`.

### Related commands

- section view
- section list

## asana section view

View section details

### Usage

```
asana section view <section-gid>
```

### Examples

```
asana section view 1234567890123456
asana section view 1234567890123456 --json
```

### Output modes

human, json, plain, jq, template

### Authentication

This command requires a credential.

### Metadata

This command can attach request metadata. In advanced-output apps, add `--include-meta` to a supported machine output mode to wrap the result as `{data, meta}`.

### Related commands

- section list
- task list
