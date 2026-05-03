# Implement live Delta Chat transport

## Summary

Wire `deltaops run` to a real Delta Chat transport backed by `deltachat-rpc-server`, while preserving hermetic default tests and the one-file release target.

## Requirements

- Add a live transport implementation behind the existing `internal/notify.Transport` boundary.
- Manage the Delta Chat RPC helper as an explicit packaged runtime dependency.
- Provision the bot account from a validated chatmail `dcaccount:` URL without logging the URL.
- Adapt the transport to runtime account readiness, pairing receive, and alert send interfaces.
- Keep default tests hermetic; live Delta Chat/provider tests must require an explicit integration path or build tag.
- Update CLI `run` so supported release builds start the runtime loop instead of always returning the RPC-helper packaging error.
- Keep clear next-action errors when the helper is unavailable or unsupported for the current platform.

## Acceptance Criteria

- `mise exec -- go test ./...` passes.
- `mise exec -- go build -o bin/deltaops ./cmd/deltaops` produces the binary.
- `deltaops run` on unsupported platforms still fails before provisioning with a clear next action.
- `deltaops run` on Linux with no packaged helper fails with a clear next action and does not leak the `dcaccount:` URL.
- Unit tests cover helper extraction/selection, transport adapter behavior, provisioning redaction, and CLI runtime wiring with fakes.
- README documents the current live transport behavior and how release builds must include the matching RPC helper asset.

## Notes

- Do not commit upstream helper binaries in this slice unless a reviewed release-asset policy is added first.
- Use fakes for default tests. Any real Delta Chat account setup should be behind explicit integration controls.
