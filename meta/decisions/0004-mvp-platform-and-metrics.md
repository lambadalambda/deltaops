# 0004 MVP platform and metric sources

## Status

Accepted for MVP planning.

## Context

DeltaOps needs deterministic host metrics for the first monitor release. The Delta Chat integration also requires matching platform-specific RPC server assets, so broad platform support should wait until the packaging path is proven per target.

## Decision

The MVP collector target is Linux only.

Default metric definitions:

- `disk.used_percent`: `100 * (Blocks - Bfree) / Blocks` from `statfs`; unavailable when `Blocks == 0`.
- `disk.inodes_used_percent`: `100 * (Files - Ffree) / Files` from `statfs`; unavailable when `Files == 0`.
- `memory.pressure_percent`: `100 * (MemTotal - MemAvailable) / MemTotal` from `/proc/meminfo`; unavailable when `MemTotal == 0` or `MemAvailable` is missing.
- `load.1m`: first field of `/proc/loadavg`; unavailable when the file is missing or unparsable.

CPU utilization is deferred. The MVP uses load average instead because it is cheap to collect, stable, and directly available from `/proc/loadavg`. Threshold semantics can normalize or contextualize load against CPU count in the alerting issue if needed.

Collector plan creation fails at runtime with a clear error on unsupported operating systems before collector startup. The MVP should not promise macOS, Windows, or BSD collection until their metric sources and Delta Chat helper packaging are selected.

## Consequences

- Collector implementation can use Linux `statfs`, `/proc/meminfo`, and `/proc/loadavg` sources.
- Tests should use parser inputs and fakes rather than depending on the host running the test suite.
- Linux release builds need matching embedded `deltachat-rpc-server` assets for supported architectures.
- Cross-compiling the Go code is not enough; packaging must also include the matching Delta Chat RPC helper for each Linux target.
