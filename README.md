# asana-cli

`asana-cli` is an unofficial Asana CLI for terminals, scripts, and CI, built on
[rungrad](https://github.com/vincentsch/rungrad). Commands print text tables for
people, stable `--json` for programs, `--dry-run` previews before supported
changes, confirmation before destructive actions, and predictable errors for
non-interactive use.

Use it to inspect and manage Asana workspaces, projects, sections, tasks, users,
teams, tags, attachments, custom fields, goals, and portfolios from the shell.

`asana-cli` is not affiliated with, endorsed by, or sponsored by Asana, Inc.

## What It Provides

- Text tables by default and stable `--json` for automation.
- Copy-safe `--plain`, `--jq`, and Go-template output modes through
  [rungrad](https://github.com/vincentsch/rungrad).
- `--dry-run` previews for supported write commands before any mutation happens.
- Confirmation gates for destructive delete/remove commands.
- Name-to-GID resolution with deterministic non-interactive errors.
- A hidden `__rungrad_manifest` command and conformance-tested CLI behavior.
- Self-update checks with `asana update --check`.

## Install

macOS and Linux users can install the latest checksummed release binary with:

```bash
curl -fsSL https://raw.githubusercontent.com/vincentsch/asana-cli/main/install.sh | bash
```

Or build the current release with Go 1.22.2 or newer:

```bash
go install github.com/vincentsch/asana-cli/cmd/asana@v0.1.2
```

Release binaries are named `asana-<os>-<arch>` and are published for Linux,
macOS, and Windows on amd64 and arm64.

## Authenticate

Create a Personal Access Token in Asana:

```text
https://app.asana.com/0/my-apps
```

Then store it locally:

```bash
asana login
```

For scripts or temporary sessions, `ASANA_TOKEN` overrides the saved config:

```bash
export ASANA_TOKEN="0/your-token"
asana workspace list
```

The config file stays in the platform config directory under
`asana/config.json`, for example `~/.config/asana/config.json` on Linux.

## Common Commands

```bash
# Workspaces, users, and teams
asana workspace list
asana user me
asana team list --workspace "Company"

# Projects and sections
asana project list --workspace "Company"
asana project view "Launch Plan"
asana section list --project "Launch Plan"

# Tasks
asana task list --project "Launch Plan"
asana task view 1234567890123456
asana task create "Review launch checklist" --project "Launch Plan" --assignee me
asana task done 1234567890123456

# Preview writes and require explicit destructive confirmation
asana task update 1234567890123456 --name "Updated title" --dry-run
asana task delete 1234567890123456 --dry-run
asana task delete 1234567890123456 --confirm
```

Every command has examples and related commands in `--help`:

```bash
asana --help
asana task --help
asana task create --help
```

## Automation

Use `--json` for stable machine output:

```bash
asana task list --project "Launch Plan" --json
```

The same result model powers advanced output modes:

```bash
asana project list --workspace-gid 123 --plain
asana task list --project-gid 456 --jq '.[].gid'
asana task view 789 --template '{{.name}}'
```

API-backed commands can include safe request metadata:

```bash
asana workspace list --include-meta --json
```

Metadata is opt-in and contains diagnostics such as endpoint path, request IDs,
pagination state, retry attempts, and allowlisted rate-limit headers. It does not
include credentials.

## Command Reference

- [Complete command index](manual/commands/index.md)
- [Authentication](manual/commands/auth.md)
- [Configuration](manual/configuration.md)
- [Installation](manual/installation.md)
- [Development](manual/development.md)

Generated command pages live under [`manual/commands/`](manual/commands/).

## Development

```bash
go test ./...
go vet ./...
make docs-check
make conformance
```

`make conformance` builds and scores the CLI against `rungrad-spec/1` with local
mock fixtures. It does not need a live Asana token and does not make mutation
requests.

## Status

The CLI is pre-1.0. Command names and JSON shapes are intended to be stable, but
there may still be compatibility changes before a stable release.

## License

MIT
