# Asana CLI Manual

Reference documentation for the Asana CLI.

## Contents

- [Installation](installation.md) - All installation options
- [Development](development.md) - Building from source and contributing
- [Configuration](configuration.md) - Authentication and settings

## Command Reference

- [Complete command index](commands/index.md) - Every command path and global flag
- [auth](commands/auth.md) / [login](commands/login.md) - Store an Asana personal access token
- [config](commands/config.md) - Read and update CLI configuration
- [workspace](commands/workspace.md) - List and view workspaces
- [project](commands/project.md) - Manage projects, sections, and members
- [task](commands/task.md) - Create, update, and organize tasks
- [section](commands/section.md) - Manage project sections
- [tag](commands/tag.md) - Create and manage tags
- [user](commands/user.md) - List users and view profiles
- [team](commands/team.md) - Manage teams and members (organizations only)
- [attachment](commands/attachment.md) - Upload and manage attachments
- [custom-field](commands/custom-field.md) - Manage custom fields (premium)
- [goal](commands/goal.md) - Track goals and metrics (premium)
- [portfolio](commands/portfolio.md) - Manage portfolios
- [completion](commands/completion.md) - Generate shell completion scripts
- [update](commands/update.md) - Update the installed CLI
- [version](commands/version.md) - Print build and version information

## Global Flags

These flags work with all commands:

| Flag | Description |
|------|-------------|
| `--help`, `-h` | Show help for any command |
| `--json` | Output JSON instead of text tables |
| `--config` | Path to config file (Linux default: `~/.config/asana/config.json`) |
| `--no-prompt` | Disable interactive prompts |
| `--dry-run` | Preview supported changes without performing them |
| `--plain` | Print copy-safe text where supported |
| `--jq`, `--template` | Transform stable machine output where supported |
| `--include-meta` | Include request metadata where supported |
| `--quiet` | Suppress non-essential output |
| `--no-color`, `--no-ansi`, `--no-pager` | Control terminal presentation |

## Output Formats

### Text Tables (default)

Commands print formatted tables by default:

```
$ asana project list
NAME                 GID
CLI Test Project     1212591887790722
My First Project     1212606992980859
```

### JSON (`--json`)

Use `--json` for scripting and automation:

```json
$ asana project list --json
[
  {"gid": "1212591887790722", "name": "CLI Test Project"},
  {"gid": "1212606992980859", "name": "My First Project"}
]
```

JSON uses the command's stable result model. The same model powers the advanced
script output modes:

```bash
# Headerless, tab-separated rows in the human table's core column order
asana project list --workspace-gid 123 --plain

# jq produces one deterministic JSON value per match
asana task list --project-gid 456 --jq '.[].gid'

# Go templates receive the same value as --json
asana task view 789 --template '{{.name}}'
```

Plain fields escape literal backslashes, tabs, carriage returns, and newlines as
`\\`, `\t`, `\r`, and `\n`, keeping each resource on one physical TSV row.
Invalid jq and template expressions fail before an API request is made. The
machine modes are mutually exclusive with `--plain`.

### Request metadata (`--include-meta`)

API-backed commands can wrap machine output in a `{data, meta}` envelope:

```bash
asana workspace list --include-meta --json
asana workspace list --include-meta --jq '.meta.request_id'
```

Metadata is opt-in and contains only safe request diagnostics such as endpoint
path, pagination state, request IDs, retry attempts, and allowlisted rate-limit
headers. Use it only with `--json`, `--jq`, or `--template`; it cannot be
combined with `--plain` or `--dry-run`.

## Dry Run Mode

Write commands support `--dry-run` to preview changes without applying them:

```
$ asana task create "New task" --project-gid 1212591887790722 --dry-run
DRY RUN: would POST /tasks
  body:
    name = New task
    projects = ["1212591887790722"]
  no changes were made
```

Add `--json` to receive the stable preview model. Dry runs may perform safe
lookup requests to validate names and relationships, but never send the
mutation. On read commands, `--dry-run` does not alter output or prompt behavior.

## Destructive Commands

Delete and remove commands prompt before sending a mutation. Automation,
non-interactive use, machine output, and `--no-prompt` require an explicit
`--confirm`. Use `--dry-run` instead when you only want the request preview:

```bash
asana task delete 1234567890123456 --dry-run
asana task delete 1234567890123456 --confirm
```

## Safe Update Checks

Check release availability without changing the executable:

```bash
asana update --check
asana update --check --json
```

## Name Resolution

Most commands accept names instead of GIDs. The CLI automatically resolves names:

```
$ asana task list --project "My Project"
$ asana task update 123 --assignee "jane@example.com"
$ asana section create "Done" --project "My Project"
```

If multiple resources match a name, you'll be prompted to choose unless
`--no-prompt` is set.
