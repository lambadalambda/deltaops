# Collect disk, memory, and load metrics

## Summary

Collect the initial system health metrics needed to detect disk exhaustion, high memory pressure, and any selected CPU or load condition.

## Requirements

- Collect `disk.used_percent` for configured Linux filesystems or mount points as `100 * (Blocks - Bfree) / Blocks` using `statfs`; return unavailable when `Blocks == 0`.
- Collect `disk.inodes_used_percent` for configured Linux filesystems or mount points as `100 * (Files - Ffree) / Files` using `statfs`; return unavailable when `Files == 0`.
- Collect `memory.pressure_percent` from Linux `/proc/meminfo` as `100 * (MemTotal - MemAvailable) / MemTotal`; return unavailable when `MemTotal == 0` or `MemAvailable` is missing.
- Collect `load.1m` from the first field of Linux `/proc/loadavg`; return unavailable when the file is missing or unparsable.
- Reject unsupported operating systems with the selected platform validation error.
- Keep metric collection behind interfaces for deterministic tests.
- Handle unavailable metrics with clear errors instead of panics.

## Acceptance Criteria

- Tests cover parsing or collecting representative Linux disk, inode, memory, and load samples.
- Tests cover unavailable metric sources.
- The default check set includes disk and memory, plus load if selected for MVP.
- The README documents the supported operating systems for the MVP.

## Notes

- Avoid broad platform promises until the metric source is selected and tested.
- Depends on the MVP operating-system and metric-source decision.
