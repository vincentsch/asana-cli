# Configuration

## Config File Location

The CLI stores configuration at:
- Linux: `$XDG_CONFIG_HOME/asana/config.json` (usually `~/.config/asana/config.json`)
- macOS: `~/Library/Application Support/asana/config.json`
- Windows: `%AppData%\asana\config.json`

Override with `--config /path/to/config.json` or `ASANA_CONFIG`.

## Authentication

### Interactive Login

```bash
asana login
```

Prompts for a Personal Access Token, validates it with Asana, and saves it.

### Manual Token Setup

Get a Personal Access Token from https://app.asana.com/0/my-apps.

Prefer `asana login` for local setup. Passing a token as a command argument may
leave it in your shell history.

```bash
asana config set token YOUR_TOKEN
```

Or edit the config file directly:

```json
{
  "version": 1,
  "token": "0/abc123...",
  "defaults": {
    "workspace_gid": "",
    "project_gid": ""
  }
}
```

### Environment Variable

Set `ASANA_TOKEN` to override the config file:

```bash
export ASANA_TOKEN="0/abc123..."
asana task list
```

## Default Workspace

Set a default workspace for commands that need one:

```bash
# By name
asana config set workspace "My Workspace"
```

The resolved GID is stored in the config file.

## View Current Config

```bash
asana config list
```

## Config Options

| Key | Description |
|-----|-------------|
| `token` | Asana Personal Access Token |
| `defaults.workspace_gid` | Default workspace GID |
| `defaults.project_gid` | Default project GID |

## Example Config File

```json
{
  "version": 1,
  "token": "0/1234567890abcdef...",
  "defaults": {
    "workspace_gid": "123456789012345",
    "project_gid": "987654321098765"
  }
}
```

## Troubleshooting

### "unauthorized" errors

Your token may be expired or invalid. Run `asana login` again.

### "workspace required" errors

Set a default workspace:

```bash
asana config set workspace "My Workspace"
```

Or provide it per-command:

```bash
asana task list -w "My Workspace"
```
