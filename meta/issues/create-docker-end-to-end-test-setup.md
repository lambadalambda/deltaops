# Create Docker end-to-end test setup

## Summary

Create a reproducible Docker-based test environment that can exercise DeltaOps with Delta Chat RPC integration and realistic runtime behavior.

## Requirements

- Provide a Docker or Compose setup that builds DeltaOps with the embedded RPC helper path.
- Exercise account provisioning, contact binding, metric evaluation, alert sending, cooldown behavior, and recovery notifications where practical.
- Keep the setup opt-in so default `go test ./...` remains hermetic and fast.
- Avoid committing real account credentials, binding state, or provider secrets.
- Document required external inputs if a fully hermetic Delta Chat provider is not practical yet.

## Acceptance Criteria

- A documented command starts the end-to-end environment from a clean checkout.
- The setup can reproduce at least one alert and one recovery notification through the live Delta Chat transport or a documented local equivalent.
- The setup stores all sensitive state in disposable Docker volumes or temporary paths.
- Failure modes produce actionable logs for provisioning, binding, transport startup, and notification delivery.
- The README or a dedicated test document explains how to run and reset the environment.

## Notes

- Prefer a hermetic setup if practical. If a real chatmail provider is required, make that explicit and keep it out of default CI.
