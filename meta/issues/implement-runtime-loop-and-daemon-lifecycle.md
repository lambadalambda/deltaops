# Implement runtime loop and daemon lifecycle

## Summary

Connect account setup, binding, metric collection, threshold evaluation, and notification delivery into a long-running process.

## Requirements

- Define startup sequencing from config and state load through account readiness, binding, and monitoring.
- Use a configurable polling interval with safe defaults.
- Handle `SIGINT` and `SIGTERM` with graceful shutdown.
- Reconnect or restart transport sessions when they fail, using bounded backoff.
- Keep the run loop testable with fake collectors, clocks, signal sources, and notifiers.

## Acceptance Criteria

- Tests cover startup with existing binding and startup waiting for pairing.
- Tests cover one or more polling iterations without sleeping.
- Tests cover graceful shutdown.
- Tests cover transport reconnect or backoff behavior with fakes.

## Notes

- Depends on config/state, binding, metrics, alert-state, and Delta Chat transport decisions.

## Resolution

- Runtime orchestration is implemented in `internal/runtime` behind fakeable account, pairer, collector, evaluator, notifier, sleeper, and signal-source interfaces.
- Startup waits for account readiness with bounded backoff, then uses an existing binding or waits for pairing before monitoring.
- Defaults are `1m` polling interval, `1s` initial backoff, and `1m` max backoff.
- Negative polling and backoff durations are rejected during runner construction.
- The loop collects metrics, evaluates alert decisions, and sends non-noop decisions to the bound contact.
- Notification failures trigger account readiness checks and bounded retry backoff until success or shutdown.
- Context cancellation and signal-source cancellation exit gracefully.
- `NewOSSignalSource` adapts `SIGINT` and `SIGTERM` on Unix-like systems; Windows currently adapts `os.Interrupt`.
- Tests cover existing binding startup, waiting for pairing, multiple polling iterations without sleeping, signal shutdown, readiness backoff, notification retry backoff, and invalid duration validation.
