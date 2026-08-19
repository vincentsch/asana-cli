# Release Checklist

Use this checklist before publishing a public source release.

## Prepare

1. Update public-facing files when needed: `README.md`, `manual/`,
   `SECURITY.md`, `CONTRIBUTING.md`, and this checklist.
2. Confirm `go.mod` uses a public tagged `rungrad` version.
3. Run:

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

4. Build the release matrix:

```bash
make build-all VERSION=vX.Y.Z
```

5. Scan the tracked tree:

```bash
git ls-files | rg '(^|/)(\.env|\.netrc|credentials|token|secret)'
git grep -n -i -E 'token|secret'
```

Review every hit. Auth, config, and redaction tests may be legitimate; docs,
scripts, and unexpected files need closer review.

## Tag

Use annotated tags:

```bash
git tag -a vX.Y.Z -m "Release vX.Y.Z"
git push origin vX.Y.Z
```

Go module tags are effectively immutable once the module proxy and checksum
database observe them. Never move or recreate a public tag. Publish a new patch
tag instead.

## Verify

Use a clean environment with public module settings and an empty module cache:

```bash
go install github.com/vincentsch/asana-cli/cmd/asana@vX.Y.Z
asana version
asana update --check
```

Then verify the release assets and checksums on GitHub.
