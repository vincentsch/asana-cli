# asana team

Manage teams

## Usage

```
asana team
```

## Related commands

- workspace list
- project list
- user list

## asana team create

Create a new team

### Usage

```
asana team create <name>
```

### Examples

```
asana team create "Engineering" -w "My Organization"
asana team create "Design" -w "My Organization" --visibility public --description "Design team"
asana team create "Test Team" -w "My Organization" --dry-run
```

### Flags

- `--description` Team description
- `--visibility` Team visibility (secret, request_to_join, public)
- `--workspace` Organization name
- `--workspace-gid` Organization GID

### Output modes

human, json, plain, jq, template

### Authentication

This command requires a credential.

### Changes state

This command changes state and honors `--dry-run`.

### Metadata

This command can attach request metadata. In advanced-output apps, add `--include-meta` to a supported machine output mode to wrap the result as `{data, meta}`.

### Related commands

- team list
- team member add

## asana team list

List teams in an organization

### Usage

```
asana team list
```

### Examples

```
asana team list -w "My Organization"
asana team list -w "My Organization" --limit 10
asana team list -w "My Organization" --json
```

### Flags

- `--limit` Limit number of teams in output
- `--workspace` Organization/workspace name
- `--workspace-gid` Organization/workspace GID

### Output modes

human, json, plain, jq, template

### Authentication

This command requires a credential.

### Metadata

This command can attach request metadata. In advanced-output apps, add `--include-meta` to a supported machine output mode to wrap the result as `{data, meta}`.

### Related commands

- team view
- team create
- project list

## asana team member

Manage team members

### Usage

```
asana team member
```

### Related commands

- team view
- user list

### asana team member add

Add member to team

#### Usage

```
asana team member add <team-gid>
```

#### Examples

```
asana team member add 1234567890123456 --user 9876543210987654
asana team member add 1234567890123456 --user jane@example.com -w "My Organization"
asana team member add 1234567890123456 --user 111 --user 222
asana team member add 1234567890123456 --user 9876543210987654 --dry-run
```

#### Flags

- `--user` User(s) to add (GID or email)
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

- team member list
- team member remove

### asana team member list

List team members

#### Usage

```
asana team member list <team-gid>
```

#### Examples

```
asana team member list 1234567890123456
asana team member list 1234567890123456 --limit 20
asana team member list 1234567890123456 --json
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

- team member add
- team view

### asana team member remove

Remove member from team

#### Usage

```
asana team member remove <team-gid>
```

#### Examples

```
asana team member remove 1234567890123456 --user 9876543210987654 --confirm
asana team member remove 1234567890123456 --user jane@example.com -w "My Organization" --confirm
asana team member remove 1234567890123456 --user 9876543210987654 --dry-run
```

#### Flags

- `--confirm` Confirm the destructive action without a prompt
- `--user` User(s) to remove (GID or email)
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

- team member list
- team member add

## asana team view

View team details

### Usage

```
asana team view <team-gid>
```

### Examples

```
asana team view 1234567890123456
asana team view 1234567890123456 --json
```

### Output modes

human, json, plain, jq, template

### Authentication

This command requires a credential.

### Metadata

This command can attach request metadata. In advanced-output apps, add `--include-meta` to a supported machine output mode to wrap the result as `{data, meta}`.

### Related commands

- team list
- team member list
