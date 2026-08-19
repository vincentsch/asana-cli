# asana config

Manage configuration

## Usage

```
asana config
```

## Related commands

- auth login
- workspace list

## asana config get

Get a config value

### Usage

```
asana config get <key>
```

### Examples

```
asana config get workspace
asana config get project
asana config get token
```

### Output modes

human, json, plain, jq, template

### Authentication

This command requires a credential.

### Metadata

This command can attach request metadata. In advanced-output apps, add `--include-meta` to a supported machine output mode to wrap the result as `{data, meta}`.

### Related commands

- config set
- config list

## asana config list

List config values

### Usage

```
asana config list
```

### Examples

```
asana config list
asana config list --json
```

### Output modes

human, json, plain, jq, template

### Authentication

This command requires a credential.

### Metadata

This command can attach request metadata. In advanced-output apps, add `--include-meta` to a supported machine output mode to wrap the result as `{data, meta}`.

### Related commands

- config set
- config get

## asana config set

Set a config value

### Usage

```
asana config set <key> <value>
```

### Examples

```
asana config set workspace "My Workspace"
asana config set project "My Project" -w "My Workspace"
asana config set token YOUR_TOKEN
```

### Flags

- `--workspace` Workspace name (scopes project lookup)
- `--workspace-gid` Workspace GID (scopes project lookup)

### Output modes

human, json, plain, jq, template

### Changes state

This command changes state and honors `--dry-run`.

### Metadata

This command can attach request metadata. In advanced-output apps, add `--include-meta` to a supported machine output mode to wrap the result as `{data, meta}`.

### Related commands

- config get
- config list
- workspace list
