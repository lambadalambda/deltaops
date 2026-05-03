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
