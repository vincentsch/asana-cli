# Installation

## Go Install

Requires Go 1.22.2 or newer:

```bash
go install github.com/vincentsch/asana-cli/cmd/asana@latest
```

Verify the binary is on your `PATH`:

```bash
asana version
```

Expected shape:

```text
asana vX.Y.Z
```

## Release Installer

When GitHub release assets are available, macOS and Linux users can install the
latest checksummed binary with:

```bash
curl -fsSL https://raw.githubusercontent.com/vincentsch/asana-cli/main/install.sh | bash
```

The script detects your platform, downloads the matching release asset, and
verifies it against the release checksums. It installs `asana` to
`/usr/local/bin` unless `INSTALL_DIR` is set:

```bash
curl -fsSL https://raw.githubusercontent.com/vincentsch/asana-cli/main/install.sh |
  INSTALL_DIR="$HOME/.local/bin" bash
```

## Manual Download

Download pre-built binaries from the
[releases page](https://github.com/vincentsch/asana-cli/releases):

| Platform | Architecture | Asset |
| --- | --- | --- |
| macOS | Apple Silicon | `asana-darwin-arm64` |
| macOS | Intel | `asana-darwin-amd64` |
| Linux | x64 | `asana-linux-amd64` |
| Linux | ARM64 | `asana-linux-arm64` |
| Windows | x64 | `asana-windows-amd64.exe` |
| Windows | ARM64 | `asana-windows-arm64.exe` |

On macOS or Linux:

```bash
chmod +x asana-*
sudo mv asana-* /usr/local/bin/asana
```

On Windows, rename the downloaded `.exe` to `asana.exe` if desired and place it
on your `PATH`.

## Authentication

Create an Asana Personal Access Token:

```text
https://app.asana.com/0/my-apps
```

Then run:

```bash
asana login
```

You can also use an environment token:

```bash
export ASANA_TOKEN="0/your-token"
asana workspace list
```

## Updates

Check release availability without changing the installed executable:

```bash
asana update --check
asana update --check --json
```

Run `asana update` only when you want the CLI to download and install an
available release. Automatic self-replacement is not supported on Windows; use
the manual download path there.

## Uninstall

If installed with the release script:

```bash
sudo rm /usr/local/bin/asana
rm -rf ~/.config/asana
```

If installed with Go:

```bash
rm "$(go env GOPATH)/bin/asana"
rm -rf ~/.config/asana
```
