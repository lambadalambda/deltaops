# Evaluate thresholds and alert state

## Summary

Turn metric samples into alert, recovery, and no-op decisions with sane defaults and low noise.

## Requirements

- Define default warning and critical thresholds for disk and memory.
- Support configurable thresholds.
- Track active alert state to avoid repeated notifications every polling interval.
- Send recovery notifications when a condition returns to normal.
- Include enough context in every alert for the operator to act.
- Use an injected clock for cooldowns and recovery transitions.

## Acceptance Criteria

- Tests cover normal, alert, repeated alert suppression, threshold escalation, and recovery transitions.
- Alert messages include host, check name, observed value, threshold, and severity.
- Defaults are documented.
- Cooldown behavior is deterministic in tests.
- A clock interface is used for all time-dependent behavior.

## Notes

- Inject time instead of sleeping in tests.
