# MomentumTicker CLI

Public normal-user command line client for MomentumTicker Folios, reports, research runs, scheduled report emails, and shared Folio workflows.

This repository intentionally contains only the normal-user API client. Owner/debug Super Admin operations stay in the private `theinventor/kevins-model` Rails repo under `cli/mmsa` and require `MOMENTUMTICKER_SUPER_ADMIN_TOKEN`.

## Install

Install the latest tagged Go module:

```sh
go install github.com/theinventor/momentumticker-cli/cmd/momentum@latest
```

That installs the `momentum` binary. If you install the repository root instead, Go will name the binary `momentumticker-cli`; use the `cmd/momentum` path above for the intended command name.

Build from source with `go build -o momentum ./cmd/momentum`.

Release builds publish Goreleaser assets named like `momentum_0.1.0_linux_amd64.tar.gz`, `momentum_0.1.0_darwin_arm64.tar.gz`, `momentum_0.1.0_windows_amd64.zip`, plus `checksums.txt`.

## Authenticate

Create a MomentumTicker API token from the app settings page, then save it:

```sh
momentum auth save \
  --profile dev \
  --base http://127.0.0.1:3007 \
  --token "$MOMENTUMTICKER_TOKEN"
```

Resolution order for normal commands:

1. `--profile NAME`
2. `MOMENTUMTICKER_TOKEN` and `MOMENTUMTICKER_URL`
3. the saved default profile in `$XDG_CONFIG_HOME/momentumticker/config.json`

New profiles use the OS keychain when available. Use `--storage=file` for headless servers or CI. File-backed profiles are written with mode `0600`.

Useful auth commands:

```sh
momentum auth status
momentum auth list
momentum auth use dev
momentum auth logout dev
momentum auth migrate
momentum whoami
momentum doctor
```

`auth status`, `whoami`, and `doctor` only print token fingerprints, never raw token values.

## Commands

Folios:

```sh
momentum folio list
momentum folio show 42
momentum folio create --name "AI Research" --entries "NVDA 2\nAMD 3"
momentum folio update 42 --name "Updated Folio"
momentum folio holdings get 42
momentum folio holdings set 42 < entries.txt
```

Sharing:

```sh
momentum folio share list
momentum folio share invite 42 --email friend@example.com --permission read
momentum folio share accept 7
momentum folio share decline 7
momentum folio share revoke 7
```

Indexes and reports:

```sh
momentum index list
momentum index show sp500
momentum report
momentum report --folio master
momentum report --folio 42
```

Research:

```sh
momentum research run --name Semis --tickers NVDA,AMD,AVGO,TSM
momentum research run --name Watch < tickers.txt
momentum research list
momentum research show 1 --sort overall
```

Schedules and email:

```sh
momentum schedule get
momentum schedule set --email kevin@example.com --enabled --frequency weekdays --hour 7
momentum email kevin@example.com
```

Compatibility:

```sh
momentum watchlist get
momentum watchlist set < entries.txt
```

`watchlist` is a deprecated Master Folio compatibility alias.

Most read commands support `--json` for structured output:

```sh
momentum --json folio list
momentum --json report --folio sp500
```

## Updates

The CLI includes MonsterMailbox-style self-update support:

```sh
momentum update --check
momentum update --check --no-cache
momentum update
momentum update --to v0.1.0
```

Update checks use the GitHub Releases API and cache the latest release snapshot for 24 hours at `$XDG_CONFIG_HOME/momentumticker/update-check.json`. Installs verify the platform archive against `checksums.txt`, extract the `momentum` binary, and atomically replace the running executable. If the executable path is not writable, the command prints a clear writable-path error.

`--to` is accepted for a tag that matches the latest release. Installing older tags is intentionally not implemented yet because it needs a separate GitHub release-by-tag lookup.

## Development

Run the full local checks:

```sh
go mod tidy
go test ./...
go vet ./...
go build -o momentum ./cmd/momentum
./momentum --help
./momentum update --help
```

Release config validation, when Goreleaser is installed:

```sh
goreleaser check
```

## Security Boundary

This CLI uses normal MomentumTicker API tokens and the `/api/v1/*` user API. It does not include owner/debug commands, Super Admin tokens, or Super Admin endpoints. Normal tokens can only operate on Folios the user owns or has accepted shared access to; write operations remain server-enforced by MomentumTicker permissions.
