# DeltaOps

DeltaOps is planned as a small Go system monitor that sends alerts through Delta Chat. The target deployment model is one portable `deltaops` binary that can be copied onto a host, run with minimal setup, and paired by messaging the bot from the operator's Delta Chat account.

Status: Delta Chat integration, account provisioning, state layout, pairing logic, MVP metric-source decisions, the CLI entrypoint, live Delta Chat transport wiring, and macOS development support are in place. Release binaries still require prepared embedded RPC helper assets before they can run the live transport.

## Goals

- Build one portable Go binary for low-setup host monitoring.
- Use Delta Chat as the only notification transport.
- Create, register, or configure the bot's Delta Chat account on first run with as little operator input as the selected provider flow allows.
- Print the bot contact data and a one-time pairing code during setup.
- Bind alerts to the first contact that sends the valid pairing code, then persist that binding.
- Alert on practical host-health failures such as disk exhaustion, high memory pressure, and other checks selected for the MVP.
- Prefer safe defaults, clear recovery messages, and low alert noise.

## Quickstart

This is the intended operator flow for a Linux release build that embeds the matching `deltachat-rpc-server` helper. The current source tree does not commit helper binaries, so development builds validate startup and then exit with a next action until a release asset is prepared.

1. Install the `deltaops` binary on the monitored Linux host, for example at `/usr/local/bin/deltaops`.
2. Create a private state directory, for example `/var/lib/deltaops`, owned by the service user and mode `0700`.
3. Provide either a chatmail provider URL such as `https://nine.testrun.org/` or a full `dcaccount:` setup URL with `--dcaccount-url`, `DELTAOPS_DCACCOUNT_URL`, or `delta_chat.dcaccount_url` in the config file.
4. Start `deltaops run --state-dir /var/lib/deltaops`.
5. Read the startup output for the bot contact data and one-time pairing code.
6. Send the pairing code to the bot from the Delta Chat account that should receive alerts.
7. Leave the process running under a service manager; runtime lifecycle logs are structured JSON when the packaged runtime is active, while startup and operator errors are plain text.

Minimal config file shape:

```yaml
delta_chat:
  dcaccount_url: https://nine.testrun.org/
```

Avoid passing full `dcaccount:` setup URLs directly in shell history on shared hosts. Prefer a private config file or service environment file with mode `0600`.

## Delta Chat Integration

- DeltaOps will use `github.com/chatmail/rpc-client-go/v2` and a managed `deltachat-rpc-server` subprocess for the MVP path.
- Release builds should keep the operator experience to one copied `deltaops` file by embedding the matching platform-specific RPC server helper and extracting it at runtime.
- This is not a pure-Go binary internally. Each supported OS and architecture needs a matching Delta Chat RPC server asset.
- The MVP-supported account setup path is explicit operator input via either a chatmail provider URL such as `https://nine.testrun.org/` or a full chatmail `dcaccount:` setup URL. Provider-neutral arbitrary email account registration is out of scope.
- The `internal/notify/dcrpc` package opens the embedded helper with `DC_ACCOUNTS_PATH` set to `<state>/deltachat-accounts`, creates or reuses one Delta Chat account, configures bot mode from the normalized account setup URL when needed, receives pairing messages, and sends alert text to the persisted contact ID.
- Default tests use fakes for the RPC boundary. Live provider-dependent tests must stay behind explicit integration controls.
- The full decision is recorded in `meta/decisions/0001-delta-chat-integration.md`.

## Account Provisioning

The MVP setup input is either a chatmail provider URL or a full chatmail `dcaccount:` setup URL. DeltaOps will accept it from these sources, in order:

1. `--dcaccount-url https://nine.testrun.org/` or `--dcaccount-url dcaccount:...`.
2. `DELTAOPS_DCACCOUNT_URL=https://nine.testrun.org/` or `DELTAOPS_DCACCOUNT_URL=dcaccount:...`.
3. `delta_chat.dcaccount_url` in the config file.

Provider homepage URLs are normalized to the Delta Chat account setup link before transport setup. For example, `https://nine.testrun.org/` becomes `DCACCOUNT:https://nine.testrun.org/new`, matching the account link published on that provider homepage. If no setup input is provided, startup should fail with a next action telling the operator to provide one of those inputs. Existing IMAP/SMTP credentials and OAuth setup are deferred, not fallback behavior for the MVP.

The MVP treats the provisioning input as startup input. Keep it available for restarts until the live Delta Chat account setup path proves that the input can be safely removed after first configuration.

After account setup, DeltaOps should print the bot Delta Chat contact or secure-join URI, the bot email address if available, and the local pairing code. It should not print the consumed account setup URL.

The provisioning decision is recorded in `meta/decisions/0002-account-provisioning.md`.

## Config And State

DeltaOps should run without a config file when required setup input is supplied by flag or environment.

