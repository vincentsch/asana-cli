# asana goal

Manage goals

## Usage

```
asana goal
```

## Related commands

- portfolio
- project

## asana goal create

Create a goal

### Usage

```
asana goal create <name>
```

### Examples

```
asana goal create "Increase revenue 20%" -w "My Workspace"
asana goal create "Launch v2.0" -w "My Workspace" --owner me --due-on 2024-12-31
asana goal create "Improve NPS" -w "My Workspace" --team "Customer Success"
asana goal create "Test Goal" -w "My Workspace" --dry-run
```

### Flags

- `--due-on` Due date (YYYY-MM-DD)
- `--notes` Goal notes
- `--owner` Owner (me, GID, or email)
- `--start-on` Start date (YYYY-MM-DD)
- `--team` Team name
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

- goal list
- goal update
- goal metric set

## asana goal delete

Delete a goal

### Usage

```
asana goal delete <goal-gid>
```

### Examples

```
asana goal delete 1234567890123456 --confirm
asana goal delete 1234567890123456 --dry-run
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

- goal list

## asana goal list

List goals

### Usage

```
asana goal list
```

### Examples

```
asana goal list -w "My Workspace"
asana goal list -w "My Workspace" --team "Engineering"
asana goal list -w "My Workspace" --limit 10
asana goal list -w "My Workspace" --json
```

### Flags

- `--limit` Limit number of goals in output
- `--team` Team name
- `--team-gid` Team GID
- `--time-period` Time period GID
- `--workspace` Workspace name
- `--workspace-gid` Workspace GID

### Output modes

human, json, plain, jq, template

### Authentication

This command requires a credential.

### Metadata

This command can attach request metadata. In advanced-output apps, add `--include-meta` to a supported machine output mode to wrap the result as `{data, meta}`.

### Related commands

- goal view
- goal create

## asana goal metric

Manage goal metrics

### Usage

```
asana goal metric
```

### Related commands

- goal view
- goal update

### asana goal metric set

Set or update goal metric

#### Usage

```
asana goal metric set <goal-gid>
```

#### Examples

```
asana goal metric set 1234567890123456 --current-value 75
asana goal metric set 1234567890123456 --current-value 0 --target-value 100 --unit percentage
asana goal metric set 1234567890123456 --current-value 0 --target-value 1000000 --unit currency --precision 2
asana goal metric set 1234567890123456 --current-value 50 --dry-run
```

#### Flags

- `--current-value` Current metric value (required)
- `--initial-value` Initial metric value
- `--precision` Decimal precision
- `--target-value` Target metric value (creates/replaces metric)
- `--unit` Metric unit (none, currency, percentage)

#### Output modes

human, json, plain, jq, template

#### Authentication

This command requires a credential.

#### Changes state

This command changes state and honors `--dry-run`.

#### Metadata

This command can attach request metadata. In advanced-output apps, add `--include-meta` to a supported machine output mode to wrap the result as `{data, meta}`.

#### Related commands

- goal view

## asana goal update

Update a goal

### Usage

```
asana goal update <goal-gid>
```

### Examples

```
asana goal update 1234567890123456 --status on_track
asana goal update 1234567890123456 --name "New Goal Name" --notes "Updated description"
asana goal update 1234567890123456 --due-on 2025-01-31
asana goal update 1234567890123456 --status at_risk --dry-run
```

### Flags

- `--due-on` Due date (YYYY-MM-DD)
- `--name` New goal name
- `--notes` Goal notes
- `--start-on` Start date (YYYY-MM-DD)
- `--status` Goal status (green, yellow, red, on_track, at_risk, off_track)

### Output modes

human, json, plain, jq, template

### Authentication

This command requires a credential.

### Changes state

This command changes state and honors `--dry-run`.

### Metadata

This command can attach request metadata. In advanced-output apps, add `--include-meta` to a supported machine output mode to wrap the result as `{data, meta}`.

### Related commands

- goal view
- goal metric set

## asana goal view

View goal details

### Usage

```
asana goal view <goal-gid>
```

### Examples

```
asana goal view 1234567890123456
asana goal view 1234567890123456 --json
```

### Output modes

human, json, plain, jq, template

### Authentication

This command requires a credential.

### Metadata

This command can attach request metadata. In advanced-output apps, add `--include-meta` to a supported machine output mode to wrap the result as `{data, meta}`.

### Related commands

- goal list
- goal metric set
