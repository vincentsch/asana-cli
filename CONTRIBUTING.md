# Contributing

`asana-cli` is pre-1.0. Command names and JSON shapes are intended to be stable,
but compatibility may still change before a stable release.

## Local Checks

Run these before sending a change:

```bash
gofmt -l .
git diff --check
go vet ./...
go build ./...
go test ./...
go test -race ./...
make docs-check
make conformance
```

Keep generated command docs and help goldens in sync when the command surface
changes:

```bash
go run ./cmd/generate-command-docs
go test ./internal/cmd -run TestCommandHelpGoldensAreInSync -update
```

## Style

Prefer small changes with tests that exercise the public CLI surface. Keep docs
plain and direct. Do not commit credentials, local Asana config, local workflow
state, generated caches, or files that depend on a personal machine.

## Asana API Behavior

This project is an unofficial Asana client. Keep API-specific behavior local to
this repository. Shared CLI mechanics such as output modes, redaction,
confirmation, manifests, docs gates, and conformance scoring come from
[`rungrad`](https://github.com/vincentsch/rungrad).
