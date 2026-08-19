# asana attachment

Manage task attachments

## Usage

```
asana attachment
```

## Related commands

- task view

## asana attachment delete

Delete an attachment

### Usage

```
asana attachment delete <attachment-gid>
```

### Examples

```
asana attachment delete 1234567890123456 --confirm
asana attachment delete 1234567890123456 --dry-run
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

- attachment list
- attachment view

## asana attachment list

List attachments on a task

### Usage

```
asana attachment list
```

### Examples

```
asana attachment list --task 1234567890123456
asana attachment list --task 1234567890123456 --limit 10
asana attachment list --task 1234567890123456 --json
```

### Flags

- `--limit` Limit number of attachments in output
- `--task` Task GID
- `--task-gid` Task GID

### Output modes

human, json, plain, jq, template

### Authentication

This command requires a credential.

### Metadata

This command can attach request metadata. In advanced-output apps, add `--include-meta` to a supported machine output mode to wrap the result as `{data, meta}`.

### Related commands

- attachment view
- attachment upload

## asana attachment upload

Upload file or URL attachment to task

### Usage

```
asana attachment upload [file]
```

### Examples

```
asana attachment upload /path/to/file.pdf --task 1234567890123456
asana attachment upload /path/to/file.pdf --task 1234567890123456 --name "Report Q4"
asana attachment upload --task 1234567890123456 --url "https://example.com/doc" --name "Design Doc"
asana attachment upload /path/to/file.pdf --task 1234567890123456 --dry-run
```

### Flags

- `--connect-to-app` Connect URL attachment to app
- `--name` Display name (required for URL, optional for file)
- `--task` Task GID
- `--task-gid` Task GID
- `--url` External URL to attach

### Output modes

human, json, plain, jq, template

### Authentication

This command requires a credential.

### Changes state

This command changes state and honors `--dry-run`.

### Metadata

This command can attach request metadata. In advanced-output apps, add `--include-meta` to a supported machine output mode to wrap the result as `{data, meta}`.

### Related commands

- attachment list
- task view

## asana attachment view

View attachment details

### Usage

```
asana attachment view <attachment-gid>
```

### Examples

```
asana attachment view 1234567890123456
asana attachment view 1234567890123456 --json
```

### Output modes

human, json, plain, jq, template

### Authentication

This command requires a credential.

### Metadata

This command can attach request metadata. In advanced-output apps, add `--include-meta` to a supported machine output mode to wrap the result as `{data, meta}`.

### Related commands

- attachment list
- attachment delete