Default paths, using absolute XDG locations only:

1. Config file: `$XDG_CONFIG_HOME/deltaops/config.yaml`, or `$HOME/.config/deltaops/config.yaml` when `XDG_CONFIG_HOME` is unset.
2. State directory: `$XDG_STATE_HOME/deltaops`, or `$HOME/.local/state/deltaops` when `XDG_STATE_HOME` is unset.
3. Delta Chat accounts: `<state>/deltachat-accounts`.
4. Extracted Delta Chat RPC helper: `<state>/deltachat-rpc-helper`.
5. Bound contact: `<state>/binding.json`.

The CLI supports `--config` and `--state-dir` overrides. State directories are created with `0700` permissions and sensitive files with `0600` permissions on POSIX-style filesystems where supported.

MVP warning: local Delta Chat state is accepted as plaintext on disk behind filesystem permissions. That state can include account credentials, message databases, and contact binding data. Protect and back up the state directory accordingly.

The state layout decision is recorded in `meta/decisions/0003-config-and-state-layout.md`.

## Pairing And Reset

When no contact is bound, DeltaOps should show a one-time setup code alongside the bot contact data. The first inbound message containing that setup code becomes the alert recipient. Messages without the setup code are ignored while unbound, and messages from later contacts cannot replace an existing binding.

The binding is stored at `<state>/binding.json`. The MVP reset path is deliberate local access: stop DeltaOps, delete `<state>/binding.json`, then restart with a new setup code. Future CLI wiring can expose the same reset operation as a command.

## Platform And Metrics

The MVP deployment collector target is Linux. macOS is also supported as a local development mode so the live Delta Chat product path can be exercised on a developer workstation.

Default Linux metrics planned for the MVP:

1. `disk.used_percent`: `100 * (Blocks - Bfree) / Blocks` from `statfs`; unavailable when `Blocks == 0`.
2. `disk.inodes_used_percent`: `100 * (Files - Ffree) / Files` from `statfs`; unavailable when `Files == 0`.
3. `memory.pressure_percent`: `100 * (MemTotal - MemAvailable) / MemTotal` from `/proc/meminfo`; unavailable when `MemTotal == 0` or `MemAvailable` is missing.
4. `load.1m`: first field of `/proc/loadavg`; unavailable when the file is missing or unparsable.

CPU utilization is deferred for the MVP; load average is the initial CPU-pressure signal. Linux release builds must include matching embedded Delta Chat RPC server assets for their target architecture.

macOS development mode currently collects only filesystem capacity metrics via `statfs`:

1. `disk.used_percent`.
2. `disk.inodes_used_percent`.

macOS does not collect Linux `/proc` memory pressure or load metrics yet. It is intended for validating packaging, provisioning, pairing, transport startup, and runtime behavior from this workstation, not as the first production deployment target.

The `internal/collector` package implements Linux samples behind fakeable filesystem and `/proc` interfaces, plus macOS filesystem samples behind the same filesystem boundary. Unavailable metric sources return clear unavailable errors instead of panics.

The platform and metric-source decision is recorded in `meta/decisions/0004-mvp-platform-and-metrics.md`.

## Alert Defaults

The `internal/alert` package evaluates samples with an injected clock, active alert state, recovery decisions, and a default `30m` repeat cooldown. The evaluator is safe for concurrent calls, but the runtime must still queue or retry emitted notification decisions when delivery fails.

Default warning and critical thresholds:

1. `disk.used_percent`: warning `85`, critical `95`.
2. `disk.inodes_used_percent`: warning `85`, critical `95`.
3. `memory.pressure_percent`: warning `80`, critical `90`.
4. `load.1m`: warning `1`, critical `2`. These are conservative absolute-load defaults and should be configured for larger multi-core hosts.

Alert and recovery messages include host, check, target, observed value, threshold, severity, and state.

## Runtime Loop

The `internal/runtime` package wires account readiness, pairing, collection, alert evaluation, and notification delivery behind interfaces for deterministic tests.

Startup order:

1. Wait for account or transport readiness with bounded backoff.
2. Use an existing bound contact, or wait for pairing when unbound.
3. Send a status report to the bound contact. First-time pairing reports use `reason=paired`; restarts with an existing binding use `reason=startup`.
4. Run metric collection and alert evaluation on the configured polling interval.
5. Send non-noop alert and recovery decisions to the bound contact.

Status reports include host, reason, metric names, targets, and observed values for all samples collected at that moment. They are separate from threshold alerts and do not suppress later alert or recovery decisions.

Defaults are a `1m` polling interval, `1s` initial backoff, and `1m` maximum backoff. Negative durations are rejected. `NewOSSignalSource` adapts `SIGINT` and `SIGTERM` into the runtime signal source on Unix-like systems so shutdown cancels the loop cleanly.

