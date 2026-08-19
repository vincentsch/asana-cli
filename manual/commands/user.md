# asana user

Manage users

## Usage

```
asana user
```

## Related commands

- task follower add
- team member list

## asana user list

List users in a workspace

### Usage

```
asana user list
```

### Examples

```
asana user list -w "My Workspace"
asana user list -w "My Workspace" --limit 50
asana user list -w "My Workspace" --json
```

### Flags

- `--limit` Limit number of users in output
- `--workspace` Workspace name
- `--workspace-gid` Workspace GID

### Output modes

human, json, plain, jq, template

### Authentication

This command requires a credential.

### Metadata

This command can attach request metadata. In advanced-output apps, add `--include-meta` to a supported machine output mode to wrap the result as `{data, meta}`.

### Related commands

- user view
- task follower add

## asana user me

Show current authenticated user

### Usage

```
asana user me
```

### Examples

```
asana user me
asana user me --json
```

### Output modes

human, json, plain, jq, template

### Authentication

This command requires a credential.

### Metadata

This command can attach request metadata. In advanced-output apps, add `--include-meta` to a supported machine output mode to wrap the result as `{data, meta}`.

### Related commands

- auth login
- config set

## asana user view

View user details

### Usage

```
asana user view <user-gid>
```

### Examples

```
asana user view 1234567890123456
asana user view me
asana user view 1234567890123456 --json
```

### Output modes

human, json, plain, jq, template

### Authentication

This command requires a credential.

### Metadata

This command can attach request metadata. In advanced-output apps, add `--include-meta` to a supported machine output mode to wrap the result as `{data, meta}`.
