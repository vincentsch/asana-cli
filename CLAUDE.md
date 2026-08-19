# asana-cli

CLI for Asana inspired by GitHub's `gh`. Go + Cobra. Deterministic output for humans and automation.

## Planning

Public contributor-facing workflow lives in `README.md`, `CONTRIBUTING.md`, and
`manual/development.md`. Local private planning and release notes live under
`.vroni/wip/` when that directory is present; `.vroni/` is intentionally local
state and must not be committed to the public repo.

## Code Style

- **File comments**: Each `.go` file should have a brief package/file comment explaining its purpose
- **Inline comments**: Add comments for non-obvious logic, complex conditionals, and API quirks—not for self-explanatory code
- Keep comments maintainable: explain *why*, not *what*

## rungrad

- This CLI is built on `github.com/vincentsch/rungrad`.
- Keep rungrad pinned to a public tagged version in `go.mod`.
- Do not add a local filesystem `replace` for rungrad in committed files.
