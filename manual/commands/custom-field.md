# asana custom-field

Manage custom fields

## Usage

```
asana custom-field
```

## Related commands

- task view
- task update

## asana custom-field create

Create a custom field

### Usage

```
asana custom-field create <name>
```

### Examples

```
asana custom-field create "Notes" -w "My Workspace" --type text
asana custom-field create "Priority" -w "My Workspace" --type enum --enum-options "Low,Medium,High"
asana custom-field create "Story Points" -w "My Workspace" --type number --precision 0
asana custom-field create "Test" -w "My Workspace" --type text --dry-run
```

### Flags

- `--description` Field description
- `--enum-options` Enum option names (required for enum/multi_enum)
- `--precision` Decimal precision for number type
- `--type` Field type (text, enum, multi_enum, number, date, people, reference)
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

- custom-field list
- custom-field view

## asana custom-field delete

Delete a custom field

### Usage

```
asana custom-field delete <custom-field-gid>
```

### Examples

```
asana custom-field delete 1234567890123456 --confirm
asana custom-field delete 1234567890123456 --dry-run
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

- custom-field list

## asana custom-field list

List custom fields in a workspace

### Usage

```
asana custom-field list
```

### Examples

```
asana custom-field list -w "My Workspace"
asana custom-field list -w "My Workspace" --limit 10
asana custom-field list -w "My Workspace" --json
```

### Flags

- `--limit` Limit number of custom fields in output
- `--workspace` Workspace name
- `--workspace-gid` Workspace GID

### Output modes

human, json, plain, jq, template

### Authentication

This command requires a credential.

### Metadata

This command can attach request metadata. In advanced-output apps, add `--include-meta` to a supported machine output mode to wrap the result as `{data, meta}`.

### Related commands

- custom-field view
- custom-field create

## asana custom-field update

Update a custom field

### Usage

```
asana custom-field update <custom-field-gid>
```

### Examples

```
asana custom-field update 1234567890123456 --name "New Name"
asana custom-field update 1234567890123456 --enabled false
asana custom-field update 1234567890123456 --name "Test" --dry-run
```

### Flags

- `--description` Field description
- `--enabled` Enable or disable field (true/false)
- `--name` New field name

### Output modes

human, json, plain, jq, template

### Authentication

This command requires a credential.

### Changes state

This command changes state and honors `--dry-run`.

### Metadata

This command can attach request metadata. In advanced-output apps, add `--include-meta` to a supported machine output mode to wrap the result as `{data, meta}`.

### Related commands

- custom-field view

## asana custom-field view

View custom field details

### Usage

```
asana custom-field view <custom-field-gid>
```

### Examples

```
asana custom-field view 1234567890123456
asana custom-field view 1234567890123456 --json
```

### Output modes

human, json, plain, jq, template

### Authentication

This command requires a credential.

### Metadata

This command can attach request metadata. In advanced-output apps, add `--include-meta` to a supported machine output mode to wrap the result as `{data, meta}`.

### Related commands

- custom-field list
- custom-field update
