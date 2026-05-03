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

## Resolution

- Alert evaluation is implemented in `internal/alert` with an injected `Clock` interface.
- Defaults are `30m` cooldown, disk warning/critical `85/95`, inode warning/critical `85/95`, memory warning/critical `80/90`, and load warning/critical `1/2`.
- Custom thresholds can override any metric or individual threshold field while unmentioned metrics and fields keep defaults.
- Active alert state suppresses repeats before cooldown, repeats after cooldown, escalates warning to critical immediately, and emits recovery when samples return to normal.
- Critical/warning flapping is suppressed until cooldown unless the last notified severity is exceeded.
- Evaluator state is mutex-protected; the runtime still needs to queue or retry emitted notification decisions when transport delivery fails.
- Alert and recovery messages include host, check, target, observed value, threshold, severity, and state.
- Tests cover normal no-op, first alert, repeat suppression, cooldown repeat, escalation, flapping suppression, recovery, custom thresholds, and default/custom threshold merging.
