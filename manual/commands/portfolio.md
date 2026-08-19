# asana portfolio

Manage portfolios

## Usage

```
asana portfolio
```

## Related commands

- project
- goal

## asana portfolio create

Create a portfolio

### Usage

```
asana portfolio create <name>
```

### Examples

```
asana portfolio create "Q1 Projects" -w "My Workspace"
asana portfolio create "Engineering" -w "My Workspace" --color dark-blue
asana portfolio create "Test" -w "My Workspace" --dry-run
```

### Flags

- `--color` Portfolio color
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

- portfolio list
- portfolio project add

## asana portfolio delete

Delete a portfolio

### Usage

```
asana portfolio delete <portfolio-gid>
```

### Examples

```
asana portfolio delete 1234567890123456 --confirm
asana portfolio delete 1234567890123456 --dry-run
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

- portfolio list

## asana portfolio list

List portfolios

### Usage

```
asana portfolio list
```

### Examples

```
asana portfolio list -w "My Workspace"
asana portfolio list -w "My Workspace" --owner all
asana portfolio list -w "My Workspace" --json
```

### Flags

- `--limit` Limit number of portfolios in output
- `--owner` Portfolio owner (me, GID, or email)
- `--workspace` Workspace name
- `--workspace-gid` Workspace GID

### Output modes

human, json, plain, jq, template

### Authentication

This command requires a credential.

### Metadata

This command can attach request metadata. In advanced-output apps, add `--include-meta` to a supported machine output mode to wrap the result as `{data, meta}`.

### Related commands

- portfolio view
- portfolio create

## asana portfolio project

Manage portfolio projects

### Usage

```
asana portfolio project
```

### Related commands

- portfolio view
- project list

### asana portfolio project add

Add project to portfolio

#### Usage

```
asana portfolio project add <portfolio-gid>
```

#### Examples

```
asana portfolio project add 1234567890123456 --project-gid 9876543210987654
asana portfolio project add 1234567890123456 --project "Q1 Sprint" -w "My Workspace"
asana portfolio project add 1234567890123456 --project-gid 9876543210987654 --dry-run
```

#### Flags

- `--project` Project name
- `--project-gid` Project GID
- `--workspace` Workspace name (for project resolution)
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

- portfolio project list
- portfolio project remove

### asana portfolio project list

List projects in portfolio

#### Usage

```
asana portfolio project list <portfolio-gid>
```

#### Examples

```
asana portfolio project list 1234567890123456
asana portfolio project list 1234567890123456 --limit 10
asana portfolio project list 1234567890123456 --json
```

#### Flags

- `--limit` Limit number of projects in output

#### Output modes

human, json, plain, jq, template

#### Authentication

This command requires a credential.

#### Metadata

This command can attach request metadata. In advanced-output apps, add `--include-meta` to a supported machine output mode to wrap the result as `{data, meta}`.

#### Related commands

- portfolio project add
- portfolio view

### asana portfolio project remove

Remove project from portfolio

#### Usage

```
asana portfolio project remove <portfolio-gid>
```

#### Examples

```
asana portfolio project remove 1234567890123456 --project-gid 9876543210987654 --confirm
asana portfolio project remove 1234567890123456 --project-gid 9876543210987654 --dry-run
```

#### Flags

- `--confirm` Confirm the destructive action without a prompt
- `--project` Project name
- `--project-gid` Project GID
- `--workspace` Workspace name (for project resolution)
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

- portfolio project list
- portfolio project add

## asana portfolio view

View portfolio details

### Usage

```
asana portfolio view <portfolio-gid>
```

### Examples

```
asana portfolio view 1234567890123456
asana portfolio view 1234567890123456 --json
```

### Output modes

human, json, plain, jq, template

### Authentication

This command requires a credential.

### Metadata

This command can attach request metadata. In advanced-output apps, add `--include-meta` to a supported machine output mode to wrap the result as `{data, meta}`.

### Related commands

- portfolio list
- portfolio project list
