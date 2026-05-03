# Prove Delta Chat integration for a single Go binary

## Summary

Validate the Delta Chat integration approach before building the monitor around it. The key risk is combining automatic account setup, Go, and a portable single-binary distribution.

## Requirements

- Identify the Delta Chat account creation flow DeltaOps will support first.
- Prove whether the selected Delta Chat library or protocol path can be used from Go without violating the single-binary goal.
- Evaluate `deltachat-rpc-server` as a subprocess, cgo or Rust embedding, embedded platform-specific helper binaries, and any credible pure-Go alternative.
- Define how the bot account credentials and contact data are created, stored, and displayed.
- Keep the monitoring core decoupled from the concrete Delta Chat implementation behind a small interface.
- Capture known platform limitations, especially if cgo, native libraries, or an external helper process are required.
- Document cross-compilation constraints for each viable option.

## Acceptance Criteria

- A small spike proves account creation or documents the exact blocker with a recommended alternative.
- Tests cover the Delta Chat adapter contract using a fake implementation.
- The README states the supported account setup path and any platform constraints.
- A decision is recorded for whether the MVP remains a single binary, uses cgo, or must change scope.
- If a sidecar or embedded helper is required, the decision record states whether that still satisfies the user-facing single-binary requirement.

## Notes

- Do this before implementing monitor checks.
- If automatic registration depends on a specific provider class, document that as an MVP constraint instead of hiding it behind generic wording.

## Resolution

- Decision recorded in `../decisions/0001-delta-chat-integration.md`.
- MVP path is `github.com/chatmail/rpc-client-go/v2` with a managed `deltachat-rpc-server` subprocess.
- One-file operator distribution remains possible by embedding the platform-specific RPC server helper into each release binary, but the implementation is not pure Go.
- The MVP account provisioning path is explicit operator input with a chatmail `dcaccount:` URL; provider-neutral email account auto-registration is out of scope.
- End-to-end account configuration was not exercised because this spike did not have a usable `dcaccount:` provisioning URL; the next account provisioning issue should test `AddTransportFromQr` with a real or hermetic provisioning input.
- The Delta Chat adapter contract is represented by `internal/notify.Transport` and covered with a fake-backed test.
