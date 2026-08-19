# asana workspace

Manage workspaces

## Usage

```
asana workspace
```

## Related commands

- project
- team
- config set

## asana workspace list

List workspaces

### Usage

```
asana workspace list
```

### Examples

```
asana workspace list
asana workspace list --json
asana workspace list
asana config set workspace "My Workspace"
```

### Output modes

human, json, plain, jq, template

### Authentication

This command requires a credential.

### Metadata

This command can attach request metadata. In advanced-output apps, add `--include-meta` to a supported machine output mode to wrap the result as `{data, meta}`.

### Related commands

- project list
- team list
- config set

## asana workspace user

Manage workspace users

### Usage

```
asana workspace user
```

### Related commands

- user list
- team member

### asana workspace user add

Add a user to a workspace

#### Usage

```
asana workspace user add
```

#### Examples

```
asana workspace user add -w "My Workspace" --user jane@example.com
asana workspace user add -w "My Workspace" --user jane@example.com --dry-run
```

#### Flags

- `--user` User email or GID
- `--workspace` Workspace name
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

- workspace user list
- workspace user remove

### asana workspace user list

List users in a workspace

#### Usage

```
asana workspace user list
```

#### Examples

```
asana workspace user list -w "My Workspace"
asana workspace user list -w "My Workspace" --limit 20
asana workspace user list -w "My Workspace" --json
```

#### Flags

- `--limit` Limit number of users in output
- `--workspace` Workspace name
- `--workspace-gid` Workspace GID

#### Output modes

human, json, plain, jq, template

#### Authentication

This command requires a credential.

#### Metadata

This command can attach request metadata. In advanced-output apps, add `--include-meta` to a supported machine output mode to wrap the result as `{data, meta}`.

#### Related commands

- workspace user add
- user list

### asana workspace user remove

Remove a user from a workspace

#### Usage

```
asana workspace user remove
```

#### Examples

```
asana workspace user remove -w "My Workspace" --user jane@example.com --confirm
asana workspace user remove -w "My Workspace" --user jane@example.com --dry-run
```

#### Flags

- `--confirm` Confirm the destructive action without a prompt
- `--user` User email or GID
- `--workspace` Workspace name
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

- workspace user list
- workspace user add

## asana workspace view

View workspace details

### Usage

```
asana workspace view <workspace-gid>
```

### Examples

```
asana workspace view 1234567890123456
asana workspace view 1234567890123456 --json
```

### Output modes

human, json, plain, jq, template

### Authentication

This command requires a credential.

### Metadata

This command can attach request metadata. In advanced-output apps, add `--include-meta` to a supported machine output mode to wrap the result as `{data, meta}`.

### Related commands

- workspace list
- workspace user list
