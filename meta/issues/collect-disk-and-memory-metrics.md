# Collect disk, memory, and load metrics

## Summary

Collect the initial system health metrics needed to detect disk exhaustion, high memory pressure, and any selected CPU or load condition.

## Requirements

- Collect disk usage for configured filesystems or mount points.
- Decide whether inode exhaustion is included for supported filesystems.
- Collect memory usage in a way that reflects practical pressure, not just raw allocation.
- Collect CPU utilization or load average if selected for the MVP metric set.
- Follow the selected MVP operating systems and metric sources.
- Keep metric collection behind interfaces for deterministic tests.
- Handle unavailable metrics with clear errors instead of panics.

## Acceptance Criteria

- Tests cover parsing or collecting representative disk, memory, and selected load samples.
- Tests cover unavailable metric sources.
- The default check set includes disk and memory, plus load if selected for MVP.
- The README documents the supported operating systems for the MVP.

## Notes

- Avoid broad platform promises until the metric source is selected and tested.
- Depends on the MVP operating-system and metric-source decision.
