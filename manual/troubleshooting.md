# Troubleshooting

## The shell cannot find asana

Check whether the executable is on your PATH:

```bash
command -v asana
```

The release installer uses `/usr/local/bin` by default. If you used a custom
directory, add that directory to your shell's PATH. For example:

```bash
export PATH="$HOME/.local/bin:$PATH"
asana version
```

Put the export in your shell configuration to keep it for new terminals.
For Go installations, the executable is in `GOBIN`, or `$(go env GOPATH)/bin`
when `GOBIN` is unset. See [Installation](installation.md) for Windows setup.

## A saved token is not being used

`ASANA_TOKEN` takes precedence over the token saved by `asana login`. An old
environment variable can therefore override a newer login. In a Bash or Zsh
session, remove that override and try a read command:

```bash
unset ASANA_TOKEN
asana workspace list
```

Keep the token out of bug reports, terminal screenshots and shared logs.

## More than one project has the same name

The CLI reports matching candidates instead of guessing. Restrict the lookup
to a workspace, or use the project's GID:

```bash
asana task list --workspace-gid 123 --project "Launch Plan" --json
asana task list --project-gid 456 --json
```

Replace `123` and `456` with real IDs from your account. Use either `--project`
or `--project-gid` in a command, not both.

## Task search is unavailable on your plan

`asana task search` uses Asana's premium search API. To read a known project's
tasks without that search endpoint, use:

```bash
asana task list --project-gid 456 --json
```

Your token still needs access to the project. The CLI cannot grant additional
permissions or bypass Asana plan limits.

## Report a problem

Include the output of `asana version`, your OS, the command with credentials
and private data removed, and the error text. Describe what you expected and
what happened. [Open an issue](https://github.com/vincentsch/asana-cli/issues).
