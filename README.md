# DeltaOps

DeltaOps is a small host monitor that sends alerts and recovery notices through Delta Chat.

It is meant for small servers where you want one simple binary, a private chat recipient, and actionable messages when basic host-health checks cross a threshold.

## What It Does

- Runs as a single `deltaops` binary.
- Uses Delta Chat as the notification channel.
- Sets up a Delta Chat bot account from a chatmail provider URL or `dcaccount:` setup URL.
- Pairs alerts to the first operator who sends the startup pairing code.
- Sends pairing/startup status reports, threshold alerts, and recovery notices.
- Monitors Linux disk usage, inode usage, memory pressure, and 1-minute load.

Linux is the main deployment target. macOS release builds are available for local development and transport testing with filesystem metrics only.

## Install

Download a release from GitHub:

https://github.com/lambadalambda/deltaops/releases

Release targets:

- `linux-amd64`
- `linux-arm64`
- `darwin-amd64`
- `darwin-arm64`

Example for Linux amd64:

```sh
version=v0.1.0
target=linux-amd64

curl -LO "https://github.com/lambadalambda/deltaops/releases/download/${version}/deltaops-${version}-${target}.tar.gz"
curl -LO "https://github.com/lambadalambda/deltaops/releases/download/${version}/deltaops-${version}-${target}.tar.gz.sha256"
sha256sum -c "deltaops-${version}-${target}.tar.gz.sha256"
tar -xzf "deltaops-${version}-${target}.tar.gz"
sudo install -m 0755 deltaops /usr/local/bin/deltaops
```

## First Run

For a quick manual run, create a private state directory in your home directory:

```sh
install -d -m 0700 "$HOME/.local/state/deltaops"
```

Start DeltaOps with a chatmail provider URL:

```sh
deltaops run --state-dir "$HOME/.local/state/deltaops" --dcaccount-url https://nine.testrun.org/
```

On startup, DeltaOps prints the bot contact details and a one-time pairing code. Send that code to the bot from the Delta Chat account that should receive alerts.

After pairing, DeltaOps persists that recipient and sends a status report. On later restarts it sends a startup status report before normal alert polling.

## Configuration

You can provide the setup URL by flag, environment variable, or config file:

```sh
DELTAOPS_DCACCOUNT_URL=https://nine.testrun.org/ deltaops run --state-dir "$HOME/.local/state/deltaops"
```

```yaml
delta_chat:
  dcaccount_url: https://nine.testrun.org/
```

Full `dcaccount:` setup URLs should be treated as secrets. Prefer a private config file or service environment file with mode `0600` on shared hosts.

## Day-To-Day Use

- Run DeltaOps under a service manager such as systemd.
- Keep the state directory private and backed up to encrypted storage only.
- Reset the alert recipient by stopping DeltaOps, deleting `<state>/binding.json`, and starting it again.
- Do not delete the whole state directory unless you also want to reprovision the Delta Chat bot account.

See the [operations guide](docs/operations.md) for service setup, state paths, reset steps, backups, and default alert thresholds.

## Development

This repository uses `mise` to pin the Go toolchain:

```sh
mise install
mise exec -- go test ./...
```

Build the CLI locally:

```sh
mise exec -- go build -o bin/deltaops ./cmd/deltaops
```

Local live-transport builds need a prepared Delta Chat RPC helper asset before `deltaops run` can start the real transport. See the [development guide](docs/development.md).

## More Docs

- [Operations guide](docs/operations.md)
- [Architecture notes](docs/architecture.md)
- [Development guide](docs/development.md)
- [Docker E2E harness](test/e2e/README.md)
- [Local issue plan](meta/issues.md)