## Logging And Delivery Failures

Runtime logging is structured JSON when `NewJSONLogger` is used. Logged lifecycle events include startup, account readiness, pairing, status report delivery, alert decisions, notification delivery failures, retries, queue-limit failures, sent notifications, and shutdown.

Log fields with names that look like secrets, setup codes, provisioning URLs, message text, message bodies, errors, or causes are redacted. Runtime alert-decision logs include only safe metadata such as metric, target, kind, and severity, not raw message contents or bound contact IDs.

Notification delivery uses bounded retries for both status reports and alert/recovery notifications. Defaults are `3` notification attempts and at most `32` pending notification decisions per polling iteration. Account-readiness checks after a send failure are also bounded by the remaining delivery retry budget. If delivery is exhausted or the pending notification bound is exceeded, the runtime returns an operator-facing error with a next action and leaves useful local logs for diagnosis.

Heartbeat messages are deferred for the MVP. The first version sends pairing/startup status reports plus threshold-based alerts and recoveries.

## CLI And Packaging

Commands:

1. `deltaops` or `deltaops run`: start with safe defaults, validate startup inputs, and prepare state.
2. `deltaops version`: print version metadata.
3. `deltaops run --help`: print supported startup flags.

Startup accepts a chatmail provider URL or full `dcaccount:` setup URL from `--dcaccount-url`, `DELTAOPS_DCACCOUNT_URL`, or `delta_chat.dcaccount_url` in the config file. The CLI also supports `--config` and `--state-dir` overrides. The config reader intentionally supports only the current minimal key shape needed by the MVP, not arbitrary YAML configuration.

Build the current CLI binary with:

```sh
mise exec -- go build -o bin/deltaops ./cmd/deltaops
```

The source tree does not commit upstream `deltachat-rpc-server` binaries. On Linux, a valid development `run` prepares the state layout and then exits with a clear next action explaining that a release must be built with the matching Delta Chat RPC helper asset. This preserves the one-file operator target while keeping the missing runtime dependency explicit.

The MVP deployment platform is Linux. macOS `run` is supported for development with filesystem-only metrics. Other operating systems can compile developer commands such as `version`, but `run` rejects unsupported operating systems before collector startup. Cross-compilation requires more than setting `GOOS` and `GOARCH`: each supported architecture release must embed the corresponding upstream `deltachat-rpc-server` artifact built for that target.

For `github.com/chatmail/rpc-client-go/v2` v2.49.0, the currently recognized helper asset names and checksums are:

1. `linux/amd64`: `deltachat-rpc-server-x86_64-linux`, SHA-256 `28e10b40518f55fa8ce20edd119fa743dd29a22df372b58443ec53eb753cb50c`.
2. `linux/arm64`: `deltachat-rpc-server-aarch64-linux`, SHA-256 `33acdc048060fcd51bc585f2eefdaa2cf93cca9306440f45be8c5936024732cf`.
3. `linux/386`: `deltachat-rpc-server-i686-linux`, SHA-256 `6fe6831f0bcd84316dafa416883249aba623eb392b7795769d7b9f635dc069b6`.
4. `darwin/arm64`: `deltachat-rpc-server-aarch64-macos`, SHA-256 `3ea30551ddaa67c2691c1cfbf0087ad95b799c5192269aada232ca2569891789`.
5. `darwin/amd64`: `deltachat-rpc-server-x86_64-macos`, SHA-256 `a8885769dc24eacd605b32593332de138fc77d97550b709c330d4fd4479b48c9`.

Prepare selected helpers before building a release:

```sh
sh scripts/prepare-dcrpc-assets.sh linux/amd64
```

Prepare one helper target per release build so each `deltaops` binary embeds only the helper it needs. Use `sh scripts/prepare-dcrpc-assets.sh all` only for local cache/testing because a subsequent build embeds every prepared helper. The script downloads from `https://github.com/chatmail/core/releases/tag/v2.49.0`, verifies SHA-256 checksums, and places helpers under `internal/notify/dcrpc/assets/`. Embedded helper bytes are also checksum-validated before extraction. The helper is embedded into `deltaops`, extracted at runtime to `<state>/deltachat-rpc-helper` with executable `0700` permissions, and launched as a managed subprocess. Helper binaries are ignored by git because they are large upstream artifacts.

For local macOS testing on Apple Silicon, prepare the Darwin helper before building:

```sh
sh scripts/prepare-dcrpc-assets.sh darwin/arm64
mise exec -- go build -o bin/deltaops ./cmd/deltaops
```

Use `darwin/amd64` on Intel macOS. Without the prepared helper, `bin/deltaops run --dcaccount-url https://nine.testrun.org/` should reach the normal helper-packaging error instead of rejecting macOS as unsupported.

## Linux Service Example

