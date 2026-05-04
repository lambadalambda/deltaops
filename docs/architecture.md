# Architecture Notes

DeltaOps keeps the monitor core separate from Delta Chat transport details so alert behavior can be tested without a live provider.

## Delivery Shape

The operator-facing shape is one copied `deltaops` binary. Internally, live Delta Chat support uses a platform-specific `deltachat-rpc-server` helper embedded into that binary and extracted at runtime.

This means DeltaOps is not pure Go internally. Each supported release target needs a matching upstream RPC helper asset.

## Delta Chat Transport

The live transport uses `github.com/chatmail/rpc-client-go/v2` and a managed `deltachat-rpc-server` subprocess.

The `internal/notify/dcrpc` package:

- Selects the embedded helper for the current OS and architecture.
- Validates helper bytes against the expected SHA-256 checksum.
- Extracts the helper under `<state>/deltachat-rpc-helper` with executable `0700` permissions.
- Runs the helper with `DC_ACCOUNTS_PATH` set to `<state>/deltachat-accounts`.
- Creates or reuses one Delta Chat account.
- Receives pairing messages and sends status, alert, and recovery text.

Default tests use fakes at the RPC boundary. Live provider-dependent tests stay out of the default test path.

## Runtime Loop

The `internal/runtime` package wires account readiness, pairing, status reports, collection, alert evaluation, and notification delivery behind interfaces.

Startup order:

1. Wait for account or transport readiness with bounded backoff.
2. Use an existing bound contact, or wait for pairing when unbound.
3. Send a `reason=paired` or `reason=startup` status report.
4. Run metric collection and alert evaluation on the polling interval.
5. Send non-noop alert and recovery decisions to the bound contact.

Defaults are a `1m` polling interval, `1s` initial backoff, and `1m` maximum backoff. Negative durations are rejected.

## Metrics

Linux is the MVP deployment collector target.

Linux metrics:

| Metric | Source |
| --- | --- |
| `disk.used_percent` | `statfs` |
| `disk.inodes_used_percent` | `statfs` |
| `memory.pressure_percent` | `/proc/meminfo` |
| `load.1m` | `/proc/loadavg` |

CPU utilization is deferred. The initial CPU-pressure signal is 1-minute load average.

macOS development mode currently collects filesystem capacity metrics only:

- `disk.used_percent`
- `disk.inodes_used_percent`

macOS does not collect Linux `/proc` memory pressure or load metrics. It is intended for packaging, provisioning, pairing, transport startup, and runtime testing from a developer workstation.

Metric sources return clear unavailable errors instead of panics when source data is missing or unparsable.

## Alert Evaluation

The `internal/alert` package evaluates metric samples with an injected clock, active alert state, recovery decisions, and a default `30m` repeat cooldown.

Alerting behavior is independent from the Delta Chat transport. The runtime is responsible for retrying or surfacing notification delivery failures.

## Logging And Delivery Failures

Runtime logging is structured JSON when the packaged runtime is active. Logged lifecycle events include startup, account readiness, pairing, status report delivery, alert decisions, notification delivery failures, retries, queue-limit failures, sent notifications, and shutdown.

Fields whose names look like secrets, setup codes, provisioning URLs, message text, message bodies, errors, or causes are redacted. Alert-decision logs keep safe metadata such as metric, target, kind, and severity, not raw message contents or bound contact IDs.

Notification delivery uses bounded retries for status reports and alert/recovery notifications. Defaults are `3` notification attempts and at most `32` pending notification decisions per polling iteration. If delivery is exhausted or the pending notification bound is exceeded, the runtime returns an operator-facing error with a next action.

Heartbeat messages are deferred for now. The first version sends pairing/startup status reports plus threshold-based alerts and recoveries.

## Decisions

Recorded decisions live under `meta/decisions/`:

- [Delta Chat integration](../meta/decisions/0001-delta-chat-integration.md)
- [Account provisioning](../meta/decisions/0002-account-provisioning.md)
- [Config and state layout](../meta/decisions/0003-config-and-state-layout.md)
- [MVP platform and metrics](../meta/decisions/0004-mvp-platform-and-metrics.md)

The open work plan lives in [meta/issues.md](../meta/issues.md).
