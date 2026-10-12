# Changelog

## Unreleased

- Added troubleshooting for PATH, environment token overrides, ambiguous
  project names and premium task search. Corrected the manual's claim that
  ambiguous names open a selection prompt.
- Scheduled credential-free CI checks on Monday, Wednesday and Friday, with
  Dependabot checks for Go modules and GitHub Actions.
- Prepared the repository for a fresh public source release.
- Updated the shared CLI foundation to
  `github.com/vincentsch/rungrad v0.2.2`.
- Corrected installation documentation to describe only supported install paths.
- The release installer now prints "Could not determine latest version" when the
  release lookup fails, instead of exiting without a message.
- Invalid or expired Asana tokens now exit with code 3 (missing or invalid
  credentials), as the rungrad exit-code spec defines, instead of code 1.

## 2026-08-09

- Migrated the CLI to the native rungrad lifecycle, output pipeline, redaction
  boundary, generated command catalog, generated manual, update/check adapter,
  and conformance harness.
- Added deterministic hidden manifest coverage for 119 visible command paths.
- Added destructive confirmation gates and preserved dry-run validation before
  mutation previews.