Use a dedicated service user and a private environment file rather than putting secrets in the unit. Example `/etc/deltaops/deltaops.env`:

```sh
DELTAOPS_DCACCOUNT_URL=https://nine.testrun.org/
```

Restrict it with mode `0600`. Example systemd unit:

```ini
[Unit]
Description=DeltaOps host monitor
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=deltaops
Group=deltaops
EnvironmentFile=/etc/deltaops/deltaops.env
ExecStart=/usr/local/bin/deltaops run --state-dir /var/lib/deltaops
Restart=on-failure
RestartSec=30s
StateDirectory=deltaops
StateDirectoryMode=0700
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/lib/deltaops

[Install]
WantedBy=multi-user.target
```

`SIGINT` and `SIGTERM` cancel the runtime loop cleanly. Runtime signal-source tests cover this cancellation path; an end-to-end systemd stop check should be repeated once the packaged Delta Chat runtime is available.

## Operations

Reset pairing by stopping DeltaOps, deleting `<state>/binding.json`, and starting DeltaOps again with a fresh setup code. Do not delete the whole state directory for a contact reset unless the Delta Chat account should also be reprovisioned.

Back up the state directory only to encrypted storage. Stop DeltaOps before file-level backups so Delta Chat account databases and binding files are consistent. Restore the directory with the original owner and restrictive permissions before restarting the service.

Rotate the Delta Chat bot account by stopping DeltaOps, backing up or deleting the existing state directory, provisioning with a new provider or `dcaccount:` URL, and pairing the new bot contact. Rotate the alert recipient by deleting only `binding.json` and pairing a new contact.

Log rotation is handled by the service manager when stdout and stderr go to journald. If logs are redirected to files, use normal Linux log rotation and restrict file permissions because alert metadata can reveal operational state.

## Non-Goals

- Replacing full observability stacks such as Prometheus, Grafana, or agent-based SaaS platforms.
- Supporting many notification transports.
- Managing multiple operators in the first version.
- Shipping a daemon supervisor.

## Intended UX

```sh
deltaops
```

On first run, DeltaOps should:

1. Create or load its local state directory.
2. Configure a Delta Chat account using a supplied chatmail provider or `dcaccount:` URL from flag, environment, or config.
3. Print contact data and a one-time pairing code for the bot.
4. Wait for the first incoming message that contains the pairing code.
5. Persist that sender as the alert recipient.
6. Send a pairing confirmation status report with the currently collected metrics.
7. Start sending alerts and recovery notices for configured checks.

After binding, later messages from other contacts should not receive host alerts unless the local reset flow is used. On restarts with an existing binding, DeltaOps sends a startup status report before normal polling.

## Security Considerations

- Local state may contain Delta Chat account credentials, message databases, and the bound contact. It must be stored outside the repository with restrictive permissions.
- The MVP accepts plaintext local state protected by filesystem permissions; OS keyring integration is deferred.
- Treat full `dcaccount:` URLs as credentials. Do not commit them, place them in world-readable unit files, or pass them through shared shell history. Provider homepage URLs are usually public, but provider URLs with tokens or generated setup data are sensitive.
- The pairing code prevents a random first sender from taking over alert delivery during setup. Anyone who can read startup output during the unbound window can pair the monitor.
- Status reports and alert messages can reveal hostnames, resource pressure, filesystem targets, thresholds, and recovery state. The monitor should send them only to the persisted bound contact.
- Runtime logs redact fields that look like provisioning URLs, setup codes, message text, message bodies, errors, or causes. Normal lifecycle logs keep safe metadata such as metric, target, kind, and severity.
- If Delta Chat delivery is unavailable, DeltaOps logs locally, retries with backoff, and avoids unbounded queues.
- Heartbeat messages are deferred for the MVP to avoid ongoing notification noise before real alert behavior is proven.

## Development

This repository uses `mise` to pin the Go toolchain.

```sh
mise install
mise exec -- go test ./...
```

If `mise` reports that `.mise.toml` is not trusted, review the file and run:

```sh
mise trust .mise.toml
```

Build command:

```sh
mise exec -- go build -o bin/deltaops ./cmd/deltaops
```

Opt-in Docker smoke test for the embedded Linux RPC helper path:

```sh
sh scripts/e2e-docker.sh smoke
```

Live Docker runs require a real `DELTAOPS_DCACCOUNT_URL` and are documented in `test/e2e/README.md`.

## Planning

The local issue plan lives in `meta/issues.md`, with issue details in `meta/issues/`.

The first implementation steps recorded the Delta Chat integration path, single-binary constraint, account provisioning flow, config/state layout, pairing-code contact binding, MVP metric-source decision, metric collection, alert-state evaluation, runtime loop, structured transport-failure logging, CLI packaging behavior, operation/security documentation, and live Delta Chat transport wiring.
