# Security

Please report security issues privately.

Use GitHub private vulnerability reporting if it is enabled for the repository.
You can also email `vschmalbach@vschmalbach.com`.

Do not open a public issue for a vulnerability until a fix is available.

`asana-cli` stores Asana Personal Access Tokens in the platform config directory
under `asana/config.json` unless you use `ASANA_TOKEN`. Do not include actual
tokens in bug reports, test fixtures, screenshots, or command output.

`asana-cli` is pre-1.0. Security fixes may ship as patch releases without
preserving bug-compatible behavior.
