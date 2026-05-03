# Implement structured logging and transport failure handling

## Summary

Make DeltaOps diagnosable when alerts cannot be sent and prevent notification failures from crashing or exhausting the monitor.

## Requirements

- Log startup, pairing, alert-state transitions, transport failures, and shutdown without leaking secrets or message contents.
- Define retry and backoff behavior for failed Delta Chat sends.
- Bound any in-memory alert queue.
- Decide whether the MVP sends periodic heartbeat messages after binding.
- Ensure local logs remain useful when Delta Chat is unavailable.

## Acceptance Criteria

- Tests cover failed sends, bounded retries, and queue limits using fake transports.
- Logs redact account secrets, setup codes, and raw message bodies.
- Operator-facing errors include clear next actions.
- Heartbeat behavior is documented if implemented, or explicitly deferred.

## Notes

- Depends on the transport adapter interface and runtime-loop design.

## Resolution

- Structured logging is implemented in `internal/runtime` through a fakeable `Logger` interface and `NewJSONLogger`.
- Runtime logs startup, account readiness, pairing, alert decisions, notification failures, retry attempts, notification sends, queue-limit failures, and shutdown.
- Log redaction covers field names associated with secrets, setup codes, provisioning URLs, message text, message bodies, errors, and causes.
- Runtime alert-decision logs contain safe metadata only: metric, target, decision kind, and severity; bound contact IDs are not logged.
- Notification delivery is bounded by `MaxNotifyAttempts`, defaulting to `3`, with exponential backoff capped by `MaxBackoff`; account-readiness checks after send failures are bounded by the remaining delivery retry budget.
- Per-poll pending notification decisions are bounded by `MaxPendingNotifications`, defaulting to `32`.
- Delivery exhaustion and pending notification overflow return `OperatorError` values with clear next actions.
- Heartbeat messages are explicitly deferred for the MVP to avoid notification noise before real alert behavior is proven.
- Tests cover failed sends, bounded retries, bounded readiness checks after send failures, queue limits, lifecycle log events, runtime failure-path redaction, and invalid retry/queue limits using fakes.
