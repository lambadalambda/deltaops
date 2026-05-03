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
