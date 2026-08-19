# asana tag

Manage tags

## Usage

```
asana tag
```

## Related commands

- task tag add
- task tag remove

## asana tag create

Create a new tag

### Usage

```
asana tag create <name>
```

### Examples

```
asana tag create "urgent" -w "My Workspace"
asana tag create "blocked" -w "My Workspace" --color dark-red
asana tag create "needs-review" -w "My Workspace" --notes "Requires code review"
asana tag create "test" -w "My Workspace" --dry-run
```

### Flags

- `--color` Tag color
- `--notes` Tag notes/description
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

- tag list
- task tag add

## asana tag delete

Delete a tag

### Usage

```
asana tag delete <tag-gid>
```

### Examples

```
asana tag delete 1234567890123456 --confirm
asana tag delete 1234567890123456 --dry-run
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

## asana tag list

List tags in a workspace

### Usage

```
asana tag list
```

### Examples

```
asana tag list -w "My Workspace"
asana tag list -w "My Workspace" --limit 20
asana tag list -w "My Workspace" --json
```

### Flags

- `--limit` Limit number of tags in output
- `--workspace` Workspace name
- `--workspace-gid` Workspace GID

### Output modes

human, json, plain, jq, template

### Authentication

This command requires a credential.

### Metadata

This command can attach request metadata. In advanced-output apps, add `--include-meta` to a supported machine output mode to wrap the result as `{data, meta}`.

### Related commands

- tag view
- tag create
- tag tasks

## asana tag tasks

List tasks with this tag

### Usage

```
asana tag tasks <tag-gid>
```

### Examples

```
asana tag tasks 1234567890123456
asana tag tasks 1234567890123456 --limit 10
asana tag tasks 1234567890123456 --json
```

### Flags

- `--limit` Limit number of tasks in output

### Output modes

human, json, plain, jq, template

### Authentication

This command requires a credential.

### Metadata

This command can attach request metadata. In advanced-output apps, add `--include-meta` to a supported machine output mode to wrap the result as `{data, meta}`.

### Related commands

- tag list
- task tag add
- task list

## asana tag update

Update a tag

### Usage

```
asana tag update <tag-gid>
```

### Examples

```
asana tag update 1234567890123456 --name "critical"
asana tag update 1234567890123456 --color dark-orange
asana tag update 1234567890123456 --name "new-name" --dry-run
```

### Flags

- `--color` Tag color
- `--name` New tag name
- `--notes` Tag notes/description

### Output modes

human, json, plain, jq, template

### Authentication

This command requires a credential.

### Changes state

This command changes state and honors `--dry-run`.

### Metadata

This command can attach request metadata. In advanced-output apps, add `--include-meta` to a supported machine output mode to wrap the result as `{data, meta}`.

## asana tag view

View tag details

### Usage

```
asana tag view <tag-gid>
```

### Examples

```
asana tag view 1234567890123456
asana tag view 1234567890123456 --json
```

### Output modes

human, json, plain, jq, template

### Authentication

This command requires a credential.

### Metadata

This command can attach request metadata. In advanced-output apps, add `--include-meta` to a supported machine output mode to wrap the result as `{data, meta}`.

### Related commands

- tag list
- tag tasks
