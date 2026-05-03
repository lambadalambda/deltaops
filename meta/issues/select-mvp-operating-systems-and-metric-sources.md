# Select MVP operating systems and metric sources

## Summary

Choose the initial operating system targets and the metric data sources before implementing collectors.

## Requirements

- Decide whether the MVP is Linux-only or supports additional platforms.
- Compare practical metric sources such as `/proc`, platform syscalls, and Go libraries.
- Decide whether CPU load or utilization belongs in the first metric set alongside disk and memory.
- Document how cgo requirements from Delta Chat affect platform support and cross-compilation.
- Prefer deterministic collector inputs that can be tested without relying on the host running the tests.

## Acceptance Criteria

- The README states MVP operating system support.
- The collector issue references the chosen sources and metric definitions.
- The decision documents disk capacity, inode, memory pressure, and CPU or load-average scope.
- Unsupported platforms fail with clear build or runtime errors.

## Notes

- Depends on the Delta Chat integration spike if that spike constrains build targets.

## Resolution

- Decision recorded in `../decisions/0004-mvp-platform-and-metrics.md`.
- MVP collector target is Linux only.
- Unsupported operating systems fail at runtime through `collector.NewPlan` and `collector.ValidatePlatform` with a clear Linux-only error before collector startup.
- Default metric definitions are recorded in `internal/collector`:
- `disk.used_percent`: `100 * (Blocks - Bfree) / Blocks` from `statfs`; unavailable when `Blocks == 0`.
- `disk.inodes_used_percent`: `100 * (Files - Ffree) / Files` from `statfs`; unavailable when `Files == 0`.
- `memory.pressure_percent`: `100 * (MemTotal - MemAvailable) / MemTotal` from `/proc/meminfo`; unavailable when `MemTotal == 0` or `MemAvailable` is missing.
- `load.1m`: first field of `/proc/loadavg`; unavailable when the file is missing or unparsable.
- CPU utilization is deferred; load average is the MVP CPU-pressure signal.
- Cross-compilation must include the matching Linux Delta Chat RPC server helper, not just the Go binary.
