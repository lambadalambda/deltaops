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
