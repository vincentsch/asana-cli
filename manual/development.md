# Development

## Prerequisites

- Go 1.22.2 or newer
- Git
- `make`

## Clone And Build

```bash
git clone https://github.com/vincentsch/asana-cli.git
cd asana-cli
go build -o asana ./cmd/asana
```

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

`make conformance` builds a scored binary, routes it to local mock Asana and
release fixtures, and verifies the rungrad behavior contract without live
credentials or live mutations.

## Project Structure

```text
asana-cli/
├── cmd/asana/           # CLI entry point
├── cmd/generate-command-docs/
├── internal/
│   ├── api/             # Asana REST client
│   ├── cmd/             # Asana commands and rungrad adapters
│   ├── config/          # Product-owned JSON config and rungrad projection
│   ├── conformance/     # Built-binary rungrad scoring fixtures
│   ├── interactive/     # huh-based prompts and token entry
│   ├── manualdocs/      # Generated-manual layout adapter
│   └── resolve/         # Asana scoping over rungrad name resolution
├── manual/              # User documentation and generated command reference
└── go.mod
```

The executable lifecycle, global flags, output selection, transforms,
redaction, destructive confirmation, manifest, generated-doc gates, update
check model, test harness, and conformance scorer come from
[`github.com/vincentsch/rungrad`](https://github.com/vincentsch/rungrad).
Asana-specific REST behavior, JSON config, prompts, identifier semantics, and
domain commands stay in this repository.

## Adding A Command

1. Add the command implementation under `internal/cmd/`.
2. Add the top-level command to `asanaCommandTemplates` when it introduces a new
   command family. Nested commands are discovered automatically.
3. Add the path to the command contract maps only when it has exceptional
   behavior such as mutation, destructive confirmation, or unauthenticated
   execution.
4. Return one stable result model and route output through the existing rungrad
   output adapter.
5. Add focused command tests.
6. Regenerate generated docs and help goldens when the command surface changes:

```bash
go run ./cmd/generate-command-docs
go test ./internal/cmd -run TestCommandHelpGoldensAreInSync -update
```

## Style

- Keep changes small and covered by tests that exercise the public CLI surface.
- Add a brief package or file comment to each `.go` file.
- Comment API quirks and non-obvious control flow, not self-explanatory code.
- Keep docs plain, direct, and accurate about available distribution channels.
- Do not commit credentials, local config, generated caches, or machine-specific
  workflow state.

## Mocked API Testing

Unit and conformance tests use local mock servers. For manual QA against a local
mock API, set:

```bash
ASANA_API_BASE_URL=http://127.0.0.1:8080 go run ./cmd/asana workspace list
```

`ASANA_API_BASE_URL` is for tests and QA only. Normal runtime defaults to
`https://app.asana.com/api/1.0`.

## Releases

Releases are tag-triggered through GitHub Actions. See [Release checklist](../RELEASE.md).
