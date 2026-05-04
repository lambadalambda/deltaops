# Development Guide

This repository is a Go project and uses `mise` to pin the toolchain.

## Setup

```sh
mise install
mise exec -- go test ./...
```

If `mise` reports that `.mise.toml` is not trusted, review the file and run:

```sh
mise trust .mise.toml
```

## Build

Build the CLI:

```sh
mise exec -- go build -o bin/deltaops ./cmd/deltaops
```

Run version output:

```sh
bin/deltaops version
```

`deltaops` and `deltaops run` start the monitor. `deltaops run --help` prints supported startup flags.

## Delta Chat RPC Helper Assets

The source tree does not commit upstream `deltachat-rpc-server` binaries. Official release builds prepare and embed one matching helper per target.

Local live-transport builds need the matching helper prepared before building. Without it, `deltaops run` prepares startup state and then exits with a clear helper-packaging error.

Prepare one helper target:

```sh
sh scripts/prepare-dcrpc-assets.sh linux/amd64
mise exec -- go build -o bin/deltaops ./cmd/deltaops
```

Prepare Apple Silicon macOS helper:

```sh
sh scripts/prepare-dcrpc-assets.sh darwin/arm64
mise exec -- go build -o bin/deltaops ./cmd/deltaops
```

Use `darwin/amd64` on Intel macOS.

Prepare one helper target per release build so each binary embeds only the helper it needs. `sh scripts/prepare-dcrpc-assets.sh all` is useful for local cache/testing, but a subsequent build embeds every prepared helper.

For `github.com/chatmail/rpc-client-go/v2` v2.49.0, recognized helper assets are:

| Target | Asset | SHA-256 |
| --- | --- | --- |
| `linux/amd64` | `deltachat-rpc-server-x86_64-linux` | `28e10b40518f55fa8ce20edd119fa743dd29a22df372b58443ec53eb753cb50c` |
| `linux/arm64` | `deltachat-rpc-server-aarch64-linux` | `33acdc048060fcd51bc585f2eefdaa2cf93cca9306440f45be8c5936024732cf` |
| `linux/386` | `deltachat-rpc-server-i686-linux` | `6fe6831f0bcd84316dafa416883249aba623eb392b7795769d7b9f635dc069b6` |
| `darwin/arm64` | `deltachat-rpc-server-aarch64-macos` | `3ea30551ddaa67c2691c1cfbf0087ad95b799c5192269aada232ca2569891789` |
| `darwin/amd64` | `deltachat-rpc-server-x86_64-macos` | `a8885769dc24eacd605b32593332de138fc77d97550b709c330d4fd4479b48c9` |

The helper preparation script downloads from `https://github.com/chatmail/core/releases/tag/v2.49.0`, verifies SHA-256 checksums, and places helpers under `internal/notify/dcrpc/assets/`. Embedded helper bytes are checksum-validated again before runtime extraction.

## Release Workflow

The GitHub Actions release workflow:

- Runs `go test ./...` first.
- Builds `linux/amd64`, `linux/arm64`, `darwin/amd64`, and `darwin/arm64`.
- Prepares exactly one Delta Chat RPC helper per target before building.
- Uploads branch/manual build artifacts.
- Publishes GitHub Release assets and `.sha256` files when a `v*` tag is pushed.

Current public repository:

https://github.com/lambadalambda/deltaops

## Docker E2E Harness

Run the opt-in Docker smoke test for the embedded Linux RPC helper path:

```sh
sh scripts/e2e-docker.sh smoke
```

Live Docker runs require a real provider or `dcaccount:` value in `DELTAOPS_DCACCOUNT_URL`. See [test/e2e/README.md](../test/e2e/README.md).

## Tests

Run the default hermetic suite:

```sh
mise exec -- go test ./...
```

Keep live Delta Chat or provider-dependent tests out of the default test path unless they are behind explicit integration controls.

## Planning

Repository-local issues live in [meta/issues.md](../meta/issues.md). Completed issue history lives in [meta/issues_archive.md](../meta/issues_archive.md).

Issue detail files and decisions are intentionally kept under `meta/` rather than the user-facing README.
